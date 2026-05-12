# Plex Movies Page — Design Spec

## Problem

We've shipped the integrations framework: users can save a Plex token at `/integrations` and it's validated against `plex.tv/api/v2/user`. The token is sitting in `user_integrations` doing nothing. The next step is putting that token to work — give each connected user a dedicated `/movies` page that shows everything in their Plex movie libraries.

## Goals

- Authenticated users with a saved, enabled Plex token can browse all movies from their Plex Media Server (PMS) at `/movies`.
- Token never leaves the backend; the browser never sees Plex tokens or raw PMS URLs.
- UI follows the same shadcn/Radix + sidebar conventions as the rest of the app.

## Non-Goals (Deferred)

- TV shows, music, or photos. Movies only.
- Library picker / multi-server picker.
- Plex search endpoint integration; in-app filtering or genre/year facets.
- Watch status, mark-as-watched, playback.
- Cross-linking with the torrent download flow ("missing in Plex" → search).
- Persistent local cache / sync of the Plex library into our SQLite.

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Data layer | Backend proxy with on-demand PMS discovery | Token stays server-side; matches the auth-service → Go proxy pattern; no schema churn |
| Server selection | First `owned=1` resource that `provides` `server`; prefer local, then `plex.direct` HTTPS, then relay | Most likely to succeed without firewall surprises |
| Pagination | Server-side via Plex's `X-Plex-Container-Start` / `X-Plex-Container-Size`; infinite scroll on the client | Plex libraries can be thousands of items |
| Multi-library behavior | All movie libraries combined into one view; v1 paginates one library at a time and concatenates results in order | Avoids merging-sort complexity; documented quirk if a user has multiple movie libraries |
| Sort options | `addedAt:desc` (default), `titleSort:asc`, `year:desc`, `rating:desc` | Matches the four sorts Plex's own UI emphasizes |
| Image delivery | Backend proxies `/library/metadata/<id>/thumb/<v>` and sets `Cache-Control: public, max-age=86400` | Token never reaches the browser; browser cache amortizes load |
| Discovery caching | In-memory map `userID → ServerConn` with 5-minute TTL | Cheap, avoids re-discovering on every request, expires often enough to recover from PMS changes |
| Gating | All `/plex/*` routes return 412 `plex_not_configured` if the user has no token or `plex_enabled = 0` | Frontend uses this to route the user to `/integrations` |
| Sidebar visibility | "Movies" nav item is hidden until the integrations check confirms Plex is enabled + has token | Avoids dead links for unconnected users |
| Tests | Backend: httptest fixtures for plex.tv and PMS; fake `PlexClient` for handler tests. Frontend: manual verification (no frontend tests per CLAUDE.md) | Consistent with current repo conventions |

## Architecture

```
Browser
  → GET /api/plex/movies?start=0&size=50&sort=addedAt:desc
auth-service (proxies, strips/re-attaches X-User-*)
  → Go backend :8080
      middleware.RequireUser
        ↓
      plex.Handlers
        ↓
      plex.Client
        ├── integrations.Store      (read token + enabled)
        ├── plex.tv/api/v2/resources  (discovery, cached per user)
        └── PMS                       (library/sections, library/sections/:k/all, image)
```

The `plex` package owns all Plex protocol knowledge: discovery, connection selection, XML/JSON quirks, header conventions. Handlers depend on it through a small interface so they can be tested with a fake.

## Data Model

No new tables. Reads `plex_token` and `plex_enabled` from the existing `user_integrations` row via `integrations.Store`.

In-memory only:

```go
type ServerConn struct {
    BaseURL          string    // e.g. "https://1-2-3-4.<hash>.plex.direct:32400"
    MachineIdentifier string
    ResolvedAt       time.Time
}

type discoveryCache struct {
    mu      sync.Mutex
    entries map[string]ServerConn // keyed by user ID
}
```

TTL: 5 minutes. Lookup-miss → re-discover.

## Backend

### Package Layout

```
backend/plex/
  client.go         # PlexClient: List/Get movies, FetchImage
  discover.go       # ResolveServer(token) → ServerConn; connection-selection rules
  cache.go          # discoveryCache (in-memory, 5 min TTL)
  handlers.go       # Gin handlers for /plex/* routes
  types.go          # Movie, MovieDetail, ServerConn, error sentinels
  client_test.go    # httptest fixtures for plex.tv + PMS
  handlers_test.go  # uses a fake PlexClient
```

### Routes

