# Plex Movies Page Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a `/movies` page that shows a paginated grid of all movies in the authenticated user's Plex libraries, sourced live from their Plex Media Server through a backend proxy.

**Architecture:** New Go `plex` package owns PMS discovery (cached, 5-min TTL), library/movie queries, and image streaming. Gin handlers under `middleware.RequireUser()` expose `/plex/movies`, `/plex/movies/:ratingKey`, `/plex/image`. Frontend adds a `/movies` route with grid + infinite scroll, gated by `useIntegrations()`. Plex token stays in the backend; the browser only sees our proxy URLs.

**Tech Stack:** Go 1.25, Gin, jmoiron/sqlx, modernc.org/sqlite, Go `net/http/httptest` for fixtures. React 18 + TypeScript, react-router-dom v6, Radix UI primitives (Dialog, DropdownMenu, Skeleton), Tailwind CSS, lucide-react icons.

**Spec:** `docs/superpowers/specs/2026-05-12-plex-movies-page-design.md`

---

## File Structure

### Backend (new package: `backend/plex/`)
- `types.go` — `Movie`, `MovieDetail`, `ServerConn`, `ListMoviesResult`, error sentinels
- `cache.go` — `discoveryCache` (in-memory map, 5-min TTL)
- `discover.go` — `ResolveServer(userID, token)` → `ServerConn`; connection-selection rules; uses cache
- `client.go` — `PlexClient`: `ListMovies`, `GetMovie`, `FetchImage`
- `handlers.go` — `RegisterHandlers(g, store, client)`; Gin handlers for `/plex/*`
- `cache_test.go`, `discover_test.go`, `client_test.go`, `handlers_test.go`

### Backend (modified)
- `backend/main.go` — instantiate `PlexClient`, call `plex.RegisterHandlers`

### Frontend (new)
- `frontend/src/hooks/useIntegrations.ts` — module-scoped cache + fetch of `/api/integrations`
- `frontend/src/components/movies/types.ts` — `Movie`, `MovieDetail` TS types
- `frontend/src/components/movies/MoviesPage.tsx` — page wrapper, gating, sort dropdown
- `frontend/src/components/movies/MovieGrid.tsx` — responsive grid + skeleton + empty state
- `frontend/src/components/movies/MovieCard.tsx` — single poster card
- `frontend/src/components/movies/MovieDetail.tsx` — Radix Dialog with detail
- `frontend/src/components/movies/SortDropdown.tsx` — 4 sort options
- `frontend/src/components/movies/useMoviesQuery.ts` — pagination + sort hook

### Frontend (modified)
- `frontend/src/App.tsx` — `/movies` route
- `frontend/src/components/app-sidebar/nav-menu.tsx` — accept `plexEnabled`, render Movies item
- `frontend/src/components/app-sidebar/app-sidebar.tsx` — accept + pass `plexEnabled`
- `frontend/src/components/AppShell.tsx` — call `useIntegrations`, pass to `AppSidebar`

---

## Task 1: Backend types and error sentinels

**Files:**
- Create: `backend/plex/types.go`

- [ ] **Step 1: Create types.go**

```go
package plex

import (
	"errors"
	"time"
)

// ErrUnauthorized indicates Plex rejected the user's token (401).
var ErrUnauthorized = errors.New("plex: unauthorized")

// ErrServerUnreachable indicates the user's PMS could not be reached or
// no eligible server was discovered.
var ErrServerUnreachable = errors.New("plex: server unreachable")

// ErrNotConfigured indicates the user has no Plex token or has disabled
// the Plex integration.
var ErrNotConfigured = errors.New("plex: not configured")

// ServerConn is a resolved connection to a user's Plex Media Server.
type ServerConn struct {
	BaseURL           string
	MachineIdentifier string
	ResolvedAt        time.Time
}

// Movie is the summary form returned in list responses.
type Movie struct {
	RatingKey      string  `json:"ratingKey"`
	Title          string  `json:"title"`
	Year           int     `json:"year"`
	Thumb          string  `json:"thumb"`
	Art            string  `json:"art"`
	Rating         float64 `json:"rating"`
	AudienceRating float64 `json:"audienceRating"`
	Duration       int64   `json:"duration"`
	AddedAt        int64   `json:"addedAt"`
	Summary        string  `json:"summary"`
}

// MovieDetail is the expanded form returned for a single movie.
type MovieDetail struct {
	Movie
	ContentRating         string   `json:"contentRating"`
	Studio                string   `json:"studio"`
	OriginallyAvailableAt string   `json:"originallyAvailableAt"`
	Genres                []string `json:"genres"`
	Directors             []string `json:"directors"`
	Writers               []string `json:"writers"`
	Cast                  []string `json:"cast"`
}

// ListMoviesResult is the paginated list response.
type ListMoviesResult struct {
	Items []Movie `json:"items"`
	Total int     `json:"total"`
	Start int     `json:"start"`
	Size  int     `json:"size"`
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd backend && go build ./plex/...`
Expected: no output, exit 0.

- [ ] **Step 3: Commit**

```bash
git add backend/plex/types.go
git commit -m "feat(plex): add types and error sentinels"
```

---

## Task 2: Discovery cache

**Files:**
- Create: `backend/plex/cache.go`
- Create: `backend/plex/cache_test.go`

- [ ] **Step 1: Write the failing test**

```go
package plex

import (
	"testing"
	"time"
)

func TestDiscoveryCache_GetMissReturnsZero(t *testing.T) {
	c := newDiscoveryCache(5 * time.Minute)
	if _, ok := c.get("user-1"); ok {
		t.Fatal("expected miss on empty cache")
	}
}

func TestDiscoveryCache_SetAndGet(t *testing.T) {
	c := newDiscoveryCache(5 * time.Minute)
	conn := ServerConn{BaseURL: "https://example.plex.direct:32400", MachineIdentifier: "abc"}
	c.set("user-1", conn)

	got, ok := c.get("user-1")
	if !ok {
		t.Fatal("expected hit")
	}
	if got.BaseURL != conn.BaseURL || got.MachineIdentifier != conn.MachineIdentifier {
		t.Fatalf("got %+v, want %+v", got, conn)
	}
}

func TestDiscoveryCache_ExpiredEntryIsMiss(t *testing.T) {
	c := newDiscoveryCache(10 * time.Millisecond)
	c.set("user-1", ServerConn{BaseURL: "x"})
	time.Sleep(20 * time.Millisecond)
	if _, ok := c.get("user-1"); ok {
		t.Fatal("expected expired entry to be a miss")
	}
}

func TestDiscoveryCache_Invalidate(t *testing.T) {
	c := newDiscoveryCache(5 * time.Minute)
	c.set("user-1", ServerConn{BaseURL: "x"})
	c.invalidate("user-1")
	if _, ok := c.get("user-1"); ok {
		t.Fatal("expected invalidate to remove entry")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./plex/... -run TestDiscoveryCache`
Expected: FAIL — `newDiscoveryCache` undefined.

- [ ] **Step 3: Implement cache.go**

```go
package plex

import (
	"sync"
	"time"
)

type discoveryCache struct {
	mu      sync.Mutex
	ttl     time.Duration
	entries map[string]ServerConn
}

func newDiscoveryCache(ttl time.Duration) *discoveryCache {
	return &discoveryCache{
		ttl:     ttl,
		entries: make(map[string]ServerConn),
	}
}

func (c *discoveryCache) get(userID string) (ServerConn, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	conn, ok := c.entries[userID]
	if !ok {
		return ServerConn{}, false
	}
	if time.Since(conn.ResolvedAt) > c.ttl {
		delete(c.entries, userID)
		return ServerConn{}, false
	}
	return conn, true
}

func (c *discoveryCache) set(userID string, conn ServerConn) {
	if conn.ResolvedAt.IsZero() {
		conn.ResolvedAt = time.Now()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[userID] = conn
}

func (c *discoveryCache) invalidate(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, userID)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./plex/... -run TestDiscoveryCache -v`
Expected: 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/plex/cache.go backend/plex/cache_test.go
git commit -m "feat(plex): add discovery cache with TTL"
```

---

## Task 3: PMS resource discovery

**Files:**
- Create: `backend/plex/discover.go`
- Create: `backend/plex/discover_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package plex

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const resourcesPrefersLocalFixture = `[
  {
    "name": "Home Server",
    "clientIdentifier": "abc123",
    "provides": "server",
    "owned": true,
    "connections": [
      {"protocol": "https", "address": "1.2.3.4", "port": 32400, "uri": "https://relay.example", "local": false, "relay": true},
      {"protocol": "https", "address": "192.168.1.10", "port": 32400, "uri": "https://192-168-1-10.plex.direct:32400", "local": true, "relay": false},
      {"protocol": "https", "address": "5.6.7.8", "port": 32400, "uri": "https://5-6-7-8.plex.direct:32400", "local": false, "relay": false}
    ]
  }
]`

const resourcesPrefersHttpsDirectFixture = `[
  {
    "name": "Home Server",
    "clientIdentifier": "abc123",
    "provides": "server",
    "owned": true,
    "connections": [
      {"protocol": "https", "address": "1.2.3.4", "port": 32400, "uri": "https://relay.example", "local": false, "relay": true},
      {"protocol": "https", "address": "5.6.7.8", "port": 32400, "uri": "https://5-6-7-8.plex.direct:32400", "local": false, "relay": false}
    ]
  }
]`

const resourcesRelayOnlyFixture = `[
  {
    "name": "Home Server",
    "clientIdentifier": "abc123",
    "provides": "server",
    "owned": true,
    "connections": [
      {"protocol": "https", "address": "1.2.3.4", "port": 32400, "uri": "https://relay.example", "local": false, "relay": true}
    ]
  }
]`

const resourcesNotOwnedFixture = `[
  {"name": "Friend Server", "clientIdentifier": "x", "provides": "server", "owned": false, "connections": []},
  {
    "name": "My Server",
    "clientIdentifier": "abc123",
    "provides": "server",
    "owned": true,
    "connections": [
      {"protocol": "https", "address": "5.6.7.8", "port": 32400, "uri": "https://5-6-7-8.plex.direct:32400", "local": false, "relay": false}
    ]
  }
]`

const resourcesNoServersFixture = `[
  {"name": "Plex Cloud", "clientIdentifier": "x", "provides": "client,player", "owned": true, "connections": []}
]`

