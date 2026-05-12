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
