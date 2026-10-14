package domain

import (
	"testing"
)

func TestProcessQuery(t *testing.T) {
	repo := NewRepository()

	// 1. Тест додавання піци в кошик
	resp := repo.ProcessQuery("pizzeria", "Хочу замовити піцу Пепероні", "uk")
	if resp.Intent != "add_to_cart" {
		t.Fatalf("expected intent add_to_cart, got %s", resp.Intent)
	}
	if len(resp.UIActions) < 2 {
		t.Fatalf("expected at least 2 ui actions, got %d", len(resp.UIActions))
	}

	// 2. Тест рекомендації гострого
	respHot := repo.ProcessQuery("pizzeria", "Порадь щось гостре", "uk")
	if respHot.Intent != "recommendation" {
		t.Fatalf("expected intent recommendation, got %s", respHot.Intent)
	}

	// 3. Тест відкриття чека
	respCheckout := repo.ProcessQuery("pizzeria", "Хочу оплатити замовлення", "uk")
	if respCheckout.Intent != "open_checkout" {
		t.Fatalf("expected intent open_checkout, got %s", respCheckout.Intent)
	}
}