func newResourcesServer(t *testing.T, body string, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Plex-Token") == "" {
			http.Error(w, "missing token", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

func TestResolveServer_PrefersLocalHTTPS(t *testing.T) {
	srv := newResourcesServer(t, resourcesPrefersLocalFixture, http.StatusOK)
	defer srv.Close()

	d := newDiscoverer(srv.URL, http.DefaultClient, newDiscoveryCache(time.Minute))
	conn, err := d.resolve("user-1", "token-1")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !strings.Contains(conn.BaseURL, "192-168-1-10") {
		t.Fatalf("expected local URL, got %q", conn.BaseURL)
	}
}

func TestResolveServer_PrefersPlexDirectOverRelay(t *testing.T) {
	srv := newResourcesServer(t, resourcesPrefersHttpsDirectFixture, http.StatusOK)
	defer srv.Close()

	d := newDiscoverer(srv.URL, http.DefaultClient, newDiscoveryCache(time.Minute))
	conn, err := d.resolve("user-1", "token-1")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !strings.Contains(conn.BaseURL, "5-6-7-8") {
		t.Fatalf("expected plex.direct URL, got %q", conn.BaseURL)
	}
}

func TestResolveServer_FallsBackToRelay(t *testing.T) {
	srv := newResourcesServer(t, resourcesRelayOnlyFixture, http.StatusOK)
	defer srv.Close()

	d := newDiscoverer(srv.URL, http.DefaultClient, newDiscoveryCache(time.Minute))
	conn, err := d.resolve("user-1", "token-1")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !strings.Contains(conn.BaseURL, "relay.example") {
		t.Fatalf("expected relay URL, got %q", conn.BaseURL)
	}
}

func TestResolveServer_SkipsUnownedResources(t *testing.T) {
	srv := newResourcesServer(t, resourcesNotOwnedFixture, http.StatusOK)
	defer srv.Close()

	d := newDiscoverer(srv.URL, http.DefaultClient, newDiscoveryCache(time.Minute))
	conn, err := d.resolve("user-1", "token-1")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if conn.MachineIdentifier != "abc123" {
		t.Fatalf("expected owned server, got identifier %q", conn.MachineIdentifier)
	}
}

func TestResolveServer_NoEligibleServer(t *testing.T) {
	srv := newResourcesServer(t, resourcesNoServersFixture, http.StatusOK)
	defer srv.Close()

	d := newDiscoverer(srv.URL, http.DefaultClient, newDiscoveryCache(time.Minute))
	_, err := d.resolve("user-1", "token-1")
	if err != ErrServerUnreachable {
		t.Fatalf("expected ErrServerUnreachable, got %v", err)
	}
}

func TestResolveServer_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	d := newDiscoverer(srv.URL, http.DefaultClient, newDiscoveryCache(time.Minute))
	_, err := d.resolve("user-1", "token-1")
	if err != ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestResolveServer_UsesCacheOnSecondCall(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(resourcesPrefersHttpsDirectFixture))
	}))
	defer srv.Close()

	d := newDiscoverer(srv.URL, http.DefaultClient, newDiscoveryCache(time.Minute))
	if _, err := d.resolve("user-1", "token-1"); err != nil {
		t.Fatalf("first resolve: %v", err)
	}
	if _, err := d.resolve("user-1", "token-1"); err != nil {
		t.Fatalf("second resolve: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 upstream call, got %d", calls)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./plex/... -run TestResolveServer`
Expected: FAIL — `newDiscoverer` undefined.

- [ ] **Step 3: Implement discover.go**

```go
package plex

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const defaultResourcesURL = "https://plex.tv/api/v2/resources?includeHttps=1&includeRelay=1"

type plexConnection struct {
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
	URI      string `json:"uri"`
	Local    bool   `json:"local"`
	Relay    bool   `json:"relay"`
}

type plexResource struct {
	Name             string           `json:"name"`
	ClientIdentifier string           `json:"clientIdentifier"`
	Provides         string           `json:"provides"`
	Owned            bool             `json:"owned"`
	Connections      []plexConnection `json:"connections"`
}

type discoverer struct {
	resourcesURL string
	httpClient   *http.Client
	cache        *discoveryCache
}

func newDiscoverer(resourcesURL string, httpClient *http.Client, cache *discoveryCache) *discoverer {
	if resourcesURL == "" {
		resourcesURL = defaultResourcesURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &discoverer{
		resourcesURL: resourcesURL,
		httpClient:   httpClient,
		cache:        cache,
	}
}

func (d *discoverer) resolve(userID, token string) (ServerConn, error) {
	if conn, ok := d.cache.get(userID); ok {
		return conn, nil
	}

	req, err := http.NewRequest(http.MethodGet, d.resourcesURL, nil)
	if err != nil {
		return ServerConn{}, fmt.Errorf("build resources request: %w", err)
	}
	req.Header.Set("X-Plex-Token", token)
	req.Header.Set("Accept", "application/json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return ServerConn{}, ErrServerUnreachable
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return ServerConn{}, ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		return ServerConn{}, ErrServerUnreachable
	}

	var resources []plexResource
	if err := json.NewDecoder(resp.Body).Decode(&resources); err != nil {
		return ServerConn{}, ErrServerUnreachable
	}

	for _, r := range resources {
		if !r.Owned || !containsServer(r.Provides) {
			continue
		}
		uri := pickConnection(r.Connections)
		if uri == "" {
			continue
		}
		conn := ServerConn{
			BaseURL:           uri,
			MachineIdentifier: r.ClientIdentifier,
			ResolvedAt:        time.Now(),
		}
		d.cache.set(userID, conn)
		return conn, nil
	}

	return ServerConn{}, ErrServerUnreachable
}

func containsServer(provides string) bool {
	for _, p := range splitCSV(provides) {
		if p == "server" {
			return true
		}
	}
	return false
}

func splitCSV(s string) []string {
	out := []string{}
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return out
}

// pickConnection applies the connection-selection priority:
//  1. local + https
//  2. https + !relay (plex.direct)
//  3. relay
func pickConnection(cs []plexConnection) string {
	for _, c := range cs {
		if c.Local && c.Protocol == "https" {
			return c.URI
		}
	}
	for _, c := range cs {
		if c.Protocol == "https" && !c.Relay {
			return c.URI
		}
	}
	for _, c := range cs {
		if c.Relay {
			return c.URI
		}
	}
	return ""
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./plex/... -run TestResolveServer -v`
Expected: 7 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/plex/discover.go backend/plex/discover_test.go
git commit -m "feat(plex): PMS discovery with connection selection and cache"
```

---

## Task 4: PlexClient.ListMovies — single library

**Files:**
- Create: `backend/plex/client.go`
- Create: `backend/plex/client_test.go`

- [ ] **Step 1: Write the failing test**

```go
package plex

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// sectionsOneMovieLibFixture: a server with one Movies and one Shows library
const sectionsOneMovieLibFixture = `{
  "MediaContainer": {
    "Directory": [
      {"key": "1", "type": "movie", "title": "Movies"},
      {"key": "2", "type": "show",  "title": "TV"}
    ]
  }
}`

const moviesPage1Fixture = `{
  "MediaContainer": {
    "size": 2,
    "totalSize": 3,
    "offset": 0,
    "Metadata": [
      {"ratingKey": "10", "title": "Movie A", "year": 2020, "thumb": "/library/metadata/10/thumb/1", "art": "/library/metadata/10/art/1", "rating": 7.5, "audienceRating": 8.0, "duration": 5400000, "addedAt": 1700000000, "summary": "A movie."},
      {"ratingKey": "11", "title": "Movie B", "year": 2021, "thumb": "/library/metadata/11/thumb/1", "art": "/library/metadata/11/art/1", "rating": 6.0, "audienceRating": 7.0, "duration": 6000000, "addedAt": 1700000100, "summary": "B."}
    ]
  }
}`

const moviesPage2Fixture = `{
  "MediaContainer": {
    "size": 1,
    "totalSize": 3,
    "offset": 2,
    "Metadata": [
      {"ratingKey": "12", "title": "Movie C", "year": 2022, "thumb": "/library/metadata/12/thumb/1", "addedAt": 1700000200, "summary": "C."}
    ]
  }
}`

// fakePMS routes /library/sections and /library/sections/:k/all responses.
func fakePMS(t *testing.T, handlers map[string]func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for path, h := range handlers {
		mux.HandleFunc(path, h)
	}
	return httptest.NewServer(mux)
}

func TestListMovies_SingleLibrary_FirstPage(t *testing.T) {
	pms := fakePMS(t, map[string]func(w http.ResponseWriter, r *http.Request){
		"/library/sections": func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Plex-Token") != "tok" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(sectionsOneMovieLibFixture))
		},
		"/library/sections/1/all": func(w http.ResponseWriter, r *http.Request) {
			if got := r.URL.Query().Get("X-Plex-Container-Start"); got != "0" {
				t.Errorf("start: got %q, want 0", got)
			}
			if got := r.URL.Query().Get("X-Plex-Container-Size"); got != "2" {
				t.Errorf("size: got %q, want 2", got)
			}
			if got := r.URL.Query().Get("sort"); got != "addedAt:desc" {
				t.Errorf("sort: got %q, want addedAt:desc", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(moviesPage1Fixture))
		},
	})
	defer pms.Close()

	client := newClient(http.DefaultClient)
	conn := ServerConn{BaseURL: pms.URL, ResolvedAt: time.Now()}

	res, err := client.ListMovies(conn, "tok", 0, 2, "addedAt:desc")
	if err != nil {
		t.Fatalf("ListMovies: %v", err)
	}
	if res.Total != 3 {
		t.Errorf("total: got %d, want 3", res.Total)
	}
	if len(res.Items) != 2 {
		t.Fatalf("items: got %d, want 2", len(res.Items))
	}
	if res.Items[0].RatingKey != "10" || res.Items[0].Title != "Movie A" {
		t.Errorf("item[0]: got %+v", res.Items[0])
	}
	if res.Items[0].Duration != 5400000 || res.Items[0].AddedAt != 1700000000 {
		t.Errorf("item[0] duration/addedAt: got %+v", res.Items[0])
	}
}

func TestListMovies_Unauthorized(t *testing.T) {
	pms := fakePMS(t, map[string]func(w http.ResponseWriter, r *http.Request){
		"/library/sections": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		},
	})
	defer pms.Close()

	client := newClient(http.DefaultClient)
	conn := ServerConn{BaseURL: pms.URL, ResolvedAt: time.Now()}

	_, err := client.ListMovies(conn, "tok", 0, 50, "addedAt:desc")
	if err != ErrUnauthorized {
		t.Fatalf("got %v, want ErrUnauthorized", err)
	}
}

func TestListMovies_NoMovieLibrary_EmptyResult(t *testing.T) {
	pms := fakePMS(t, map[string]func(w http.ResponseWriter, r *http.Request){
		"/library/sections": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"MediaContainer":{"Directory":[{"key":"1","type":"show"}]}}`))
		},
	})
	defer pms.Close()

	client := newClient(http.DefaultClient)
	conn := ServerConn{BaseURL: pms.URL, ResolvedAt: time.Now()}

	res, err := client.ListMovies(conn, "tok", 0, 50, "addedAt:desc")
	if err != nil {
		t.Fatalf("ListMovies: %v", err)
	}
	if res.Total != 0 || len(res.Items) != 0 {
		t.Fatalf("want empty, got total=%d items=%d", res.Total, len(res.Items))
	}
}

// moviesPage2Fixture is consumed by tests added in Task 5; keep it defined here
// so the multi-library tests can reference it without restating fixture data.
var _ = moviesPage2Fixture
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./plex/... -run TestListMovies`
Expected: FAIL — `newClient` undefined.

- [ ] **Step 3: Implement client.go (single-library path)**

```go
package plex

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	clientIdentifier = "torrent-ui"
	clientProduct    = "TorrentUI"
)

type PlexClient struct {
	httpClient *http.Client
	discoverer *discoverer
}

func newClient(httpClient *http.Client) *PlexClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &PlexClient{httpClient: httpClient}
}

// New constructs a fully-wired client with cached discovery.
func New(httpClient *http.Client) *PlexClient {
	c := newClient(httpClient)
	c.discoverer = newDiscoverer("", httpClient, newDiscoveryCache(5*time.Minute))
	return c
}

