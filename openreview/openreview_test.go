package openreview

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestClient(baseURL string) *Client {
	cfg := DefaultConfig()
	cfg.BaseURL = baseURL
	cfg.Rate = 0
	return NewClient(cfg)
}

func TestGetSendsUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	body, err := c.get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{}` {
		t.Errorf("body = %q", body)
	}
}

func TestGetRetriesOn503(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	cfg.Retries = 5
	c := NewClient(cfg)

	start := time.Now()
	body, err := c.get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{}` {
		t.Errorf("body = %q after retries", body)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
}

func TestSearch(t *testing.T) {
	resp := apiResp{
		Count: 1,
		Notes: []wireNote{
			{
				ID:     "test123",
				Number: 42,
				Content: mustMarshal(map[string]any{
					"title":    map[string]any{"value": "Attention Is All You Need"},
					"authors":  map[string]any{"value": []string{"Vaswani", "Shazeer"}},
					"keywords": map[string]any{"value": []string{"transformer", "attention", "nlp"}},
					"venue":    map[string]any{"value": "ICLR 2024"},
					"decision": map[string]any{"value": "Accept: Poster"},
				}),
			},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/notes/search" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Error(err)
		}
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	papers, count, err := c.Search(context.Background(), "transformer", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
	if len(papers) != 1 {
		t.Fatalf("got %d papers, want 1", len(papers))
	}
	p := papers[0]
	if p.ID != "test123" {
		t.Errorf("ID = %q, want %q", p.ID, "test123")
	}
	if p.Title != "Attention Is All You Need" {
		t.Errorf("Title = %q", p.Title)
	}
	if p.Authors != "Vaswani; Shazeer" {
		t.Errorf("Authors = %q", p.Authors)
	}
	if p.Keywords != "transformer, attention, nlp" {
		t.Errorf("Keywords = %q", p.Keywords)
	}
	if p.Decision != "Accept: Poster" {
		t.Errorf("Decision = %q", p.Decision)
	}
	if p.URL != "https://openreview.net/forum?id=test123" {
		t.Errorf("URL = %q", p.URL)
	}
}

func TestNotes(t *testing.T) {
	resp := apiResp{
		Count: 2,
		Notes: []wireNote{
			{ID: "a1", Number: 1, Content: mustMarshal(map[string]any{
				"title": map[string]any{"value": "Paper A"},
			})},
			{ID: "b2", Number: 2, Content: mustMarshal(map[string]any{
				"title": map[string]any{"value": "Paper B"},
			})},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/notes" {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("invitation"); got != "ICLR.cc/2024/Conference/-/Blind_Submission" {
			t.Errorf("invitation = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Error(err)
		}
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	papers, count, err := c.Notes(context.Background(), "ICLR.cc/2024/Conference/-/Blind_Submission", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
	if len(papers) != 2 {
		t.Fatalf("got %d papers, want 2", len(papers))
	}
	if papers[0].Title != "Paper A" {
		t.Errorf("papers[0].Title = %q", papers[0].Title)
	}
}

func TestNote(t *testing.T) {
	resp := apiResp{
		Count: 1,
		Notes: []wireNote{
			{ID: "xyz789", Number: 7, Content: mustMarshal(map[string]any{
				"title":   map[string]any{"value": "My Paper"},
				"authors": map[string]any{"value": []string{"Alice"}},
			})},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Error(err)
		}
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	p, err := c.Note(context.Background(), "xyz789")
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != "xyz789" {
		t.Errorf("ID = %q", p.ID)
	}
	if p.Title != "My Paper" {
		t.Errorf("Title = %q", p.Title)
	}
}

func TestNoteNotFound(t *testing.T) {
	resp := apiResp{Count: 0, Notes: nil}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Error(err)
		}
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.Note(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStrValBareString(t *testing.T) {
	var s strVal
	if err := s.UnmarshalJSON([]byte(`"hello"`)); err != nil {
		t.Fatal(err)
	}
	if s.Value != "hello" {
		t.Errorf("Value = %q, want %q", s.Value, "hello")
	}
}

func TestRawValBareSlice(t *testing.T) {
	var r rawVal
	if err := r.UnmarshalJSON([]byte(`["a","b"]`)); err != nil {
		t.Fatal(err)
	}
	if len(r.Value) != 2 || r.Value[0] != "a" {
		t.Errorf("Value = %v", r.Value)
	}
}

func mustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
