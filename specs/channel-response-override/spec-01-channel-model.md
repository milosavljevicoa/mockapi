# spec-01-revert-channel-model

## Goal

Revert the `Channel` struct to its original shape. The response override feature now
operates at the (channel, method, path) level — stored inside the `Store` as an
internal rules map — so no exported fields on `Channel` are needed.

## Context

The previous implementation added `ResponseStatus int` and `ResponseBody []byte` to
`Channel`. These fields are no longer the right home for response configuration: a
single channel can receive requests at many different paths, each of which should be
independently configurable. Storing one response on the channel would force all paths
to share it.

`store/models.go` is the only file that changes in this spec.

## Out of scope

- Any store method changes (spec-02).
- Any handler or UI changes.

## Steps

1. **`store/models.go`** — Remove `ResponseStatus int` and `ResponseBody []byte` from
   the `Channel` struct, restoring it to:
   ```go
   type Channel struct {
       Slug      string
       CreatedAt time.Time
   }
   ```

## Acceptance criteria

- `Channel` has only `Slug` and `CreatedAt`.
- `go build ./...` passes after this change alone (the store methods that reference
  those fields will be removed in spec-02, so a build break between specs is expected
  — the full build must pass only after spec-02 is applied).

## Dependencies

None.
