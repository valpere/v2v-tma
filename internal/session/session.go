// Package session keeps per-user concierge sessions (a domain plus a cart) in
// memory. Sessions are bound to the authenticated principal and expire.
package session

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/valpere/v2v-tma/internal/domain"
)

const (
	TTL         = 24 * time.Hour
	MaxSessions = 10000
)

var (
	ErrNotFound = errors.New("session not found")
	ErrFull     = errors.New("too many sessions")
)

// Session is one conversation: a domain and the cart lines of its owner.
type Session struct {
	ID        string
	Principal string
	Domain    string

	mu      sync.Mutex
	lines   []domain.CartLine
	expires time.Time
}

// Lines returns a copy of the cart lines.
func (s *Session) Lines() []domain.CartLine {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]domain.CartLine(nil), s.lines...)
}

// Add increases the quantity of an item (or appends a line).
func (s *Session) Add(itemID string, qty int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.lines {
		if s.lines[i].ItemID == itemID {
			s.lines[i].Quantity += qty
			return
		}
	}
	s.lines = append(s.lines, domain.CartLine{ItemID: itemID, Quantity: qty})
}

// Set sets an item's quantity; zero removes the line.
func (s *Session) Set(itemID string, qty int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.lines {
		if s.lines[i].ItemID == itemID {
			if qty <= 0 {
				s.lines = append(s.lines[:i], s.lines[i+1:]...)
			} else {
				s.lines[i].Quantity = qty
			}
			return
		}
	}
	if qty > 0 {
		s.lines = append(s.lines, domain.CartLine{ItemID: itemID, Quantity: qty})
	}
}

// Clear empties the cart.
func (s *Session) Clear() {
	s.mu.Lock()
	s.lines = nil
	s.mu.Unlock()
}

// Store is a concurrency-safe in-memory session registry.
type Store struct {
	mu       sync.Mutex
	sessions map[string]*Session
	now      func() time.Time
}

func NewStore() *Store {
	return &Store{sessions: make(map[string]*Session), now: time.Now}
}

// Create starts a session with an unguessable 128-bit id.
func (st *Store) Create(principal, domainID string) (*Session, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	now := st.now()
	if len(st.sessions) >= MaxSessions {
		for id, s := range st.sessions {
			if now.After(s.expires) {
				delete(st.sessions, id)
			}
		}
		if len(st.sessions) >= MaxSessions {
			return nil, ErrFull
		}
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return nil, err
	}
	s := &Session{ID: hex.EncodeToString(b[:]), Principal: principal, Domain: domainID, expires: now.Add(TTL)}
	st.sessions[s.ID] = s
	return s, nil
}

// Get returns the session only for its owner and only while it is alive; any
// other combination is indistinguishable from "no such session".
func (st *Store) Get(id, principal string) (*Session, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	s, ok := st.sessions[id]
	if !ok || s.Principal != principal {
		return nil, ErrNotFound
	}
	if st.now().After(s.expires) {
		delete(st.sessions, id)
		return nil, ErrNotFound
	}
	return s, nil
}

// Replace overwrites the cart lines (used to roll back a rejected change).
func (s *Session) Replace(lines []domain.CartLine) {
	s.mu.Lock()
	s.lines = append([]domain.CartLine(nil), lines...)
	s.mu.Unlock()
}
