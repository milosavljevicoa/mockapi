# ADR-001: In-Memory Storage with Global FIFO Cap

**Date:** 2026-10-28  
**Status:** Accepted

---

## Context

MockAPI needs to store captured HTTP requests for later inspection via the dashboard. The tool is intended for personal, internal use during development and debugging — not as a production system.

## Decision

All state is held exclusively in process memory using Go structs protected by a `sync.RWMutex`. No database, file system writes, or external dependencies are used.

A **global cap of 100 requests** (configurable) is enforced across all channels. When the cap is reached, the single globally-oldest request (by `ReceivedAt`) is evicted before the new one is inserted. This is a simple FIFO ring over the full request set.

## Rationale

- Zero operational overhead — no database to install, configure, or migrate.
- Instant startup — no connection setup or schema validation.
- For a personal webhook inspector, 100 requests across all channels covers all practical interactive debugging sessions.
- Global eviction is simpler to reason about than per-channel eviction and avoids needing per-channel limits.

## Consequences

- All data is lost on process exit. This is acceptable; the tool is used interactively and sessions are short-lived.
- A single very active channel can consume the entire cap and push out requests from other channels. At 100 requests with personal use this is not a concern in practice.
- If usage patterns change and persistence becomes necessary, a SQLite backend can be introduced with minimal interface changes — the store is accessed only through a narrow interface.
