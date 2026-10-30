# spec-04-capture-rule-lookup

## Goal

Make the webhook capture handler look up the per-path response rule instead of using
a channel-level response, so that different paths within the same channel return
independent responses.

## Context

`handler/capture.go` currently calls `st.GetChannelResponse(slug)`, which returns the
same status and body for every path on a channel. After spec-02, that method no longer
exists; the replacement is `st.GetResponseRule(slug, method, path)`, which accepts the
full method and path and returns the rule for that specific combination — defaulting to
`(200, {})` when none has been set.

The method and path are already available on the `*http.Request` as `r.Method` and
`r.URL.Path`. No other logic in the handler changes.

## Out of scope

- Changes to `store/` (specs 01–02).
- API changes (`handler/api.go`, spec-03).
- UI changes (spec-05).

## Steps

1. **`handler/capture.go`** — Replace the response lookup at the end of `ServeCapture`:

   Remove:
   ```go
   status, body, _ := st.GetChannelResponse(slug)
   ```

   Replace with:
   ```go
   status, body := st.GetResponseRule(slug, r.Method, r.URL.Path)
   ```

   The `Content-Type` header, `WriteHeader`, and `Write` calls are unchanged.

## Acceptance criteria

- A `POST` to `/api/serviceflow/stores` on a fresh channel returns `200 {}` with
  `Content-Type: application/json`.
- After `PATCH /_/api/channels/api/requests/{id-for-stores}` sets status 418 and body
  `{"teapot":true}`, a subsequent `POST` to `/api/serviceflow/stores` returns 418 with
  that body.
- A `POST` to `/api/serviceflow/stores/123` is unaffected by the rule set for
  `/api/serviceflow/stores` — it still returns its own configured response (or the
  default if not configured).
- `go build ./...` passes.

## Dependencies

- spec-02 (GetResponseRule on the store).
