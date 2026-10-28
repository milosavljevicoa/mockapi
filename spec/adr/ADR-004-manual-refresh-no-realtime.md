# ADR-004: Manual Refresh — No Real-Time Push

**Date:** 2026-10-28  
**Status:** Accepted

---

## Context

When a new request arrives, the dashboard could be updated in several ways:

1. **WebSockets**: bidirectional persistent connection; overkill for one-way server-to-browser push.
2. **Server-Sent Events (SSE)**: lightweight one-way push; no polling; requires a long-lived HTTP connection per browser tab.
3. **Polling**: JS `setInterval` hits the API every N seconds and re-renders if the list changed.
4. **Manual refresh**: user presses F5 or clicks a Refresh button.

## Decision

v1 uses manual refresh only. A visible **Refresh** button in the request list panel triggers a `fetch` call to reload the request list for the current channel.

## Rationale

- This is a personal developer tool, not a production monitoring dashboard. The user knows when they've triggered a webhook and can refresh on demand.
- Eliminates all long-lived connection management (reconnect logic, heartbeats, goroutine lifecycle) from the server.
- Eliminates polling timers and stale-check logic from the client.
- The resulting code surface is significantly smaller and easier to reason about.

## Consequences

- Requests are not visible until the user manually refreshes. This is the accepted trade-off.
- When real-time updates become desirable (v2), SSE is the preferred upgrade path: it requires adding one streaming endpoint and replacing the Refresh button click handler with an `EventSource` subscription. No protocol change for existing REST endpoints.
