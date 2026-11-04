// Package fibonacci implements the Fibonacci sequence domain logic.
//
// It has no knowledge of HTTP or any transport and can be reused from
// any entry point.
package fibonacci

import (
	"errors"
	"fmt"
)

// MaxN is the largest supported sequence length. The last element of a
// sequence of length MaxN is F(92) = 7540113804746346429, which is the
// largest Fibonacci number that fits in an int64.
const MaxN = 93

var (
	// ErrNegative is returned when the requested length is negative.
	ErrNegative = errors.New("n must not be negative")
	// ErrTooLarge is returned when the requested length exceeds MaxN.
	ErrTooLarge = fmt.Errorf("n must not be greater than %d", MaxN)
)

// Sequence returns the first n Fibonacci numbers, starting from 0.
// For n == 0 it returns an empty, non-nil slice.
func Sequence(n int) ([]int64, error) {
	switch {
	case n < 0:
		return nil, ErrNegative
	case n > MaxN:
		return nil, ErrTooLarge
	}

	seq := make([]int64, n)
	for i := range seq {
		if i < 2 {
			seq[i] = int64(i)
			continue
		}
		seq[i] = seq[i-1] + seq[i-2]
	}

	return seq, nil
}
