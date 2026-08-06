package sieve_test

import (
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"github.com/guerinoni/sieve"
)

// nodesInList counts the nodes reachable from the head by walking the `next`
// links, which is what String renders.
func nodesInList(dump string) int {
	dump = strings.TrimSuffix(strings.TrimPrefix(dump, "["), "]")
	if dump == "" {
		return 0
	}

	return strings.Count(dump, " -> ") + 1
}

// TestListMatchesLen drives random operation sequences and checks that the
// doubly linked list stays consistent with the map: an unlinked node must not
// stay reachable from the head and the length must never exceed the capacity.
func TestListMatchesLen(t *testing.T) {
	capacities := []int32{1, 2, 3, 5, 8}
	ttls := []time.Duration{0, time.Microsecond, time.Hour}

	for _, capacity := range capacities {
		for _, ttl := range ttls {
			for seed := range uint64(300) {
				//nolint:gosec // a deterministic seed is what makes the failures reproducible
				rnd := rand.New(rand.NewPCG(seed, 0))

				cache := sieve.New[int, int](capacity)
				if ttl > 0 {
					cache = cache.WithTTL(ttl)
				}

				const opsPerRun = 60

				for range opsPerRun {
					key := rnd.IntN(int(capacity) + 3)

					switch rnd.IntN(3) {
					case 0:
						cache.Set(key, key)
					case 1:
						cache.Get(key)
					case 2:
						cache.Delete(key)
					}

					if got, want := nodesInList(cache.String()), int(cache.Len()); got != want {
						t.Fatalf("capacity=%d ttl=%v seed=%d: list holds %d nodes but Len is %d, state %s",
							capacity, ttl, seed, got, want, cache.String())
					}

					if cache.Len() > capacity {
						t.Fatalf("capacity=%d ttl=%v seed=%d: Len is %d, state %s",
							capacity, ttl, seed, cache.Len(), cache.String())
					}
				}
			}
		}
	}
}

// TestDeleteTailOfPair is the minimal case of the bug the fuzz above found:
// removing the tail of a two element cache used to leave the head pointing at
// the removed node.
func TestDeleteTailOfPair(t *testing.T) {
	cache := sieve.New[int, int](2)
	cache.Set(1, 1)
	cache.Set(2, 2)

	cache.Delete(1)

	if got, want := cache.String(), "[2: 2]"; got != want {
		t.Fatalf("String is %q, want %q", got, want)
	}
}
