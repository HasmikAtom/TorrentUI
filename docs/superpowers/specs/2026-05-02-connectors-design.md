# Connectors (Integrations) Framework — Design Spec

## Problem

Plex integration needs per-user tokens and config. Hardcoding credentials in `.env` doesn't scale — David and Hasmik have separate Plex accounts, and future integrations (Jellyfin, Sonarr, etc.) would each need their own credentials. We need a clean way to store and manage per-user external service connections.

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Data ownership | Go backend owns its own SQLite DB | Keeps auth-service focused on auth; Go is the consumer of integration data |
| Storage pattern | Miniflux-style flat columns, one row per user | Simple, type-safe, no JSON parsing; new integrations = ALTER TABLE ADD COLUMN |
| Credential encryption | None (for now) | Self-hosted, two users, tokens are revocable; can layer encryption later |
| Token acquisition | Manual paste (PIN flow later) | Simplest first pass; Plex PIN auth flow is a follow-up |
| UI placement | User dropdown → /integrations page | Configure-once-and-forget; doesn't need top-level sidebar real estate |
| Token validation | Validate on save via Plex API | Immediate feedback; no background jobs exist yet for lazy validation |
| Config scope | Token + enabled only | No quality prefs, polling interval, or scraper source until features that need them exist |

## Data Model

Go backend gets a new SQLite database at `data/backend.sqlite`. Migration runs at startup (`CREATE TABLE IF NOT EXISTS`).

```sql
CREATE TABLE user_integrations (
    user_id         TEXT PRIMARY KEY,
    plex_enabled    INTEGER NOT NULL DEFAULT 0,
    plex_token      TEXT NOT NULL DEFAULT '',
    created_at      INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at      INTEGER NOT NULL DEFAULT (unixepoch())
);
```

- One row per user, upserted on first connector save
- Adding a new integration type (e.g. Sonarr) = migration with `ALTER TABLE ADD COLUMN sonarr_enabled INTEGER DEFAULT 0`, `sonarr_url TEXT DEFAULT ''`, `sonarr_api_key TEXT DEFAULT ''`
- Go struct maps 1:1: `PlexEnabled bool`, `PlexToken string`

## Go Backend

### SQLite Setup

- Dependency: `modernc.org/sqlite` (pure Go, no CGO)
- DB path: `data/backend.sqlite` (bind-mounted in Docker like auth's `data/auth.sqlite`)
- Migrations run at startup, same `CREATE TABLE IF NOT EXISTS` pattern as auth-service

### API Routes

All routes under `middleware.RequireUser()`:

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/integrations` | Returns user's full integration row (or defaults if none) |
| `PUT` | `/integrations/plex` | Update Plex connector. If `token` is provided, validates it first. If only `enabled` is provided, toggles without re-validation. |
| `DELETE` | `/integrations/plex` | Clear Plex token, set plex_enabled = 0 |

No generic connector registry or interface. Each integration type gets explicit handler functions. Adding a new type = new route pair + validation function.

### Plex Token Validation

`PUT /integrations/plex` validates by calling `GET https://plex.tv/api/v2/user` with the header `X-Plex-Token: <submitted_token>`.

- 200 → token is valid, save it
- 401 → return 400 `{"error": "Invalid Plex token"}`
- Network error → return 502 `{"error": "Could not verify token — try again later"}`

## Frontend

### Access Point

New "Integrations" menu item in the user dropdown (nav-user.tsx), between ThemeSubmenu and Sign out. Uses a `Plug` icon from lucide-react. Clicking navigates to `/integrations`.

### Route

New `<Route path="/integrations" element={<IntegrationsPage />} />` in App.tsx. Available to all authenticated users (not admin-gated).

### Page Layout

Inline list with expand/collapse per integration type. One row per integration.

### UI States

1. **Not connected** — muted integration icon, "Connect" button on the right
2. **Entering token** — row expands to show token input field, Save/Cancel buttons, help text for finding the token
3. **Connected** — solid icon, green "Connected" status indicator, enable/disable toggle switch
4. **Editing** — row expands to show masked token field, Update and Disconnect buttons

### Flow

1. Not connected → click "Connect" → form expands
2. Paste token → click "Save" → `PUT /api/integrations/plex` validates → success → collapse to "Connected" state
3. Click row to expand → edit token or disconnect
4. Toggle switch → `PUT /api/integrations/plex` with `enabled: true/false` (no re-validation)
5. "Disconnect" → `DELETE /api/integrations/plex` → revert to "Not connected"

### Error Handling

- Invalid token → inline error message below the input field
- Plex API unreachable → inline error "Could not verify token — try again later"

### Components

- `IntegrationsPage.tsx` — page wrapper, fetches `GET /api/integrations`, renders list
- `PlexIntegrationRow.tsx` — Plex-specific row with expand/collapse, form, toggle

## End-to-End Flow

1. User clicks "Integrations" in sidebar user dropdown → navigates to `/integrations`
2. `GET /api/integrations` → auth-service proxies to Go → Go reads `user_integrations` row (or returns defaults)
3. User clicks "Connect" on Plex → expands token form → pastes token → clicks "Save"
4. `PUT /api/integrations/plex` with `{"token": "xyz"}` → Go validates against Plex API → upserts row → returns updated integration state
5. UI updates to "Connected" state with toggle
6. Toggle → `PUT /api/integrations/plex` with `{"enabled": false}` → updates without re-validation
7. "Disconnect" → `DELETE /api/integrations/plex` → clears token, sets disabled

## Out of Scope (Deferred)

- Credential encryption at rest
- Plex PIN-based OAuth flow
- Plex config fields (quality, polling interval, scraper source, auto-download)
- Background polling / automation (Plex Auto-Tracker, Watchlist)
- Other integration types (Sonarr, Radarr, Jellyfin)
