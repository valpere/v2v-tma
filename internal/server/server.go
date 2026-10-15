// Package server serves the Mini App and forwards /v1 to the engine. The
// Mini App authenticates every call with Telegram initData in the Authorization
// header (no cookie, so there is no ambient credential to forge a request
// with); the engine validates it. This server holds no secrets.
package server

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// Config is what New needs.
type Config struct {
	EngineURL string // base URL of the engine's end-user /v1 API (v2v-demo WEB_ADDR)
	WebDir    string // static files on disk (development)
	Static    fs.FS  // the embedded client; wins over WebDir
}

// New returns the handler: /healthz, /v1/* -> engine, everything else static.
func New(cfg Config) (http.Handler, error) {
	if cfg.EngineURL == "" {
		return nil, errors.New("ENGINE_URL is required")
	}
	target, err := url.Parse(cfg.EngineURL)
	if err != nil || (target.Scheme != "http" && target.Scheme != "https") || target.Host == "" {
		return nil, fmt.Errorf("ENGINE_URL %q: want http(s)://host[:port]", cfg.EngineURL)
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.Out.Header.Del("X-Principal") // only the trusted-frontend scheme may set it, and that is not us
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":"engine unavailable"}`))
		},
	}
	var static http.FileSystem = http.Dir(cfg.WebDir)
	if cfg.Static != nil {
		static = http.FS(cfg.Static)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.Handle("/v1/", proxy)
	mux.Handle("/", http.FileServer(static))
	return headers(mux), nil
}

// headers sets what is safe for a page Telegram embeds in a webview (so no
// frame-ancestors / X-Frame-Options here).
func headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
