package ringbuf

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRingBuffer_PushBack(t *testing.T) {
	tests := []struct {
		name   string
		maxCap int
		data   []int
		// expected intermediary states
		states [][]int
	}{
		{
			"Maximum capacity 1",
			1,
			[]int{1, 2, 3, 4, 5, 6},
			[][]int{
				{1}, {2}, {3}, {4}, {5}, {6},
			},
		},
		{
			"Maximum capacity 2",
			2,
			[]int{1, 2, 3, 4, 5, 6},
			[][]int{
				{1, 0}, {1, 2}, {3, 2}, {3, 4}, {5, 4}, {5, 6},
			},
		},
		{
			"Maximum capacity 3",
			3,
			[]int{1, 2, 3, 4, 5, 6},
			[][]int{
				{1, 0, 0}, {1, 2, 0}, {1, 2, 3}, {4, 2, 3}, {4, 5, 3}, {4, 5, 6},
			},
		},
		{
			"Maximum capacity equals len(buf)",
			6,
			[]int{1, 2, 3, 4, 5, 6},
			[][]int{
				{1, 0, 0, 0, 0, 0},
				{1, 2, 0, 0, 0, 0},
				{1, 2, 3, 0, 0, 0},
				{1, 2, 3, 4, 0, 0},
				{1, 2, 3, 4, 5, 0},
				{1, 2, 3, 4, 5, 6},
			},
		},
		{
			"Maximum capacity bigger than len(buf)",
			8,
			[]int{1, 2, 3, 4, 5, 6},
			[][]int{
				{1, 0, 0, 0, 0, 0, 0, 0},
				{1, 2, 0, 0, 0, 0, 0, 0},
				{1, 2, 3, 0, 0, 0, 0, 0},
				{1, 2, 3, 4, 0, 0, 0, 0},
				{1, 2, 3, 4, 5, 0, 0, 0},
				{1, 2, 3, 4, 5, 6, 0, 0},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rb := NewRingBuffer[int](tc.maxCap)
			for i := 0; i < len(tc.data); i++ {
				rb.PushBack(tc.data[i])

				assert.Equal(t, tc.states[i], rb.buf)
			}
		})
	}
}

func TestRingBuffer_PopFront(t *testing.T) {
	t.Run("All elements taken FIFO", func(t *testing.T) {
		var (
			// setup internal state directly to avoid calling PushBack()
			// and tainting the test coverage
			rb = &RingBuffer[int]{
				buf:   []int{1, 2, 3, 4, 5, 6},
				count: 6,
			}

			exp = []int{1, 2, 3, 4, 5, 6}
		)

		for i := 0; i < len(exp); i++ {
			assert.Equal(t, exp[i], rb.PopFront())
		}
	})

	t.Run("Panics when buffer empty", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("PropFront() did not panic")
			}
		}()
		// maximum capacity doesn't matter for this test
		rb := NewRingBuffer[int](rand.Intn(32)) //nolint:gosec
		rb.PopFront()
	})
}

func TestRingBuffer_Len(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		// maximum capacity doesn't matter for this test
		var rb *RingBuffer[int]
		assert.Equal(t, 0, rb.Len())
	})

	t.Run("Empty", func(t *testing.T) {
		// maximum capacity doesn't matter for this test
		rb := NewRingBuffer[int](rand.Intn(32)) //nolint:gosec
		assert.Equal(t, 0, rb.Len())
	})
}

func TestRingBuffer_PushPopLen(t *testing.T) {
	t.Run("PushPop within bounds", func(t *testing.T) {
		var (
			sz = rand.Intn(32) //nolint:gosec
			rb = NewRingBuffer[int](sz)
		)

		for i := 1; i <= sz; i++ {
			rb.PushBack(i)
			assert.Equal(t, i, rb.Len())
		}
		for i := 1; i <= sz; i++ {
			assert.Equal(t, i, rb.PopFront())
			assert.Equal(t, sz-i, rb.Len())
		}
	})

	t.Run("Len does not exceed maximum capacity", func(t *testing.T) {
		var (
			sz = rand.Intn(32) //nolint:gosec
			rb = NewRingBuffer[int](sz)
		)

		// rb.Len() cannot exceed the maximumCapacity
		for i := 1; i <= sz*3; i++ {
			rb.PushBack(i)
		}
	})
}

func Benchmark_PushBack(b *testing.B) {
	rb := NewRingBuffer[int](b.N)

	for i := 0; i < b.N; i++ {
		rb.PushBack(i)
	}
}

func Benchmark_FIFO(b *testing.B) {
	rb := NewRingBuffer[int](b.N)

	for i := 0; i < b.N; i++ {
		rb.PushBack(i)
	}
	for i := 0; i < b.N; i++ {
		rb.PopFront()
	}
}
