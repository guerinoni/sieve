// Those test use the same pkg because we need to mock the var `now` in sieve.go.
package sieve

import (
	"bufio"
	"os"
	"testing"
	"time"
)

func TestOneElementWithTTL(t *testing.T) {
	s := New[int, struct{}](4).WithTTL(1 * time.Second)

	// fake now
	sec := 1
	now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, sec, 0, time.UTC) }

	s.Set(7, struct{}{})

	_, ok := s.Get(7)
	if !ok {
		t.Errorf("expected key 7 to be in the cache")
	}

	// simulate time passing
	sec = 2
	now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, sec, 0, time.UTC) }

	_, ok = s.Get(7)
	if !ok {
		t.Errorf("expected key 7 to be in the cache")
	}

	// simulate time passing
	sec = 3
	now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, sec, 0, time.UTC) }

	_, ok = s.Get(7)
	if !ok {
		t.Errorf("expected key 7 to be in the cache")
	}

	// simulate time passing
	sec = 5
	now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, sec, 0, time.UTC) }

	_, ok = s.Get(7)
	if ok {
		t.Errorf("expected key 7 to be expired")
	}
}

func TestTwoElementWithTLL(t *testing.T) {
	s := New[int, struct{}](4).WithTTL(1 * time.Second)

	t.Run("first evict tail", func(t *testing.T) {
		// fake now
		sec := 1
		now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, sec, 0, time.UTC) }

		s.Set(7, struct{}{})
		s.Set(8, struct{}{})

		// simulate time passing
		sec = 2
		now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, sec, 0, time.UTC) }

		s.Get(7) // keep 7 alive

		// simulate time passing
		sec = 3
		now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, sec, 0, time.UTC) }

		_, ok := s.Get(7)
		if !ok {
			t.Errorf("expected key 7 to be in the cache")
		}

		_, ok = s.Get(8)
		if ok {
			t.Errorf("expected key 8 to be expired")
		}

		if s.Len() != 1 {
			t.Errorf("expected len to be 1")
		}
	})

	s.Flush()

	t.Run("first evict head", func(t *testing.T) {
		// fake now
		sec := 1
		now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, sec, 0, time.UTC) }

		s.Set(7, struct{}{})
		s.Set(8, struct{}{})

		// simulate time passing
		sec = 2
		now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, sec, 0, time.UTC) }

		s.Get(8) // keep 8 alive

		// simulate time passing
		sec = 3
		now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, sec, 0, time.UTC) }

		_, ok := s.Get(7)
		if ok {
			t.Errorf("expected key 7 to be expired")
		}

		_, ok = s.Get(8)
		if !ok {
			t.Errorf("expected key 8 to be in the cache")
		}

		if s.Len() != 1 {
			t.Errorf("expected len to be 1")
		}
	})
}

