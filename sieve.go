// Package sieve implements the SIEVE cache eviction algorithm.
package sieve

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type node[K comparable, V any] struct {
	key   K
	value V

	prev *node[K, V]
	next *node[K, V]

	visited bool
	access  time.Time
}

func newNode[K comparable, V any](key K, value V, access time.Time) *node[K, V] {
	return &node[K, V]{
		key:     key,
		value:   value,
		prev:    nil,
		next:    nil,
		visited: false,
		access:  access,
	}
}

// Cache is a data structure working as a cache with a fixed size.
type Cache[K comparable, V any] struct {
	head *node[K, V]
	tail *node[K, V]
	// hand is a pointer to the current node that is going to be evicted.
	hand *node[K, V]

	// m is a map that holds the key-value pairs.
	m map[K]*node[K, V]

	capacity int32
	len      atomic.Int32
	ttl      time.Duration

	mu sync.Locker
}

// WithTTL is a builder function used to add the expiration management for keys.
// It must be called before the cache is used or shared between goroutines.
func (s *Cache[K, V]) WithTTL(ttl time.Duration) *Cache[K, V] {
	s.ttl = ttl

	return s
}

// New returns a new sieve.
// The size parameter is the maximum number of elements that the sieve can hold.
// If the size is less than or equal to zero, it panics.
func New[K comparable, V any](size int32) *Cache[K, V] {
	if size <= 0 {
		panic("sieve: size must be greater than zero")
	}

	return &Cache[K, V]{
		head:     nil,
		tail:     nil,
		hand:     nil,
		m:        make(map[K]*node[K, V]),
		capacity: size,
		len:      atomic.Int32{},
		ttl:      0,
		mu:       &sync.Mutex{},
	}
}

// NewSingleThread returns a new sieve that is safe for single-threaded use.
func NewSingleThread[K comparable, V any](size int32) *Cache[K, V] {
	c := New[K, V](size)

	c.mu = noopMutex{}

	return c
}

// Len returns the number of elements in the sieve.
func (s *Cache[K, V]) Len() int32 {
	return s.len.Load()
}

// Set inserts a new key-value pair in the sieve.
// If the key already exists, it does nothing.
// The order of the insert will be something like:
// [head] -> [node] -> [node] -> ... -> [tail]
// The hand pointer is moving from the tail to the head.
// The `next` it to the tail, and the `prev` is to the head.
func (s *Cache[K, V]) Set(key K, value V) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var atNow time.Time
	if s.ttl > 0 {
		atNow = time.Now()
	}

	// key already exists
	if v, ok := s.m[key]; ok {
		// mark the node visited
		v.visited = true

		// update the value
		v.value = value

		// update the access time
		v.access = atNow

		return
	}

	// cache is full
	if s.Len() >= s.capacity {
		s.evictNode(atNow)
	}

	n := newNode(key, value, atNow)

	// insert into the cache
	s.m[key] = n

	s.len.Add(1)

	// point to the current head
	n.next = s.head

	if s.head != nil {
		// update the prev link of the current head
		s.head.prev = n
	}

	// now head is the new node
	s.head = n

	if s.tail == nil {
		// if the tail is nil, then the new node is also the tail
		// because the cache is empty
		s.tail = n

		// also the hand is the tail
		s.hand = n
	}
}

func (s *Cache[K, V]) evictNode(atNow time.Time) {
	h := s.hand

	for h.visited {
		// if the node is visited but is expired, then we can evict it
		if s.ttl > 0 && atNow.Sub(h.access) > s.ttl {
			break
		}

		// don't evict the node, just mark it as not visited
		h.visited = false

		// move hand towards the head
		h = h.prev

		// wrap around if we go beyond the head
		if h == nil {
			h = s.tail
		}
	}

	s.hand = h
	s.removeNodeFromLinkedList(h)

	delete(s.m, h.key)

	s.len.Add(-1)
}

func (s *Cache[K, V]) removeNodeFromLinkedList(n *node[K, V]) {
	// the hand moves towards the head, so it falls back on the previous node
	if s.hand == n {
		s.hand = n.prev
	}

	if n.prev != nil {
		n.prev.next = n.next
	} else {
		s.head = n.next
	}

	if n.next != nil {
		n.next.prev = n.prev
	} else {
		s.tail = n.prev
	}

	// wrap to the end if we go beyond the head
	if s.hand == nil {
		s.hand = s.tail
	}

	// help the GC to collect the node
	n.prev = nil
	n.next = nil
}

// Get returns the value associated with the key.
// If the key does not exist, it returns zero value an false, otherwise the value and true.
func (s *Cache[K, V]) Get(key K) (V, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var zeroValue V

	n, ok := s.m[key]

	if !ok {
		return zeroValue, false
	}

	if s.ttl > 0 {
		atNow := time.Now()

		if atNow.Sub(n.access) > s.ttl {
			s.removeNodeFromLinkedList(n)

			// remove the node from the cache
			delete(s.m, n.key)

			// decrease length
			s.len.Add(-1)

			return zeroValue, false
		}

		// update the access time
		n.access = atNow
	}

	// mark the node as visited
	n.visited = true

	return n.value, true
}

// Delete removes the element with the given key from the sieve.
// It returns true if the key was found and removed, false otherwise.
func (s *Cache[K, V]) Delete(key K) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	n, ok := s.m[key]
	if !ok {
		return false
	}

	s.removeNodeFromLinkedList(n)

	delete(s.m, n.key)

	s.len.Add(-1)

	return true
}

// Flush removes all elements from the sieve and dealloc the internal structs.
func (s *Cache[K, V]) Flush() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.head = nil
	s.tail = nil
	s.hand = nil
	s.m = make(map[K]*node[K, V])
	s.len.Store(0)
}

type noopMutex struct{}

func (noopMutex) Lock()   {}
func (noopMutex) Unlock() {}

func (s *Cache[K, V]) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	var str strings.Builder

	str.WriteString("[")

	for n := s.head; n != nil; n = n.next {
		fmt.Fprintf(&str, "%v: %v", n.key, n.value)

		if n.next != nil {
			str.WriteString(" -> ")
		}
	}

	str.WriteString("]")

	return str.String()
}
