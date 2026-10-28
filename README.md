# MockAPI

A single-binary Go tool for inspecting inbound HTTP requests. Point any webhook sender at a channel URL, then open the browser dashboard to see exactly what was sent — headers, body, method, path, query params, everything.

No database. No cloud dependency. No setup beyond running one binary.

---

## Quick Start

```bash
# Build
go build -o mockapi .

# Run (default port 9000, cap 100 requests)
./mockapi

# Open the dashboard
open http://localhost:9000
```

Send a webhook to any channel slug:

```bash
curl -X POST http://localhost:9000/my-channel \
  -H "Content-Type: application/json" \
  -d '{"event": "push", "ref": "refs/heads/main"}'
```

Then open the dashboard and click `my-channel` to inspect the request.

---

## How It Works

Channels are created automatically when the first request arrives at `/{slug}`. Every HTTP method is accepted. The server always responds `200 OK` to the sender — its only job is to capture and display.

```
┌──────────────────────────────────────────────────────┐
│                    Go Binary                         │
│                                                      │
│  ┌────────────┐   ┌──────────────┐   ┌───────────┐   │
│  │  Capture   │   │   In-Memory  │   │ Dashboard │   │
│  │  Handler   │──▶│    Store     │◀──│    API    │   │
│  │ /{slug}/.. │   │  (ring buf)  │   │  /_/...   │   │
│  └────────────┘   └──────────────┘   └───────────┘   │
└──────────────────────────────────────────────────────┘
```

---

## Dashboard

A three-panel browser UI served from `/_/`:

- **Left** — channel list, sorted by most recent activity. Create new channels via the **+ New Channel** button.
- **Middle** — request list for the selected channel, newest first. Each row shows method, path, timestamp, and body size.
- **Right** — request detail with tabs for Overview, Headers, Query Params, and Body (pretty-printed JSON or raw).

Per-channel actions: **Copy URL** and **Clear** (removes all requests, keeps the channel).

---

## Configuration

| Flag | Default | Description |
|---|---|---|
| `--port` | `9000` | Port to listen on |
| `--cap` | `100` | Max requests held in memory (oldest evicted first) |
| `--verbose` | `false` | Print headers and body to stderr per captured request |


```bash
# Custom port
./mockapi --port 8080

# Verbose console logging
./mockapi --verbose
```

---

## Console Logging

Every captured request produces a log line on stderr:

```
2026/10/28 14:32:01 [POST] /hooks/abc?token=xyz | 200 | 42ms
```

With `--verbose`, headers and the formatted body follow each line. JSON bodies are pretty-printed; other content types are printed as-is.

Only capture requests are logged — dashboard and API calls produce no output.

---

## URL Routing

| Pattern | Description |
|---|---|
| `GET /` | Redirect to `/_/` |
| `GET /_/` | Dashboard SPA |
| `GET /_/api/channels` | List channels |
| `POST /_/api/channels` | Create channel |
| `GET /_/api/channels/{slug}/requests` | List requests |
| `GET /_/api/channels/{slug}/requests/{id}` | Get request detail |
| `DELETE /_/api/channels/{slug}/requests` | Clear channel requests |
| `ANY /{slug}` | Capture request |
| `ANY /{slug}/*` | Capture request (sub-path) |

The `/_/` prefix is reserved. Slugs must be 3–50 lowercase alphanumeric characters or hyphens, and cannot start with `_`.

---

## Tech Stack

- **Language:** Go 1.22+
- **Router:** `net/http` `ServeMux` (no external dependencies)
- **UI:** Vanilla JS + Tailwind CSS (CDN), embedded via Go's `embed` package
- **IDs:** `crypto/rand` UUID v4
- **Storage:** In-memory ring buffer, no persistence

---

## License

MIT
