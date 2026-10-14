package domain

import (
	"errors"
	"fmt"
)

const (
	MaxLineQty   = 20
	MaxCartLines = 20
)

// CartLine is the only thing a client may state about an order line.
type CartLine struct {
	ItemID   string `json:"item_id"`
	Quantity int    `json:"quantity"`
}

// CartItem is a priced cart line, computed on the server.
type CartItem struct {
	ItemID     string `json:"item_id"`
	Title      string `json:"title"`
	Icon       string `json:"icon"`
	PriceUAH   int    `json:"price_uah"`
	PriceStars int    `json:"price_stars"`
	Quantity   int    `json:"quantity"`
}

// Cart is the authoritative priced view of a session's lines.
type Cart struct {
	Items      []CartItem `json:"items"`
	Count      int        `json:"count"`
	TotalUAH   int        `json:"total_uah"`
	TotalStars int        `json:"total_stars"`
}

var (
	ErrUnknownItem = errors.New("unknown item")
	ErrBadQuantity = errors.New("bad quantity")
	ErrCartFull    = errors.New("cart is full")
)

// Quote prices lines against the catalog of domainID. It rejects unknown ids,
// quantities outside 1..MaxLineQty and more than MaxCartLines lines.
func (r *Repository) Quote(domainID string, lines []CartLine) (Cart, error) {
	d, ok := r.domains[domainID]
	if !ok {
		return Cart{}, fmt.Errorf("unknown domain %q", domainID)
	}
	if len(lines) > MaxCartLines {
		return Cart{}, ErrCartFull
	}
	byID := make(map[string]Item, len(d.Items))
	for _, it := range d.Items {
		byID[it.ID] = it
	}
	cart := Cart{Items: make([]CartItem, 0, len(lines))}
	for _, l := range lines {
		it, ok := byID[l.ItemID]
		if !ok {
			return Cart{}, fmt.Errorf("%w: %q", ErrUnknownItem, l.ItemID)
		}
		if l.Quantity < 1 || l.Quantity > MaxLineQty {
			return Cart{}, fmt.Errorf("%w: %d", ErrBadQuantity, l.Quantity)
		}
		cart.Items = append(cart.Items, CartItem{
			ItemID: it.ID, Title: it.Title, Icon: it.Icon,
			PriceUAH: it.PriceUAH, PriceStars: it.PriceStars, Quantity: l.Quantity,
		})
		cart.Count += l.Quantity
		cart.TotalUAH += it.PriceUAH * l.Quantity
		cart.TotalStars += it.PriceStars * l.Quantity
	}
	return cart, nil
}