All registered under `api := r.Group("/", middleware.RequireUser())`:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/plex/movies` | Paginated movie list. Query params: `start` (int, default 0), `size` (int, default 50, max 200), `sort` (one of `addedAt:desc`, `titleSort:asc`, `year:desc`, `rating:desc`; default `addedAt:desc`). |
| GET | `/plex/movies/:ratingKey` | Movie detail (summary, genres, directors, writers, top cast, duration, audience rating). |
| GET | `/plex/image` | Streams image bytes. Query param: `path` (must start with `/library/metadata/`; reject other paths to prevent SSRF). Sets `Content-Type` from upstream, `Cache-Control: public, max-age=86400`. |

### Gating

Each handler calls a helper:

```go
func resolveClient(c *gin.Context) (*PlexClient, ServerConn, bool) {
    userID := c.GetString("userId")
    row, err := store.GetIntegrations(userID)
    if err != nil { c.JSON(500, ...); return ..., false }
    if !row.PlexEnabled || row.PlexToken == "" {
        c.JSON(412, gin.H{"error": "plex_not_configured"})
        return ..., false
    }
    conn, err := plexClient.ResolveServer(userID, row.PlexToken)
    if err != nil { c.JSON(502, gin.H{"error": "plex_server_unreachable"}); return ..., false }
    return plexClient, conn, true
}
```

### Plex API Conventions

Required headers on every PMS request:
- `X-Plex-Token: <token>`
- `Accept: application/json`
- `X-Plex-Client-Identifier: torrent-ui`
- `X-Plex-Product: TorrentUI`

Discovery: `GET https://plex.tv/api/v2/resources?includeHttps=1&includeRelay=1`.
- Filter to resources with `owned: true` and `provides` containing `"server"`.
- For the first such resource, walk `connections` and pick the first that matches in this priority order:
  1. `local: true` and `protocol: "https"`
  2. `protocol: "https"` and `relay: false` (these are `plex.direct` URLs)
  3. `relay: true`
- Returned `ServerConn{ BaseURL, MachineIdentifier }`.

Movie libraries: `GET <BaseURL>/library/sections` → walk `MediaContainer.Directory[]`, keep entries where `type == "movie"`.

Movies per library: `GET <BaseURL>/library/sections/<key>/all?type=1&X-Plex-Container-Start=N&X-Plex-Container-Size=M&sort=<sort>`. Plex echoes pagination back via `MediaContainer.size`, `totalSize`, `offset`. Item fields used: `ratingKey`, `title`, `year`, `thumb`, `art`, `rating`, `audienceRating`, `duration`, `addedAt`, `summary`.

Movie detail: `GET <BaseURL>/library/metadata/<ratingKey>` → first `Metadata` entry. Pull `summary`, `Genre[]`, `Director[]`, `Writer[]`, `Role[]` (top 6), `duration`, `contentRating`, `studio`, `originallyAvailableAt`.

Image: `GET <BaseURL>/photo/:/transcode?width=300&height=450&minSize=1&upscale=1&url=<urlencoded path>&X-Plex-Token=...`. Backend can either transcode (smaller payload) or pass through raw `/library/metadata/<id>/thumb/<v>`. v1: pass through raw to keep code simple; revisit if posters are too heavy.

### Multi-Library Behavior

If the user has multiple movie libraries:
- `/plex/movies` paginates **within a single library at a time**. The handler iterates libraries in the order Plex returns them; once a library is exhausted, the next page starts on the next library at offset 0.
- `total` returned to the frontend is the **sum across all movie libraries**.
- This avoids implementing a merge-sort across libraries. Documented quirk: sort order is per-library, not globally interleaved. Acceptable for v1 since most users have one movie library.

### Error Mapping

| Upstream | Returned to client |
|----------|--------------------|
| Plex token rejected (401 from plex.tv or PMS) | 401 `{"error": "plex_unauthorized"}` |
| Discovery network error or no eligible resource | 502 `{"error": "plex_server_unreachable"}` |
| PMS network error / 5xx | 502 `{"error": "plex_server_unreachable"}` |
| No movie libraries | 200 `{ items: [], total: 0, start: 0, size: 0 }` (not an error) |
| Image upstream error | 502 (text body) |

Retry policy: one retry on transport-level network errors only. No retry on 4xx/5xx HTTP responses. Per-request timeout: 15s.

### SSRF Guard on `/plex/image`

`path` query param MUST start with `/library/metadata/` after URL decoding. Anything else → 400. Prevents a malicious caller from coercing the backend into fetching arbitrary URLs by smuggling a host in `path`.

## Frontend

### Routing

`App.tsx` gains:

```tsx
<Route path="/movies" element={<MoviesPage />} />
```

### Sidebar Visibility

`nav-menu.tsx` is extended to accept a `plexEnabled: boolean` prop. When true, a "Movies" item (Film icon from lucide-react) is inserted between "Home" and admin items. A small `useIntegrations` hook (in `frontend/src/hooks/`) fetches `/api/integrations` once and caches the result in module-scoped state so `AppShell` and `MoviesPage` share the same fetch. `AppShell` passes `plexEnabled && plexHasToken` down to the sidebar.

### Components

```
frontend/src/components/
  movies/
    MoviesPage.tsx       # page wrapper: gating, sort dropdown, infinite scroll
    MovieGrid.tsx        # responsive grid of MovieCard
    MovieCard.tsx        # poster + title + year
    MovieDetail.tsx      # modal for a single movie (Radix Dialog)
    SortDropdown.tsx     # Radix DropdownMenu wrapping the four sort options
    useMoviesQuery.ts    # custom hook: pages state, fetch next page, sort change resets
```

