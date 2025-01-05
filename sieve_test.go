package sieve_test

import (
	"testing"

	"github.com/guerinoni/sieve"
	"github.com/stretchr/testify/assert"
)

func TestPanicWithSizeZero(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			if r != "sieve: size must be greater than zero" {
				t.Errorf("expected panic message 'something went wrong', got '%v'", r)
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
			if r != "sieve: size must be greater than zero" {
				t.Errorf("expected panic message 'something went wrong', got '%v'", r)
			}
		} else {
			t.Errorf("expected panic but got none")
		}
	}()

	sieve.New[string, int](-10)
}

func TestEasy(t *testing.T) {
	s := sieve.New[int, string](2)
	assert.Equal(t, 0, s.Len())

	s.Insert(1, "one")
	assert.Equal(t, 1, s.Len())

	s.Insert(1, "one") // duplicate
	assert.Equal(t, 1, s.Len())

	s.Insert(2, "two")
	assert.Equal(t, 2, s.Len())

	{
		v, ok := s.Get(3)
		assert.False(t, ok)
		assert.Equal(t, "", v)
	}

	{
		v, ok := s.Get(1)
		assert.True(t, ok)
		assert.Equal(t, "one", v)
	}

	// now we start evicting

	s.Insert(3, "three")
	assert.Equal(t, 2, s.Len())

	{
		v, ok := s.Get(1)
		assert.True(t, ok)
		assert.Equal(t, "one", v)
	}

	{
		v, ok := s.Get(2)
		assert.False(t, ok)
		assert.Equal(t, "", v)
	}
}

func TestAllAreVisited(t *testing.T) {
	s := sieve.New[int, string](2)

	s.Insert(1, "one")
	s.Insert(2, "two")

	s.Get(1)
	s.Get(2)

	// so now we have all nodes visited

	s.Insert(3, "three")
	assert.Equal(t, 2, s.Len())

	{
		v, ok := s.Get(3)
		assert.True(t, ok)
		assert.Equal(t, "three", v)
	}

	{
		v, ok := s.Get(2)
		assert.True(t, ok)
		assert.Equal(t, "two", v)
	}

	{
		v, ok := s.Get(1)
		assert.False(t, ok)
		assert.Equal(t, "", v)
	}
}

func TestMoreComplex(t *testing.T) {
	s := sieve.New[int, struct{}](4)

	s.Insert(7, struct{}{})
	s.Insert(7, struct{}{})
	s.Insert(5, struct{}{})
	s.Insert(5, struct{}{})
	s.Insert(6, struct{}{})
	s.Insert(10, struct{}{})
	s.Insert(9, struct{}{})
	s.Insert(1, struct{}{})
	s.Insert(5, struct{}{})
	s.Insert(7, struct{}{})

	assert.Equal(t, 4, s.Len())

	_, ok := s.Get(7)
	assert.True(t, ok)

	_, ok = s.Get(5)
	assert.True(t, ok)

	_, ok = s.Get(9)
	assert.True(t, ok)

	_, ok = s.Get(1)
	assert.True(t, ok)
}

// BenchmarkSimple-12      16972869                72.24 ns/op           50 B/op          1 allocs/op
func BenchmarkSimple(b *testing.B) {
	b.ReportAllocs()

	s := sieve.New[int, string](10)

	for i := 0; i < b.N; i++ {
		s.Insert(i, "one")
	}
}
