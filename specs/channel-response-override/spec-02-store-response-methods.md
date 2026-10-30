# spec-02-store-response-rules

## Goal

Replace the channel-level response methods with a per-path rules map so that each
(channelSlug, method, path) triple can be configured independently.

## Context

The previous implementation stored one `ResponseStatus`/`ResponseBody` directly on
`Channel` and exposed `GetChannelResponse`/`UpdateChannelResponse`. This means every
path on a channel shares the same response — too coarse. The new model stores rules in
a dedicated map keyed by `(slug, method, path)`, decoupled from the channel record.

`store/store.go` is the only file that changes in this spec.

### What to remove

- `ResponseStatus: 200` and `ResponseBody: []byte("{}")` defaults from the `&Channel{}`
  literals in both `CreateChannel` and `GetOrCreateChannel`.
- `GetChannelResponse` method.
- `UpdateChannelResponse` method.

### What to add

An unexported struct for the rule value and a `rules` map on `Store`:

```go
type responseRule struct {
    status int
    body   []byte
}
```

Add `rules map[string]*responseRule` to the `Store` struct and initialise it in `New`:

```go
func New(cap int) *Store {
    return &Store{
        channels: make(map[string]*Channel),
        requests: make([]*Request, 0),
        rules:    make(map[string]*responseRule),
        cap:      cap,
    }
}
```

The map key is constructed as `slug + "\x00" + method + "\x00" + path` — the null
byte is chosen as a separator because it cannot appear in valid HTTP method names or
URL paths, so it prevents collisions.

## Steps

1. **`store/store.go`** — Add `responseRule` struct (unexported) and `rules` field.

2. **`store/store.go`** — Add `GetResponseRule`:
   ```go
   // GetResponseRule returns the configured response for the given channel slug,
   // HTTP method, and path. Falls back to (200, {}) when no rule has been set.
   func (s *Store) GetResponseRule(slug, method, path string) (status int, body []byte) {
       s.mu.RLock()
       defer s.mu.RUnlock()
       key := slug + "\x00" + method + "\x00" + path
       if r, ok := s.rules[key]; ok {
           bodyCopy := make([]byte, len(r.body))
           copy(bodyCopy, r.body)
           return r.status, bodyCopy
       }
       return 200, []byte("{}")
   }
   ```

3. **`store/store.go`** — Add `SetResponseRule`:
   ```go
   // SetResponseRule sets the response status and body for the given channel slug,
   // HTTP method, and path. Returns an error if the channel does not exist.
   func (s *Store) SetResponseRule(slug, method, path string, status int, body []byte) error {
       s.mu.Lock()
       defer s.mu.Unlock()
       if _, ok := s.channels[slug]; !ok {
           return errors.New("channel not found")
       }
       key := slug + "\x00" + method + "\x00" + path
       s.rules[key] = &responseRule{status: status, body: body}
       return nil
   }
   ```

4. **`store/store.go`** — Remove `GetChannelResponse`, `UpdateChannelResponse`, and
   the `ResponseStatus`/`ResponseBody` defaults from `CreateChannel` and
   `GetOrCreateChannel`.

## Acceptance criteria

- `go build ./...` passes.
- `GetResponseRule` returns `(200, []byte("{}"))` for an unconfigured key.
- After `SetResponseRule(slug, "GET", "/api/stores", 404, []byte(`{"error":"not found"}`))`,
  a subsequent `GetResponseRule(slug, "GET", "/api/stores")` returns `(404, ...)`.
- `SetResponseRule` on a non-existent slug returns a non-nil error.
- Rules for different method+path combinations within the same channel are independent.

## Dependencies

- spec-01 (Channel struct must not have ResponseStatus/ResponseBody fields).
