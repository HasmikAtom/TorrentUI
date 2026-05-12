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
