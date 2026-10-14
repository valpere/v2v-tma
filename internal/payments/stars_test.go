package payments

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMockInvoiceWithoutToken(t *testing.T) {
	s := NewStarsService("")
	link, err := s.CreateStarsInvoice("t", "d", "ord_1", 35)
	if err != nil || !strings.Contains(link, "demo_stars_invoice_ord_1_35") || !s.Mock() {
		t.Fatalf("got %q, %v", link, err)
	}
}

func TestRejectsNonPositiveAmount(t *testing.T) {
	for _, n := range []int{0, -5} {
		if _, err := NewStarsService("").CreateStarsInvoice("t", "d", "o", n); err == nil {
			t.Errorf("amount %d must be rejected", n)
		}
	}
}

func TestRealInvoiceRequest(t *testing.T) {
	var got CreateInvoiceRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bot123:TOK/createInvoiceLink" {
			t.Errorf("path = %s", r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&got)
		w.Write([]byte(`{"ok":true,"result":"https://t.me/$abc"}`))
	}))
	defer srv.Close()

	s := NewStarsService("123:TOK")
	s.baseURL = srv.URL
	link, err := s.CreateStarsInvoice("title", "desc", "ord_9", 75)
	if err != nil || link != "https://t.me/$abc" {
		t.Fatalf("got %q, %v", link, err)
	}
	if got.Currency != "XTR" || got.Payload != "ord_9" || len(got.Prices) != 1 || got.Prices[0].Amount != 75 {
		t.Fatalf("request = %+v", got)
	}
}

func TestTelegramErrorDoesNotLeakToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":false,"description":"Bad Request: nope"}`))
	}))
	s := NewStarsService("123:SECRET")
	s.baseURL = srv.URL
	_, err := s.CreateStarsInvoice("t", "d", "o", 5)
	if err == nil || !strings.Contains(err.Error(), "nope") || strings.Contains(err.Error(), "SECRET") {
		t.Fatalf("err = %v", err)
	}
	srv.Close()
	_, err = s.CreateStarsInvoice("t", "d", "o", 5) // connection refused
	if err == nil || strings.Contains(err.Error(), "SECRET") {
		t.Fatalf("transport error leaked the token: %v", err)
	}
}
