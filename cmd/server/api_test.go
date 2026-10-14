package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/valpere/v2v-tma/internal/domain"
	"github.com/valpere/v2v-tma/internal/payments"
	"github.com/valpere/v2v-tma/internal/session"
)

func testHandler(devAuth bool) http.Handler {
	mux := http.NewServeMux()
	a := &api{repo: domain.NewRepository(), sessions: session.NewStore(), stars: payments.NewStarsService(""), devAuth: devAuth, now: time.Now}
	a.routes(mux)
	return mux
}

func do(h http.Handler, method, path, body, authz string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if authz != "" {
		r.Header.Set("Authorization", authz)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func newSession(t *testing.T, h http.Handler) string {
	t.Helper()
	w := do(h, "POST", "/v1/sessions", `{"topic":"pizzeria"}`, "")
	var out struct {
		ID string `json:"session_id"`
	}
	if w.Code != http.StatusCreated || json.Unmarshal(w.Body.Bytes(), &out) != nil || out.ID == "" {
		t.Fatalf("create session: %d %s", w.Code, w.Body)
	}
	return out.ID
}

func TestAuthRequiredUnlessDevAuth(t *testing.T) {
	if w := do(testHandler(false), "POST", "/v1/sessions", `{"topic":"pizzeria"}`, ""); w.Code != http.StatusUnauthorized {
		t.Errorf("no header: %d", w.Code)
	}
	// the old "dev_demo_user" bypass must be gone
	if w := do(testHandler(false), "POST", "/v1/sessions", `{"topic":"pizzeria"}`, "tma dev_demo_user"); w.Code != http.StatusUnauthorized {
		t.Errorf("dev prefix: %d", w.Code)
	}
	if w := do(testHandler(true), "POST", "/v1/sessions", `{"topic":"pizzeria"}`, ""); w.Code != http.StatusCreated {
		t.Errorf("dev auth: %d", w.Code)
	}
}

func TestMessageFillsServerCartAndCheckoutIgnoresClientAmounts(t *testing.T) {
	h := testHandler(true)
	id := newSession(t, h)

	w := do(h, "POST", "/v1/sessions/"+id+"/messages", `{"text":"додай 2 пепероні та колу"}`, "")
	var m messageResponse
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &m) != nil {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if m.Cart.Count != 3 || m.Cart.TotalStars != 2*35+6 {
		t.Fatalf("cart = %+v", m.Cart)
	}

	// Client-supplied amounts are ignored: unknown fields change nothing.
	w = do(h, "POST", "/v1/sessions/"+id+"/checkout", `{"stars_amount":1}`, "")
	var co struct {
		Stars int  `json:"stars"`
		Mock  bool `json:"mock"`
	}
	json.Unmarshal(w.Body.Bytes(), &co)
	if w.Code != 200 || co.Stars != 76 || !co.Mock {
		t.Fatalf("checkout: %d %s", w.Code, w.Body)
	}
}

func TestCheckoutEmptyCartAndBadItems(t *testing.T) {
	h := testHandler(true)
	id := newSession(t, h)
	if w := do(h, "POST", "/v1/sessions/"+id+"/checkout", `{}`, ""); w.Code != http.StatusConflict {
		t.Errorf("empty cart: %d", w.Code)
	}
	if w := do(h, "PUT", "/v1/sessions/"+id+"/cart/items/nope", `{"quantity":1}`, ""); w.Code != http.StatusBadRequest {
		t.Errorf("unknown item: %d", w.Code)
	}
	if w := do(h, "PUT", "/v1/sessions/"+id+"/cart/items/drink_cola", `{"quantity":999}`, ""); w.Code != http.StatusBadRequest {
		t.Errorf("huge quantity: %d", w.Code)
	}
	if w := do(h, "GET", "/v1/sessions/"+id+"/cart", ``, ""); !strings.Contains(w.Body.String(), `"count":0`) {
		t.Errorf("a rejected change must leave the cart untouched: %s", w.Body)
	}
}

func TestSessionOfAnotherPrincipalIsNotFound(t *testing.T) {
	h := testHandler(true)
	id := newSession(t, h)
	a := &api{repo: domain.NewRepository(), sessions: session.NewStore(), stars: payments.NewStarsService(""), devAuth: true, now: time.Now}
	s, _ := a.sessions.Create("someone-else", "pizzeria")
	mux := http.NewServeMux()
	a.routes(mux)
	if w := do(mux, "GET", "/v1/sessions/"+s.ID+"/cart", ``, ""); w.Code != http.StatusNotFound {
		t.Errorf("foreign session: %d", w.Code)
	}
	_ = id
}

func TestConfigValidation(t *testing.T) {
	if (config{}).validate() == nil {
		t.Error("no token, no DEV_AUTH, no ENGINE_URL must refuse to start")
	}
	for _, c := range []config{{DevAuth: true}, {BotToken: "1:x"}, {EngineURL: "http://x"}} {
		if err := c.validate(); err != nil {
			t.Errorf("%+v: %v", c, err)
		}
	}
}

func TestEngineURLProxiesV1(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("engine:" + r.URL.Path + ":" + r.Header.Get("Authorization")))
	}))
	defer up.Close()
	h, err := newHandler(config{EngineURL: up.URL, WebDir: "../../web"})
	if err != nil {
		t.Fatal(err)
	}
	if w := do(h, "GET", "/v1/topics", "", "tma abc"); w.Body.String() != "engine:/v1/topics:tma abc" {
		t.Errorf("got %q", w.Body)
	}
}
