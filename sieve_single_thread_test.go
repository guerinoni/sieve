package sieve_test

import (
	"testing"

	"github.com/guerinoni/sieve"
)

func TestPanicWithSizeZeroSingleThread(t *testing.T) {
	checkPanicWithSizeZero(t, newST)
}

func TestPanicWithSizeLessThanZeroSingleThread(t *testing.T) {
	checkPanicWithSizeLessThanZero(t, newSTNeg)
}

func TestEasySingleThread(t *testing.T) {
	runBasicChecks(t, newST)
}

func TestAllAreVisitedSingleThread(t *testing.T) {
	runAllVisitedChecks(t, newST)
}

func TestHandWrapAroundSingleThread(t *testing.T) {
	runHandWrapAroundChecks(t, newST)
}

func TestMoreComplexSingleThread(t *testing.T) {
	runMoreComplexChecks(t, newSTEmpty)
}

func TestDeleteNonExistentKey(t *testing.T) {
	s := sieve.NewSingleThread[int, string](2)
	s.Set(1, one)

	ok := s.Delete(99)
	if ok {
		t.Errorf("expected Delete to return false for non-existent key")
	}

	if s.Len() != 1 {
		t.Errorf("expected length 1, got %d", s.Len())
	}
}

func TestDeleteOnlyElement(t *testing.T) {
	s := sieve.NewSingleThread[int, string](2)
	s.Set(1, one)

	ok := s.Delete(1)
	if !ok {
		t.Errorf("expected Delete to return true")
	}

	if s.Len() != 0 {
		t.Errorf("expected length 0, got %d", s.Len())
	}

	_, ok = s.Get(1)
	if ok {
		t.Errorf("expected key 1 to not exist after delete")
	}
}

func TestDeleteHead(t *testing.T) {
	s := sieve.NewSingleThread[int, string](3)
	s.Set(1, one)
	s.Set(2, two)
	s.Set(3, three)

	// head is 3 (last inserted)
	ok := s.Delete(3)
	if !ok {
		t.Errorf("expected Delete to return true")
	}

	if s.Len() != 2 {
		t.Errorf("expected length 2, got %d", s.Len())
	}

	v, ok := s.Get(1)
	if !ok || v != one {
		t.Errorf("expected key 1 to be 'one', got '%s'", v)
	}

	v, ok = s.Get(2)
	if !ok || v != two {
		t.Errorf("expected key 2 to be 'two', got '%s'", v)
	}
}

func TestDeleteTail(t *testing.T) {
	s := sieve.NewSingleThread[int, string](3)
	s.Set(1, one)
	s.Set(2, two)
	s.Set(3, three)

	// tail is 1 (first inserted)
	ok := s.Delete(1)
	if !ok {
		t.Errorf("expected Delete to return true")
	}

	if s.Len() != 2 {
		t.Errorf("expected length 2, got %d", s.Len())
	}

	v, ok := s.Get(2)
	if !ok || v != two {
		t.Errorf("expected key 2 to be 'two', got '%s'", v)
	}

	v, ok = s.Get(3)
	if !ok || v != three {
		t.Errorf("expected key 3 to be 'three', got '%s'", v)
	}
}

func TestDeleteMiddle(t *testing.T) {
	s := sieve.NewSingleThread[int, string](3)
	s.Set(1, one)
	s.Set(2, two)
	s.Set(3, three)

	ok := s.Delete(2)
	if !ok {
		t.Errorf("expected Delete to return true")
	}

	if s.Len() != 2 {
		t.Errorf("expected length 2, got %d", s.Len())
	}

	v, ok := s.Get(1)
	if !ok || v != one {
		t.Errorf("expected key 1 to be 'one', got '%s'", v)
	}

	v, ok = s.Get(3)
	if !ok || v != three {
		t.Errorf("expected key 3 to be 'three', got '%s'", v)
	}
}

func TestDeleteThenInsert(t *testing.T) {
	s := sieve.NewSingleThread[int, string](2)
	s.Set(1, one)
	s.Set(2, two)

	s.Delete(1)
	s.Set(3, three)

	// should not have evicted since delete freed a slot
	if s.Len() != 2 {
		t.Errorf("expected length 2, got %d", s.Len())
	}

	v, ok := s.Get(2)
	if !ok || v != two {
		t.Errorf("expected key 2 to be 'two', got '%s'", v)
	}

	v, ok = s.Get(3)
	if !ok || v != three {
		t.Errorf("expected key 3 to be 'three', got '%s'", v)
	}
}

func TestFlush(t *testing.T) {
	s := sieve.NewSingleThread[int, string](3)
	s.Set(1, one)
	s.Set(2, two)
	s.Set(3, three)

	s.Flush()

	if s.Len() != 0 {
		t.Errorf("expected length 0 after flush, got %d", s.Len())
	}

	if _, ok := s.Get(1); ok {
		t.Errorf("expected key 1 to not exist after flush")
	}

	// ensure cache is usable after flush
	s.Set(4, "four")

	v, ok := s.Get(4)
	if !ok || v != "four" {
		t.Errorf("expected key 4 to be 'four' after flush and re-insert, got '%s'", v)
	}
}

func BenchmarkSimpleSingleThread(b *testing.B) {
	b.ReportAllocs()

	s := sieve.NewSingleThread[int, string](10)

	for i := range b.N {
		s.Set(i, one)
	}
}

func BenchmarkBigInputSingleThread(b *testing.B) {
	b.ReportAllocs()

	lines := benchInput(b)

	s := sieve.NewSingleThread[string, string](1000)

	b.ResetTimer()

	for i := range b.N {
		d := lines[i%len(lines)]
		if _, ok := s.Get(d); !ok {
			s.Set(d, d)
		}
	}
}