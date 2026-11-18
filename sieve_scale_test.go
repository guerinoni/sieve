package sieve_test

import (
	"math/rand/v2"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

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

// The Zipf benchmarks model a large web cache with the power-law popularity of
// the paper: a million entries in front of 16M requests over about 1.5M distinct
// keys. A warm cache misses on about 6% of them, in the 2-7% range of the
// paper's throughput runs.
const (
	zipfSkew     = 1.1
	zipfKeys     = 1 << 23
	zipfCapacity = 1 << 20
	zipfTraceLen = 1 << 24
)

func zipfTrace() []uint64 {
	//nolint:gosec // a fixed seed keeps the trace identical across runs
	zipf := rand.NewZipf(rand.New(rand.NewPCG(1, 2)), zipfSkew, 1, zipfKeys-1)

	trace := make([]uint64, zipfTraceLen)
	for i := range trace {
		trace[i] = zipf.Uint64()
	}

	return trace
}

type zipfCache struct {
	name       string
	new        func() *sieve.Cache[uint64, uint64]
	concurrent bool
}

func zipfCaches() []zipfCache {
	return []zipfCache{
		{"New", func() *sieve.Cache[uint64, uint64] { return sieve.New[uint64, uint64](zipfCapacity) }, true},
		{"NewSingleThread", func() *sieve.Cache[uint64, uint64] {
			return sieve.NewSingleThread[uint64, uint64](zipfCapacity)
		}, false},
		{"WithTTL", func() *sieve.Cache[uint64, uint64] {
			return sieve.New[uint64, uint64](zipfCapacity).WithTTL(time.Hour)
		}, true},
	}
}

// getOrSet is the cache-aside pattern every benchmark here replays.
func getOrSet(s *sieve.Cache[uint64, uint64], k uint64) bool {
	if _, ok := s.Get(k); ok {
		return true
	}

	s.Set(k, k)

	return false
}

// warm replays the whole trace once so the timed loop starts on a full cache.
func warm(s *sieve.Cache[uint64, uint64], trace []uint64) {
	for _, k := range trace {
		getOrSet(s, k)
	}
}

func BenchmarkZipf(b *testing.B) {
	trace := zipfTrace()

	for _, bc := range zipfCaches() {
		b.Run(bc.name, func(b *testing.B) {
			b.ReportAllocs()

			s := bc.new()
			warm(s, trace)

			misses, i := 0, 0

			for ; b.Loop(); i++ {
				if !getOrSet(s, trace[i%len(trace)]) {
					misses++
				}
			}

			b.ReportMetric(float64(misses)/float64(i), "misses/op")
		})
	}
}

func BenchmarkZipfParallel(b *testing.B) {
	trace := zipfTrace()

	for _, bc := range zipfCaches() {
		if !bc.concurrent {
			continue
		}

		b.Run(bc.name, func(b *testing.B) {
			b.ReportAllocs()

			s := bc.new()
			warm(s, trace)

			var workers, misses atomic.Int64

			// RunParallel, unlike b.Loop, does not reset the timer after the warm up
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				// spread the goroutines over the trace so they do not replay it in lockstep
				i := int(workers.Add(1)) * len(trace) / runtime.GOMAXPROCS(0)

				var local int64

				for ; pb.Next(); i++ {
					if !getOrSet(s, trace[i%len(trace)]) {
						local++
					}
				}

				misses.Add(local)
			})

			b.ReportMetric(float64(misses.Load())/float64(b.N), "misses/op")
		})
	}
}
