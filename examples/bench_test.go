package main

import (
	"testing"

	lru "github.com/hashicorp/golang-lru/v2"
	s3fifo "github.com/scalalang2/golang-fifo/s3fifo"
	golangsieve "github.com/scalalang2/golang-fifo/sieve"
)

// The benchmarks of the other caches live here and not next to the library so
// that the library module keeps an empty require list. They mirror the shape of
// the ones in the root module and read the same input file, so the two runs can
// be compared directly.

const one = "one"

const (
	simpleCapacity   = 10
	bigInputCapacity = 1000
)

func benchInput(b *testing.B) []string {
	b.Helper()

	lines, err := readInput()
	if err != nil {
		b.Fatalf("could not read %s: %v", fileName, err)
	}

	return lines
}

func BenchmarkSimpleLRU(b *testing.B) {
	b.ReportAllocs()

	s, err := lru.New[int, string](simpleCapacity)
	if err != nil {
		b.Fatalf("could not create lru: %v", err)
	}

	for i := range b.N {
		s.Add(i, one)
	}
}

func BenchmarkBigInputLRU(b *testing.B) {
	b.ReportAllocs()

	lines := benchInput(b)

	s, err := lru.New[string, string](bigInputCapacity)
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

	s := s3fifo.New[int, string](simpleCapacity, 0)

	for i := range b.N {
		s.Set(i, one)
	}
}

func BenchmarkBigInputS3FIFO(b *testing.B) {
	b.ReportAllocs()

	lines := benchInput(b)

	s := s3fifo.New[string, string](bigInputCapacity, 0)

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

	s := golangsieve.New[int, string](simpleCapacity, 0)

	for i := range b.N {
		s.Set(i, one)
	}
}

func BenchmarkBigInputGolangSieve(b *testing.B) {
	b.ReportAllocs()

	lines := benchInput(b)

	s := golangsieve.New[string, string](bigInputCapacity, 0)

	b.ResetTimer()

	for i := range b.N {
		d := lines[i%len(lines)]
		if _, ok := s.Get(d); !ok {
			s.Set(d, d)
		}
	}
}
