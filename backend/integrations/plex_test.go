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
