package state

import (
	"encoding/json"
	"log/slog"
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
	oldQty := s.snapshot.Position.Qty
	s.snapshot.Position = position
	if oldQty != position.Qty {
		slog.Info("position updated", "old_qty", oldQty, "new_qty", position.Qty, "avg_entry", position.AvgEntry)
	}
}

func (s *Store) SetOpenOrders(orders map[string]OpenOrder) {
	s.mu.Lock()
	defer s.mu.Unlock()
	oldCount := len(s.snapshot.OpenOrders)
	s.snapshot.OpenOrders = orders
	if oldCount != len(orders) {
		slog.Info("open orders updated", "old_count", oldCount, "new_count", len(orders))
	}
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
		slog.Error("state save failed", "path", path, "error", err)
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		slog.Error("state save failed", "path", path, "error", err)
		return err
	}
	slog.Info("state saved", "path", path, "position_qty", s.snapshot.Position.Qty, "open_orders", len(s.snapshot.OpenOrders))
	return nil
}

func (s *Store) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Error("state load failed", "path", path, "error", err)
		return err
	}
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		slog.Error("state load failed", "path", path, "error", err)
		return err
	}
	if snapshot.OpenOrders == nil {
		snapshot.OpenOrders = map[string]OpenOrder{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshot = snapshot
	slog.Info("state loaded", "path", path, "position_qty", snapshot.Position.Qty, "open_orders", len(snapshot.OpenOrders), "last_trade", snapshot.LastTradeTime.Format(time.RFC3339))
	return nil
}
