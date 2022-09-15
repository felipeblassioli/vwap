package ringbuf

import (
	"sync"
)

// RingBuffer represents a single instance of the ring buffer
// data structure.
type RingBuffer[T any] struct {
	mu sync.Mutex

	buf    []T
	count  int
	end    int
	start  int
	maxCap int
}

// NewRingBuffer creates a new RingBuffer with a given maximum capacity.
func NewRingBuffer[T any](maxCap int) *RingBuffer[T] {
	return &RingBuffer[T]{
		buf:    make([]T, maxCap),
		end:    0,
		start:  0,
		maxCap: maxCap,
	}
}

// PushBack appends an element to the back of the queue.
// If the ring buffer is empty, the call panics
func (r *RingBuffer[T]) PushBack(item T) {
	r.mu.Lock()
	r.buf[r.end] = item
	r.end++
	r.end %= len(r.buf)
	if r.count < r.maxCap {
		r.count++
	}
	r.mu.Unlock()
}

// PopFront removes and returns the element from the front of the queue.
// If the ring buffer is empty, the call panic
func (r *RingBuffer[T]) PopFront() T {
	if r.count <= 0 {
		panic("ringbuf: PopFront() called in an empty buffer")
	}
	r.mu.Lock()
	item := r.buf[r.start]
	r.start++
	r.start %= len(r.buf)
	r.count--
	r.mu.Unlock()

	return item
}

// Len returns the number of elements currently stored in the queue.
// If r is nil, r.Len() is zero.
func (r *RingBuffer[T]) Len() int {
	if r == nil {
		return 0
	}
	return r.count
}
