# Integrations Framework Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add per-user integration management to TorrentUI, starting with Plex token storage and validation.

**Architecture:** Go backend gets its own SQLite database (`data/backend.sqlite`) with a `user_integrations` table (Miniflux-style: one row per user, flat columns per integration type). Frontend adds an Integrations page accessible from the user dropdown, with inline expand/collapse rows per integration.

**Tech Stack:** Go + `modernc.org/sqlite`, Gin HTTP framework, React + TypeScript, shadcn/ui components, Tailwind CSS

---

## File Map

### Backend (Go)

| File | Action | Responsibility |
|------|--------|---------------|
| `backend/db/db.go` | Create | SQLite connection init, migration runner |
| `backend/db/db_test.go` | Create | Tests for DB init and migration |
| `backend/integrations/store.go` | Create | CRUD operations for `user_integrations` table |
| `backend/integrations/store_test.go` | Create | Tests for store operations |
| `backend/integrations/plex.go` | Create | Plex token validation via Plex API |
| `backend/integrations/plex_test.go` | Create | Tests for Plex validation (with HTTP mock) |
| `backend/integrations/handlers.go` | Create | Gin handlers: GET /integrations, PUT /integrations/plex, DELETE /integrations/plex |
| `backend/integrations/handlers_test.go` | Create | HTTP tests for integration handlers |
| `backend/main.go` | Modify | Init DB on startup, mount integration routes |
| `backend/go.mod` | Modify | Add `modernc.org/sqlite` dependency |
| `docker-compose.dev.yml` | Modify | Add `./data` volume mount to backend-dev |

### Frontend (React)

| File | Action | Responsibility |
|------|--------|---------------|
| `frontend/src/components/IntegrationsPage.tsx` | Create | Page wrapper, fetches integration state, renders rows |
| `frontend/src/components/PlexIntegrationRow.tsx` | Create | Plex row with expand/collapse, token form, toggle |
| `frontend/src/components/app-sidebar/nav-user.tsx` | Modify | Add "Integrations" menu item to user dropdown |
| `frontend/src/App.tsx` | Modify | Add `/integrations` route |
| `frontend/src/components/ui/switch.tsx` | Create | Install shadcn Switch component (for enable/disable toggle) |

---

### Task 1: Go SQLite Setup — DB Package

**Files:**
- Create: `backend/db/db.go`
- Create: `backend/db/db_test.go`
- Modify: `backend/go.mod`

- [ ] **Step 1: Add the SQLite dependency**

```bash
cd backend && go get modernc.org/sqlite github.com/jmoiron/sqlx
```

We use `modernc.org/sqlite` (pure Go, no CGO) as the driver and `jmoiron/sqlx` for ergonomic struct scanning.

- [ ] **Step 2: Write the failing test for DB init and migration**

Create `backend/db/db_test.go`:

```go
package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpen_CreatesFileAndRunsMigrations(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.sqlite")

	d, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer d.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("expected database file to exist")
	}

	var tableName string
	err = d.DB.Get(&tableName, "SELECT name FROM sqlite_master WHERE type='table' AND name='user_integrations'")
	if err != nil {
		t.Fatalf("user_integrations table not found: %v", err)
	}
	if tableName != "user_integrations" {
		t.Fatalf("expected 'user_integrations', got %q", tableName)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd backend && go test ./db/ -v
```

Expected: compilation failure — package `db` doesn't exist yet.

- [ ] **Step 4: Implement the db package**

Create `backend/db/db.go`:

```go
package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

type Database struct {
	DB *sqlx.DB
}

func Open(path string) (*Database, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	conn, err := sqlx.Open("sqlite", path+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(on)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	d := &Database{DB: conn}
	if err := d.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return d, nil
}

func (d *Database) Close() error {
	return d.DB.Close()
}

func (d *Database) migrate() error {
	_, err := d.DB.Exec(`
		CREATE TABLE IF NOT EXISTS user_integrations (
			user_id      TEXT PRIMARY KEY,
			plex_enabled INTEGER NOT NULL DEFAULT 0,
			plex_token   TEXT    NOT NULL DEFAULT '',
			created_at   INTEGER NOT NULL DEFAULT (unixepoch()),
			updated_at   INTEGER NOT NULL DEFAULT (unixepoch())
		);
	`)
	return err
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd backend && go test ./db/ -v
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
cd backend && git add db/ go.mod go.sum
git commit -m "feat: add Go SQLite database package with user_integrations migration"
```

