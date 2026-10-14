package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/valpere/v2v-tma/internal/auth"
	"github.com/valpere/v2v-tma/internal/domain"
	"github.com/valpere/v2v-tma/internal/payments"
)

type Server struct {
	repo         *domain.Repository
	starsService *payments.StarsService
	botToken     string
}

func main() {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8085"
	}

	repo := domain.NewRepository()
	starsService := payments.NewStarsService(botToken)

	srv := &Server{
		repo:         repo,
		starsService: starsService,
		botToken:     botToken,
	}

	mux := http.NewServeMux()

	// API ендпоінти
	mux.HandleFunc("GET /api/domains", srv.handleListDomains)
	mux.HandleFunc("POST /api/chat", srv.handleChat)
	mux.HandleFunc("POST /api/checkout/stars", srv.handleCheckoutStars)

	// Статика TMA
	fs := http.FileServer(http.Dir("web"))
	mux.Handle("/", fs)

	log.Printf("🚀 v2v-tma сервер запущено на http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func (s *Server) handleListDomains(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	domains := s.repo.ListDomains()
	json.NewEncoder(w).Encode(domains)
}

type ChatRequest struct {
	DomainID string `json:"domain_id"`
	Text     string `json:"text"`
	Lang     string `json:"lang"`
	InitData string `json:"init_data"`
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Перевірка initData Telegram
	if valid, err := auth.ValidateInitData(req.InitData, s.botToken); !valid || err != nil {
		log.Printf("Warning: invalid initData: %v", err)
	}

	if req.Lang == "" {
		req.Lang = "uk"
	}

	resp := s.repo.ProcessQuery(req.DomainID, req.Text, req.Lang)
	json.NewEncoder(w).Encode(resp)
}

type CheckoutStarsRequest struct {
	DomainID    string        `json:"domain_id"`
	StarsAmount int           `json:"stars_amount"`
	CartItems   []interface{} `json:"cart_items"`
	InitData    string        `json:"init_data"`
}

func (s *Server) handleCheckoutStars(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req CheckoutStarsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	orderID := fmt.Sprintf("ord_%d", os.Getpid())
	title := "Оплата замовлення"
	description := fmt.Sprintf("Оплата кошика (%d ⭐ Stars)", req.StarsAmount)

	invoiceLink, err := s.starsService.CreateStarsInvoice(title, description, orderID, req.StarsAmount)
	if err != nil {
		http.Error(w, "Failed to create stars invoice", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":          true,
		"invoice_url": invoiceLink,
		"stars":       req.StarsAmount,
	})
}
