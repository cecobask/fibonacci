package fibonacci_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/cecobask/fibonacci/internal/fibonacci"
)

func TestSequence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		n       int
		want    []int64
		wantErr error
	}{
		{name: "zero", n: 0, want: []int64{}},
		{name: "one", n: 1, want: []int64{0}},
		{name: "two", n: 2, want: []int64{0, 1}},
		{name: "five", n: 5, want: []int64{0, 1, 1, 2, 3}},
		{name: "ten", n: 10, want: []int64{0, 1, 1, 2, 3, 5, 8, 13, 21, 34}},
		{name: "negative", n: -1, wantErr: fibonacci.ErrNegative},
		{name: "too large", n: fibonacci.MaxN + 1, wantErr: fibonacci.ErrTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := fibonacci.Sequence(tt.n)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Sequence(%d) error = %v, want %v", tt.n, err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got == nil {
				t.Fatalf("Sequence(%d) returned nil slice, want non-nil", tt.n)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("Sequence(%d) = %v, want %v", tt.n, got, tt.want)
			}
		})
	}
}

func TestSequenceMaxN(t *testing.T) {
	t.Parallel()

	got, err := fibonacci.Sequence(fibonacci.MaxN)
	if err != nil {
		t.Fatalf("Sequence(MaxN) unexpected error: %v", err)
	}
	if len(got) != fibonacci.MaxN {
		t.Fatalf("len = %d, want %d", len(got), fibonacci.MaxN)
	}

	const f92 int64 = 7540113804746346429
	if last := got[len(got)-1]; last != f92 {
		t.Errorf("last element = %d, want %d", last, f92)
	}
}

func FuzzSequence(f *testing.F) {
	for _, n := range []int{-1, 0, 1, 2, 5, fibonacci.MaxN, fibonacci.MaxN + 1} {
		f.Add(n)
	}

	f.Fuzz(func(t *testing.T, n int) {
		seq, err := fibonacci.Sequence(n)
		if n < 0 || n > fibonacci.MaxN {
			if err == nil {
				t.Fatalf("Sequence(%d) expected error", n)
			}
			return
		}
		if err != nil {
			t.Fatalf("Sequence(%d) unexpected error: %v", n, err)
		}
		if len(seq) != n {
			t.Fatalf("len = %d, want %d", len(seq), n)
		}
		for i := 2; i < len(seq); i++ {
			if seq[i] != seq[i-1]+seq[i-2] {
				t.Fatalf("seq[%d] = %d, want %d", i, seq[i], seq[i-1]+seq[i-2])
			}
			if seq[i] < seq[i-1] {
				t.Fatalf("overflow at index %d", i)
			}
		}
	})
}

func BenchmarkSequence(b *testing.B) {
	for b.Loop() {
		_, _ = fibonacci.Sequence(fibonacci.MaxN)
	}
}
