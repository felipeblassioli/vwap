package vwap

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/felipeblassioli/vwap/pkg/ringbuf"
)

// Calculator is a Volume-weighted average (VWAP) price calculator.
//
// The calculation is done within a sliding window of data points.
type Calculator struct {
	mu sync.Mutex

	// windowWidth defines the maximum number of (Price, Quantity) pairs used
	// for the computation of the current VWAP.
	windowWidth int

	// pqs holds all Price x Quantity values used so far for the calculation of
	// the current VWAP.
	pqs *ringbuf.RingBuffer[*big.Float]

	// cumulativeTypicalPrice is the summation of all prices multiplied by the
	// quantity of the traded asset used for the calculation of the current VWAP.
	cumulativeTypicalPrice *big.Float

	// qs holds all quantities used for the calculation of the current VWAP.
	qs *ringbuf.RingBuffer[*big.Float]

	// cumulativeVolume is the summation of all quantities used for the
	// calculation of the current VWAP
	cumulativeVolume *big.Float

	// vwap is the result of the VWAP calculation for `windowWidth`
	// (Price, Quantity) pairs.
	vwap *big.Float
}

// By setting the desired precision to 24 or 53 and using matching rounding
// mode (typically ToNearestEven), Float operations produce the same results
// as the corresponding float32 or float64 IEEE-754 arithmetic for operands
// that correspond to normal (i.e., not denormal) float32 or float64 numbers.
// Exponent underflow and overflow lead to a 0 or an Infinity for different
// values than IEEE-754 because Float exponents have a much larger range.
const (
	prec = uint(53)
	mode = big.ToNearestEven
	// Printing a float64 IEEE-754 type results in 16 precision digits
	numPrecDigits = 16
)

var (
	ErrFloatParse = floatParseError{msg: "failed to parse value into float"}
)

type floatParseError struct {
	msg string
	// value is the string that failed to be parsed into a big.Float
	value string
}

func (e floatParseError) Error() string { return e.msg }

// Value returns the string that failed to be parsed into a big.Float
func (e floatParseError) Value() string { return e.value }

func NewCalculator(windowWidth int) *Calculator {
	return &Calculator{
		windowWidth:            windowWidth,
		pqs:                    ringbuf.NewRingBuffer[*big.Float](windowWidth),
		cumulativeTypicalPrice: new(big.Float).SetPrec(prec).SetMode(mode),
		qs:                     ringbuf.NewRingBuffer[*big.Float](windowWidth),
		cumulativeVolume:       new(big.Float).SetPrec(prec).SetMode(mode),
		vwap:                   new(big.Float).SetPrec(prec).SetMode(mode),
	}
}

// Update receives a pair of (Price, Quantity) and calculates the new VWAP
// value. If the number of pairs used for the calculation so far exceeds the
// windowWidth, it discards the oldest pair from the calculation and substitutes
// it by the received pair.
func (c *Calculator) Update(price, quantity string) (string, error) {
	p, ok := new(big.Float).SetPrec(prec).SetMode(mode).SetString(price)
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrFloatParse, price)
	}

	q, ok := new(big.Float).SetPrec(prec).SetMode(mode).SetString(quantity)
	if !ok {
		return "", fmt.Errorf("%w %s", ErrFloatParse, quantity)
	}
	pq := new(big.Float).SetPrec(prec).SetMode(mode).Mul(p, q)

	c.mu.Lock()
	if c.pqs.Len() == c.windowWidth {
		oldPQ := c.pqs.PopFront()
		c.cumulativeTypicalPrice.Sub(c.cumulativeTypicalPrice, oldPQ)

		oldQ := c.qs.PopFront()
		c.cumulativeVolume = c.cumulativeVolume.Sub(c.cumulativeVolume, oldQ)
	}

	c.cumulativeTypicalPrice.Add(c.cumulativeTypicalPrice, pq)
	c.pqs.PushBack(pq)

	c.cumulativeVolume.Add(c.cumulativeVolume, q)
	c.qs.PushBack(q)

	c.vwap.Quo(c.cumulativeTypicalPrice, c.cumulativeVolume)
	c.mu.Unlock()

	return c.vwap.Text('f', numPrecDigits), nil
}
