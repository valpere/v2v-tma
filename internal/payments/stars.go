// Package payments creates Telegram Stars (XTR) invoice links. Confirming the
// payment (pre_checkout_query / successful_payment) needs a bot update loop,
// which belongs to the engine (v2v-demo), not to this static-client repo.
package payments

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const telegramAPI = "https://api.telegram.org"

type StarsService struct {
	botToken   string
	baseURL    string
	httpClient *http.Client
}

func NewStarsService(botToken string) *StarsService {
	return &StarsService{
		botToken:   botToken,
		baseURL:    telegramAPI,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Mock reports whether invoices are fake links (no bot token configured).
func (s *StarsService) Mock() bool { return s.botToken == "" }

type LabeledPrice struct {
	Label  string `json:"label"`
	Amount int    `json:"amount"` // in XTR (stars)
}

type CreateInvoiceRequest struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Payload     string         `json:"payload"`
	Currency    string         `json:"currency"` // "XTR"
	Prices      []LabeledPrice `json:"prices"`
}

type CreateInvoiceResponse struct {
	OK          bool   `json:"ok"`
	Result      string `json:"result"` // invoice link
	Description string `json:"description"`
}

// CreateStarsInvoice returns a payment link for starsAmount stars. Without a
// bot token it returns a clearly fake link so the demo flow can run offline.
func (s *StarsService) CreateStarsInvoice(title, description, orderID string, starsAmount int) (string, error) {
	if starsAmount <= 0 {
		return "", fmt.Errorf("invalid stars amount %d", starsAmount)
	}
	if s.Mock() {
		return fmt.Sprintf("https://t.me/$demo_stars_invoice_%s_%d", orderID, starsAmount), nil
	}

	body, err := json.Marshal(CreateInvoiceRequest{
		Title:       title,
		Description: description,
		Payload:     orderID,
		Currency:    "XTR",
		Prices:      []LabeledPrice{{Label: "Разом", Amount: starsAmount}},
	})
	if err != nil {
		return "", err
	}
	resp, err := s.httpClient.Post(fmt.Sprintf("%s/bot%s/createInvoiceLink", s.baseURL, s.botToken), "application/json", bytes.NewReader(body))
	if err != nil {
		// The URL embeds the bot token; never surface it in an error.
		return "", fmt.Errorf("createInvoiceLink request failed")
	}
	defer resp.Body.Close()

	var tg CreateInvoiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&tg); err != nil {
		return "", fmt.Errorf("createInvoiceLink: bad response (HTTP %d)", resp.StatusCode)
	}
	if !tg.OK {
		return "", fmt.Errorf("createInvoiceLink: %s", tg.Description)
	}
	return tg.Result, nil
}
