package engine

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"ats/internal/strategy"
)

type Decision struct {
	RunID          string          `json:"run_id"`
	Timestamp      time.Time       `json:"timestamp"`
	BarTime        time.Time       `json:"bar_time"`
	Symbol         string          `json:"symbol"`
	Close          float64         `json:"close"`
	SMA            float64         `json:"sma"`
	Intent         strategy.Action `json:"intent"`
	IntentQty      int             `json:"intent_qty"`
	Reason         string          `json:"reason"`
	Result         string          `json:"result"`
	ApprovalReason string          `json:"approval_reason,omitempty"`
	RejectReason   string          `json:"reject_reason,omitempty"`
	OrderID        string          `json:"order_id,omitempty"`
	ClientOrderID  string          `json:"client_order_id,omitempty"`
}

type DecisionLogger struct {
	runID  string
	file   *os.File
	writer *bufio.Writer
	mu     sync.Mutex
}

func NewDecisionLogger(path string, runID string) (*DecisionLogger, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &DecisionLogger{
		runID:  runID,
		file:   file,
		writer: bufio.NewWriter(file),
	}, nil
}

func (d *DecisionLogger) RunID() string {
	return d.runID
}

func (d *DecisionLogger) Append(decision Decision) {
	d.mu.Lock()
	defer d.mu.Unlock()
	payload, err := json.Marshal(decision)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal decision: %v\n", err)
		return
	}
	if _, err := d.writer.Write(append(payload, '\n')); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write decision: %v\n", err)
		return
	}
	if err := d.writer.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to flush decision log: %v\n", err)
	}
}

func (d *DecisionLogger) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.writer.Flush(); err != nil {
		_ = d.file.Close()
		return err
	}
	return d.file.Close()
}
