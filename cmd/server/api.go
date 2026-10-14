package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/valpere/v2v-tma/internal/auth"
	"github.com/valpere/v2v-tma/internal/domain"
	"github.com/valpere/v2v-tma/internal/payments"
	"github.com/valpere/v2v-tma/internal/session"
)

const (
	maxBody    = 16 << 10
	maxTextLen = 500
	authScheme = "tma "
)

// api is the stub implementation of the engine's /v1 contract.
type api struct {
	repo     *domain.Repository
	sessions *session.Store
	stars    *payments.StarsService
	botToken string
	devAuth  bool
	now      func() time.Time
}

func (a *api) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/topics", a.topics)
	mux.HandleFunc("POST /v1/sessions", a.authed(a.createSession))
	mux.HandleFunc("POST /v1/sessions/{id}/messages", a.authed(a.withSession(a.message)))
	mux.HandleFunc("GET /v1/sessions/{id}/cart", a.authed(a.withSession(a.getCart)))
	mux.HandleFunc("PUT /v1/sessions/{id}/cart/items/{item}", a.authed(a.withSession(a.setItem)))
	mux.HandleFunc("DELETE /v1/sessions/{id}/cart", a.authed(a.withSession(a.clearCart)))
	mux.HandleFunc("POST /v1/sessions/{id}/checkout", a.authed(a.withSession(a.checkout)))
}

type principalHandler func(w http.ResponseWriter, r *http.Request, p auth.Principal)
type sessionHandler func(w http.ResponseWriter, r *http.Request, s *session.Session)

// authed resolves the caller from "Authorization: tma <initData>". An invalid
// or expired signature is always 401; a missing header is accepted only with
// DEV_AUTH=1.
func (a *api) authed(h principalHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hdr := r.Header.Get("Authorization")
		if hdr == "" {
			if a.devAuth {
				h(w, r, auth.DevPrincipal)
				return
			}
			fail(w, http.StatusUnauthorized, "missing Authorization")
			return
		}
		initData, ok := strings.CutPrefix(hdr, authScheme)
		if !ok {
			fail(w, http.StatusUnauthorized, "unsupported Authorization scheme")
			return
		}
		if a.devAuth && a.botToken == "" {
			h(w, r, auth.DevPrincipal) // nothing to verify against in dev
			return
		}
		p, err := auth.Validate(initData, a.botToken, a.now())
		if err != nil {
			log.Printf("auth rejected: %v", err)
			fail(w, http.StatusUnauthorized, "invalid initData")
			return
		}
		h(w, r, p)
	}
}

func (a *api) withSession(h sessionHandler) principalHandler {
	return func(w http.ResponseWriter, r *http.Request, p auth.Principal) {
		s, err := a.sessions.Get(r.PathValue("id"), p.ID)
		if err != nil {
			fail(w, http.StatusNotFound, "session not found")
			return
		}
		h(w, r, s)
	}
}

func (a *api) topics(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.repo.ListDomains())
}

func (a *api) createSession(w http.ResponseWriter, r *http.Request, p auth.Principal) {
	var req struct {
		Topic string `json:"topic"`
	}
	if !decode(w, r, &req) {
		return
	}
	if _, ok := a.repo.GetDomain(req.Topic); !ok {
		fail(w, http.StatusBadRequest, "unknown topic")
		return
	}
	s, err := a.sessions.Create(p.ID, req.Topic)
	if err != nil {
		fail(w, http.StatusServiceUnavailable, "too many sessions")
		return
	}
	cart, _ := a.repo.Quote(s.Domain, nil)
	writeJSON(w, http.StatusCreated, map[string]any{"session_id": s.ID, "topic": s.Domain, "cart": cart})
}

type messageResponse struct {
	domain.ChatResponse
	Cart domain.Cart `json:"cart"`
}

