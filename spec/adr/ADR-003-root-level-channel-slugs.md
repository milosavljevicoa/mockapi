# ADR-003: Root-Level Channel Slugs

**Date:** 2026-10-28  
**Status:** Accepted

---

## Context

Incoming webhook capture URLs need a routing scheme. Two options were considered:

- **Namespaced**: `/hook/{slug}` and `/hook/{slug}/*` — separates capture routes from app routes cleanly by prefix.
- **Root-level**: `/{slug}` and `/{slug}/*` — shorter, cleaner URLs that are easier to copy and share.

## Decision

Channel slugs live at the root of the URL namespace: `http://localhost:8080/{slug}`.

All internal application routes are placed under the `/_/` prefix:
- `/_/` — dashboard SPA
- `/_/static/*` — embedded static assets
- `/_/api/*` — REST API consumed by the dashboard JS

The slug `_` (and any slug beginning with `_`) is permanently reserved and rejected at channel creation time.

## Rationale

- Root-level URLs are shorter and easier to paste into a webhook configuration field.
- The `/_/` reservation is a well-established convention (used by Firebase, various CDNs) and is easy to explain and enforce.
- Slug validation (lowercase alphanumeric + hyphens, 3–50 chars, no leading/trailing hyphen, no leading underscore) prevents any collision with the reserved prefix.

## Consequences

- Any future top-level route additions must either go under `/_/` or be explicitly added to a blocklist of reserved slugs (e.g. `favicon.ico`, `robots.txt`). The implementation should maintain a small explicit denylist for these well-known browser-requested paths.
- The routing logic must match `/_/*` before falling through to the slug capture handler to ensure the priority is correct.
