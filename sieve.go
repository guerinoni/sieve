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

	// access is the time of the last access, in nanoseconds since Cache.epoch.
	// Both access and visited are atomic because Get updates them under the
	// read lock.
	access  atomic.Int64
	visited atomic.Bool
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
	// epoch is the origin of the node access times. Its monotonic clock reading
	// keeps the TTL immune to wall clock changes.
	epoch time.Time

	mu rwLocker
}

type rwLocker interface {
	sync.Locker
	RLock()
	RUnlock()
}

// WithTTL is a builder function used to add the expiration management for keys.
// It must be called before the cache is used or shared between goroutines.
func (s *Cache[K, V]) WithTTL(ttl time.Duration) *Cache[K, V] {
	s.ttl = ttl
	s.epoch = time.Now()

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
		epoch:    time.Time{},
		mu:       &sync.RWMutex{},
	}
}

// NewSingleThread returns a new sieve that is safe for single-threaded use.
func NewSingleThread[K comparable, V any](size int32) *Cache[K, V] {
	c := New[K, V](size)

	c.mu = noopMutex{}

	return c
}

// Len returns the number of elements in the sieve.
// With a TTL configured, Len counts entries that have expired but have not
// been purged yet: entries are removed lazily, on the next Get or Set call.
func (s *Cache[K, V]) Len() int32 {
	return s.len.Load()
}

// Set inserts a new key-value pair in the sieve.
// If the key already exists, it updates the value, marks the node as
// visited and, with a TTL configured, refreshes its access time.
// The order of the insert will be something like:
// [head] -> [node] -> [node] -> ... -> [tail]
// The hand pointer is moving from the tail to the head.
// The `next` it to the tail, and the `prev` is to the head.
func (s *Cache[K, V]) Set(key K, value V) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()

	// key already exists
	if v, ok := s.m[key]; ok {
		// mark the node visited
		v.visited.Store(true)

		// update the value
		v.value = value

		// update the access time
		v.access.Store(now)

		return
	}

	var n *node[K, V]

	// a full cache recycles the evicted node, so it inserts without allocating
	if s.Len() >= s.capacity {
		n = s.evictNode(now)
	} else {
		n = new(node[K, V])
	}

	n.key = key
	n.value = value
	n.visited.Store(false)
	n.access.Store(now)

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

// now returns the nanoseconds elapsed since the epoch. Without a TTL it returns
// 0, so the clock is only read when it matters.
func (s *Cache[K, V]) now() int64 {
	if s.ttl <= 0 {
		return 0
	}

	return int64(time.Since(s.epoch))
}

func (s *Cache[K, V]) expired(n *node[K, V], now int64) bool {
	return s.ttl > 0 && now-n.access.Load() > int64(s.ttl)
}

// evictNode unlinks the next victim and returns it, so Set can reuse it.
func (s *Cache[K, V]) evictNode(now int64) *node[K, V] {
	h := s.hand

	for h.visited.Load() {
		// if the node is visited but is expired, then we can evict it
		if s.expired(h, now) {
			break
		}

		// don't evict the node, just mark it as not visited
		h.visited.Store(false)

		// move hand towards the head
		h = h.prev

		// wrap around if we go beyond the head
		if h == nil {
			h = s.tail
		}
	}

	s.hand = h
	s.remove(h)

	return h
}

// remove drops the node from the linked list and the map.
func (s *Cache[K, V]) remove(n *node[K, V]) {
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

	delete(s.m, n.key)

	s.len.Add(-1)
}

// Get returns the value associated with the key.
// If the key does not exist, it returns zero value and false, otherwise the value and true.
// A hit only sets the visited bit and never moves the node, so concurrent Get
// calls share a read lock, as section 3 of the SIEVE paper points out.
func (s *Cache[K, V]) Get(key K) (V, bool) {
	s.mu.RLock()

	n, ok := s.m[key]
	if !ok {
		s.mu.RUnlock()

		var zeroValue V

		return zeroValue, false
	}

	now := s.now()

	if s.expired(n, now) {
		s.mu.RUnlock()

		return s.expire(key)
	}

	if s.ttl > 0 {
		// update the access time
		n.access.Store(now)
	}

	// a hot node is already visited: skipping the store keeps its cache line
	// shared between the cores
	if !n.visited.Load() {
		n.visited.Store(true)
	}

	value := n.value

	s.mu.RUnlock()

	return value, true
}

// expire removes the key under the write lock. It checks the node again because
// another goroutine can refresh or remove it once Get releases the read lock.
func (s *Cache[K, V]) expire(key K) (V, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if n, ok := s.m[key]; ok && s.expired(n, s.now()) {
		s.remove(n)
	}

	var zeroValue V

	return zeroValue, false
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

	s.remove(n)

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

func (noopMutex) Lock()    {}
func (noopMutex) Unlock()  {}
func (noopMutex) RLock()   {}
func (noopMutex) RUnlock() {}

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
