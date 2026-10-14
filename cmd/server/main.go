// Command server serves the Mini App static files and the /v1 API. With
// ENGINE_URL set it proxies /v1 to the real engine (v2v-demo); otherwise it
// answers /v1 itself with a deterministic stub of the same contract.
package main

import (
	"errors"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/valpere/v2v-tma/internal/domain"
	"github.com/valpere/v2v-tma/internal/payments"
	"github.com/valpere/v2v-tma/internal/session"
)

type config struct {
	Port      string
	BotToken  string
	DevAuth   bool   // DEV_AUTH=1: accept requests without initData (local development only)
	EngineURL string // ENGINE_URL: proxy /v1 to the real engine
	WebDir    string
}

func loadConfig() config {
	c := config{
		Port:      getenv("PORT", "8085"),
		BotToken:  os.Getenv("TELEGRAM_BOT_TOKEN"),
		DevAuth:   os.Getenv("DEV_AUTH") == "1",
		EngineURL: os.Getenv("ENGINE_URL"),
		WebDir:    getenv("WEB_DIR", "web"),
	}
	return c
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func (c config) validate() error {
	if c.EngineURL == "" && c.BotToken == "" && !c.DevAuth {
		return errors.New("set TELEGRAM_BOT_TOKEN, or DEV_AUTH=1 for local development, or ENGINE_URL")
	}
	return nil
}

func newHandler(c config) (http.Handler, error) {
	mux := http.NewServeMux()
	if c.EngineURL != "" {
		target, err := url.Parse(c.EngineURL)
		if err != nil || target.Host == "" {
			return nil, errors.New("ENGINE_URL must be an absolute URL")
		}
		mux.Handle("/v1/", &httputil.ReverseProxy{Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target) // Authorization (the initData) is forwarded untouched
		}})
	} else {
		a := &api{
			repo:     domain.NewRepository(),
			sessions: session.NewStore(),
			stars:    payments.NewStarsService(c.BotToken),
			botToken: c.BotToken,
			devAuth:  c.DevAuth,
			now:      time.Now,
		}
		a.routes(mux)
	}
	mux.Handle("/", http.FileServer(http.Dir(c.WebDir)))
	return mux, nil
}

func main() {
	c := loadConfig()
	if err := c.validate(); err != nil {
		log.Fatal(err)
	}
	h, err := newHandler(c)
	if err != nil {
		log.Fatal(err)
	}
	srv := &http.Server{
		Addr:              ":" + c.Port,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	mode := "stub engine"
	if c.EngineURL != "" {
		mode = "proxy to " + c.EngineURL
	}
	log.Printf("v2v-tma listening on http://localhost:%s (%s, dev_auth=%t)", c.Port, mode, c.DevAuth)
	log.Fatal(srv.ListenAndServe())
}