---

### Task 2: Integrations Store — CRUD Operations

**Files:**
- Create: `backend/integrations/store.go`
- Create: `backend/integrations/store_test.go`

- [ ] **Step 1: Write failing tests for store operations**

Create `backend/integrations/store_test.go`:

```go
package integrations

import (
	"path/filepath"
	"testing"

	"github.com/hasmikatom/torrent/db"
)

func setupTestDB(t *testing.T) *db.Database {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func TestStore_GetIntegrations_ReturnsDefaultsForNewUser(t *testing.T) {
	d := setupTestDB(t)
	store := NewStore(d)

	row, err := store.GetIntegrations("user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if row.PlexEnabled {
		t.Error("expected plex_enabled=false for new user")
	}
	if row.PlexToken != "" {
		t.Error("expected empty plex_token for new user")
	}
}

func TestStore_UpsertPlex_InsertsNewRow(t *testing.T) {
	d := setupTestDB(t)
	store := NewStore(d)

	err := store.UpsertPlex("user-1", "my-token", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	row, err := store.GetIntegrations("user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !row.PlexEnabled {
		t.Error("expected plex_enabled=true")
	}
	if row.PlexToken != "my-token" {
		t.Errorf("expected 'my-token', got %q", row.PlexToken)
	}
}

func TestStore_UpsertPlex_UpdatesExistingRow(t *testing.T) {
	d := setupTestDB(t)
	store := NewStore(d)

	store.UpsertPlex("user-1", "token-1", true)
	store.UpsertPlex("user-1", "token-2", false)

	row, _ := store.GetIntegrations("user-1")
	if row.PlexToken != "token-2" {
		t.Errorf("expected 'token-2', got %q", row.PlexToken)
	}
	if row.PlexEnabled {
		t.Error("expected plex_enabled=false after update")
	}
}

func TestStore_SetPlexEnabled_TogglesWithoutChangingToken(t *testing.T) {
	d := setupTestDB(t)
	store := NewStore(d)

	store.UpsertPlex("user-1", "my-token", true)
	err := store.SetPlexEnabled("user-1", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	row, _ := store.GetIntegrations("user-1")
	if row.PlexEnabled {
		t.Error("expected plex_enabled=false")
	}
	if row.PlexToken != "my-token" {
		t.Errorf("expected token preserved, got %q", row.PlexToken)
	}
}

func TestStore_DeletePlex_ClearsTokenAndDisables(t *testing.T) {
	d := setupTestDB(t)
	store := NewStore(d)

	store.UpsertPlex("user-1", "my-token", true)
	err := store.DeletePlex("user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	row, _ := store.GetIntegrations("user-1")
	if row.PlexEnabled {
		t.Error("expected plex_enabled=false after delete")
	}
	if row.PlexToken != "" {
		t.Errorf("expected empty token after delete, got %q", row.PlexToken)
	}
}

func TestStore_DeletePlex_NoopForNonexistentUser(t *testing.T) {
	d := setupTestDB(t)
	store := NewStore(d)

	err := store.DeletePlex("nonexistent")
	if err != nil {
		t.Fatalf("expected no error for nonexistent user, got: %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd backend && go test ./integrations/ -v
```

Expected: compilation failure — `NewStore`, `Store`, `UserIntegrations` not defined.

- [ ] **Step 3: Implement the store**

Create `backend/integrations/store.go`:

