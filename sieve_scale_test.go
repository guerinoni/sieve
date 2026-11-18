package sieve_test

import (
	"testing"

	"github.com/guerinoni/sieve"
)

// TestMissCountMatchesReference replays the one million requests of the sample
// trace and pins the miss counts. The figures are the ones produced by
// github.com/scalalang2/golang-fifo/sieve, an independent implementation of the
// paper, so any drift in the eviction order shows up here even when the small
// hand-written cases still pass.
func TestMissCountMatchesReference(t *testing.T) {
	lines := sampleInput(t)

	for _, tc := range []struct {
		capacity int32
		misses   int
	}{
		{1, 970518},
		{10, 656714},
		{100, 328766},
		{500, 105255},
		{999, 1163},
	} {
		for name, cache := range map[string]*sieve.Cache[string, struct{}]{
			"New":             sieve.New[string, struct{}](tc.capacity),
			"NewSingleThread": sieve.NewSingleThread[string, struct{}](tc.capacity),
		} {
			misses := 0

			for _, l := range lines {
				if _, ok := cache.Get(l); !ok {
					misses++

					cache.Set(l, struct{}{})
				}
			}

			if misses != tc.misses {
				t.Errorf("%s(%d): %d misses, want %d", name, tc.capacity, misses, tc.misses)
			}
		}
	}
}
