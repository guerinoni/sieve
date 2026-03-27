package sieve_test

import (
	"bufio"
	"os"
	"testing"
	"testing/synctest"
	"time"

	"github.com/guerinoni/sieve"
)

func TestOneElementWithTTL(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := sieve.New[int, struct{}](4).WithTTL(1 * time.Second)

		s.Set(7, struct{}{})
		synctest.Wait()

		time.Sleep(500 * time.Millisecond)
		synctest.Wait()

		_, ok := s.Get(7)
		if !ok {
			t.Errorf("expected key 7 to be in the cache")
		}

		time.Sleep(2 * time.Second)
		synctest.Wait()

		_, ok = s.Get(7)
		if ok {
			t.Errorf("expected key 7 to be expired")
		}
	})
}

func TestTwoElementWithTLLEvictTail(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := sieve.New[int, struct{}](4).WithTTL(1 * time.Second)

		s.Set(7, struct{}{})
		s.Set(8, struct{}{})
		synctest.Wait()

		time.Sleep(900 * time.Millisecond)
		synctest.Wait()

		s.Get(7) // keep 7 alive
		synctest.Wait()

		time.Sleep(200 * time.Millisecond)
		synctest.Wait()

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
}

func TestTwoElementWithTLLEvictHead(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := sieve.New[int, struct{}](4).WithTTL(1 * time.Second)

		s.Set(7, struct{}{})
		s.Set(8, struct{}{})
		synctest.Wait()

		time.Sleep(900 * time.Millisecond)
		synctest.Wait()

		s.Get(8) // keep 8 alive
		synctest.Wait()

		time.Sleep(200 * time.Millisecond)
		synctest.Wait()

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

func TestThreeElementWithTTLEvictHead(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := sieve.New[int, struct{}](4).WithTTL(1 * time.Second)

		s.Set(7, struct{}{})
		s.Set(8, struct{}{})
		s.Set(9, struct{}{})
		synctest.Wait()

		time.Sleep(900 * time.Millisecond)
		synctest.Wait()

		s.Get(7) // keep 7 alive
		s.Get(8) // keep 8 alive
		synctest.Wait()

		time.Sleep(200 * time.Millisecond)
		synctest.Wait()

		_, ok7 := s.Get(7)
		_, ok8 := s.Get(8)
		if !ok7 || !ok8 {
			t.Errorf("expected 7 and 8 keys to be in the cache, got 7=%v 8=%v", ok7, ok8)
		}

		if _, ok9 := s.Get(9); ok9 {
			t.Errorf("expected key 9 to be expired")
		}
	})
}

func TestThreeElementWithTTLEvictMiddle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := sieve.New[int, struct{}](4).WithTTL(1 * time.Second)

		s.Set(7, struct{}{})
		s.Set(8, struct{}{})
		s.Set(9, struct{}{})
		synctest.Wait()

		time.Sleep(900 * time.Millisecond)
		synctest.Wait()

		s.Get(7) // keep 7 alive
		s.Get(9) // keep 9 alive
		synctest.Wait()

		time.Sleep(200 * time.Millisecond)
		synctest.Wait()

		_, ok7 := s.Get(7)
		_, ok9 := s.Get(9)
		if !ok7 || !ok9 {
			t.Errorf("expected 7 and 9 keys to be in the cache, got 7=%v 9=%v", ok7, ok9)
		}

		if _, ok8 := s.Get(8); ok8 {
			t.Errorf("expected key 8 to be expired")
		}
	})
}

func TestThreeElementWithTTLEvictTail(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := sieve.New[int, struct{}](4).WithTTL(1 * time.Second)

		s.Set(7, struct{}{})
		synctest.Wait()

		time.Sleep(500 * time.Millisecond)
		synctest.Wait()

		s.Set(8, struct{}{})
		s.Set(9, struct{}{})
		synctest.Wait()

		time.Sleep(600 * time.Millisecond)
		synctest.Wait()

		_, ok := s.Get(7)
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
	synctest.Test(t, func(t *testing.T) {
		s := sieve.New[int, struct{}](4).WithTTL(1 * time.Second)

		s.Set(7, struct{}{})
		s.Set(8, struct{}{})
		s.Set(9, struct{}{})
		s.Set(10, struct{}{})
		synctest.Wait()

		s.Get(7) // keep 7 inside cache
		s.Set(11, struct{}{})
		synctest.Wait()

		time.Sleep(2 * time.Second)
		synctest.Wait()

		if _, ok := s.Get(9); ok {
			t.Errorf("expected key 9 to be expired")
		}
	})
}

func TestSetWithAllExpired(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := sieve.New[int, struct{}](4).WithTTL(1 * time.Second)

		s.Set(7, struct{}{})
		s.Set(8, struct{}{})
		s.Set(9, struct{}{})
		s.Set(10, struct{}{})
		synctest.Wait()

		s.Get(7) // now hand should start after 7, because 7 is marked `visited`
		synctest.Wait()

		time.Sleep(2 * time.Second)
		synctest.Wait()

		s.Set(11, struct{}{})

		if expected := `[11: {} -> 10: {} -> 9: {} -> 8: {}]`; s.String() != expected {
			t.Errorf("expected %s, got %s", expected, s.String())
		}
	})
}

func TestHandNilPanicAfterTTLEvictions(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := sieve.New[int, struct{}](3).WithTTL(1 * time.Second)

		s.Set(1, struct{}{})
		s.Set(2, struct{}{})
		s.Set(3, struct{}{})
		synctest.Wait()

		time.Sleep(10 * time.Second)
		synctest.Wait()

		s.Get(1)
		s.Get(2)
		s.Get(3)
		synctest.Wait()

		s.Set(4, struct{}{})
		s.Set(5, struct{}{})
		synctest.Wait()

		s.Get(4)
		synctest.Wait()

		s.Set(6, struct{}{})
		synctest.Wait()

		s.Set(7, struct{}{})
		synctest.Wait()

		if s.Len() != 3 {
			t.Errorf("expected length 3, got %d", s.Len())
		}
	})
}

func BenchmarkSimpleWithTTL(b *testing.B) {
	b.ReportAllocs()

	s := sieve.NewSingleThread[int, string](10).WithTTL(100 * time.Millisecond)

	for i := range b.N {
		s.Set(i, "one")
	}
}

func BenchmarkSimpleConcurrentWithTTL(b *testing.B) {
	b.ReportAllocs()

	s := sieve.New[int, string](10).WithTTL(100 * time.Millisecond)

	for i := range 100 {
		go func(i int) {
			s.Set(i, "one")
		}(i)

		go func(i int) {
			s.Get(i)
		}(i)
	}
}

func BenchmarkBigInputWithTTL(b *testing.B) {
	b.ReportAllocs()

	s := sieve.New[string, string](1000).WithTTL(100 * time.Millisecond)

	file := testInputFile

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