// ResolveServer returns the user's PMS connection. Used by handlers.
func (c *PlexClient) ResolveServer(userID, token string) (ServerConn, error) {
	if c.discoverer == nil {
		return ServerConn{}, ErrServerUnreachable
	}
	return c.discoverer.resolve(userID, token)
}

// InvalidateServer drops the cached PMS for a user (e.g. on 401).
func (c *PlexClient) InvalidateServer(userID string) {
	if c.discoverer != nil {
		c.discoverer.cache.invalidate(userID)
	}
}

type mediaContainer struct {
	Size      int             `json:"size"`
	TotalSize int             `json:"totalSize"`
	Offset    int             `json:"offset"`
	Directory []sectionEntry  `json:"Directory"`
	Metadata  []metadataEntry `json:"Metadata"`
}

type sectionsResponse struct {
	MediaContainer mediaContainer `json:"MediaContainer"`
}

type sectionEntry struct {
	Key   string `json:"key"`
	Type  string `json:"type"`
	Title string `json:"title"`
}

type metadataEntry struct {
	RatingKey      string         `json:"ratingKey"`
	Title          string         `json:"title"`
	Year           int            `json:"year"`
	Thumb          string         `json:"thumb"`
	Art            string         `json:"art"`
	Rating         float64        `json:"rating"`
	AudienceRating float64        `json:"audienceRating"`
	Duration       int64          `json:"duration"`
	AddedAt        int64          `json:"addedAt"`
	Summary        string         `json:"summary"`
	ContentRating  string         `json:"contentRating"`
	Studio         string         `json:"studio"`
	OriginallyAt   string         `json:"originallyAvailableAt"`
	Genre          []taggedEntry  `json:"Genre"`
	Director       []taggedEntry  `json:"Director"`
	Writer         []taggedEntry  `json:"Writer"`
	Role           []taggedEntry  `json:"Role"`
}

type taggedEntry struct {
	Tag string `json:"tag"`
}

// ListMovies queries movie libraries on the user's PMS. v1: paginates within
// the first movie library; if exhausted, advances to the next library.
func (c *PlexClient) ListMovies(conn ServerConn, token string, start, size int, sort string) (ListMoviesResult, error) {
	libs, err := c.listMovieLibraries(conn, token)
	if err != nil {
		return ListMoviesResult{}, err
	}
	if len(libs) == 0 {
		return ListMoviesResult{Items: []Movie{}, Total: 0, Start: start, Size: 0}, nil
	}

	// Single library: just query it.
	if len(libs) == 1 {
		return c.queryLibraryPage(conn, token, libs[0].Key, start, size, sort)
	}
	return c.queryAcrossLibraries(conn, token, libs, start, size, sort)
}

func (c *PlexClient) listMovieLibraries(conn ServerConn, token string) ([]sectionEntry, error) {
	var resp sectionsResponse
	if err := c.getJSON(conn.BaseURL+"/library/sections", token, nil, &resp); err != nil {
		return nil, err
	}
	out := []sectionEntry{}
	for _, d := range resp.MediaContainer.Directory {
		if d.Type == "movie" {
			out = append(out, d)
		}
	}
	return out, nil
}

func (c *PlexClient) queryLibraryPage(conn ServerConn, token, libKey string, start, size int, sort string) (ListMoviesResult, error) {
	q := url.Values{}
	q.Set("type", "1")
	q.Set("X-Plex-Container-Start", strconv.Itoa(start))
	q.Set("X-Plex-Container-Size", strconv.Itoa(size))
	q.Set("sort", sort)

	var resp sectionsResponse
	endpoint := fmt.Sprintf("%s/library/sections/%s/all", conn.BaseURL, libKey)
	if err := c.getJSON(endpoint, token, q, &resp); err != nil {
		return ListMoviesResult{}, err
	}
	items := make([]Movie, 0, len(resp.MediaContainer.Metadata))
	for _, m := range resp.MediaContainer.Metadata {
		items = append(items, toMovie(m))
	}
	return ListMoviesResult{
		Items: items,
		Total: resp.MediaContainer.TotalSize,
		Start: start,
		Size:  len(items),
	}, nil
}

// queryAcrossLibraries is implemented in Task 5.
func (c *PlexClient) queryAcrossLibraries(conn ServerConn, token string, libs []sectionEntry, start, size int, sort string) (ListMoviesResult, error) {
	return ListMoviesResult{}, fmt.Errorf("multi-library not implemented yet")
}

