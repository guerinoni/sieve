package sieve_test

import (
	"bufio"
	"os"
	"testing"

	"github.com/guerinoni/sieve"
)

const panicError = "sieve: size must be greater than zero"

func TestPanicWithSizeZero(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			if r != panicError {
				t.Errorf("expected panic message 'sieve: size must be greater than zero', got '%v'", r)
			}
		} else {
			t.Errorf("expected panic but got none")
		}
	}()

	sieve.New[int, string](0)
}

func TestPanicWithSizeLessThanZero(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			if r != panicError {
				t.Errorf("expected panic message 'something went wrong', got '%v'", r)
			}
		} else {
			t.Errorf("expected panic but got none")
		}
	}()

	sieve.New[string, int](-10)
}

const one = "one"

func TestEasy(t *testing.T) { //nolint: cyclop
	s := sieve.New[int, string](2)
	if s.Len() != 0 {
		t.Errorf("expected length 0, got %d", s.Len())
	}

	s.Set(1, one)
	if s.Len() != 1 {
		t.Errorf("expected length 1, got %d", s.Len())
	}

	s.Set(1, one) // duplicate
	if s.Len() != 1 {
		t.Errorf("expected length 1 after duplicate, got %d", s.Len())
	}

	s.Set(2, "two")
	if s.Len() != 2 {
		t.Errorf("expected length 2, got %d", s.Len())
	}

	v, ok := s.Get(3)
	if ok {
		t.Errorf("expected key 3 to not exist, but it does")
	}
	if v != "" {
		t.Errorf("expected value for key 3 to be '', got '%s'", v)
	}

	v, ok = s.Get(1)
	if !ok {
		t.Errorf("expected key 1 to exist, but it does not")
	}
	if v != one {
		t.Errorf("expected value for key 1 to be 'one', got '%s'", v)
	}

	// now we start evicting

	s.Set(3, "three")
	if s.Len() != 2 {
		t.Errorf("expected length 2 after eviction, got %d", s.Len())
	}

	v, ok = s.Get(1)
	if !ok {
		t.Errorf("expected key 1 to exist, but it does not")
	}
	if v != one {
		t.Errorf("expected value for key 1 to be 'one', got '%s'", v)
	}

	v, ok = s.Get(2)
	if ok {
		t.Errorf("expected key 2 to not exist, but it does")
	}
	if v != "" {
		t.Errorf("expected value for key 2 to be '', got '%s'", v)
	}
}

func TestAllAreVisited(t *testing.T) {
	s := sieve.New[int, string](2)

	s.Set(1, one)
	s.Set(2, "two")
	s.Get(2)

	// so now we have all nodes visited

	s.Set(3, "three")
	if s.Len() != 2 {
		t.Errorf("expected length 2 after eviction, got %d", s.Len())
	}

	v, ok := s.Get(3)
	if !ok {
		t.Errorf("expected key 3 to exist, but it does not")
	}
	if v != "three" {
		t.Errorf("expected value for key 3 to be 'three', got '%s'", v)
	}

	v, ok = s.Get(2)
	if !ok {
		t.Errorf("expected key 2 to exist, but it does not")
	}
	if v != "two" {
		t.Errorf("expected value for key 2 to be 'two', got '%s'", v)
	}

	v, ok = s.Get(1)
	if ok {
		t.Errorf("expected key 1 to not exist, but it does")
	}
	if v != "" {
		t.Errorf("expected value for key 1 to be '', got '%s'", v)
	}
}

func TestHandWrapAround(t *testing.T) {
	s := sieve.New[int, string](2)

	s.Set(1, one)
	s.Set(2, "two")
	_, ok := s.Get(1)
	if !ok {
		t.Errorf("expected to find 1")
	}

	s.Set(3, "three")
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

func TestMoreComplex(t *testing.T) { //nolint: dupl
	s := sieve.New[int, struct{}](4)
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
	for i := range 100 {
		go func(i int) {
			s.Set(i, one)
		}(i)

		go func(i int) {
			s.Get(i)
		}(i)
	}
}

func BenchmarkBigInput(b *testing.B) {
	b.ReportAllocs()

	s := sieve.New[string, string](1000)

	file := "./examples/input"
	f, err := os.Open(file)
	if err != nil {
		b.Errorf("could not open file %s: %v", file, err)

		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	for read := scanner.Scan(); read; read = scanner.Scan() {
		d := scanner.Text()
		if _, ok := s.Get(d); !ok {
			s.Set(d, d)
		}
	}
}
