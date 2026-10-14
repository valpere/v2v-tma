package session

import (
	"errors"
	"testing"
	"time"
)

func TestSessionIsBoundToItsOwner(t *testing.T) {
	st := NewStore()
	s, err := st.Create("alice", "pizzeria")
	if err != nil {
		t.Fatal(err)
	}
	if len(s.ID) != 32 {
		t.Fatalf("id %q is not 128-bit hex", s.ID)
	}
	if _, err := st.Get(s.ID, "alice"); err != nil {
		t.Fatalf("owner: %v", err)
	}
	if _, err := st.Get(s.ID, "mallory"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other principal: err = %v, want ErrNotFound", err)
	}
	if _, err := st.Get("deadbeef", "alice"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown id: err = %v", err)
	}
}

func TestSessionExpires(t *testing.T) {
	st := NewStore()
	clock := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	st.now = func() time.Time { return clock }
	s, _ := st.Create("alice", "pizzeria")
	clock = clock.Add(TTL + time.Second)
	if _, err := st.Get(s.ID, "alice"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired session: err = %v", err)
	}
}

func TestStoreCapacity(t *testing.T) {
	st := NewStore()
	for i := 0; i < MaxSessions; i++ {
		if _, err := st.Create("u", "pizzeria"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := st.Create("u", "pizzeria"); !errors.Is(err, ErrFull) {
		t.Fatalf("err = %v, want ErrFull", err)
	}
}

func TestCartLines(t *testing.T) {
	s, _ := NewStore().Create("u", "pizzeria")
	s.Add("a", 1)
	s.Add("a", 2)
	s.Add("b", 1)
	s.Set("b", 5)
	if got := s.Lines(); len(got) != 2 || got[0].Quantity != 3 || got[1].Quantity != 5 {
		t.Fatalf("got %+v", got)
	}
	s.Set("a", 0)
	if got := s.Lines(); len(got) != 1 || got[0].ItemID != "b" {
		t.Fatalf("got %+v", got)
	}
	s.Clear()
	if len(s.Lines()) != 0 {
		t.Fatal("cart not cleared")
	}
}