func (a *api) message(w http.ResponseWriter, r *http.Request, s *session.Session) {
	var req struct {
		Text string `json:"text"`
		Lang string `json:"lang"`
	}
	if !decode(w, r, &req) {
		return
	}
	text := strings.TrimSpace(req.Text)
	if text == "" || utf8.RuneCountInString(text) > maxTextLen {
		fail(w, http.StatusBadRequest, fmt.Sprintf("text must be 1..%d characters", maxTextLen))
		return
	}
	lang := "uk"
	if req.Lang == "en" {
		lang = "en"
	}

	resp := a.repo.ProcessQuery(s.Domain, text, lang)

	// The server owns the cart: apply ADD_TO_CART here, roll back if the
	// result would break a cart limit.
	prev := s.Lines()
	for _, act := range resp.UIActions {
		if act.Type == "ADD_TO_CART" {
			s.Add(act.Payload.ItemID, act.Payload.Quantity)
		}
	}
	cart, err := a.repo.Quote(s.Domain, s.Lines())
	if err != nil {
		s.Replace(prev)
		cart, _ = a.repo.Quote(s.Domain, prev)
		resp.UIActions = withoutAdds(resp.UIActions)
		resp.Reply = fmt.Sprintf("Не вдалося додати: у кошику може бути до %d одиниць однієї позиції та до %d позицій.", domain.MaxLineQty, domain.MaxCartLines)
	}
	writeJSON(w, http.StatusOK, messageResponse{ChatResponse: resp, Cart: cart})
}

func withoutAdds(in []domain.UIAction) []domain.UIAction {
	out := make([]domain.UIAction, 0, len(in))
	for _, a := range in {
		if a.Type != "ADD_TO_CART" {
			out = append(out, a)
		}
	}
	return out
}

func (a *api) getCart(w http.ResponseWriter, r *http.Request, s *session.Session) {
	cart, _ := a.repo.Quote(s.Domain, s.Lines())
	writeJSON(w, http.StatusOK, cart)
}

func (a *api) setItem(w http.ResponseWriter, r *http.Request, s *session.Session) {
	var req struct {
		Quantity int `json:"quantity"`
	}
	if !decode(w, r, &req) {
		return
	}
	prev := s.Lines()
	s.Set(r.PathValue("item"), req.Quantity)
	cart, err := a.repo.Quote(s.Domain, s.Lines())
	if err != nil {
		s.Replace(prev)
		fail(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cart)
}

func (a *api) clearCart(w http.ResponseWriter, r *http.Request, s *session.Session) {
	s.Clear()
	cart, _ := a.repo.Quote(s.Domain, nil)
	writeJSON(w, http.StatusOK, cart)
}

// checkout prices the server-side cart and creates a Stars invoice for it. The
// request carries no amounts at all.
func (a *api) checkout(w http.ResponseWriter, r *http.Request, s *session.Session) {
	cart, err := a.repo.Quote(s.Domain, s.Lines())
	if err != nil || cart.Count == 0 {
		fail(w, http.StatusConflict, "cart is empty")
		return
	}
	orderID, err := newOrderID()
	if err != nil {
		fail(w, http.StatusInternalServerError, "internal error")
		return
	}
	d, _ := a.repo.GetDomain(s.Domain)
	link, err := a.stars.CreateStarsInvoice(d.Name, fmt.Sprintf("Замовлення (%d позицій)", cart.Count), orderID, cart.TotalStars)
	if err != nil {
		log.Printf("checkout %s: %v", orderID, err)
		fail(w, http.StatusBadGateway, "failed to create invoice")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"order_id":    orderID,
		"invoice_url": link,
		"stars":       cart.TotalStars,
		"total_uah":   cart.TotalUAH,
		"mock":        a.stars.Mock(),
	})
}

func newOrderID() (string, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "ord_" + hex.EncodeToString(b[:]), nil
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			fail(w, http.StatusRequestEntityTooLarge, "request too large")
		} else {
			fail(w, http.StatusBadRequest, "bad request")
		}
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