func (c *PlexClient) getJSON(endpoint, token string, query url.Values, out interface{}) error {
	if query != nil && len(query) > 0 {
		endpoint = endpoint + "?" + query.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("X-Plex-Token", token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Plex-Client-Identifier", clientIdentifier)
	req.Header.Set("X-Plex-Product", clientProduct)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ErrServerUnreachable
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return ErrServerUnreachable
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return ErrServerUnreachable
	}
	return nil
}

func toMovie(m metadataEntry) Movie {
	return Movie{
		RatingKey:      m.RatingKey,
		Title:          m.Title,
		Year:           m.Year,
		Thumb:          m.Thumb,
		Art:            m.Art,
		Rating:         m.Rating,
		AudienceRating: m.AudienceRating,
		Duration:       m.Duration,
		AddedAt:        m.AddedAt,
		Summary:        m.Summary,
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./plex/... -run TestListMovies -v`
Expected: 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/plex/client.go backend/plex/client_test.go
git commit -m "feat(plex): list movies for single library with pagination"
```

---

## Task 5: PlexClient.ListMovies — multi-library

**Files:**
- Modify: `backend/plex/client.go` (replace `queryAcrossLibraries`)
- Modify: `backend/plex/client_test.go` (add tests)

- [ ] **Step 1: Add failing test for multi-library**

Append to `backend/plex/client_test.go`:

```go
const sectionsTwoMovieLibsFixture = `{
  "MediaContainer": {
    "Directory": [
      {"key": "1", "type": "movie", "title": "Movies"},
      {"key": "2", "type": "movie", "title": "Anime"}
    ]
  }
}`

// Library 1 has 3 items (offsets 0,1,2). Library 2 has 2 items (offsets 0,1).
// Page with start=2, size=2 should pull last item of lib 1 + first item of lib 2.

func TestListMovies_MultiLibrary_PageSpansBoundary(t *testing.T) {
	pms := fakePMS(t, map[string]func(w http.ResponseWriter, r *http.Request){
		"/library/sections": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(sectionsTwoMovieLibsFixture))
		},
		"/library/sections/1/all": func(w http.ResponseWriter, r *http.Request) {
			start := r.URL.Query().Get("X-Plex-Container-Start")
			size := r.URL.Query().Get("X-Plex-Container-Size")
			w.Header().Set("Content-Type", "application/json")
			// caller requests within lib 1: start=2, size=1 => returns item 102
			if start == "2" && size == "1" {
				_, _ = w.Write([]byte(`{"MediaContainer":{"size":1,"totalSize":3,"offset":2,"Metadata":[{"ratingKey":"102","title":"L1-C","year":2020,"addedAt":3}]}}`))
				return
			}
			t.Errorf("unexpected lib 1 query start=%s size=%s", start, size)
			w.WriteHeader(http.StatusInternalServerError)
		},
		"/library/sections/2/all": func(w http.ResponseWriter, r *http.Request) {
			start := r.URL.Query().Get("X-Plex-Container-Start")
			size := r.URL.Query().Get("X-Plex-Container-Size")
			w.Header().Set("Content-Type", "application/json")
			// caller fills remaining 1 item from lib 2: start=0, size=1 => returns item 200
			if start == "0" && size == "1" {
				_, _ = w.Write([]byte(`{"MediaContainer":{"size":1,"totalSize":2,"offset":0,"Metadata":[{"ratingKey":"200","title":"L2-A","year":2021,"addedAt":10}]}}`))
				return
			}
			t.Errorf("unexpected lib 2 query start=%s size=%s", start, size)
			w.WriteHeader(http.StatusInternalServerError)
		},
	})
	defer pms.Close()

	client := newClient(http.DefaultClient)
	conn := ServerConn{BaseURL: pms.URL, ResolvedAt: time.Now()}

	res, err := client.ListMovies(conn, "tok", 2, 2, "addedAt:desc")
	if err != nil {
		t.Fatalf("ListMovies: %v", err)
	}
	if res.Total != 5 {
		t.Errorf("total: got %d, want 5 (3+2)", res.Total)
	}
	if len(res.Items) != 2 {
		t.Fatalf("items: got %d, want 2", len(res.Items))
	}
	if res.Items[0].RatingKey != "102" || res.Items[1].RatingKey != "200" {
		t.Errorf("items: got [%s, %s], want [102, 200]", res.Items[0].RatingKey, res.Items[1].RatingKey)
	}
}

func TestListMovies_MultiLibrary_PastEnd(t *testing.T) {
	pms := fakePMS(t, map[string]func(w http.ResponseWriter, r *http.Request){
		"/library/sections": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(sectionsTwoMovieLibsFixture))
		},
		"/library/sections/1/all": func(w http.ResponseWriter, r *http.Request) {
			// any request: return totalSize=3, empty metadata when offset >= 3
			w.Header().Set("Content-Type", "application/json")
			start := r.URL.Query().Get("X-Plex-Container-Start")
			if start == "0" {
				_, _ = w.Write([]byte(`{"MediaContainer":{"size":0,"totalSize":3,"offset":0,"Metadata":[]}}`))
				return
			}
			_, _ = w.Write([]byte(`{"MediaContainer":{"size":0,"totalSize":3,"offset":99,"Metadata":[]}}`))
		},
		"/library/sections/2/all": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			start := r.URL.Query().Get("X-Plex-Container-Start")
			if start == "0" {
				_, _ = w.Write([]byte(`{"MediaContainer":{"size":0,"totalSize":2,"offset":0,"Metadata":[]}}`))
				return
			}
			_, _ = w.Write([]byte(`{"MediaContainer":{"size":0,"totalSize":2,"offset":99,"Metadata":[]}}`))
		},
	})
	defer pms.Close()

	client := newClient(http.DefaultClient)
	conn := ServerConn{BaseURL: pms.URL, ResolvedAt: time.Now()}

	res, err := client.ListMovies(conn, "tok", 10, 5, "addedAt:desc")
	if err != nil {
		t.Fatalf("ListMovies: %v", err)
	}
	if res.Total != 5 {
		t.Errorf("total: got %d, want 5", res.Total)
	}
	if len(res.Items) != 0 {
		t.Errorf("items: got %d, want 0", len(res.Items))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./plex/... -run TestListMovies_MultiLibrary`
Expected: FAIL — multi-library returns "not implemented".

- [ ] **Step 3: Replace `queryAcrossLibraries` in client.go**

Replace the stub with:

```go
// queryAcrossLibraries paginates virtually across multiple movie libraries
// by treating them as a concatenated stream in the order Plex returned them.
// First it sums totals across libraries (one /all?size=0 request each, cheap),
// then walks libraries until `size` items have been gathered starting at `start`.
func (c *PlexClient) queryAcrossLibraries(conn ServerConn, token string, libs []sectionEntry, start, size int, sort string) (ListMoviesResult, error) {
	libTotals := make([]int, len(libs))
	grandTotal := 0
	for i, lib := range libs {
		t, err := c.libraryTotal(conn, token, lib.Key, sort)
		if err != nil {
			return ListMoviesResult{}, err
		}
		libTotals[i] = t
		grandTotal += t
	}

	items := make([]Movie, 0, size)
	cursor := 0  // virtual offset across the concatenation
	remaining := size

	for i, lib := range libs {
		libStart := cursor       // virtual offset where this library begins
		libEnd := cursor + libTotals[i]
		cursor = libEnd

		if remaining <= 0 {
			break
		}
		if start >= libEnd {
			continue
		}
		// Translate virtual offsets to per-library offsets.
		localStart := 0
		if start > libStart {
			localStart = start - libStart
		}
		want := remaining
		available := libTotals[i] - localStart
		if want > available {
			want = available
		}
		if want <= 0 {
			continue
		}
		page, err := c.queryLibraryPage(conn, token, lib.Key, localStart, want, sort)
		if err != nil {
			return ListMoviesResult{}, err
		}
		items = append(items, page.Items...)
		remaining -= len(page.Items)
	}

	return ListMoviesResult{
		Items: items,
		Total: grandTotal,
		Start: start,
		Size:  len(items),
	}, nil
}

func (c *PlexClient) libraryTotal(conn ServerConn, token, libKey, sort string) (int, error) {
	q := url.Values{}
	q.Set("type", "1")
	q.Set("X-Plex-Container-Start", "0")
	q.Set("X-Plex-Container-Size", "0")
	q.Set("sort", sort)

	var resp sectionsResponse
	endpoint := fmt.Sprintf("%s/library/sections/%s/all", conn.BaseURL, libKey)
	if err := c.getJSON(endpoint, token, q, &resp); err != nil {
		return 0, err
	}
	return resp.MediaContainer.TotalSize, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./plex/... -run TestListMovies -v`
Expected: 5 tests PASS (3 single-library + 2 multi-library).

- [ ] **Step 5: Commit**

```bash
git add backend/plex/client.go backend/plex/client_test.go
git commit -m "feat(plex): paginate across multiple movie libraries"
```

---

## Task 6: PlexClient.GetMovie — detail

**Files:**
- Modify: `backend/plex/client.go`
- Modify: `backend/plex/client_test.go`

- [ ] **Step 1: Add failing test**

Append to `backend/plex/client_test.go`:

```go
const movieDetailFixture = `{
  "MediaContainer": {
    "Metadata": [{
      "ratingKey": "42",
      "title": "Detail Movie",
      "year": 2024,
      "thumb": "/library/metadata/42/thumb/1",
      "summary": "summary text",
      "duration": 7200000,
      "contentRating": "PG-13",
      "studio": "Studio X",
      "originallyAvailableAt": "2024-01-15",
      "Genre":    [{"tag":"Action"},{"tag":"Drama"}],
      "Director": [{"tag":"Jane Doe"}],
      "Writer":   [{"tag":"John Smith"}],
      "Role": [
        {"tag":"Actor 1"},{"tag":"Actor 2"},{"tag":"Actor 3"},
        {"tag":"Actor 4"},{"tag":"Actor 5"},{"tag":"Actor 6"},
        {"tag":"Actor 7"}
      ]
    }]
  }
}`

func TestGetMovie_ParsesDetail(t *testing.T) {
	pms := fakePMS(t, map[string]func(w http.ResponseWriter, r *http.Request){
		"/library/metadata/42": func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Plex-Token") != "tok" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(movieDetailFixture))
		},
	})
	defer pms.Close()

	client := newClient(http.DefaultClient)
	conn := ServerConn{BaseURL: pms.URL, ResolvedAt: time.Now()}

	d, err := client.GetMovie(conn, "tok", "42")
	if err != nil {
		t.Fatalf("GetMovie: %v", err)
	}
	if d.RatingKey != "42" || d.Title != "Detail Movie" {
		t.Errorf("basic fields: got %+v", d.Movie)
	}
	if len(d.Genres) != 2 || d.Genres[0] != "Action" {
		t.Errorf("genres: got %v", d.Genres)
	}
	if len(d.Directors) != 1 || d.Directors[0] != "Jane Doe" {
		t.Errorf("directors: got %v", d.Directors)
	}
	if len(d.Cast) != 6 {
		t.Errorf("cast: got %d entries, want top 6", len(d.Cast))
	}
	if d.ContentRating != "PG-13" || d.Studio != "Studio X" || d.OriginallyAvailableAt != "2024-01-15" {
		t.Errorf("extras: got %+v", d)
	}
}

func TestGetMovie_NotFound(t *testing.T) {
	pms := fakePMS(t, map[string]func(w http.ResponseWriter, r *http.Request){
		"/library/metadata/999": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		},
	})
	defer pms.Close()

	client := newClient(http.DefaultClient)
	conn := ServerConn{BaseURL: pms.URL, ResolvedAt: time.Now()}

	_, err := client.GetMovie(conn, "tok", "999")
	if err != ErrServerUnreachable {
		t.Fatalf("got %v, want ErrServerUnreachable", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./plex/... -run TestGetMovie`
Expected: FAIL — `GetMovie` undefined.

- [ ] **Step 3: Add GetMovie to client.go**

Append to `backend/plex/client.go`:

```go
// GetMovie returns the detail for a single movie by rating key.
func (c *PlexClient) GetMovie(conn ServerConn, token, ratingKey string) (MovieDetail, error) {
	var resp sectionsResponse
	endpoint := fmt.Sprintf("%s/library/metadata/%s", conn.BaseURL, ratingKey)
	if err := c.getJSON(endpoint, token, nil, &resp); err != nil {
		return MovieDetail{}, err
	}
	if len(resp.MediaContainer.Metadata) == 0 {
		return MovieDetail{}, ErrServerUnreachable
	}
	m := resp.MediaContainer.Metadata[0]

	d := MovieDetail{
		Movie:                 toMovie(m),
		ContentRating:         m.ContentRating,
		Studio:                m.Studio,
		OriginallyAvailableAt: m.OriginallyAt,
		Genres:                tagsOf(m.Genre),
		Directors:             tagsOf(m.Director),
		Writers:               tagsOf(m.Writer),
		Cast:                  tagsOf(m.Role),
	}
	if len(d.Cast) > 6 {
		d.Cast = d.Cast[:6]
	}
	return d, nil
}

func tagsOf(in []taggedEntry) []string {
	out := make([]string, 0, len(in))
	for _, t := range in {
		out = append(out, t.Tag)
	}
	return out
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./plex/... -run TestGetMovie -v`
Expected: 2 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/plex/client.go backend/plex/client_test.go
git commit -m "feat(plex): fetch single movie detail"
```

---

## Task 7: PlexClient.FetchImage

**Files:**
- Modify: `backend/plex/client.go`
- Modify: `backend/plex/client_test.go`

- [ ] **Step 1: Add failing test**

Append to `backend/plex/client_test.go`:

```go
func TestFetchImage_StreamsBytes(t *testing.T) {
	pms := fakePMS(t, map[string]func(w http.ResponseWriter, r *http.Request){
		"/library/metadata/10/thumb/1": func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Plex-Token") != "tok" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10})
		},
	})
	defer pms.Close()

	client := newClient(http.DefaultClient)
	conn := ServerConn{BaseURL: pms.URL, ResolvedAt: time.Now()}

	resp, err := client.FetchImage(conn, "tok", "/library/metadata/10/thumb/1")
	if err != nil {
		t.Fatalf("FetchImage: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if resp.Header.Get("Content-Type") != "image/jpeg" {
		t.Errorf("content-type: got %q", resp.Header.Get("Content-Type"))
	}
}

func TestFetchImage_Unauthorized(t *testing.T) {
	pms := fakePMS(t, map[string]func(w http.ResponseWriter, r *http.Request){
		"/library/metadata/10/thumb/1": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		},
	})
	defer pms.Close()

	client := newClient(http.DefaultClient)
	conn := ServerConn{BaseURL: pms.URL, ResolvedAt: time.Now()}

	_, err := client.FetchImage(conn, "tok", "/library/metadata/10/thumb/1")
	if err != ErrUnauthorized {
		t.Fatalf("got %v, want ErrUnauthorized", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./plex/... -run TestFetchImage`
Expected: FAIL — `FetchImage` undefined.

- [ ] **Step 3: Add FetchImage to client.go**

Append to `backend/plex/client.go`:

```go
// FetchImage returns an open HTTP response streaming the image bytes.
// Caller MUST close resp.Body.
func (c *PlexClient) FetchImage(conn ServerConn, token, path string) (*http.Response, error) {
	endpoint := conn.BaseURL + path
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build image request: %w", err)
	}
	req.Header.Set("X-Plex-Token", token)
	req.Header.Set("X-Plex-Client-Identifier", clientIdentifier)
	req.Header.Set("X-Plex-Product", clientProduct)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, ErrServerUnreachable
	}
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		return nil, ErrUnauthorized
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, ErrServerUnreachable
	}
	return resp, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./plex/... -v`
Expected: all plex tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/plex/client.go backend/plex/client_test.go
git commit -m "feat(plex): proxy poster/thumb images from PMS"
```

---

## Task 8: Handlers — gating + GET /plex/movies

**Files:**
- Create: `backend/plex/handlers.go`
- Create: `backend/plex/handlers_test.go`

- [ ] **Step 1: Write the failing test**

```go
package plex

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hasmikatom/torrent/db"
	"github.com/hasmikatom/torrent/integrations"
	"github.com/hasmikatom/torrent/middleware"
)

// fakeClient implements the handler-side interface for tests.
type fakeClient struct {
	resolveErr    error
	listResult    ListMoviesResult
	listErr       error
	movie         MovieDetail
	movieErr      error
	imageStatus   int
	imageBody     []byte
	imageErr      error
	lastImagePath string
	invalidated   bool

	// captured args from the most recent ListMovies call
	lastListStart int
	lastListSize  int
	lastListSort  string
}

func (f *fakeClient) ResolveServer(userID, token string) (ServerConn, error) {
	if f.resolveErr != nil {
		return ServerConn{}, f.resolveErr
	}
	return ServerConn{BaseURL: "http://fake", MachineIdentifier: "id"}, nil
}
func (f *fakeClient) InvalidateServer(userID string) { f.invalidated = true }
func (f *fakeClient) ListMovies(conn ServerConn, token string, start, size int, sort string) (ListMoviesResult, error) {
	f.lastListStart = start
	f.lastListSize = size
	f.lastListSort = sort
	if f.listErr != nil {
		return ListMoviesResult{}, f.listErr
	}
	return f.listResult, nil
}
func (f *fakeClient) GetMovie(conn ServerConn, token, ratingKey string) (MovieDetail, error) {
	if f.movieErr != nil {
		return MovieDetail{}, f.movieErr
	}
	return f.movie, nil
}
func (f *fakeClient) FetchImage(conn ServerConn, token, path string) (*http.Response, error) {
	f.lastImagePath = path
	if f.imageErr != nil {
		return nil, f.imageErr
	}
	rec := httptest.NewRecorder()
	rec.Code = f.imageStatus
	rec.Header().Set("Content-Type", "image/jpeg")
	_, _ = rec.Write(f.imageBody)
	return rec.Result(), nil
}

func setupHandlers(t *testing.T, fc *fakeClient, configurePlex func(*integrations.Store)) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	d, err := db.Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	t.Cleanup(func() { d.Close() })

	store := integrations.NewStore(d)
	if configurePlex != nil {
		configurePlex(store)
	}
	r := gin.New()
	g := r.Group("/", middleware.RequireUser())
	RegisterHandlers(g, store, fc)
	return r
}

func authed(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("X-User-Id", "user-1")
	req.Header.Set("X-User-Email", "u@example.com")
	return req
}

func TestMoviesHandler_PreconditionFailed_WhenNotConnected(t *testing.T) {
	r := setupHandlers(t, &fakeClient{}, nil) // no token saved

	w := httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodGet, "/plex/movies"))

	if w.Code != http.StatusPreconditionFailed {
		t.Fatalf("status: got %d, want 412 (body: %s)", w.Code, w.Body.String())
	}
	var body map[string]string
	_ = json.NewDecoder(w.Body).Decode(&body)
	if body["error"] != "plex_not_configured" {
		t.Errorf("error: got %q", body["error"])
	}
}

func TestMoviesHandler_ReturnsItems(t *testing.T) {
	fc := &fakeClient{
		listResult: ListMoviesResult{
			Items: []Movie{{RatingKey: "10", Title: "A", Year: 2020}},
			Total: 1, Start: 0, Size: 1,
		},
	}
	r := setupHandlers(t, fc, func(s *integrations.Store) {
		_ = s.UpsertPlex("user-1", "tok", true)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodGet, "/plex/movies?start=0&size=10&sort=titleSort:asc"))

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d (body: %s)", w.Code, w.Body.String())
	}
	var res ListMoviesResult
	_ = json.NewDecoder(w.Body).Decode(&res)
	if res.Total != 1 || len(res.Items) != 1 || res.Items[0].Title != "A" {
		t.Errorf("body: %+v", res)
	}
}

func TestMoviesHandler_Unauthorized(t *testing.T) {
	fc := &fakeClient{listErr: ErrUnauthorized}
	r := setupHandlers(t, fc, func(s *integrations.Store) {
		_ = s.UpsertPlex("user-1", "tok", true)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodGet, "/plex/movies"))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d", w.Code)
	}
	if !fc.invalidated {
		t.Error("expected cached server to be invalidated on 401")
	}
}

func TestMoviesHandler_UnreachableMapsTo502(t *testing.T) {
	fc := &fakeClient{listErr: ErrServerUnreachable}
	r := setupHandlers(t, fc, func(s *integrations.Store) {
		_ = s.UpsertPlex("user-1", "tok", true)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodGet, "/plex/movies"))

	if w.Code != http.StatusBadGateway {
		t.Fatalf("status: got %d", w.Code)
	}
}

func TestMoviesHandler_ClampsAndValidates(t *testing.T) {
	fc := &fakeClient{listResult: ListMoviesResult{Items: []Movie{}, Total: 0}}
	r := setupHandlers(t, fc, func(s *integrations.Store) {
		_ = s.UpsertPlex("user-1", "tok", true)
	})

	cases := []struct {
		name      string
		query     string
		wantStart int
		wantSize  int
		wantSort  string
	}{
		{"defaults", "", 0, 50, "addedAt:desc"},
		{"size capped at 200", "?size=500", 0, 200, "addedAt:desc"},
		{"negative start clamped to 0", "?start=-5", 0, 50, "addedAt:desc"},
		{"unknown sort falls back", "?sort=bogus:asc", 0, 50, "addedAt:desc"},
		{"valid sort respected", "?sort=titleSort:asc", 0, 50, "titleSort:asc"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, authed(http.MethodGet, "/plex/movies"+tc.query))
			if w.Code != http.StatusOK {
				t.Fatalf("status: got %d (body: %s)", w.Code, w.Body.String())
			}
			if fc.lastListStart != tc.wantStart {
				t.Errorf("start: got %d, want %d", fc.lastListStart, tc.wantStart)
			}
			if fc.lastListSize != tc.wantSize {
				t.Errorf("size: got %d, want %d", fc.lastListSize, tc.wantSize)
			}
			if fc.lastListSort != tc.wantSort {
				t.Errorf("sort: got %q, want %q", fc.lastListSort, tc.wantSort)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./plex/... -run TestMoviesHandler`
Expected: FAIL — `RegisterHandlers` undefined.

- [ ] **Step 3: Implement handlers.go**

```go
package plex

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hasmikatom/torrent/integrations"
)

// Client is the surface the handlers depend on. Both *PlexClient and test
// fakes implement it.
type Client interface {
	ResolveServer(userID, token string) (ServerConn, error)
	InvalidateServer(userID string)
	ListMovies(conn ServerConn, token string, start, size int, sort string) (ListMoviesResult, error)
	GetMovie(conn ServerConn, token, ratingKey string) (MovieDetail, error)
	FetchImage(conn ServerConn, token, path string) (*http.Response, error)
}

var validSorts = map[string]bool{
	"addedAt:desc":   true,
	"titleSort:asc":  true,
	"year:desc":      true,
	"rating:desc":    true,
}

const (
	defaultPageSize = 50
	maxPageSize     = 200
)

func RegisterHandlers(g *gin.RouterGroup, store *integrations.Store, client Client) {
	g.GET("/plex/movies", listMoviesHandler(store, client))
	// detail + image handlers added in later tasks
}

// resolveUserPlex pulls the user's token + resolves their PMS connection.
// Writes the appropriate error response and returns ok=false if anything
// is missing or unreachable.
func resolveUserPlex(c *gin.Context, store *integrations.Store, client Client) (string, ServerConn, bool) {
	userID := c.GetString("userId")
	row, err := store.GetIntegrations(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read integrations"})
		return "", ServerConn{}, false
	}
	if !row.PlexEnabled || row.PlexToken == "" {
		c.JSON(http.StatusPreconditionFailed, gin.H{"error": "plex_not_configured"})
		return "", ServerConn{}, false
	}
	conn, err := client.ResolveServer(userID, row.PlexToken)
	if err == ErrUnauthorized {
		client.InvalidateServer(userID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "plex_unauthorized"})
		return "", ServerConn{}, false
	}
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "plex_server_unreachable"})
		return "", ServerConn{}, false
	}
	return row.PlexToken, conn, true
}

func listMoviesHandler(store *integrations.Store, client Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, conn, ok := resolveUserPlex(c, store, client)
		if !ok {
			return
		}

		start, _ := strconv.Atoi(c.DefaultQuery("start", "0"))
		if start < 0 {
			start = 0
		}
		size, _ := strconv.Atoi(c.DefaultQuery("size", strconv.Itoa(defaultPageSize)))
		if size <= 0 {
			size = defaultPageSize
		}
		if size > maxPageSize {
			size = maxPageSize
		}
		sort := strings.TrimSpace(c.DefaultQuery("sort", "addedAt:desc"))
		if !validSorts[sort] {
			sort = "addedAt:desc"
		}

		res, err := client.ListMovies(conn, token, start, size, sort)
		if err == ErrUnauthorized {
			client.InvalidateServer(c.GetString("userId"))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "plex_unauthorized"})
			return
		}
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "plex_server_unreachable"})
			return
		}
		if res.Items == nil {
			res.Items = []Movie{}
		}
		c.JSON(http.StatusOK, res)
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./plex/... -run TestMoviesHandler -v`
Expected: 4 top-level tests PASS (the clamp test has 5 subtests, all PASS).

- [ ] **Step 5: Commit**

```bash
git add backend/plex/handlers.go backend/plex/handlers_test.go
git commit -m "feat(plex): /plex/movies handler with gating and sort validation"
```

---

## Task 9: Handler — GET /plex/movies/:ratingKey

**Files:**
- Modify: `backend/plex/handlers.go`
- Modify: `backend/plex/handlers_test.go`

- [ ] **Step 1: Add failing test**

Append to `backend/plex/handlers_test.go`:

```go
func TestMovieDetailHandler_ReturnsDetail(t *testing.T) {
	fc := &fakeClient{
		movie: MovieDetail{
			Movie:  Movie{RatingKey: "42", Title: "Movie", Year: 2024},
			Genres: []string{"Action"},
		},
	}
	r := setupHandlers(t, fc, func(s *integrations.Store) {
		_ = s.UpsertPlex("user-1", "tok", true)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodGet, "/plex/movies/42"))

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d (body: %s)", w.Code, w.Body.String())
	}
	var d MovieDetail
	_ = json.NewDecoder(w.Body).Decode(&d)
	if d.RatingKey != "42" || d.Genres[0] != "Action" {
		t.Errorf("body: %+v", d)
	}
}

func TestMovieDetailHandler_NotConfigured(t *testing.T) {
	r := setupHandlers(t, &fakeClient{}, nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodGet, "/plex/movies/42"))

	if w.Code != http.StatusPreconditionFailed {
		t.Fatalf("status: got %d", w.Code)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./plex/... -run TestMovieDetailHandler`
Expected: FAIL — 404 from gin (route not registered).

- [ ] **Step 3: Wire the handler in handlers.go**

Modify `RegisterHandlers` and add `movieDetailHandler`:

```go
func RegisterHandlers(g *gin.RouterGroup, store *integrations.Store, client Client) {
	g.GET("/plex/movies", listMoviesHandler(store, client))
	g.GET("/plex/movies/:ratingKey", movieDetailHandler(store, client))
	// image handler added in next task
}

func movieDetailHandler(store *integrations.Store, client Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, conn, ok := resolveUserPlex(c, store, client)
		if !ok {
			return
		}

		ratingKey := c.Param("ratingKey")
		if ratingKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ratingKey required"})
			return
		}

		d, err := client.GetMovie(conn, token, ratingKey)
		if err == ErrUnauthorized {
			client.InvalidateServer(c.GetString("userId"))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "plex_unauthorized"})
			return
		}
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "plex_server_unreachable"})
			return
		}
		c.JSON(http.StatusOK, d)
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./plex/... -run TestMovieDetailHandler -v`
Expected: 2 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/plex/handlers.go backend/plex/handlers_test.go
git commit -m "feat(plex): /plex/movies/:ratingKey detail handler"
```

---

## Task 10: Handler — GET /plex/image with SSRF guard

**Files:**
- Modify: `backend/plex/handlers.go`
- Modify: `backend/plex/handlers_test.go`

- [ ] **Step 1: Add failing test**

Append to `backend/plex/handlers_test.go`:

```go
func TestImageHandler_ProxiesBytes(t *testing.T) {
	fc := &fakeClient{
		imageStatus: 200,
		imageBody:   []byte{0xDE, 0xAD, 0xBE, 0xEF},
	}
	r := setupHandlers(t, fc, func(s *integrations.Store) {
		_ = s.UpsertPlex("user-1", "tok", true)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodGet, "/plex/image?path=/library/metadata/10/thumb/1"))

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d (body: %s)", w.Code, w.Body.String())
	}
	if w.Header().Get("Content-Type") != "image/jpeg" {
		t.Errorf("content-type: got %q", w.Header().Get("Content-Type"))
	}
	if w.Header().Get("Cache-Control") != "public, max-age=86400" {
		t.Errorf("cache-control: got %q", w.Header().Get("Cache-Control"))
	}
	if fc.lastImagePath != "/library/metadata/10/thumb/1" {
		t.Errorf("path passed to client: got %q", fc.lastImagePath)
	}
	if len(w.Body.Bytes()) != 4 {
		t.Errorf("body bytes: got %d, want 4", len(w.Body.Bytes()))
	}
}

func TestImageHandler_RejectsBadPath(t *testing.T) {
	r := setupHandlers(t, &fakeClient{}, func(s *integrations.Store) {
		_ = s.UpsertPlex("user-1", "tok", true)
	})

	cases := []string{
		"",
		"/etc/passwd",
		"http://evil.example/x",
		"//evil.example/x",
		"library/metadata/10/thumb/1",
	}
	for _, p := range cases {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, authed(http.MethodGet, "/plex/image?path="+p))
		if w.Code != http.StatusBadRequest {
			t.Errorf("path %q: got %d, want 400", p, w.Code)
		}
	}
}

func TestImageHandler_NotConfigured(t *testing.T) {
	r := setupHandlers(t, &fakeClient{}, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodGet, "/plex/image?path=/library/metadata/10/thumb/1"))
	if w.Code != http.StatusPreconditionFailed {
		t.Fatalf("status: got %d", w.Code)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./plex/... -run TestImageHandler`
Expected: FAIL — route not registered.

- [ ] **Step 3: Add image handler to handlers.go**

Update `RegisterHandlers` and add the handler:

```go
func RegisterHandlers(g *gin.RouterGroup, store *integrations.Store, client Client) {
	g.GET("/plex/movies", listMoviesHandler(store, client))
	g.GET("/plex/movies/:ratingKey", movieDetailHandler(store, client))
	g.GET("/plex/image", imageHandler(store, client))
}

func imageHandler(store *integrations.Store, client Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Query("path")
		if !strings.HasPrefix(path, "/library/metadata/") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
			return
		}

		token, conn, ok := resolveUserPlex(c, store, client)
		if !ok {
			return
		}

		resp, err := client.FetchImage(conn, token, path)
		if err == ErrUnauthorized {
			client.InvalidateServer(c.GetString("userId"))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "plex_unauthorized"})
			return
		}
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "plex_server_unreachable"})
			return
		}
		defer resp.Body.Close()

		ct := resp.Header.Get("Content-Type")
		if ct == "" {
			ct = "image/jpeg"
		}
		c.Header("Content-Type", ct)
		c.Header("Cache-Control", "public, max-age=86400")
		c.Status(http.StatusOK)
		_, _ = io.Copy(c.Writer, resp.Body)
	}
}
```

Add `"io"` to the import block at the top of handlers.go.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./plex/... -v`
Expected: all plex tests PASS (handlers + client + discover + cache).

