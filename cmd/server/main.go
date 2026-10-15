// Command server serves the Telegram Mini App and forwards its /v1 calls to
// the engine (v2v-demo, WEB_ADDR). It holds no business logic and no secrets.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	v2vtma "github.com/valpere/v2v-tma"
	"github.com/valpere/v2v-tma/internal/server"
)

func main() {
	env := func(k, def string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return def
	}
	cfg := server.Config{EngineURL: os.Getenv("ENGINE_URL"), WebDir: os.Getenv("WEB_DIR")}
	if cfg.WebDir == "" { // production: the client embedded in this binary
		cfg.Static = v2vtma.Client()
	}
	h, err := server.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	srv := &http.Server{
		Addr: env("ADDR", "127.0.0.1:8085"), Handler: h,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
		WriteTimeout: 90 * time.Second, // a dialogue turn can take a while
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	}()
	log.Printf("v2v-tma: listening on http://%s, engine %s", srv.Addr, cfg.EngineURL)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
