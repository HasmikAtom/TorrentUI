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
			// totals query: start=0 size=0
			if start == "0" && size == "0" {
				_, _ = w.Write([]byte(`{"MediaContainer":{"size":0,"totalSize":3,"offset":0,"Metadata":[]}}`))
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
			// totals query: start=0 size=0
			if start == "0" && size == "0" {
				_, _ = w.Write([]byte(`{"MediaContainer":{"size":0,"totalSize":2,"offset":0,"Metadata":[]}}`))
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
		"/library/sections/1/all": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"MediaContainer":{"size":0,"totalSize":3,"offset":0,"Metadata":[]}}`))
		},
		"/library/sections/2/all": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"MediaContainer":{"size":0,"totalSize":2,"offset":0,"Metadata":[]}}`))
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
