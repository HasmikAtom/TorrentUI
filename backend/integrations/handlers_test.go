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
