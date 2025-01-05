package sieve

type node[K comparable, V any] struct {
	key   K
	value V

	prev *node[K, V]
	next *node[K, V]

	visited bool
}

func newNode[K comparable, V any](key K, value V) *node[K, V] {
	return &node[K, V]{
		key:     key,
		value:   value,
		prev:    nil,
		next:    nil,
		visited: false,
	}
}

// Sieve is a data structure working as a cache with a fixed size.
type Sieve[K comparable, V any] struct {
	head *node[K, V]
	tail *node[K, V]
	hand *node[K, V]

	m map[K]*node[K, V]

	capacity int
}

// New returns a new sieve.
// The size parameter is the maximum number of elements that the sieve can hold.
// If the size is less than or equal to zero, it panics.
func New[K comparable, V any](size int) Sieve[K, V] {
	if size <= 0 {
		panic("sieve: size must be greater than zero")
	}

	return Sieve[K, V]{
		head:     nil,
		tail:     nil,
		hand:     nil,
		m:        make(map[K]*node[K, V], size),
		capacity: size,
	}
}

// Len returns the number of elements in the sieve.
func (s *Sieve[K, V]) Len() int {
	return len(s.m)
}

// Insert inserts a new key-value pair in the sieve.
// If the key already exists, it does nothing.
func (s *Sieve[K, V]) Insert(key K, value V) {
	// key already exists
	if v, ok := s.m[key]; ok {
		// mark the node visited
		v.visited = true

		// update the value
		v.value = value

		return
	}

	// cache is full
	if len(s.m) == s.capacity {
		h := s.hand

		for h.visited {
			// don't evict the node, just mark it as not visited
			h.visited = false

			// move hand torwards the head
			h = h.prev

			// wrap around if we go beyond the head
			if h == nil {
				h = s.tail
			}
		}

		s.hand = s.hand.prev

		if h.next != nil {
			h.next.prev = h.prev
		} else { // so we are the last node
			s.tail = h.prev
		}

		if h.prev != nil {
			h.prev.next = h.next
		} else { // so we are the first node
			s.head = h.next
		}

		delete(s.m, h.key)
	}

	n := newNode(key, value)

	// insert into the cache
	s.m[key] = n

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

// Get returns the value associated with the key.
// If the key does not exist, it returns zero value an false, otherwise the value and true.
func (s *Sieve[K, V]) Get(key K) (V, bool) {
	if n, ok := s.m[key]; ok {
		// mark the node visited
		n.visited = true

		return n.value, true
	}

	var v V // zero value
	return v, false
}
