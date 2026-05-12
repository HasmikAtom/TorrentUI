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
