package v2vtma_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	v2vtma "github.com/valpere/v2v-tma"
	"github.com/valpere/v2v-tma/internal/server"
)

// The embedded client must be served with the types a browser needs to run it
// (ES modules are refused unless served as JavaScript) and must not depend on a runtime CDN.
func TestEmbeddedClientIsServable(t *testing.T) {
	h, err := server.New(server.Config{EngineURL: "http://127.0.0.1:1", Static: v2vtma.Client()})
	if err != nil {
		t.Fatal(err)
	}
	get := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		return w
	}
	for path, typ := range map[string]string{"/assets/app.mjs": "javascript", "/assets/core.mjs": "javascript", "/assets/recorder.mjs": "javascript", "/assets/tailwind.css": "text/css"} {
		w := get(path)
		if w.Code != http.StatusOK || !strings.Contains(w.Header().Get("Content-Type"), typ) {
			t.Errorf("%s: %d %q", path, w.Code, w.Header().Get("Content-Type"))
		}
	}
	if body := get("/").Body.String(); strings.Contains(body, "cdn.tailwindcss.com") || !strings.Contains(body, "/assets/tailwind.css") {
		t.Error("index.html must use the precompiled stylesheet, not the Tailwind CDN")
	}
}
