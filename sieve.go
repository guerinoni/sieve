package sieve

import (
	"sync"
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

func (n *node[K, V]) withTTL(now time.Time) *node[K, V] {
	n.access = now

	return n
}

func newNode[K comparable, V any](key K, value V) *node[K, V] {
	return &node[K, V]{
		key:     key,
		value:   value,
		prev:    nil,
		next:    nil,
		visited: false,
		access:  time.Time{},
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

	capacity int
	len      int
	ttl      time.Duration

	mu sync.Locker
}

// WithTTL is a builder function used to add the expiration management for keys.
func (s *Cache[K, V]) WithTTL(ttl time.Duration) *Cache[K, V] {
	s.ttl = ttl

	return s
}

// New returns a new sieve.
// The size parameter is the maximum number of elements that the sieve can hold.
// If the size is less than or equal to zero, it panics.
func New[K comparable, V any](size int) *Cache[K, V] {
	if size <= 0 {
		panic("sieve: size must be greater than zero")
	}

	return &Cache[K, V]{
		head:     nil,
		tail:     nil,
		hand:     nil,
		m:        make(map[K]*node[K, V]),
		capacity: size,
		len:      0,
		ttl:      0,
		mu:       &sync.Mutex{},
	}
}

// NewSingleThread returns a new sieve that is safe for single-threaded use.
func NewSingleThread[K comparable, V any](size int) *Cache[K, V] {
	c := New[K, V](size)

	c.mu = noopMutex{}

	return c
}

// Len returns the number of elements in the sieve.
func (s *Cache[K, V]) Len() int {
	return s.len
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

	// key already exists
	if v, ok := s.m[key]; ok {
		// mark the node visited
		v.visited = true

		// update the value
		v.value = value

		// update the access time
		v.access = now()

		return
	}

	// cache is full
	if s.Len() == s.capacity {
		s.evictNode()
	}

	n := newNode(key, value)

	if s.ttl > 0 {
		n = n.withTTL(now())
	}

	// insert into the cache
	s.m[key] = n

	s.len++

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

func (s *Cache[K, V]) evictNode() {
	h := s.hand

	for h.visited {
		// if the node is visited but is expired, then we can evict it
		if s.ttl > 0 && now().Sub(h.access) > s.ttl {
			break
		}

		// don't evict the node, just mark it as not visited
		s.hand.visited = false

		// move hand torwards the head
		h = h.prev

		// wrap around if we go beyond the head
		if h == nil {
			h = s.tail
		}
	}

	s.hand = h.prev

	// wrap to the end if we go beyond the head
	if s.hand == nil {
		s.hand = s.tail
	}

	if h.next != nil {
		h.next.prev = h.prev
	} else { // so we are the last node
		s.tail = h.prev
	}

	if h.prev != nil {
		h.prev.next = h.next
	} else { // so we are the first node
		s.hand = h.next
	}

	if s.head == h {
		s.head = h.next
	}

	delete(s.m, h.key)

	s.len--
}

func (s *Cache[K, V]) isExpired(n *node[K, V]) bool {
	return s.ttl > 0 && now().Sub(n.access) > s.ttl
}

func (s *Cache[K, V]) removeNodeFromLinkedList(n *node[K, V]) {
	if s.len == 1 {
		// just reset everything
		s.hand = nil
		s.head = nil
		s.tail = nil
	}

	if s.len == 2 {
		if s.head == n {
			s.head = s.tail
		} else { // so n == s.tail
			s.tail = s.head
		}

		s.hand = s.tail
	}

	if s.len >= 3 {
		// remove from the linked list

		switch n {
		case s.head:
			n.next.prev = nil
			s.head = n.next
		case s.tail:
			if s.hand == n {
				s.hand = n.prev
			}

			n.prev.next = nil
			s.tail = n.prev
		default:
			if s.hand == n {
				s.hand = n.prev
			}

			n.prev.next = n.next
			n.next.prev = n.prev
		}
	}
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

	if s.isExpired(n) {
		s.removeNodeFromLinkedList(n)

		// remove the node from the cache
		delete(s.m, n.key)

		// decrease length
		s.len--

		return zeroValue, false
	}

	// update the access time
	n.access = now()

	// mark the node as visited
	n.visited = true

	return n.value, true
}

// Flush removes all elements from the sieve and dealloc the internal structs.
func (s *Cache[K, V]) Flush() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.head = nil
	s.tail = nil
	s.hand = nil
	s.m = make(map[K]*node[K, V])
	s.len = 0
}

type noopMutex struct{}

func (noopMutex) Lock()   {}
func (noopMutex) Unlock() {}

// now is real `time.Now` function.
// It is a variable to make it easier to mock in tests.
var now = time.Now
