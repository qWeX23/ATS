package md

import "errors"

type RingBuffer struct {
	values []float64
	size   int
	index  int
	filled bool
}

func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		values: make([]float64, size),
		size:   size,
	}
}

func (r *RingBuffer) Add(value float64) {
	r.values[r.index] = value
	r.index = (r.index + 1) % r.size
	if r.index == 0 {
		r.filled = true
	}
}

func (r *RingBuffer) Len() int {
	if r.filled {
		return r.size
	}
	return r.index
}

func (r *RingBuffer) Values() []float64 {
	length := r.Len()
	result := make([]float64, 0, length)
	if length == 0 {
		return result
	}
	if r.filled {
		result = append(result, r.values[r.index:]...)
	}
	result = append(result, r.values[:r.index]...)
	return result
}

func (r *RingBuffer) SMA(window int) (float64, error) {
	if window <= 0 {
		return 0, errors.New("window must be positive")
	}
	values := r.Values()
	if len(values) < window {
		return 0, errors.New("not enough data for SMA")
	}
	start := len(values) - window
	sum := 0.0
	for _, v := range values[start:] {
		sum += v
	}
	return sum / float64(window), nil
}