`MoviesPage.tsx`:
1. On mount, `GET /api/integrations`. If `!plexEnabled || !plexHasToken`, render a centered card: "Connect Plex to see your movies" with a button linking to `/integrations`.
2. Otherwise, render `<SortDropdown>` + `<MovieGrid items={items} loading={loading} />`.
3. Infinite scroll: `IntersectionObserver` on a sentinel div at the bottom; when intersecting and not loading and `items.length < total`, fetch next page.
4. Sort change resets `items=[]`, `start=0`, refetches.

`MovieCard.tsx`:
- `<img src={`/api/plex/image?path=${encodeURIComponent(thumb)}`} loading="lazy" />`
- Aspect ratio 2:3, rounded corners, subtle shadow.
- Title (1 line, truncate) + year underneath.
- Click → opens `<MovieDetail ratingKey={...} />`.

`MovieDetail.tsx`:
- On open, `GET /api/plex/movies/:ratingKey`.
- Renders backdrop (via `art`), poster, title, year, runtime (mm), audience rating, genres as pills, top 6 cast, directors, summary.
- Uses the existing `Dialog` from `components/ui/dialog`.

### Error States

| Backend response | UI |
|------------------|----|
| 412 `plex_not_configured` | Same CTA as above (route to /integrations) |
| 401 `plex_unauthorized` | Toast + inline banner: "Your Plex token is invalid. Reconnect at Integrations." |
| 502 `plex_server_unreachable` | Inline error card with "Retry" button |
| 200 with empty items on first page | "No movies found in your Plex libraries." |

### Loading States

- First page: skeleton grid (12 placeholder cards).
- Subsequent pages (infinite scroll): spinner at the bottom sentinel.
- Image errors: fall back to a `<div>` with the title centered on a muted background.

## End-to-End Flow

1. User clicks "Movies" in the sidebar → `/movies`.
2. `MoviesPage` fetches `/api/integrations`; Plex is enabled, so it proceeds.
3. `GET /api/plex/movies?start=0&size=50&sort=addedAt:desc`.
4. Auth-service proxies to Go with `X-User-Id`.
5. Go's plex handler: reads token from `user_integrations`, resolves the user's PMS via cached discovery (or fresh `plex.tv/api/v2/resources` on miss), queries the first movie library's `/library/sections/<key>/all`, returns 50 items.
6. Frontend renders the grid. Each `MovieCard` requests `/api/plex/image?path=...`; backend streams the thumbnail.
7. User scrolls; sentinel intersects; frontend requests `start=50`.
8. User clicks a card; `MovieDetail` opens and fetches `/api/plex/movies/:ratingKey`; details render in a dialog.
9. User changes sort to "Title A-Z"; state resets, `start=0` refetch with `sort=titleSort:asc`.

## Testing

### Backend

- `discover_test.go`:
  - Connection selection prefers local HTTPS > plex.direct HTTPS > relay.
  - Skips resources that aren't owned or don't provide server.
  - Network error from plex.tv → wrapped error.
- `client_test.go`:
  - Lists movies with pagination (httptest server returns fixture JSON with `MediaContainer.size`/`totalSize`).
  - Multi-library iteration: two movie libraries, requested page spans the boundary, items concatenated correctly.
  - Movie detail parsing: genres, directors, top 6 cast.
  - 401 from PMS → `ErrUnauthorized` sentinel.
- `handlers_test.go` (using a fake `PlexClient`):
  - 412 when integration row is disabled or missing token.
  - 401 when the fake returns `ErrUnauthorized`.
  - 502 when the fake returns transport error.
  - SSRF guard on `/plex/image` rejects paths outside `/library/metadata/`.
  - Pagination query params parsed and clamped (size capped at 200, negative start rejected).

### Frontend

- Manual verification in browser per CLAUDE.md (no frontend tests):
  - Cold load with Plex connected → grid renders, posters load, infinite scroll works.
  - Toggle Plex off in /integrations → /movies shows the CTA.
  - Invalidate token (manually edit DB) → 401 path renders banner.
  - Stop PMS → 502 path renders retry card.
  - Sort change resets and refetches.
  - Detail modal opens with full content.

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| PMS unreachable behind a strict firewall | Prefer `plex.direct` HTTPS (works cross-network); fall back to relay |
| Image bandwidth (1000s of posters) | Browser cache via `Cache-Control: public, max-age=86400`; revisit transcoding to smaller sizes if needed |
| Discovery rate-limit on plex.tv | 5-min in-memory TTL on the per-user cache; only re-discover on miss or PMS error |
| Token rotation / revocation | 401 path surfaces a clear "reconnect at /integrations" message |
| Multi-library users seeing per-library sort | Document the v1 quirk; consider proper merge later |

## Open Questions

None blocking. Items to revisit after shipping:
- Should `/plex/image` transcode to thumbnail size (~300×450) to cut bandwidth?
- Should we add a small "last refreshed" hint to indicate live data?
- Should the discovery cache be persisted across restarts? (Probably not — it's cheap.)
