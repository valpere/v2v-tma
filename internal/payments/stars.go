package payments

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type StarsService struct {
	botToken   string
	httpClient *http.Client
}

func NewStarsService(botToken string) *StarsService {
	return &StarsService{
		botToken: botToken,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type LabeledPrice struct {
	Label  string `json:"label"`
	Amount int    `json:"amount"` // у XTR (зірках)
}

type CreateInvoiceRequest struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Payload     string         `json:"payload"`
	Currency    string         `json:"currency"` // "XTR"
	Prices      []LabeledPrice `json:"prices"`
}

type CreateInvoiceResponse struct {
	OK     bool   `json:"ok"`
	Result string `json:"result"` // посилання на інвойс t.me/$...
}

// CreateStarsInvoice створює посилання на оплату Telegram Stars
func (s *StarsService) CreateStarsInvoice(title, description, orderID string, starsAmount int) (string, error) {
	if s.botToken == "" {
		// Mock link для демо-режиму
		return fmt.Sprintf("https://t.me/$demo_stars_invoice_%s_%d", orderID, starsAmount), nil
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/createInvoiceLink", s.botToken)

	reqBody := CreateInvoiceRequest{
		Title:       title,
		Description: description,
		Payload:     orderID,
		Currency:    "XTR",
		Prices: []LabeledPrice{
			{Label: "Разом", Amount: starsAmount},
		},
	}

	payloadBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := s.httpClient.Post(apiURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var tgResp CreateInvoiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&tgResp); err != nil {
		return "", err
	}

	if !tgResp.OK {
		return "", fmt.Errorf("telegram API error creating invoice")
	}

	return tgResp.Result, nil
}