```go
package integrations

import (
	"database/sql"

	"github.com/hasmikatom/torrent/db"
)

type UserIntegrations struct {
	UserID      string `json:"userId" db:"user_id"`
	PlexEnabled bool   `json:"plexEnabled" db:"plex_enabled"`
	PlexToken   string `json:"plexToken,omitempty" db:"plex_token"`
	CreatedAt   int64  `json:"createdAt" db:"created_at"`
	UpdatedAt   int64  `json:"updatedAt" db:"updated_at"`
}

type Store struct {
	db *db.Database
}

func NewStore(d *db.Database) *Store {
	return &Store{db: d}
}

func (s *Store) GetIntegrations(userID string) (UserIntegrations, error) {
	var row UserIntegrations
	err := s.db.DB.Get(&row, "SELECT * FROM user_integrations WHERE user_id = ?", userID)
	if err == sql.ErrNoRows {
		return UserIntegrations{UserID: userID}, nil
	}
	return row, err
}

func (s *Store) UpsertPlex(userID, token string, enabled bool) error {
	_, err := s.db.DB.Exec(`
		INSERT INTO user_integrations (user_id, plex_token, plex_enabled, updated_at)
		VALUES (?, ?, ?, unixepoch())
		ON CONFLICT(user_id) DO UPDATE SET
			plex_token   = excluded.plex_token,
			plex_enabled = excluded.plex_enabled,
			updated_at   = unixepoch()
	`, userID, token, enabled)
	return err
}

func (s *Store) SetPlexEnabled(userID string, enabled bool) error {
	_, err := s.db.DB.Exec(`
		UPDATE user_integrations
		SET plex_enabled = ?, updated_at = unixepoch()
		WHERE user_id = ?
	`, enabled, userID)
	return err
}

func (s *Store) DeletePlex(userID string) error {
	_, err := s.db.DB.Exec(`
		UPDATE user_integrations
		SET plex_token = '', plex_enabled = 0, updated_at = unixepoch()
		WHERE user_id = ?
	`, userID)
	return err
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd backend && go test ./integrations/ -v
```

Expected: all 6 tests PASS.

- [ ] **Step 5: Commit**

```bash
cd backend && git add integrations/store.go integrations/store_test.go
git commit -m "feat: add integrations store with CRUD operations"
```

---

### Task 3: Plex Token Validation

**Files:**
- Create: `backend/integrations/plex.go`
- Create: `backend/integrations/plex_test.go`

- [ ] **Step 1: Write failing test for Plex validation**

Create `backend/integrations/plex_test.go`:

```go
package integrations

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidatePlexToken_ValidToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Plex-Token") != "valid-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"username":"testuser"}`))
	}))
	defer server.Close()

	err := ValidatePlexToken("valid-token", server.URL)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidatePlexToken_InvalidToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	err := ValidatePlexToken("bad-token", server.URL)
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestValidatePlexToken_EmptyToken(t *testing.T) {
	err := ValidatePlexToken("", "")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./integrations/ -run TestValidatePlex -v
```

Expected: compilation failure — `ValidatePlexToken` not defined.

- [ ] **Step 3: Implement Plex validation**

Create `backend/integrations/plex.go`:

```go
package integrations

import (
	"fmt"
	"net/http"
	"time"
)

const defaultPlexAPIURL = "https://plex.tv/api/v2/user"

var plexHTTPClient = &http.Client{Timeout: 10 * time.Second}

// ValidatePlexToken checks a Plex token against the Plex API.
// Pass an empty baseURL to use the default (https://plex.tv/api/v2/user).
func ValidatePlexToken(token string, baseURL string) error {
	if token == "" {
		return fmt.Errorf("token is required")
	}

	url := baseURL
	if url == "" {
		url = defaultPlexAPIURL
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("X-Plex-Token", token)
	req.Header.Set("Accept", "application/json")

	resp, err := plexHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("could not verify token — try again later")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid Plex token")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected Plex API response: %d", resp.StatusCode)
	}

	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd backend && go test ./integrations/ -run TestValidatePlex -v
```

Expected: all 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
cd backend && git add integrations/plex.go integrations/plex_test.go
git commit -m "feat: add Plex token validation against Plex API"
```

---

### Task 4: Integration HTTP Handlers

**Files:**
- Create: `backend/integrations/handlers.go`
- Create: `backend/integrations/handlers_test.go`

- [ ] **Step 1: Write failing tests for handlers**

Create `backend/integrations/handlers_test.go`:

```go
package integrations

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hasmikatom/torrent/db"
	"github.com/hasmikatom/torrent/middleware"
)

func setupHandlerTest(t *testing.T) (*gin.Engine, *Store) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	d, err := db.Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	t.Cleanup(func() { d.Close() })

	store := NewStore(d)
	r := gin.New()
	g := r.Group("/", middleware.RequireUser())
	RegisterHandlers(g, store, "")
	return r, store
}

func authedRequest(method, path string, body interface{}) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "user-1")
	req.Header.Set("X-User-Email", "test@example.com")
	return req
}

func TestGetIntegrations_ReturnsDefaults(t *testing.T) {
	r, _ := setupHandlerTest(t)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, authedRequest(http.MethodGet, "/integrations", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		PlexEnabled  bool `json:"plexEnabled"`
		PlexHasToken bool `json:"plexHasToken"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.PlexEnabled {
		t.Error("expected plexEnabled=false")
	}
	if resp.PlexHasToken {
		t.Error("expected plexHasToken=false")
	}
}

