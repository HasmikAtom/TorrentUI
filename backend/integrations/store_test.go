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
