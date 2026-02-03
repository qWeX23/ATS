package state

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

type Position struct {
	Qty      int
	AvgEntry float64
}

type OpenOrder struct {
	ClientOrderID string
	OrderID       string
	Status        string
}

type Snapshot struct {
	Position      Position
	OpenOrders    map[string]OpenOrder
	LastTradeTime time.Time
	LastBarTime   time.Time
}

type Store struct {
	mu       sync.RWMutex
	snapshot Snapshot
}

func NewStore() *Store {
	return &Store{
		snapshot: Snapshot{
			OpenOrders: map[string]OpenOrder{},
		},
	}
}

func (s *Store) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	copy := s.snapshot
	copy.OpenOrders = make(map[string]OpenOrder, len(s.snapshot.OpenOrders))
	for k, v := range s.snapshot.OpenOrders {
		copy.OpenOrders[k] = v
	}
	return copy
}

func (s *Store) UpdatePosition(position Position) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshot.Position = position
}

func (s *Store) SetOpenOrders(orders map[string]OpenOrder) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshot.OpenOrders = orders
}

func (s *Store) SetLastTradeTime(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshot.LastTradeTime = t
}

func (s *Store) SetLastBarTime(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshot.LastBarTime = t
}

func (s *Store) Save(path string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := json.MarshalIndent(s.snapshot, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (s *Store) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return err
	}
	if snapshot.OpenOrders == nil {
		snapshot.OpenOrders = map[string]OpenOrder{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshot = snapshot
	return nil
}
