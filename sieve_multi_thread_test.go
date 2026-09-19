package sieve_test

import (
	"bufio"
	"os"
	"testing"
	"time"

	"github.com/guerinoni/sieve"
)

const panicError = "sieve: size must be greater than zero"
const testInputFile = "./examples/input"

const one = "one"
const two = "two"
const three = "three"
const uno = "uno"

// Factories let the common checks run against both cache constructors so the
// single- and multi-threaded variants stay behaviorally identical.
func newMT(size int32) *sieve.Cache[int, string]     { return sieve.New[int, string](size) }
func newST(size int32) *sieve.Cache[int, string]     { return sieve.NewSingleThread[int, string](size) }
func newMTNeg(size int32) *sieve.Cache[string, int]  { return sieve.New[string, int](size) }
func newSTNeg(size int32) *sieve.Cache[string, int]  { return sieve.NewSingleThread[string, int](size) }
func newMTEmpty(size int32) *sieve.Cache[int, struct{}] { return sieve.New[int, struct{}](size) }
func newSTEmpty(size int32) *sieve.Cache[int, struct{}] { return sieve.NewSingleThread[int, struct{}](size) }

// benchInput loads the whole sample input once so that the benchmarks measure
// the cache and not the file reading.
func benchInput(b *testing.B) []string {
	b.Helper()

	f, err := os.Open(testInputFile)
	if err != nil {
		b.Fatalf("could not open file %s: %v", testInputFile, err)
	}

	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	var lines []string

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	err = scanner.Err()
	if err != nil {
		b.Fatalf("could not read file %s: %v", testInputFile, err)
	}

	return lines
}

func checkPanicWithSizeZero(t *testing.T, newCache func(int32) *sieve.Cache[int, string]) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			if r != panicError {
				t.Errorf("expected panic message 'sieve: size must be greater than zero', got '%v'", r)
			}
		} else {
			t.Errorf("expected panic but got none")
		}
	}()

	_ = newCache(0)
}

func checkPanicWithSizeLessThanZero(t *testing.T, newCache func(int32) *sieve.Cache[string, int]) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			if r != panicError {
				t.Errorf("expected panic message 'sieve: size must be greater than zero', got '%v'", r)
			}
		} else {
			t.Errorf("expected panic but got none")
		}
	}()

	_ = newCache(-10)
}

// runBasicChecks covers insert, duplicate Set (which refreshes the value),
// get, and eviction on a size-two cache.
func runBasicChecks(t *testing.T, newCache func(int32) *sieve.Cache[int, string]) {
	t.Helper()

	s := newCache(2)
	if s.Len() != 0 {
		t.Errorf("expected length 0, got %d", s.Len())
	}

	s.Set(1, one)

	if s.Len() != 1 {
		t.Errorf("expected length 1, got %d", s.Len())
	}

	s.Set(1, uno) // duplicate: refreshes the value

	if s.Len() != 1 {
		t.Errorf("expected length 1 after duplicate, got %d", s.Len())
	}

	v, ok := s.Get(1)
	if !ok || v != uno {
		t.Errorf("expected duplicate Set to update the value to 'uno', got '%s'", v)
	}

	s.Set(2, two)

	if s.Len() != 2 {
		t.Errorf("expected length 2, got %d", s.Len())
	}

	v, ok = s.Get(3)
	if ok {
		t.Errorf("expected key 3 to not exist, but it does")
	}

	if v != "" {
		t.Errorf("expected value for key 3 to be '', got '%s'", v)
	}

	v, ok = s.Get(1)
	if !ok || v != uno {
		t.Errorf("expected key 1 to hold 'uno', got '%s'", v)
	}

	// now we start evicting

	s.Set(3, three)

	if s.Len() != 2 {
		t.Errorf("expected length 2 after eviction, got %d", s.Len())
	}

	v, ok = s.Get(1)
	if !ok || v != uno {
		t.Errorf("expected key 1 to hold 'uno', got '%s'", v)
	}

	v, ok = s.Get(2)
	if ok {
		t.Errorf("expected key 2 to not exist, but it does")
	}

	if v != "" {
		t.Errorf("expected value for key 2 to be '', got '%s'", v)
	}
}

// runAllVisitedChecks exercises the eviction scan when every node is visited.
func runAllVisitedChecks(t *testing.T, newCache func(int32) *sieve.Cache[int, string]) {
	t.Helper()

	s := newCache(2)

	s.Set(1, one)
	s.Set(2, two)
	s.Get(2)

	// so now we have all nodes visited

	s.Set(3, three)

	if s.Len() != 2 {
		t.Errorf("expected length 2 after eviction, got %d", s.Len())
	}

	v, ok := s.Get(3)
	if !ok || v != three {
		t.Errorf("expected key 3 to be 'three', got '%s'", v)
	}

	v, ok = s.Get(2)
	if !ok || v != two {
		t.Errorf("expected key 2 to be 'two', got '%s'", v)
	}

	v, ok = s.Get(1)
	if ok {
		t.Errorf("expected key 1 to not exist, but it does")
	}

	if v != "" {
		t.Errorf("expected value for key 1 to be '', got '%s'", v)
	}
}

