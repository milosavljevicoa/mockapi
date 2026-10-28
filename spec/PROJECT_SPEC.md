# MockAPI — Project Specification

**Version:** 1.0  
**Date:** 2026-10-28  
**Status:** Approved

---

## Overview

MockAPI is a single-binary Go web server for inspecting inbound HTTP requests. You point a webhook sender at a channel URL, then open the dashboard in your browser to see exactly what was sent — headers, body, method, path, query params, everything. No database, no cloud dependency, no setup beyond running one binary.

---

## Goals

- Capture every detail of incoming HTTP requests in real time.
- Provide a clean browser dashboard to inspect captured requests.
- Ship as a single self-contained binary with zero runtime dependencies.

---

## Non-Goals (v1)

- Configurable response per channel (planned v2).
- Real-time dashboard push via SSE or WebSockets (planned v2).
- Request replay to a target URL (planned v2).

---

## Architecture

```
┌──────────────────────────────────────────────────────┐
│                    Go Binary                         │
│                                                      │
│  ┌────────────┐   ┌──────────────┐   ┌───────────┐  │
│  │  Capture   │   │   In-Memory  │   │ Dashboard │  │
│  │  Handler   │──▶│    Store     │◀──│    API    │  │
│  │ /{slug}/.. │   │  (global     │   │  /_/...   │  │
│  └────────────┘   │   ring buf)  │   └───────────┘  │
│                   └──────────────┘         │         │
│                                      ┌─────▼──────┐  │
│                                      │ Embedded   │  │
│                                      │ HTML/JS/   │  │
│                                      │ Tailwind   │  │
│                                      └────────────┘  │
└──────────────────────────────────────────────────────┘
```

### Components

| Component | Responsibility |
|---|---|
| **Capture Handler** | Matches any request not starting with `/_/`, extracts slug from first path segment, records request, returns `200 OK`. |
| **In-Memory Store** | Thread-safe store holding channels and requests. Enforces the global cap via FIFO eviction. |
| **Dashboard API** | REST endpoints under `/_/api/` consumed by the embedded JS. |
| **Embedded UI** | Static HTML/CSS/JS embedded via Go's `embed` package, served from `/_/`. |

---

## URL Routing

| Pattern | Handler |
|---|---|
| `GET /` | Redirect to dashboard (`/_/`) |
| `GET /_/` | Dashboard SPA |
| `GET /_/static/*` | Embedded static assets |
| `GET /_/api/channels` | List all channels |
| `POST /_/api/channels` | Create channel explicitly |
| `GET /_/api/channels/{slug}` | Get channel metadata |
| `DELETE /_/api/channels/{slug}/requests` | Clear all requests for a channel |
| `GET /_/api/channels/{slug}/requests` | List requests for a channel |
| `GET /_/api/channels/{slug}/requests/{id}` | Get single request detail |
| `ANY /{slug}` | Capture request |
| `ANY /{slug}/*` | Capture request (sub-path) |

### Reserved Prefixes

The prefix `/_/` is permanently reserved for internal routes and cannot be used as a channel slug. Slug validation rejects any name starting with `_`.

---

## Data Model

### Channel

```go
type Channel struct {
    Slug      string    // URL-safe identifier
    CreatedAt time.Time
}
```

### Request

```go
type Request struct {
    ID          string              // UUID v4
    ChannelSlug string
    ReceivedAt  time.Time
    Method      string              // GET, POST, etc.
    Path        string              // full path including sub-path
    Query       map[string][]string // parsed query parameters
    Headers     http.Header         // all request headers
    Body        []byte              // raw request body
    ContentType string              // extracted from Content-Type header
    BodySize    int                 // bytes
}
```

---

## Channel Lifecycle

```
                    ┌─────────────────┐
                    │  Does slug      │
                    │  exist?         │
                    └──────┬──────────┘
              No ──────────┘    └────── Yes
               │                         │
               ▼                         ▼
        Auto-create channel       Use existing channel
               │                         │
               └────────────┬────────────┘
                            │
                            ▼
                   Record request in channel
                            │
                            ▼
                   Global cap exceeded?
                     │           │
                    Yes          No
                     │           │
                     ▼           ▼
              Evict globally   Done
              oldest request
```

### Slug Rules

- Lowercase alphanumeric characters and hyphens only.
- 3–50 characters.
- Cannot start or end with a hyphen.
- Cannot start with `_`.
- Must be unique within the running instance.

### Auto-Generated Slugs

Format: `{adjective}-{noun}-{4-digit-number}`  
Example: `silent-river-4821`

Generated at channel creation time. Uniqueness is checked against existing channels; regenerated on collision.

---

## Request Capture

Every HTTP method is accepted. The full request is read and stored:

- Method
- Full path (e.g. `/my-channel/users/123`)
- Raw query string, parsed into key-value map
- All request headers (no filtering)
- Full request body (read completely before responding)
- `Content-Type` header value
- Body size in bytes
- Timestamp (`time.Time`, stored as UTC)

