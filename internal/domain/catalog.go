package domain

import (
	"embed"
	"encoding/json"
	"fmt"
	"sort"
)

//go:embed catalog/*.json
var catalogFS embed.FS

// Item is one orderable catalog entry. The catalog files are the single source
// of truth for prices; clients only ever send item ids and quantities.
type Item struct {
	ID          string   `json:"item_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	PriceUAH    int      `json:"price_uah"`
	PriceStars  int      `json:"price_stars"`
	Category    string   `json:"category"`
	Icon        string   `json:"icon"`
	Badge       string   `json:"badge,omitempty"`
	Keywords    []string `json:"keywords,omitempty"` // lower-case word prefixes the stub matcher looks for
}

// DomainInfo is one business vertical (a "topic" in the engine API).
type DomainInfo struct {
	Order       int      `json:"order"`
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Tagline     string   `json:"tagline"`
	Icon        string   `json:"icon"`
	Welcome     string   `json:"welcome"`
	VoiceSample string   `json:"voice_sample"`
	Prompts     []string `json:"quick_prompts"`
	Items       []Item   `json:"items"`
}

// ActionPayload carries the arguments of a UIAction; unused fields are omitted.
type ActionPayload struct {
	ItemID     string `json:"item_id,omitempty"`
	Quantity   int    `json:"quantity,omitempty"`
	DurationMS int    `json:"duration_ms,omitempty"`
	Domain     string `json:"domain,omitempty"`
}

// UIAction is an instruction for the client UI (HIGHLIGHT_ITEM, ADD_TO_CART,
// OPEN_CHECKOUT). The server applies ADD_TO_CART to the session cart itself;
// the client only animates it.
type UIAction struct {
	Type    string        `json:"type"`
	Payload ActionPayload `json:"payload"`
}

// ChatResponse is the result of one concierge turn.
type ChatResponse struct {
	Reply     string     `json:"reply"`
	Signal    string     `json:"signal"` // continue | escalate (the engine's lead_ready is not used by the stub)
	Language  string     `json:"language"`
	Intent    string     `json:"intent"`
	UIActions []UIAction `json:"ui_actions"`
}

// Repository is the read-only in-memory catalog.
type Repository struct {
	domains map[string]DomainInfo
}

// NewRepository loads the embedded catalog files. A malformed embedded file is
// a build-time defect, so it panics rather than returning an error.
func NewRepository() *Repository {
	r := &Repository{domains: make(map[string]DomainInfo)}
	files, err := catalogFS.ReadDir("catalog")
	if err != nil {
		panic(fmt.Sprintf("domain: read embedded catalog: %v", err))
	}
	for _, f := range files {
		raw, err := catalogFS.ReadFile("catalog/" + f.Name())
		if err != nil {
			panic(fmt.Sprintf("domain: read %s: %v", f.Name(), err))
		}
		var d DomainInfo
		if err := json.Unmarshal(raw, &d); err != nil {
			panic(fmt.Sprintf("domain: parse %s: %v", f.Name(), err))
		}
		r.domains[d.ID] = d
	}
	return r
}

func (r *Repository) GetDomain(id string) (DomainInfo, bool) {
	d, ok := r.domains[id]
	return d, ok
}

// ListDomains returns the domains in their catalog "order".
func (r *Repository) ListDomains() []DomainInfo {
	list := make([]DomainInfo, 0, len(r.domains))
	for _, d := range r.domains {
		list = append(list, d)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Order < list[j].Order })
	return list
}
