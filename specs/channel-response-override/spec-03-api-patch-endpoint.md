# spec-03-api-request-response-endpoint

## Goal

Remove the channel-level PATCH endpoint and the response fields from `channelJSON`.
Add `response_status` and `response_body` to `requestJSON` and expose a new
`PATCH /_/api/channels/{slug}/requests/{id}` endpoint so callers can configure the
response for a specific request's method+path.

## Context

`handler/api.go` currently has `ResponseStatus`/`ResponseBody` on `channelJSON` and a
`PATCH /_/api/channels/{slug}` handler — both must be removed. The right surface is the
request object: each captured request knows its method and path, which together form
the rule key. Patching a request means "set the rule for this request's method+path".

All existing requests for that channel+method+path combination will receive the updated
response on their next capture, including ones received before the rule was set.

`store.Store` must expose `SetResponseRule` and `GetResponseRule` (provided by spec-02)
before this spec can compile.

## Out of scope

- Changes to `store/` (specs 01–02).
- Changes to `handler/capture.go` (spec-04).
- UI changes (spec-05).

## Steps

1. **`handler/api.go`** — Revert `channelJSON` to its original shape:
   ```go
   type channelJSON struct {
       Slug      string    `json:"slug"`
       CreatedAt time.Time `json:"created_at"`
   }
   ```
   Revert `toChannelJSON` to return only `Slug` and `CreatedAt`.

2. **`handler/api.go`** — Extend `requestJSON` with two new fields:
   ```go
   type requestJSON struct {
       // ... all existing fields unchanged ...
       ResponseStatus int             `json:"response_status"`
       ResponseBody   json.RawMessage `json:"response_body"`
   }
   ```

3. **`handler/api.go`** — Update `toRequestJSON` to accept the rule values:
   ```go
   func toRequestJSON(r *store.Request, respStatus int, respBody []byte) requestJSON {
       headers := make(map[string][]string, len(r.Headers))
       for k, v := range r.Headers {
           headers[k] = v
       }
       return requestJSON{
           ID:             r.ID,
           ChannelSlug:    r.ChannelSlug,
           ReceivedAt:     r.ReceivedAt,
           Method:         r.Method,
           Path:           r.Path,
           Query:          r.Query,
           Headers:        headers,
           Body:           r.Body,
           ContentType:    r.ContentType,
           BodySize:       r.BodySize,
           ResponseStatus: respStatus,
           ResponseBody:   json.RawMessage(respBody),
       }
   }
   ```

   Update every call site of `toRequestJSON` to supply the rule:
   - In `GET /_/api/channels/{slug}/requests`: for each `req`, call
     `st.GetResponseRule(slug, req.Method, req.Path)` and pass the result.
   - In `GET /_/api/channels/{slug}/requests/{id}`: same pattern.

4. **`handler/api.go`** — Remove `PATCH /_/api/channels/{slug}` and its handler.

5. **`handler/api.go`** — Register `PATCH /_/api/channels/{slug}/requests/{id}`:
   ```
   "PATCH /_/api/channels/{slug}/requests/{id}"
   ```
   Handler logic:
   - Look up the request with `st.GetRequest(slug, id)`; return 404 if not found.
   - Decode request body:
     ```go
     var body struct {
         ResponseStatus *int            `json:"response_status"`
         ResponseBody   json.RawMessage `json:"response_body"`
     }
     ```
   - Return 400 `"response_status is required"` if pointer is nil.
   - Return 400 `"response_body is required"` if `ResponseBody` is nil.
   - Return 400 `"response_status must be between 100 and 599"` if out of range.
   - Return 400 `"response_body must be valid JSON"` if `!json.Valid(ResponseBody)`.
   - Call `st.SetResponseRule(slug, req.Method, req.Path, *body.ResponseStatus, []byte(body.ResponseBody))`.
   - If store returns error, return 404.
   - Re-fetch the rule and return `toRequestJSON(req, status, body)` with 200 OK.

## Acceptance criteria

- `GET /_/api/channels/{slug}/requests` includes `response_status` and `response_body`
  on each request object; unconfigured requests show `200` and `{}`.
- `GET /_/api/channels/{slug}/requests/{id}` includes the same fields.
- `GET /_/api/channels` and `GET /_/api/channels/{slug}` no longer include
  `response_status` or `response_body`.
- `PATCH /_/api/channels/{slug}` no longer exists (returns 405 or 404).
- `PATCH /_/api/channels/{slug}/requests/{id}` with valid body returns 200 with
  updated `response_status` and `response_body`.
- Patching request A (path `/api/stores`) and then fetching request B (same channel,
  same method, same path) reflects the same rule values on B.
- Patching request A (path `/api/stores`) does not affect request C (path `/api/stores/123`).
- All 400/404 validation cases work as specified.
- `go build ./...` passes.

## Dependencies

- spec-01 (Channel struct reverted).
- spec-02 (SetResponseRule and GetResponseRule on the store).
