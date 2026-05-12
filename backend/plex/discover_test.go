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