- [ ] **Step 5: Commit**

```bash
git add backend/plex/handlers.go backend/plex/handlers_test.go
git commit -m "feat(plex): /plex/image proxy with SSRF guard and cache headers"
```

---

## Task 11: Wire up plex package in main.go

**Files:**
- Modify: `backend/main.go`

- [ ] **Step 1: Modify backend/main.go**

Add to the import block:

```go
"github.com/hasmikatom/torrent/plex"
```

After `integrations.RegisterHandlers(api, intStore, "")` (line ~101), add:

```go
plexClient := plex.New(&http.Client{Timeout: 15 * time.Second})
plex.RegisterHandlers(api, intStore, plexClient)
```

The relevant section becomes:

```go
intStore := integrations.NewStore(backendDB)
integrations.RegisterHandlers(api, intStore, "")

plexClient := plex.New(&http.Client{Timeout: 15 * time.Second})
plex.RegisterHandlers(api, intStore, plexClient)
```

- [ ] **Step 2: Verify the backend builds**

Run: `cd backend && go build ./...`
Expected: no output, exit 0.

- [ ] **Step 3: Verify all tests still pass**

Run: `cd backend && go test ./...`
Expected: all tests PASS.

- [ ] **Step 4: Commit**

```bash
git add backend/main.go
git commit -m "feat(plex): wire plex handlers into backend main"
```

