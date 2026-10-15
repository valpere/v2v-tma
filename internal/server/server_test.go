package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func newEngine(t *testing.T) (*httptest.Server, *http.Request) {
	t.Helper()
	var got http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = *r.Clone(r.Context())
		io.WriteString(w, `{"ok":true}`)
	}))
	t.Cleanup(srv.Close)
	return srv, &got
}

func handler(t *testing.T, engineURL string) http.Handler {
	t.Helper()
	h, err := New(Config{EngineURL: engineURL, Static: fstest.MapFS{"index.html": {Data: []byte("<h1>tma</h1>")}}})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestServesTheClientAndHealth(t *testing.T) {
	h := handler(t, "http://127.0.0.1:1")
	for path, want := range map[string]string{"/": "<h1>tma</h1>", "/healthz": ""} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 || !strings.Contains(w.Body.String(), want) {
			t.Errorf("%s: %d %q", path, w.Code, w.Body)
		}
	}
}

func TestProxiesV1WithTheInitDataUntouched(t *testing.T) {
	srv, got := newEngine(t)
	h := handler(t, srv.URL)
	r := httptest.NewRequest("POST", "/v1/sessions", strings.NewReader(`{"topic":"pizzeria"}`))
	r.Header.Set("Authorization", "tma query_id=1&hash=abc")
	r.Header.Set("X-Principal", "web:forged")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || got.URL.Path != "/v1/sessions" {
		t.Fatalf("%d, engine saw %q", w.Code, got.URL.Path)
	}
	if got.Header.Get("Authorization") != "tma query_id=1&hash=abc" {
		t.Errorf("Authorization = %q", got.Header.Get("Authorization"))
	}
	if got.Header.Get("X-Principal") != "" {
		t.Errorf("a client-supplied X-Principal must not reach the engine: %q", got.Header.Get("X-Principal"))
	}
}

func TestOnlyV1IsProxied(t *testing.T) {
	srv, got := newEngine(t)
	h := handler(t, srv.URL)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	if w.Code != http.StatusNotFound || got.URL != nil {
		t.Fatalf("%d, engine reached: %v", w.Code, got.URL != nil)
	}
}

func TestEngineDownIsAJSON502(t *testing.T) {
	h := handler(t, "http://127.0.0.1:1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/v1/topics", nil))
	if w.Code != http.StatusBadGateway || !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("%d %v", w.Code, w.Header())
	}
}

func TestNewRejectsABadEngineURL(t *testing.T) {
	for _, u := range []string{"", "nope", "ftp://x"} {
		if _, err := New(Config{EngineURL: u}); err == nil {
			t.Errorf("ENGINE_URL %q accepted", u)
		}
	}
}

func TestSecurityHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	handler(t, "http://127.0.0.1:1").ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Header().Get("X-Content-Type-Options") != "nosniff" || w.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatalf("%v", w.Header())
	}
}