**Response to sender:** always `200 OK` with an empty body, regardless of channel or content.

---

## In-Memory Store

- **Global cap:** 100 requests (configurable via `--cap` / `CAP` env var).
- **Eviction:** When the cap is reached, the single globally-oldest request (by `ReceivedAt`) is removed before the new one is inserted.
- **Thread safety:** All store operations protected by a `sync.RWMutex`.
- **No persistence:** All data is lost on process exit.

---

## Dashboard

### Layout

```
┌─────────────────────────────────────────────────────────┐
│  MockAPI                              [+ New Channel]    │
├──────────────────┬──────────────────────────────────────┤
│                  │                                       │
│  CHANNELS        │  REQUESTS          REQUEST DETAIL     │
│  ──────────      │  ──────────        ──────────────     │
│  > silent-river  │  POST /users  ──▶  Method: POST       │
│    fuzzy-wolf    │  GET  /       │    Path: /users        │
│                  │  DELETE /id   │    Time: 14:32:01      │
│                  │               │    Headers: ...        │
│                  │               │    Body: { ... }       │
│                  │                                       │
└──────────────────┴──────────────────────────────────────┘
```

### Channel Sidebar (left panel)

- Lists all channels, sorted by most recently active.
- Each entry shows the slug name.
- Clicking a channel loads its request list.
- **"+ New Channel"** button opens a modal:
  - Text field for custom slug (optional).
  - If left blank, a slug is auto-generated on submit.
  - Shows the generated webhook URL after creation.
- Per-channel actions (accessible via a `⋯` menu or inline buttons):
  - **Copy URL** — copies `http://{host}/{slug}` to clipboard.
  - **Clear** — deletes all requests for this channel; channel remains.

### Request List (middle panel)

- Lists all requests for the selected channel, newest first.
- Each row shows:
  - Method badge (color-coded: green=GET, blue=POST, yellow=PUT/PATCH, red=DELETE, grey=other)
  - Path (sub-path portion; `/` if none)
  - Absolute timestamp (`2026-10-28 14:32:01`)
  - Body size (`1.2 KB`)
- Clicking a row loads the request detail panel.
- **Refresh** button at the top reloads the request list from the server.

### Request Detail (right panel)

Tabs:
1. **Overview** — method, full path, timestamp, body size, content-type.
2. **Headers** — all request headers as a key-value table.
3. **Query Params** — parsed query string as a key-value table. Hidden if no query params.
4. **Body** — body content with two sub-tabs:
   - **Pretty** (default): JSON is syntax-highlighted and formatted; `application/x-www-form-urlencoded` shown as key-value pairs; all other types shown as plain text.
   - **Raw**: unmodified body bytes as a string.

---

## Configuration

| Flag | Env Var | Default | Description |
|---|---|---|---|
| `--port` | `PORT` | `8080` | Port to listen on |
| `--cap` | `CAP` | `100` | Global request cap |
| `--verbose` | — | `false` | Print request headers and body to the console |

CLI flags take precedence over env vars. `--verbose` has no env var equivalent.

---

## Console Logging

Every captured webhook request is logged to `stderr` via the standard `log` package.

### Default output

One line per request:

```
2026/10/28 14:32:01 [POST] /hooks/abc?token=xyz | 200 | 42ms
```

Fields in order: HTTP method, full path with query string, status code, response time.

### Verbose output (`--verbose`)

The one-liner is followed by all request headers and the body:

```
2026/10/28 14:32:01 [POST] /hooks/abc?token=xyz | 200 | 42ms
  Headers:
    Content-Type: application/json
    X-Hub-Signature: sha256=abc123
    X-Github-Event: push
  Body:
    {
      "event": "push",
      "ref": "refs/heads/main"
    }
```

**Header display:** all headers are shown exactly as received, with no filtering.

**Body formatting:** if `Content-Type` is `application/json`, the body is pretty-printed as indented JSON. For all other content types, the raw body string is printed as-is.

**Body truncation:** none — the full body is always printed.

### Scope

Only webhook capture requests (`/{slug}` and `/{slug}/*`) are logged. Dashboard API calls (`/_/api/`) and embedded UI requests (`/_/`) produce no console output.

---

## Tech Stack

| Layer | Choice |
|---|---|
| Language | Go 1.22+ |
| Router | `net/http` with `ServeMux` (no external router) |
| UI delivery | Go `embed` package |
| CSS | Tailwind CSS (CDN Play script) |
| JS | Vanilla JS (no framework, no build step) |
| IDs | `crypto/rand` UUID v4 |

---

## Binary Usage

```bash
# Default (port 8080, cap 100)
./mockapi

# Custom port
./mockapi --port 9000

# Custom cap via env
CAP=50 ./mockapi

# Both
PORT=9090 CAP=200 ./mockapi

# Verbose console logging (headers + body per request)
./mockapi --verbose

# Verbose with custom port
./mockapi --port 9000 --verbose
```