func TestPutPlex_SavesValidToken(t *testing.T) {
	plexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Plex-Token") == "valid-token" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer plexServer.Close()

	gin.SetMode(gin.TestMode)
	d, err := db.Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer d.Close()

	store := NewStore(d)
	r := gin.New()
	g := r.Group("/", middleware.RequireUser())
	RegisterHandlers(g, store, plexServer.URL)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, authedRequest(http.MethodPut, "/integrations/plex", map[string]interface{}{
		"token": "valid-token",
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	row, _ := store.GetIntegrations("user-1")
	if row.PlexToken != "valid-token" {
		t.Errorf("expected token saved, got %q", row.PlexToken)
	}
	if !row.PlexEnabled {
		t.Error("expected plex_enabled=true after save")
	}
}

func TestPutPlex_RejectsInvalidToken(t *testing.T) {
	plexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer plexServer.Close()

	gin.SetMode(gin.TestMode)
	d, err := db.Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer d.Close()

	store := NewStore(d)
	r := gin.New()
	g := r.Group("/", middleware.RequireUser())
	RegisterHandlers(g, store, plexServer.URL)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, authedRequest(http.MethodPut, "/integrations/plex", map[string]interface{}{
		"token": "bad-token",
	}))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPutPlex_TogglesEnabledWithoutToken(t *testing.T) {
	r, store := setupHandlerTest(t)

	store.UpsertPlex("user-1", "my-token", true)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, authedRequest(http.MethodPut, "/integrations/plex", map[string]interface{}{
		"enabled": false,
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	row, _ := store.GetIntegrations("user-1")
	if row.PlexEnabled {
		t.Error("expected plex_enabled=false after toggle")
	}
	if row.PlexToken != "my-token" {
		t.Error("expected token preserved after toggle")
	}
}

func TestDeletePlex_ClearsToken(t *testing.T) {
	r, store := setupHandlerTest(t)

	store.UpsertPlex("user-1", "my-token", true)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, authedRequest(http.MethodDelete, "/integrations/plex", nil))

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	row, _ := store.GetIntegrations("user-1")
	if row.PlexToken != "" {
		t.Errorf("expected empty token, got %q", row.PlexToken)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd backend && go test ./integrations/ -run TestGetIntegrations\|TestPutPlex\|TestDeletePlex -v
```

Expected: compilation failure — `RegisterHandlers` not defined.

- [ ] **Step 3: Implement handlers**

Create `backend/integrations/handlers.go`:

```go
package integrations

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type plexRequest struct {
	Token   *string `json:"token"`
	Enabled *bool   `json:"enabled"`
}

func RegisterHandlers(g *gin.RouterGroup, store *Store, plexAPIURL string) {
	g.GET("/integrations", func(c *gin.Context) {
		userID := c.GetString("userId")
		row, err := store.GetIntegrations(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load integrations"})
			return
		}
		// Never expose the raw token to the frontend — only whether one is set
		c.JSON(http.StatusOK, gin.H{
			"userId":       row.UserID,
			"plexEnabled":  row.PlexEnabled,
			"plexHasToken": row.PlexToken != "",
		})
	})

	g.PUT("/integrations/plex", func(c *gin.Context) {
		userID := c.GetString("userId")
		var req plexRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		// Toggle-only: no token provided, just flip enabled
		if req.Token == nil && req.Enabled != nil {
			if err := store.SetPlexEnabled(userID, *req.Enabled); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update"})
				return
			}
			row, _ := store.GetIntegrations(userID)
			c.JSON(http.StatusOK, gin.H{
				"userId":       row.UserID,
				"plexEnabled":  row.PlexEnabled,
				"plexHasToken": row.PlexToken != "",
			})
			return
		}

		// Token save: validate first
		if req.Token == nil || *req.Token == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
			return
		}

		if err := ValidatePlexToken(*req.Token, plexAPIURL); err != nil {
			status := http.StatusBadRequest
			if err.Error() == "could not verify token — try again later" {
				status = http.StatusBadGateway
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}

		enabled := true
		if req.Enabled != nil {
			enabled = *req.Enabled
		}

		if err := store.UpsertPlex(userID, *req.Token, enabled); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save"})
			return
		}

		row, _ := store.GetIntegrations(userID)
		c.JSON(http.StatusOK, gin.H{
			"userId":       row.UserID,
			"plexEnabled":  row.PlexEnabled,
			"plexHasToken": row.PlexToken != "",
		})
	})

	g.DELETE("/integrations/plex", func(c *gin.Context) {
		userID := c.GetString("userId")
		if err := store.DeletePlex(userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete"})
			return
		}
		c.Status(http.StatusNoContent)
	})
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd backend && go test ./integrations/ -v
```

Expected: all handler tests PASS.

- [ ] **Step 5: Commit**

```bash
cd backend && git add integrations/handlers.go integrations/handlers_test.go
git commit -m "feat: add integration HTTP handlers (GET, PUT, DELETE)"
```

---

### Task 5: Wire Up Backend — main.go + Docker

**Files:**
- Modify: `backend/main.go`
- Modify: `docker-compose.dev.yml`

- [ ] **Step 1: Update main.go to init DB and mount routes**

In `backend/main.go`, add the import and initialization. After the `scraper.LoadScraperConfig()` line in `init()`, add DB opening. Then mount integration routes alongside existing routes.

Updated `backend/main.go` (showing the changes — the full file with modifications):

Add imports:
```go
import (
	// ... existing imports ...
	"github.com/hasmikatom/torrent/db"
	"github.com/hasmikatom/torrent/integrations"
)
```

Add a package-level variable after the existing `client` var:
```go
var backendDB *db.Database
```

Add to `init()` after `scraper.LoadScraperConfig()`:
```go
	var err error
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./data/backend.sqlite"
	}
	backendDB, err = db.Open(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
```

Add integration routes in `main()` after the existing `api` group block:
```go
	intStore := integrations.NewStore(backendDB)
	integrations.RegisterHandlers(api, intStore, "")
```

Add DB close in the shutdown section, before `srv.Shutdown`:
```go
	if backendDB != nil {
		backendDB.Close()
	}
```

- [ ] **Step 2: Add data volume mount to docker-compose.dev.yml**

In `docker-compose.dev.yml`, the `backend-dev` service needs access to the `./data` directory. Add this volume to the `backend-dev` volumes list:

```yaml
      - ./data:/data
```

And add the environment variable:
```yaml
      - DATABASE_PATH=/data/backend.sqlite
```

- [ ] **Step 3: Verify the backend compiles**

```bash
cd backend && go build .
```

Expected: successful compilation, no errors.

- [ ] **Step 4: Run all backend tests**

```bash
cd backend && go test ./...
```

Expected: all tests pass (db, integrations, middleware).

- [ ] **Step 5: Commit**

```bash
git add backend/main.go docker-compose.dev.yml
git commit -m "feat: wire up integrations DB and routes in backend startup"
```

---

### Task 6: Frontend — Install Switch Component + IntegrationsPage

**Files:**
- Create: `frontend/src/components/ui/switch.tsx` (via shadcn CLI)
- Create: `frontend/src/components/IntegrationsPage.tsx`

- [ ] **Step 1: Install shadcn Switch component**

```bash
cd frontend && pnpm dlx shadcn@latest add switch --overwrite
```

This creates `frontend/src/components/ui/switch.tsx`.

- [ ] **Step 2: Create the IntegrationsPage component**

Create `frontend/src/components/IntegrationsPage.tsx`:

```tsx
import { useEffect, useState } from "react";
import { apiFetch } from "@/services";
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

- [ ] **Step 3: Verify TypeScript compiles (will fail — PlexIntegrationRow doesn't exist yet)**

```bash
cd frontend && ./node_modules/.bin/tsc --noEmit
```

Expected: error about missing `PlexIntegrationRow` module. This is expected — we create it in the next task.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/IntegrationsPage.tsx frontend/src/components/ui/switch.tsx
git commit -m "feat: add IntegrationsPage and install shadcn Switch component"
```

---

### Task 7: Frontend — PlexIntegrationRow Component

**Files:**
- Create: `frontend/src/components/PlexIntegrationRow.tsx`

- [ ] **Step 1: Create the PlexIntegrationRow component**

Create `frontend/src/components/PlexIntegrationRow.tsx`:

```tsx
import { useState } from "react";
import { Plug } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { apiFetch } from "@/services";

type Props = {
  enabled: boolean;
  hasToken: boolean;
  onUpdate: () => void;
};

export function PlexIntegrationRow({ enabled, hasToken, onUpdate }: Props) {
  const [expanded, setExpanded] = useState(false);
  const [token, setToken] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const connected = hasToken;

  async function handleSave() {
    setBusy(true);
    setError("");
    try {
      const res = await apiFetch("/api/integrations/plex", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ token }),
      });
      if (!res.ok) {
        const data = await res.json();
        setError(data.error || "Failed to save");
        return;
      }
      setToken("");
      setExpanded(false);
      onUpdate();
    } finally {
      setBusy(false);
    }
  }

  async function handleToggle(checked: boolean) {
    setBusy(true);
    try {
      await apiFetch("/api/integrations/plex", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ enabled: checked }),
      });
      onUpdate();
    } finally {
      setBusy(false);
    }
  }

  async function handleDisconnect() {
    setBusy(true);
    setError("");
    try {
      await apiFetch("/api/integrations/plex", { method: "DELETE" });
      setExpanded(false);
      onUpdate();
    } finally {
      setBusy(false);
    }
  }

  return (
    <div>
      <div
        className="flex items-center p-4 gap-3 cursor-pointer hover:bg-accent/50 transition-colors"
        onClick={() => setExpanded(!expanded)}
      >
        <div
          className={`flex size-8 items-center justify-center rounded-md ${
            connected ? "bg-[#e5a00d] text-black" : "bg-[#e5a00d]/20 text-[#e5a00d]"
          }`}
        >
          <Plug className="size-4" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="font-semibold text-sm">Plex</div>
          <div className="text-xs text-muted-foreground">Media server integration</div>
        </div>
        {connected ? (
          <div className="flex items-center gap-3">
            <span className="text-xs text-green-500">● Connected</span>
            <Switch
              checked={enabled}
              onCheckedChange={handleToggle}
              disabled={busy}
              onClick={(e) => e.stopPropagation()}
            />
          </div>
        ) : (
          <Button
            size="sm"
            variant="outline"
            onClick={(e) => {
              e.stopPropagation();
              setExpanded(true);
            }}
          >
            Connect
          </Button>
        )}
      </div>

      {expanded && (
        <div className="px-4 pb-4 pl-[60px] border-t bg-muted/30">
          <div className="pt-4 space-y-3">
            <div>
              <label className="text-xs text-muted-foreground block mb-1.5">
                Plex Token
              </label>
              <div className="flex gap-2">
                <Input
                  type="password"
                  placeholder={connected ? "••••••••••••••••" : "Paste your Plex token"}
                  value={token}
                  onChange={(e) => setToken(e.target.value)}
                  disabled={busy}
                />
                <Button onClick={handleSave} disabled={busy || !token} size="sm">
                  {connected ? "Update" : "Save"}
                </Button>
                {connected ? (
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={handleDisconnect}
                    disabled={busy}
                  >
                    Disconnect
                  </Button>
                ) : (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      setExpanded(false);
                      setToken("");
                      setError("");
                    }}
                  >
                    Cancel
                  </Button>
                )}
              </div>
              {error && (
                <p className="text-xs text-destructive mt-1.5">{error}</p>
              )}
            </div>
            <p className="text-[11px] text-muted-foreground">
              Find your token at plex.tv — go to Settings → Devices, click a
              device, and look for the token in the XML URL.
            </p>
          </div>
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 2: Verify TypeScript compiles**

```bash
cd frontend && ./node_modules/.bin/tsc --noEmit
```

Expected: PASS (no type errors).

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/PlexIntegrationRow.tsx
git commit -m "feat: add PlexIntegrationRow with token form and toggle"
```

---

### Task 8: Frontend — Route + Nav Link Wiring

**Files:**
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/components/app-sidebar/nav-user.tsx`

- [ ] **Step 1: Add the /integrations route to App.tsx**

In `frontend/src/App.tsx`, add the import and route:

Add import at the top:
```tsx
import { IntegrationsPage } from "@/components/IntegrationsPage";
```

Add the route after the `/admin` route and before the catch-all `*` route:
```tsx
          <Route path="/integrations" element={<IntegrationsPage />} />
```

- [ ] **Step 2: Add "Integrations" link to user dropdown**

In `frontend/src/components/app-sidebar/nav-user.tsx`:

Add imports:
```tsx
import { useNavigate } from "react-router-dom";
import { Plug } from "lucide-react";
```

(The `Plug` import goes alongside the existing lucide imports on line 1.)

In the `NavUser` component, add `useNavigate`:
```tsx
const navigate = useNavigate();
```

In the `DropdownMenuContent`, between `<ThemeSubmenu />` and `<DropdownMenuSeparator />`, add:
```tsx
            <DropdownMenuItem onSelect={() => navigate("/integrations")}>
              <Plug className="mr-2 h-4 w-4" />
              Integrations
            </DropdownMenuItem>
```

- [ ] **Step 3: Verify TypeScript compiles**

```bash
cd frontend && ./node_modules/.bin/tsc --noEmit
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/App.tsx frontend/src/components/app-sidebar/nav-user.tsx
git commit -m "feat: add /integrations route and nav link in user dropdown"
```

---

### Task 9: End-to-End Smoke Test

**Files:** None (testing only)

- [ ] **Step 1: Run all backend tests**

```bash
cd backend && go test ./... -v
```

Expected: all tests pass across `db`, `integrations`, and `middleware` packages.

- [ ] **Step 2: Run frontend typecheck**

```bash
cd frontend && ./node_modules/.bin/tsc --noEmit
```

Expected: no type errors.

- [ ] **Step 3: Run frontend lint**

```bash
cd frontend && pnpm lint
```

Expected: no lint errors (or only pre-existing warnings).

- [ ] **Step 4: Start the dev environment and test manually**

Start the frontend dev server:
```bash
cd frontend && pnpm dev
```

Test the following in the browser:

1. Open the app, sign in
2. Click your user avatar in the sidebar footer → dropdown should show "Integrations" item
3. Click "Integrations" → should navigate to `/integrations`
4. Plex row should show "Not connected" with a "Connect" button
5. Click "Connect" → form expands with token input
6. Enter an invalid token → click "Save" → should show "Invalid Plex token" error
7. Click "Cancel" → form collapses
8. Enter a valid Plex token → click "Save" → should show "Connected" with toggle
9. Toggle the switch off and on → should toggle without re-validation
10. Expand the row → click "Disconnect" → should revert to "Not connected"

- [ ] **Step 5: Commit any fixes from smoke testing**

If any issues were found and fixed during smoke testing, commit them:

```bash
git add -A
git commit -m "fix: address issues found during integration smoke test"
```

---