func TestThreeElementWithTTL(t *testing.T) { //nolint: cyclop
	s := New[int, struct{}](4).WithTTL(1 * time.Second)

	t.Run("first evict head", func(t *testing.T) { //nolint: dupl
		// fake now
		sec := 1
		now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, sec, 0, time.UTC) }
		s.Set(7, struct{}{})

		sec = 2
		now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, sec, 0, time.UTC) }

		s.Set(8, struct{}{})
		s.Set(9, struct{}{}) // head is 9 here since is the latest inserted

		if s.Len() != 3 {
			t.Errorf("expected len to be 3")
		}

		s.Get(7) // keep element 7 alive

		sec = 3
		now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, sec, 0, time.UTC) }

		_, ok7 := s.Get(7)
		_, ok8 := s.Get(8)
		if !ok7 || !ok8 {
			t.Errorf("expected 7 and 8 keys to be in the cache")
		}

		sec = 4
		now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, sec, 0, time.UTC) }

		{
			_, ok7 := s.Get(7)
			_, ok8 := s.Get(8)
			if !ok7 || !ok8 {
				t.Errorf("expected 7 and 8 keys to be in the cache")
			}

			_, ok9 := s.Get(9)
			if ok9 {
				t.Errorf("expected key 9 to be expired")
			}
		}
	})

	s.Flush()

	t.Run("first evict middle item", func(t *testing.T) { //nolint: dupl
		// fake now
		sec := 1
		now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, sec, 0, time.UTC) }
		s.Set(7, struct{}{})

		sec = 2
		now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, sec, 0, time.UTC) }

		s.Set(8, struct{}{})
		s.Set(9, struct{}{})

		if s.Len() != 3 {
			t.Errorf("expected len to be 3")
		}

		s.Get(7) // keep element 7 alive

		sec = 3
		now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, sec, 0, time.UTC) }

		_, ok7 := s.Get(7)
		_, ok9 := s.Get(9)
		if !ok7 || !ok9 {
			t.Errorf("expected 7 and 9 keys to be in the cache")
		}

		sec = 4
		now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, sec, 0, time.UTC) }

		{
			_, ok7 := s.Get(7)
			_, ok9 := s.Get(9)
			if !ok7 || !ok9 {
				t.Errorf("expected 7 and 8 keys to be in the cache")
			}

			_, ok8 := s.Get(8)
			if ok8 {
				t.Errorf("expected key 8 to be expired")
			}
		}
	})

	s.Flush()

	t.Run("first evict tail", func(t *testing.T) {
		// fake now
		sec := 1
		now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, sec, 0, time.UTC) }
		s.Set(7, struct{}{}) // tail here is 7 since is the first inserted

		sec = 2
		now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, sec, 0, time.UTC) }

		s.Set(8, struct{}{})
		s.Set(9, struct{}{})

		if s.Len() != 3 {
			t.Errorf("expected len to be 3")
		}

		sec = 3
		now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, sec, 0, time.UTC) }

		_, ok := s.Get(7) // expired because entered at sec=1
		if ok {
			t.Errorf("expected key 7 to be expired")
		}

		_, ok = s.Get(8)
		if !ok {
			t.Errorf("expected key 8 to be in the cache")
		}

		_, ok = s.Get(9)
		if !ok {
			t.Errorf("expected key 9 to be in the cache")
		}
	})
}

func TestMoreElementWithTTL(t *testing.T) {
	s := New[int, struct{}](4).WithTTL(1 * time.Second)

	t.Run("hand is in the middle of linked list", func(t *testing.T) {
		sec := 1
		now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, sec, 0, time.UTC) }

		s.Set(7, struct{}{})
		s.Set(8, struct{}{})
		s.Set(9, struct{}{})
		s.Set(10, struct{}{})

		s.Get(7) // keep 7 inside cache

		// 8 is evicted
		s.Set(11, struct{}{}) // 11 10 9 7

		sec = 3
		now = func() time.Time { return time.Date(2025, 1, 1, 0, 0, sec, 0, time.UTC) }

		_, ok := s.Get(9)
		if ok {
			t.Errorf("expected key 8 to be expired")
		}
	})
}

func BenchmarkSimpleWithTTL(b *testing.B) {
	b.ReportAllocs()

	s := NewSingleThread[int, string](10).WithTTL(100 * time.Millisecond)

	for i := range b.N {
		s.Set(i, "one")
	}
}

func BenchmarkSimpleConcurrentWithTTL(b *testing.B) {
	b.ReportAllocs()

	s := New[int, string](10).WithTTL(100 * time.Millisecond)
	for i := range 100 {
		go func(i int) {
			s.Set(i, "one")
		}(i)

		go func(i int) {
			s.Get(i)
		}(i)
	}
}

// BenchmarkBigInputWithTTL-12             1000000000               0.05654 ns/op         0 B/op          0 allocs/op.
func BenchmarkBigInputWithTTL(b *testing.B) {
	b.ReportAllocs()

	s := New[string, string](1000).WithTTL(100 * time.Millisecond)

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
