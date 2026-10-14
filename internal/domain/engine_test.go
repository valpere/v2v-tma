package domain

import (
	"errors"
	"strings"
	"testing"
)

func addedLines(resp ChatResponse) map[string]int {
	got := map[string]int{}
	for _, a := range resp.UIActions {
		if a.Type == "ADD_TO_CART" {
			got[a.Payload.ItemID] += a.Payload.Quantity
		}
	}
	return got
}

func TestProcessQueryIntents(t *testing.T) {
	repo := NewRepository()
	tests := []struct {
		domain, text, intent string
		wantAction           string
	}{
		{"pizzeria", "Хочу замовити піцу Пепероні", "add_to_cart", "ADD_TO_CART"},
		{"pizzeria", "Порадь щось гостре", "recommendation", "HIGHLIGHT_ITEM"},
		{"pizzeria", "Хочу оплатити замовлення", "open_checkout", "OPEN_CHECKOUT"},
		{"pizzeria", "Скільки коштує Маргарита?", "query_price", "HIGHLIGHT_ITEM"},
		{"lyapko", "Який аплікатор для спини?", "recommendation", "HIGHLIGHT_ITEM"},
		{"pizzeria", "розкажи анекдот", "general", ""},
	}
	for _, tc := range tests {
		resp := repo.ProcessQuery(tc.domain, tc.text, "uk")
		if resp.Intent != tc.intent {
			t.Errorf("%q: intent = %s, want %s", tc.text, resp.Intent, tc.intent)
		}
		if resp.Signal != "continue" {
			t.Errorf("%q: signal = %s, want continue", tc.text, resp.Signal)
		}
		if tc.wantAction == "" {
			continue
		}
		found := false
		for _, a := range resp.UIActions {
			found = found || a.Type == tc.wantAction
		}
		if !found {
			t.Errorf("%q: no %s action in %+v", tc.text, tc.wantAction, resp.UIActions)
		}
	}
}

// Regression: the price reply used string(rune(price)), which printed a
// control character instead of the number.
func TestPriceQueryReplyContainsDecimalPrice(t *testing.T) {
	resp := NewRepository().ProcessQuery("pizzeria", "Скільки коштує Маргарита?", "uk")
	if !strings.Contains(resp.Reply, "280") || !strings.Contains(resp.Reply, "30") {
		t.Fatalf("reply must contain the decimal prices 280 and 30, got %q", resp.Reply)
	}
}

func TestAddSeveralItemsWithQuantities(t *testing.T) {
	repo := NewRepository()
	got := addedLines(repo.ProcessQuery("pizzeria", "Додай Пепероні та Колу", "uk"))
	if got["pizza_peperoni"] != 1 || got["drink_cola"] != 1 || len(got) != 2 {
		t.Fatalf("got %v", got)
	}
	got = addedLines(repo.ProcessQuery("pizzeria", "Додай 2 пепероні та одну колу", "uk"))
	if got["pizza_peperoni"] != 2 || got["drink_cola"] != 1 {
		t.Fatalf("got %v", got)
	}
	got = addedLines(repo.ProcessQuery("pizzeria", "додай 500 пепероні", "uk"))
	if got["pizza_peperoni"] != MaxLineQty {
		t.Fatalf("quantity must be capped at %d, got %v", MaxLineQty, got)
	}
}

// "стрижку бороди" is the beard item, not the haircut: the longer phrase wins.
func TestLongestKeywordWins(t *testing.T) {
	got := addedLines(NewRepository().ProcessQuery("barbershop", "Запиши на стрижку бороди", "uk"))
	if len(got) != 1 || got["beard_trim"] != 1 {
		t.Fatalf("got %v", got)
	}
}

// "кола" must not match inside a longer unrelated word.
func TestKeywordsMatchWordPrefixOnly(t *testing.T) {
	got := addedLines(NewRepository().ProcessQuery("pizzeria", "додай школа", "uk"))
	if len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestUnknownDomainFallsBackToPizzeria(t *testing.T) {
	resp := NewRepository().ProcessQuery("nope", "додай пепероні", "uk")
	if addedLines(resp)["pizza_peperoni"] != 1 {
		t.Fatalf("got %+v", resp.UIActions)
	}
}

func TestListDomainsOrderIsStable(t *testing.T) {
	var ids []string
	for _, d := range NewRepository().ListDomains() {
		ids = append(ids, d.ID)
	}
	if strings.Join(ids, ",") != "pizzeria,barbershop,auto,lyapko" {
		t.Fatalf("got %v", ids)
	}
}

func TestQuote(t *testing.T) {
	repo := NewRepository()
	cart, err := repo.Quote("pizzeria", []CartLine{{"pizza_peperoni", 2}, {"drink_cola", 1}})
	if err != nil {
		t.Fatal(err)
	}
	if cart.Count != 3 || cart.TotalUAH != 2*340+60 || cart.TotalStars != 2*35+6 {
		t.Fatalf("got %+v", cart)
	}

	bad := []struct {
		name  string
		lines []CartLine
		want  error
	}{
		{"unknown item", []CartLine{{"haircut_classic", 1}}, ErrUnknownItem}, // item of another domain
		{"zero qty", []CartLine{{"drink_cola", 0}}, ErrBadQuantity},
		{"negative qty", []CartLine{{"drink_cola", -3}}, ErrBadQuantity},
		{"huge qty", []CartLine{{"drink_cola", MaxLineQty + 1}}, ErrBadQuantity},
		{"too many lines", make([]CartLine, MaxCartLines+1), ErrCartFull},
	}
	for _, tc := range bad {
		if _, err := repo.Quote("pizzeria", tc.lines); !errors.Is(err, tc.want) {
			t.Errorf("%s: err = %v, want %v", tc.name, err, tc.want)
		}
	}
	if _, err := repo.Quote("nope", nil); err == nil {
		t.Error("unknown domain must error")
	}
}