// runHandWrapAroundChecks exercises the hand wrapping past the head.
func runHandWrapAroundChecks(t *testing.T, newCache func(int32) *sieve.Cache[int, string]) {
	t.Helper()

	s := newCache(2)

	s.Set(1, one)
	s.Set(2, two)

	_, ok := s.Get(1)
	if !ok {
		t.Errorf("expected to find 1")
	}

	s.Set(3, three)

	_, ok = s.Get(3)
	if !ok {
		t.Errorf("expected to find 3")
	}

	s.Set(4, "four")

	_, ok = s.Get(3)
	if !ok {
		t.Errorf("expected to find 3")
	}

	_, ok = s.Get(4)
	if !ok {
		t.Errorf("expected to find 4")
	}

	s.Set(5, "five")
}

// runMoreComplexChecks drives a longer mixed insert sequence.
func runMoreComplexChecks(t *testing.T, newCache func(int32) *sieve.Cache[int, struct{}]) {
	t.Helper()

	s := newCache(4)
	s.Set(7, struct{}{})
	s.Set(7, struct{}{})
	s.Set(5, struct{}{})
	s.Set(5, struct{}{})
	s.Set(6, struct{}{})
	s.Set(10, struct{}{})
	s.Set(9, struct{}{})
	s.Set(1, struct{}{})
	s.Set(5, struct{}{})
	s.Set(7, struct{}{})

	if s.Len() != 4 {
		t.Errorf("expected 4, got %d", s.Len())
	}

	_, ok := s.Get(7)
	if !ok {
		t.Errorf("expected to find 7")
	}

	_, ok = s.Get(5)
	if !ok {
		t.Errorf("expected to find 5")
	}

	_, ok = s.Get(9)
	if !ok {
		t.Errorf("expected to find 9")
	}

	_, ok = s.Get(1)
	if !ok {
		t.Errorf("expected to find 1")
	}
}

func TestPanicWithSizeZero(t *testing.T) {
	checkPanicWithSizeZero(t, newMT)
}

func TestPanicWithSizeLessThanZero(t *testing.T) {
	checkPanicWithSizeLessThanZero(t, newMTNeg)
}

func TestEasy(t *testing.T) {
	runBasicChecks(t, newMT)
}

func TestAllAreVisited(t *testing.T) {
	runAllVisitedChecks(t, newMT)
}

func TestHandWrapAround(t *testing.T) {
	runHandWrapAroundChecks(t, newMT)
}

func TestMoreComplex(t *testing.T) {
	runMoreComplexChecks(t, newMTEmpty)
}

func TestDeleteConcurrent(t *testing.T) {
	s := sieve.New[int, string](100)

	for i := range 100 {
		s.Set(i, "v")
	}

	done := make(chan struct{})

	go func() {
		for i := range 50 {
			s.Delete(i)
		}

		done <- struct{}{}
	}()

	go func() {
		for i := 50; i < 100; i++ {
			s.Delete(i)
		}

		done <- struct{}{}
	}()

	<-done
	<-done

	if s.Len() != 0 {
		t.Errorf("expected length 0, got %d", s.Len())
	}
}

func TestDeleteConcurrentWithSetAndGet(t *testing.T) {
	s := sieve.New[int, string](50)

	for i := range 50 {
		s.Set(i, "v")
	}

	done := make(chan struct{})

	go func() {
		for i := range 50 {
			s.Delete(i)
		}

		done <- struct{}{}
	}()

	go func() {
		for i := 50; i < 100; i++ {
			s.Set(i, "v")
		}

		done <- struct{}{}
	}()

	go func() {
		for i := range 100 {
			s.Get(i)
		}

		done <- struct{}{}
	}()

	<-done
	<-done
	<-done

	if s.Len() > 50 {
		t.Errorf("expected length at most 50, got %d", s.Len())
	}
}

// TestTTLConcurrentChurn drives Set, Get and Delete from several goroutines
// while entries expire underneath them. It is a race-detector exercise: the
// suite runs with -race, so a data race in the TTL path fails the test.
func TestTTLConcurrentChurn(t *testing.T) {
	s := sieve.New[int, int](32).WithTTL(50 * time.Millisecond)

	done := make(chan struct{})

	for range 4 {
		go func() {
			for i := range 20000 {
				s.Set(i % 100, i)
				s.Get(i % 100)
				s.Delete(i % 100)
			}

			done <- struct{}{}
		}()
	}

	for range 4 {
		<-done
	}

	if s.Len() > 32 {
		t.Errorf("expected length at most 32, got %d", s.Len())
	}
}

func BenchmarkSimple(b *testing.B) {
	b.ReportAllocs()

	s := sieve.New[int, string](10)

	for i := range b.N {
		s.Set(i, one)
	}
}

func BenchmarkSimpleConcurrent(b *testing.B) {
	b.ReportAllocs()

	s := sieve.New[int, string](10)

	b.RunParallel(func(pb *testing.PB) {
		for i := 0; pb.Next(); i++ {
			s.Set(i, one)
			s.Get(i)
		}
	})
}

func BenchmarkBigInput(b *testing.B) {
	b.ReportAllocs()

	lines := benchInput(b)

	s := sieve.New[string, string](1000)

	b.ResetTimer()

	for i := range b.N {
		d := lines[i%len(lines)]
		if _, ok := s.Get(d); !ok {
			s.Set(d, d)
		}
	}
}