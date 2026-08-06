package sieve_test

import (
	"bufio"
	"os"
	"testing"

	"github.com/guerinoni/sieve"
	lru "github.com/hashicorp/golang-lru/v2"
	s3fifo "github.com/scalalang2/golang-fifo/s3fifo"
	golangsieve "github.com/scalalang2/golang-fifo/sieve"
)

const panicError = "sieve: size must be greater than zero"
const testInputFile = "./examples/input"

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
const two = "two"
const three = "three"

func TestEasy(t *testing.T) {
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

	s.Set(2, two)

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

	s.Set(3, three)

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
	s.Set(2, two)
	s.Get(2)

	// so now we have all nodes visited

	s.Set(3, three)

	if s.Len() != 2 {
		t.Errorf("expected length 2 after eviction, got %d", s.Len())
	}

	v, ok := s.Get(3)
	if !ok {
		t.Errorf("expected key 3 to exist, but it does not")
	}

	if v != three {
		t.Errorf("expected value for key 3 to be 'three', got '%s'", v)
	}

	v, ok = s.Get(2)
	if !ok {
		t.Errorf("expected key 2 to exist, but it does not")
	}

	if v != two {
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

func TestDeleteConcurrentWithSetAndGet(_ *testing.T) {
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

func BenchmarkSimpleLRU(b *testing.B) {
	b.ReportAllocs()

	s, _ := lru.New[int, string](10)

	for i := range b.N {
		s.Add(i, one)
	}
}

func BenchmarkBigInputLRU(b *testing.B) {
	b.ReportAllocs()

	lines := benchInput(b)

	s, err := lru.New[string, string](1000)
	if err != nil {
		b.Fatalf("could not create lru: %v", err)
	}

	b.ResetTimer()

	for i := range b.N {
		d := lines[i%len(lines)]
		if _, ok := s.Get(d); !ok {
			s.Add(d, d)
		}
	}
}

func BenchmarkSimpleS3FIFO(b *testing.B) {
	b.ReportAllocs()

	s := s3fifo.New[int, string](10, 0)

	for i := range b.N {
		s.Set(i, one)
	}
}

func BenchmarkBigInputS3FIFO(b *testing.B) {
	b.ReportAllocs()

	lines := benchInput(b)

	s := s3fifo.New[string, string](1000, 0)

	b.ResetTimer()

	for i := range b.N {
		d := lines[i%len(lines)]
		if _, ok := s.Get(d); !ok {
			s.Set(d, d)
		}
	}
}

func BenchmarkSimpleGolangSieve(b *testing.B) {
	b.ReportAllocs()

	s := golangsieve.New[int, string](10, 0)

	for i := range b.N {
		s.Set(i, one)
	}
}

func BenchmarkBigInputGolangSieve(b *testing.B) {
	b.ReportAllocs()

	lines := benchInput(b)

	s := golangsieve.New[string, string](1000, 0)

	b.ResetTimer()

	for i := range b.N {
		d := lines[i%len(lines)]
		if _, ok := s.Get(d); !ok {
			s.Set(d, d)
		}
	}
}