---

## Task 12: Frontend useIntegrations hook

**Files:**
- Create: `frontend/src/hooks/useIntegrations.ts`

- [ ] **Step 1: Create the hook**

```ts
import { useEffect, useState } from "react";
import { apiFetch } from "@/services";

export type IntegrationState = {
  plexEnabled: boolean;
  plexHasToken: boolean;
};

const defaultState: IntegrationState = {
  plexEnabled: false,
  plexHasToken: false,
};

let cached: IntegrationState | null = null;
let inFlight: Promise<IntegrationState> | null = null;
const subscribers = new Set<(s: IntegrationState) => void>();

async function load(): Promise<IntegrationState> {
  if (cached) return cached;
  if (!inFlight) {
    inFlight = apiFetch("/api/integrations").then(async (res) => {
      if (!res.ok) return defaultState;
      const data = (await res.json()) as IntegrationState;
      cached = data;
      subscribers.forEach((cb) => cb(data));
      return data;
    });
  }
  return inFlight;
}

export function refreshIntegrations() {
  cached = null;
  inFlight = null;
  load();
}

export function useIntegrations(): {
  state: IntegrationState;
  loading: boolean;
  refresh: () => void;
} {
  const [state, setState] = useState<IntegrationState>(cached ?? defaultState);
  const [loading, setLoading] = useState(cached === null);

  useEffect(() => {
    let alive = true;
    const sub = (s: IntegrationState) => {
      if (alive) setState(s);
    };
    subscribers.add(sub);
    if (cached) {
      setState(cached);
      setLoading(false);
    } else {
      load().then((s) => {
        if (alive) {
          setState(s);
          setLoading(false);
        }
      });
    }
    return () => {
      alive = false;
      subscribers.delete(sub);
    };
  }, []);

  return { state, loading, refresh: refreshIntegrations };
}
```

- [ ] **Step 2: Verify it typechecks**

