# ADR-005: Console Request Logging

**Date:** 2026-10-28  
**Status:** Accepted

---

## Context

MockAPI currently logs only server startup and fatal errors. During development it is useful to see captured webhook requests directly in the terminal — without switching to the browser — to get immediate feedback that a sender reached the right channel and what it sent.

## Decision

Wrap the capture handler with a logging middleware that writes one line per captured request to the console using Go's standard `log` package.

**Default mode** — one line per request:
```
2026/10/28 14:32:01 [POST] /hooks/abc?token=xyz | 200 | 42ms
```

**Verbose mode** (`--verbose` flag) — one-liner followed by all request headers and the body:
```
2026/10/28 14:32:01 [POST] /hooks/abc?token=xyz | 200 | 42ms
  Headers:
    Content-Type: application/json
    X-Hub-Signature: sha256=abc123
  Body:
    {
      "event": "push",
      "ref": "refs/heads/main"
    }
```

Logging applies **only to the capture handler** (`/{slug}/*`). Dashboard API calls (`/_/api/`) and embedded UI requests (`/_/`) are not logged.

Body formatting in verbose mode is content-type aware: if `Content-Type` is `application/json`, the body is pretty-printed as JSON; otherwise it is printed as a raw string. No truncation — the full body is always shown.

## Rationale

- Using `log` (stdlib) keeps the zero-dependency constraint intact.
- Scoping to the capture handler avoids noise from internal API and UI traffic; the operator cares about inbound webhooks, not dashboard polling.
- A single `--verbose` flag is consistent with the existing `--port` / `--cap` style and keeps the default output quiet enough to leave running in a terminal without distraction.
- Showing all headers without filtering matches the project's philosophy of full fidelity capture — operators use this tool specifically to see what senders actually send.
- No body truncation is chosen deliberately: the tool is used interactively in short sessions and the operator may need to inspect large payloads verbatim. If terminal flooding becomes a concern in practice, truncation can be added as a follow-up.

## Consequences

- Log output goes to `stderr` (the `log` package default), keeping it separate from any `stdout` the binary might emit in the future.
- The `log` package prepends a date and time prefix automatically (`2026/10/28 14:32:01`); no custom timestamp formatting is needed.
- In verbose mode, large request bodies produce proportionally large terminal output. This is an accepted trade-off for a dev tool.
- The middleware wraps the capture handler at registration time in `main.go`; no changes to the handler or store are required.
