package md

import "testing"

func TestRingBufferSMA(t *testing.T) {
	buffer := NewRingBuffer(5)
	values := []float64{1, 2, 3, 4, 5}
	for _, v := range values {
		buffer.Add(v)
	}

	sma, err := buffer.SMA(3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := (3.0 + 4.0 + 5.0) / 3.0
	if sma != expected {
		t.Fatalf("expected SMA %.2f, got %.2f", expected, sma)
	}
}

func TestRingBufferSMAInsufficientData(t *testing.T) {
	buffer := NewRingBuffer(5)
	buffer.Add(1)

	if _, err := buffer.SMA(3); err == nil {
		t.Fatalf("expected error for insufficient data")
	}
}