Run: `cd frontend && pnpm build`
Expected: build succeeds (the hook isn't yet used; tsc may warn about unused exports — fine).

- [ ] **Step 3: Commit**

```bash
git add frontend/src/hooks/useIntegrations.ts
git commit -m "feat(movies): useIntegrations hook with module-scoped cache"
```

---

## Task 13: Frontend types and API client

**Files:**
- Create: `frontend/src/components/movies/types.ts`
- Create: `frontend/src/components/movies/api.ts`

- [ ] **Step 1: Create types.ts**

```ts
export type Movie = {
  ratingKey: string;
  title: string;
  year: number;
  thumb: string;
  art: string;
  rating: number;
  audienceRating: number;
  duration: number;
  addedAt: number;
  summary: string;
};

export type MovieDetail = Movie & {
  contentRating: string;
  studio: string;
  originallyAvailableAt: string;
  genres: string[];
  directors: string[];
  writers: string[];
  cast: string[];
};

export type ListMoviesResult = {
  items: Movie[];
  total: number;
  start: number;
  size: number;
};

export type SortKey =
  | "addedAt:desc"
  | "titleSort:asc"
  | "year:desc"
  | "rating:desc";

export const SORT_OPTIONS: { key: SortKey; label: string }[] = [
  { key: "addedAt:desc", label: "Recently Added" },
  { key: "titleSort:asc", label: "Title A–Z" },
  { key: "year:desc", label: "Year (Newest)" },
  { key: "rating:desc", label: "Rating" },
];
```

- [ ] **Step 2: Create api.ts**

```ts
import { apiFetch } from "@/services";
import type { ListMoviesResult, MovieDetail, SortKey } from "./types";

export type FetchError =
  | { kind: "not_configured" }
  | { kind: "unauthorized" }
  | { kind: "unreachable" }
  | { kind: "unknown"; status: number };

async function asFetchError(res: Response): Promise<FetchError> {
  let body: { error?: string } = {};
  try {
    body = await res.json();
  } catch {
    // ignore
  }
  if (res.status === 412 || body.error === "plex_not_configured") return { kind: "not_configured" };
  if (res.status === 401 || body.error === "plex_unauthorized") return { kind: "unauthorized" };
  if (res.status === 502 || body.error === "plex_server_unreachable") return { kind: "unreachable" };
  return { kind: "unknown", status: res.status };
}

export async function fetchMovies(start: number, size: number, sort: SortKey): Promise<
  { ok: true; data: ListMoviesResult } | { ok: false; error: FetchError }
> {
  const qs = new URLSearchParams({ start: String(start), size: String(size), sort });
  const res = await apiFetch(`/api/plex/movies?${qs.toString()}`);
  if (!res.ok) return { ok: false, error: await asFetchError(res) };
  const data = (await res.json()) as ListMoviesResult;
  return { ok: true, data };
}

export async function fetchMovieDetail(ratingKey: string): Promise<
  { ok: true; data: MovieDetail } | { ok: false; error: FetchError }
> {
  const res = await apiFetch(`/api/plex/movies/${encodeURIComponent(ratingKey)}`);
  if (!res.ok) return { ok: false, error: await asFetchError(res) };
  const data = (await res.json()) as MovieDetail;
  return { ok: true, data };
}

export function imageUrl(path: string): string {
  return `/api/plex/image?path=${encodeURIComponent(path)}`;
}
```

- [ ] **Step 3: Verify it typechecks**

Run: `cd frontend && pnpm build`
Expected: build succeeds.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/movies/types.ts frontend/src/components/movies/api.ts
git commit -m "feat(movies): TS types and API client for /plex endpoints"
```

---

## Task 14: useMoviesQuery hook (pagination + sort)

**Files:**
- Create: `frontend/src/components/movies/useMoviesQuery.ts`

- [ ] **Step 1: Create the hook**

```ts
import { useCallback, useEffect, useRef, useState } from "react";
import { fetchMovies, FetchError } from "./api";
import type { Movie, SortKey } from "./types";

const PAGE_SIZE = 50;

export type MoviesQueryState = {
  items: Movie[];
  total: number;
  loading: boolean;
  error: FetchError | null;
  done: boolean;
  sort: SortKey;
  setSort: (s: SortKey) => void;
  loadMore: () => void;
  retry: () => void;
};

export function useMoviesQuery(): MoviesQueryState {
  const [items, setItems] = useState<Movie[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<FetchError | null>(null);
  const [sort, setSortInternal] = useState<SortKey>("addedAt:desc");
  const startRef = useRef(0);
  const requestIdRef = useRef(0);

  const fetchPage = useCallback(
    async (start: number, currentSort: SortKey, replace: boolean) => {
      const myId = ++requestIdRef.current;
      setLoading(true);
      setError(null);
      const res = await fetchMovies(start, PAGE_SIZE, currentSort);
      if (myId !== requestIdRef.current) return; // a newer request superseded us
      setLoading(false);
      if (!res.ok) {
        setError(res.error);
        return;
      }
      setTotal(res.data.total);
      setItems((prev) => (replace ? res.data.items : [...prev, ...res.data.items]));
      startRef.current = start + res.data.items.length;
    },
    []
  );

  useEffect(() => {
    startRef.current = 0;
    fetchPage(0, sort, true);
  }, [sort, fetchPage]);

  const loadMore = useCallback(() => {
    if (loading) return;
    if (items.length >= total) return;
    fetchPage(startRef.current, sort, false);
  }, [loading, items.length, total, sort, fetchPage]);

  const retry = useCallback(() => {
    fetchPage(startRef.current, sort, items.length === 0);
  }, [fetchPage, sort, items.length]);

  const setSort = useCallback((s: SortKey) => {
    setItems([]);
    setTotal(0);
    setSortInternal(s);
  }, []);

  return {
    items,
    total,
    loading,
    error,
    done: items.length >= total && total > 0,
    sort,
    setSort,
    loadMore,
    retry,
  };
}
```

- [ ] **Step 2: Verify it typechecks**

Run: `cd frontend && pnpm build`
Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/movies/useMoviesQuery.ts
git commit -m "feat(movies): useMoviesQuery hook with pagination + sort reset"
```

---

## Task 15: MovieCard component

**Files:**
- Create: `frontend/src/components/movies/MovieCard.tsx`

- [ ] **Step 1: Create the card**

```tsx
import { useState } from "react";
import { imageUrl } from "./api";
import type { Movie } from "./types";

type Props = {
  movie: Movie;
  onClick: (ratingKey: string) => void;
};

export function MovieCard({ movie, onClick }: Props) {
  const [imgFailed, setImgFailed] = useState(false);

  return (
    <button
      type="button"
      onClick={() => onClick(movie.ratingKey)}
      className="group flex flex-col text-left focus:outline-none focus:ring-2 focus:ring-ring rounded-md"
    >
      <div className="relative aspect-[2/3] w-full overflow-hidden rounded-md bg-muted shadow-sm">
        {!imgFailed && movie.thumb ? (
          <img
            src={imageUrl(movie.thumb)}
            alt={movie.title}
            loading="lazy"
            onError={() => setImgFailed(true)}
            className="h-full w-full object-cover transition-transform group-hover:scale-[1.03]"
          />
        ) : (
          <div className="flex h-full w-full items-center justify-center p-2 text-center text-xs text-muted-foreground">
            {movie.title}
          </div>
        )}
      </div>
      <div className="mt-2 line-clamp-1 text-sm font-medium" title={movie.title}>
        {movie.title}
      </div>
      {movie.year > 0 && (
        <div className="text-xs text-muted-foreground">{movie.year}</div>
      )}
    </button>
  );
}
```

- [ ] **Step 2: Verify typecheck**

Run: `cd frontend && pnpm build`
Expected: succeeds.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/movies/MovieCard.tsx
git commit -m "feat(movies): MovieCard with poster + title + year"
```

---

## Task 16: MovieGrid with skeleton/empty states

**Files:**
- Create: `frontend/src/components/movies/MovieGrid.tsx`

- [ ] **Step 1: Create the grid**

```tsx
import { Skeleton } from "@/components/ui/skeleton";
import { MovieCard } from "./MovieCard";
import type { Movie } from "./types";

type Props = {
  items: Movie[];
  loading: boolean;
  done: boolean;
  onCardClick: (ratingKey: string) => void;
  sentinelRef: React.RefObject<HTMLDivElement>;
};

export function MovieGrid({ items, loading, done, onCardClick, sentinelRef }: Props) {
  if (loading && items.length === 0) {
    return (
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
        {Array.from({ length: 12 }).map((_, i) => (
          <div key={i} className="flex flex-col gap-2">
            <Skeleton className="aspect-[2/3] w-full rounded-md" />
            <Skeleton className="h-4 w-3/4" />
            <Skeleton className="h-3 w-1/4" />
          </div>
        ))}
      </div>
    );
  }

  if (!loading && items.length === 0) {
    return (
      <div className="flex h-48 items-center justify-center text-muted-foreground">
        No movies found in your Plex libraries.
      </div>
    );
  }

  return (
    <>
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
        {items.map((m) => (
          <MovieCard key={m.ratingKey} movie={m} onClick={onCardClick} />
        ))}
      </div>
      <div ref={sentinelRef} className="flex h-12 items-center justify-center">
        {loading && (
          <div className="text-sm text-muted-foreground">Loading…</div>
        )}
        {done && (
          <div className="text-xs text-muted-foreground">End of library</div>
        )}
      </div>
    </>
  );
}
```

- [ ] **Step 2: Verify typecheck**

Run: `cd frontend && pnpm build`
Expected: succeeds.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/movies/MovieGrid.tsx
git commit -m "feat(movies): MovieGrid with skeleton + empty + infinite-scroll sentinel"
```

---

## Task 17: SortDropdown

**Files:**
- Create: `frontend/src/components/movies/SortDropdown.tsx`

- [ ] **Step 1: Create the dropdown**

```tsx
import { ArrowUpDown } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { SORT_OPTIONS, type SortKey } from "./types";

type Props = {
  value: SortKey;
  onChange: (v: SortKey) => void;
};

export function SortDropdown({ value, onChange }: Props) {
  const current = SORT_OPTIONS.find((o) => o.key === value) ?? SORT_OPTIONS[0];

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm">
          <ArrowUpDown className="mr-2 h-4 w-4" />
          {current.label}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuRadioGroup
          value={value}
          onValueChange={(v) => onChange(v as SortKey)}
        >
          {SORT_OPTIONS.map((o) => (
            <DropdownMenuRadioItem key={o.key} value={o.key}>
              {o.label}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
```

- [ ] **Step 2: Verify typecheck**

Run: `cd frontend && pnpm build`
Expected: succeeds.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/movies/SortDropdown.tsx
git commit -m "feat(movies): SortDropdown with 4 sort options"
```

---

## Task 18: MovieDetail dialog

**Files:**
- Create: `frontend/src/components/movies/MovieDetail.tsx`

- [ ] **Step 1: Create the detail dialog**

```tsx
import { useEffect, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { fetchMovieDetail, imageUrl, FetchError } from "./api";
import type { MovieDetail as MovieDetailT } from "./types";

type Props = {
  ratingKey: string | null;
  onClose: () => void;
};

function formatRuntime(ms: number): string {
  if (!ms) return "";
  const totalMin = Math.round(ms / 60000);
  const h = Math.floor(totalMin / 60);
  const m = totalMin % 60;
  if (h === 0) return `${m}m`;
  return `${h}h ${m}m`;
}

export function MovieDetail({ ratingKey, onClose }: Props) {
  const [data, setData] = useState<MovieDetailT | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<FetchError | null>(null);

  useEffect(() => {
    if (!ratingKey) return;
    setData(null);
    setError(null);
    setLoading(true);
    let alive = true;
    fetchMovieDetail(ratingKey).then((res) => {
      if (!alive) return;
      setLoading(false);
      if (res.ok) setData(res.data);
      else setError(res.error);
    });
    return () => {
      alive = false;
    };
  }, [ratingKey]);

  const open = ratingKey !== null;

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-w-3xl">
        {loading && (
          <>
            <DialogHeader>
              <DialogTitle><Skeleton className="h-6 w-48" /></DialogTitle>
              <DialogDescription className="sr-only">Loading movie details</DialogDescription>
            </DialogHeader>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-[200px_1fr]">
              <Skeleton className="aspect-[2/3] w-full rounded-md" />
              <div className="space-y-2">
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-3/4" />
              </div>
            </div>
          </>
        )}
        {error && (
          <>
            <DialogHeader>
              <DialogTitle>Couldn't load movie</DialogTitle>
              <DialogDescription>
                {error.kind === "unauthorized"
                  ? "Your Plex token is invalid. Reconnect in Integrations."
                  : "Could not reach your Plex server."}
              </DialogDescription>
            </DialogHeader>
          </>
        )}
        {data && (
          <>
            <DialogHeader>
              <DialogTitle>
                {data.title}
                {data.year > 0 && <span className="ml-2 text-muted-foreground font-normal">({data.year})</span>}
              </DialogTitle>
              <DialogDescription className="sr-only">Movie details</DialogDescription>
            </DialogHeader>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-[200px_1fr]">
              {data.thumb ? (
                <img
                  src={imageUrl(data.thumb)}
                  alt={data.title}
                  className="aspect-[2/3] w-full rounded-md object-cover"
                />
              ) : (
                <div className="aspect-[2/3] w-full rounded-md bg-muted" />
              )}
              <div className="space-y-3 text-sm">
                <div className="flex flex-wrap gap-x-3 gap-y-1 text-muted-foreground">
                  {formatRuntime(data.duration) && <span>{formatRuntime(data.duration)}</span>}
                  {data.contentRating && <span>{data.contentRating}</span>}
                  {data.audienceRating > 0 && <span>★ {data.audienceRating.toFixed(1)}</span>}
                </div>
                {data.genres.length > 0 && (
                  <div className="flex flex-wrap gap-1.5">
                    {data.genres.map((g) => (
                      <span key={g} className="rounded-full bg-muted px-2 py-0.5 text-xs">{g}</span>
                    ))}
                  </div>
                )}
                {data.summary && <p className="leading-relaxed">{data.summary}</p>}
                {data.directors.length > 0 && (
                  <div>
                    <span className="font-medium">Directed by:</span>{" "}
                    <span className="text-muted-foreground">{data.directors.join(", ")}</span>
                  </div>
                )}
                {data.cast.length > 0 && (
                  <div>
                    <span className="font-medium">Cast:</span>{" "}
                    <span className="text-muted-foreground">{data.cast.join(", ")}</span>
                  </div>
                )}
              </div>
            </div>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
```

- [ ] **Step 2: Verify typecheck**

Run: `cd frontend && pnpm build`
Expected: succeeds.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/movies/MovieDetail.tsx
git commit -m "feat(movies): MovieDetail dialog with poster, runtime, cast, summary"
```

---

## Task 19: MoviesPage

**Files:**
- Create: `frontend/src/components/movies/MoviesPage.tsx`

- [ ] **Step 1: Create the page**

```tsx
import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { Plug } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useIntegrations } from "@/hooks/useIntegrations";
import { MovieGrid } from "./MovieGrid";
import { SortDropdown } from "./SortDropdown";
import { MovieDetail } from "./MovieDetail";
import { useMoviesQuery } from "./useMoviesQuery";

function NotConnectedCTA() {
  return (
    <div className="mx-auto mt-16 max-w-md rounded-lg border bg-card p-8 text-center">
      <Plug className="mx-auto mb-3 size-8 text-muted-foreground" />
      <h2 className="text-lg font-semibold">Connect Plex to see your movies</h2>
      <p className="mt-2 text-sm text-muted-foreground">
        Add your Plex token in Integrations and we'll show your library here.
      </p>
      <Button asChild className="mt-4">
        <Link to="/integrations">Open Integrations</Link>
      </Button>
    </div>
  );
}

export function MoviesPage() {
  const { state: integrations, loading: integrationsLoading } = useIntegrations();
  const query = useMoviesQuery();
  const [selected, setSelected] = useState<string | null>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = sentinelRef.current;
    if (!el) return;
    const observer = new IntersectionObserver((entries) => {
      if (entries[0]?.isIntersecting) query.loadMore();
    });
    observer.observe(el);
    return () => observer.disconnect();
  }, [query]);

  if (integrationsLoading) {
    return <div className="p-6 text-muted-foreground">Loading…</div>;
  }
  if (!integrations.plexEnabled || !integrations.plexHasToken) {
    return <NotConnectedCTA />;
  }
  if (query.error?.kind === "not_configured") {
    return <NotConnectedCTA />;
  }

  return (
    <div className="space-y-4 py-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Movies</h1>
        <SortDropdown value={query.sort} onChange={query.setSort} />
      </div>

      {query.error && query.error.kind !== "not_configured" && query.items.length === 0 ? (
        <div className="mx-auto mt-12 max-w-md rounded-lg border bg-card p-6 text-center">
          <p className="text-sm">
            {query.error.kind === "unauthorized"
              ? "Your Plex token is invalid. Reconnect in Integrations."
              : "Couldn't reach your Plex server."}
          </p>
          <Button onClick={query.retry} variant="outline" size="sm" className="mt-3">
            Retry
          </Button>
        </div>
      ) : (
        <MovieGrid
          items={query.items}
          loading={query.loading}
          done={query.done}
          sentinelRef={sentinelRef}
          onCardClick={setSelected}
        />
      )}

      <MovieDetail ratingKey={selected} onClose={() => setSelected(null)} />
    </div>
  );
}
```

- [ ] **Step 2: Verify typecheck**

Run: `cd frontend && pnpm build`
Expected: succeeds.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/movies/MoviesPage.tsx
git commit -m "feat(movies): MoviesPage with gating, sort, infinite scroll, error UI"
```

---

## Task 20: Sidebar navigation — pass plexEnabled down

**Files:**
- Modify: `frontend/src/components/AppShell.tsx`
- Modify: `frontend/src/components/app-sidebar/app-sidebar.tsx`
- Modify: `frontend/src/components/app-sidebar/nav-menu.tsx`

- [ ] **Step 1: Update nav-menu.tsx**

Replace the contents of `frontend/src/components/app-sidebar/nav-menu.tsx` with:

```tsx
import { Link, useLocation } from "react-router-dom";
import { Home, Shield, Film, type LucideIcon } from "lucide-react";
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";

type NavItem = {
  title: string;
  icon: LucideIcon;
  href: string;
};

const BASE_ITEMS: NavItem[] = [
  { title: "Home", icon: Home, href: "/" },
];

const MOVIES_ITEM: NavItem = { title: "Movies", icon: Film, href: "/movies" };

const ADMIN_ITEMS: NavItem[] = [
  { title: "Admin", icon: Shield, href: "/admin" },
];

export function NavMenu({ isAdmin, plexEnabled }: { isAdmin: boolean; plexEnabled: boolean }) {
  const location = useLocation();
  const items: NavItem[] = [...BASE_ITEMS];
  if (plexEnabled) items.push(MOVIES_ITEM);
  if (isAdmin) items.push(...ADMIN_ITEMS);

  return (
    <SidebarGroup>
      <SidebarGroupLabel>Navigation</SidebarGroupLabel>
      <SidebarGroupContent>
        <SidebarMenu>
          {items.map((item) => (
            <SidebarMenuItem key={item.title}>
              <SidebarMenuButton
                asChild
                isActive={location.pathname === item.href}
                tooltip={item.title}
              >
                <Link to={item.href}>
                  <item.icon />
                  <span>{item.title}</span>
                </Link>
              </SidebarMenuButton>
            </SidebarMenuItem>
          ))}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  );
}
```

- [ ] **Step 2: Update app-sidebar.tsx**

Modify `frontend/src/components/app-sidebar/app-sidebar.tsx`. Change the prop signature and pass `plexEnabled` through:

Find:
```tsx
export function AppSidebar({ user }: { user: User }) {
```

Replace with:
```tsx
export function AppSidebar({ user, plexEnabled }: { user: User; plexEnabled: boolean }) {
```

Find:
```tsx
<NavMenu isAdmin={user.role === "admin"} />
```

Replace with:
```tsx
<NavMenu isAdmin={user.role === "admin"} plexEnabled={plexEnabled} />
```

- [ ] **Step 3: Update AppShell.tsx**

Replace `frontend/src/components/AppShell.tsx` with:

```tsx
import { SidebarInset, SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/app-sidebar";
import { Separator } from "@/components/ui/separator";
import { useIntegrations } from "@/hooks/useIntegrations";

type User = {
  id: string;
  email: string;
  name?: string | null;
  image?: string | null;
  role?: string | null;
};

export function AppShell({ user, children }: { user: User; children: React.ReactNode }) {
  const { state: integrations } = useIntegrations();
  const plexConnected = integrations.plexEnabled && integrations.plexHasToken;

  return (
    <SidebarProvider>
      <AppSidebar user={user} plexEnabled={plexConnected} />
      <SidebarInset>
        <header className="flex h-12 shrink-0 items-center gap-2 border-b px-4">
          <SidebarTrigger className="-ml-1" />
          <Separator orientation="vertical" className="mr-2 !h-4" />
          <span className="text-sm text-muted-foreground">TorrentUI</span>
        </header>
        <main className="flex-1 px-4 md:px-6">{children}</main>
      </SidebarInset>
    </SidebarProvider>
  );
}
```

- [ ] **Step 4: Verify typecheck**

Run: `cd frontend && pnpm build`
Expected: succeeds.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/AppShell.tsx frontend/src/components/app-sidebar/app-sidebar.tsx frontend/src/components/app-sidebar/nav-menu.tsx
git commit -m "feat(movies): show Movies nav item when Plex is connected"
```

---

## Task 21: Add /movies route to App.tsx

**Files:**
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: Add the route**

Find:
```tsx
import { IntegrationsPage } from "@/components/IntegrationsPage";
```

Add below it:
```tsx
import { MoviesPage } from "@/components/movies/MoviesPage";
```

Find:
```tsx
<Route path="/integrations" element={<IntegrationsPage />} />
```

Add below it:
```tsx
<Route path="/movies" element={<MoviesPage />} />
```

The Routes block should now contain `/`, `/admin`, `/integrations`, `/movies`, and `*`.

- [ ] **Step 2: Verify build**

Run: `cd frontend && pnpm build`
Expected: succeeds.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/App.tsx
git commit -m "feat(movies): register /movies route"
```

---

## Task 22: Refresh integrations cache after token changes

**Files:**
- Modify: `frontend/src/components/IntegrationsPage.tsx`

We need the sidebar "Movies" item to appear immediately after a user connects/disconnects Plex from the Integrations page — currently the cache is set on first load and never invalidated.

- [ ] **Step 1: Update IntegrationsPage.tsx**

Replace the file's contents with:

```tsx
import { useEffect, useState } from "react";
import { apiFetch } from "@/services";
import { refreshIntegrations } from "@/hooks/useIntegrations";
import { PlexIntegrationRow } from "./PlexIntegrationRow";

type IntegrationState = {
  plexEnabled: boolean;
  plexHasToken: boolean;
};

export function IntegrationsPage() {
  const [state, setState] = useState<IntegrationState | null>(null);
  const [loading, setLoading] = useState(true);

  async function load() {
    const res = await apiFetch("/api/integrations");
    if (res.ok) {
      setState(await res.json());
    }
    setLoading(false);
    refreshIntegrations();
  }

  useEffect(() => {
    load();
  }, []);

  if (loading) {
    return (
      <div className="p-6 max-w-2xl mx-auto">
        <div className="text-muted-foreground">Loading…</div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-2xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Integrations</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Connect external services to your account.
        </p>
      </div>
      <div className="border rounded-lg overflow-hidden">
        <PlexIntegrationRow
          enabled={state?.plexEnabled ?? false}
          hasToken={state?.plexHasToken ?? false}
          onUpdate={load}
        />
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Verify build**

Run: `cd frontend && pnpm build`
Expected: succeeds.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/IntegrationsPage.tsx
git commit -m "feat(movies): invalidate integrations cache on Plex token change"
```

---

## Task 23: Manual verification

**Files:** None modified.

The goal is end-to-end smoke testing in a real browser before declaring done.

- [ ] **Step 1: Start the stack**

Run from project root: `make dev-build && make dev-logs`
In another terminal: `cd frontend && pnpm dev`
Open http://localhost:5173 in a browser and sign in.

- [ ] **Step 2: Verify gating — Plex not connected**

- Navigate to http://localhost:5173/movies directly.
- Expected: the "Connect Plex to see your movies" CTA renders. The sidebar does NOT show a "Movies" item.

- [ ] **Step 3: Connect Plex**

- Open user dropdown → Integrations.
- Paste a valid Plex token, click Save.
- Expected: row flips to "Connected"; the sidebar gains a "Movies" item without a page reload.

- [ ] **Step 4: Verify the grid loads**

- Click "Movies" in the sidebar.
- Expected: skeleton grid (12 placeholders) briefly, then a grid of real posters loads. Title and year display under each poster.
- Open devtools → Network. Requests to `/api/plex/movies?...` return 200; requests to `/api/plex/image?path=...` stream image bytes with `Cache-Control: public, max-age=86400`.

- [ ] **Step 5: Verify infinite scroll**

- Scroll to the bottom of the grid.
- Expected: a "Loading…" indicator appears at the sentinel, then another page of items appends. Once `items.length >= total`, the indicator becomes "End of library".

- [ ] **Step 6: Verify sort changes**

- Click the sort dropdown and pick "Title A–Z".
- Expected: grid resets to the first page sorted alphabetically. Inspect `/api/plex/movies?sort=titleSort:asc` in Network.

- [ ] **Step 7: Verify detail dialog**

- Click any card.
- Expected: a modal dialog opens. Skeleton briefly, then poster, runtime, content rating, audience rating, genres (pills), summary, directors, top-6 cast render.
- Close with Esc or backdrop click.

- [ ] **Step 8: Verify error states**

- In a separate terminal, run `sqlite3 data/backend.sqlite "UPDATE user_integrations SET plex_token='broken' WHERE user_id='<your-user-id>';"`.
- Reload /movies.
- Expected: 401 path renders "Your Plex token is invalid. Reconnect in Integrations." with a Retry button.
- Restore the real token via the Integrations UI.

- [ ] **Step 9: Verify disconnect hides the nav item**

- Toggle Plex off in Integrations.
- Expected: the sidebar Movies item disappears (after the cache refreshes) and visiting /movies shows the CTA.

- [ ] **Step 10: Verify all backend tests still pass**

Run: `cd backend && go test ./...`
Expected: all tests PASS.

- [ ] **Step 11: Final build of the full app**

Run from project root: `cd frontend && pnpm build && cd ../backend && go build ./...`
Expected: both succeed.

- [ ] **Step 12: Final commit (only if anything changed during verification)**

If verification surfaced bugs and you made fixes, commit them with an explanatory message. Otherwise nothing to commit.

---

## Spec-coverage summary

| Spec section | Implemented in |
|---|---|
| Backend `plex` package layout | Tasks 1–10 |
| `GET /plex/movies` route | Task 8 |
| `GET /plex/movies/:ratingKey` | Task 9 |
| `GET /plex/image` with SSRF guard | Task 10 |
| PMS discovery + connection selection | Task 3 |
| Discovery cache (5-min TTL) | Task 2 |
| Multi-library pagination | Task 5 |
| Error mapping (412/401/502) | Tasks 8, 9, 10 |
| Required Plex headers (`X-Plex-Token`, `X-Plex-Client-Identifier`, `X-Plex-Product`) | Tasks 4, 7 |
| Image cache headers | Task 10 |
| Backend wiring | Task 11 |
| Frontend `useIntegrations` hook | Task 12 |
| Frontend types + API client | Task 13 |
| `useMoviesQuery` + sort reset + infinite scroll | Tasks 14, 19 |
| MovieCard / MovieGrid / SortDropdown / MovieDetail | Tasks 15–18 |
| MoviesPage with gating + retry UI | Task 19 |
| Sidebar "Movies" item gated by `plexEnabled && plexHasToken` | Task 20 |
| `/movies` route | Task 21 |
| Cache invalidation after token change | Task 22 |
| Manual verification matrix | Task 23 |
