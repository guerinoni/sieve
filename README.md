# sieve

In memory cache with sieve eviction algorithm in pure Go.

- [x] thread-safe
- [x] opt-out safety to use in single thread with more performance
- [x] zero deps
- [x] no CGO
- [x] coverage 100%
- [x] opt-in TTL (evict expired on get/set)

## Usage

```go
s := sieve.New[int, string](2)

s.Insert(1, "one")
s.Insert(2, "two")

v, ok := s.Get(1)
if !ok {
    // do something
}

_ = v // use value
```

## Single thread version

```go
s := sieve.NewSingleThread[int, string](2)

s.Insert(1, "one")
s.Insert(2, "two")

v, ok := s.Get(1)
if !ok {
    // do something
}

_ = v // use value
```

## With TTL

This is an opt-in feature for both single and multi thread.

```go
s := sieve.New[int, string](2).WithTTL(1 *time.Second)

s.Insert(1, "one")
s.Insert(2, "two")

// ... wait 0.5s

v, ok := s.Get(1) // bump the access timestamp
if !ok {
    // do something
}

_ = v // use value


// ... wait another 1s
_, ok = s.Get(1) // value is still here

// ... wait 2s
v, ok := s.Get(1) // value is gone
```


## How it works

[This is the paper](https://yazhuozhang.com/assets/publication/nsdi24-sieve.pdf)

#### TL;DR

Web cache workloads commonly exhibit Power-law (generalized Zipfian) distributions [20, 26, 27, 34, 49, 52, 55, 81, 82, 97], where a small subset of objects accounts for the majority of requests. This skew in access patterns heavily influences cache management strategies.

Promotion and demotion are internal cache operations designed to maintain an efficient logical ordering of cached objects based on their access frequency or recency:

1.	Lazy promotion refers to deferring the promotion of cached objects until eviction time, minimizing the effort required to manage cache state. For instance, adding a reinsertion mechanism to a FIFO (First-In-First-Out) policy introduces lazy promotion. Unlike FIFO, which lacks promotion entirely, or LRU (Least Recently Used), which performs eager promotion by moving objects to the head of the cache on every hit, lazy promotion balances computational efficiency with better-informed eviction decisions. By deferring promotion, it can improve:
	•	Throughput, as it reduces computational overhead during hits.
	•	Efficiency, as decisions are made with more data about an object’s popularity.

2.	Quick demotion involves rapidly removing objects soon after insertion, particularly if they exhibit low popularity. This strategy is especially effective in handling workloads where objects are frequently scanned but rarely reused, as discussed in prior studies [16, 60, 67, 70, 75, 77]. Recent research [94] extends this concept to web cache workloads, demonstrating that quick demotion is beneficial because these workloads also follow Power-law distributions. With most objects being unpopular, quick demotion helps optimize cache usage by prioritizing valuable storage for high-demand content.

## Comparison

Running the [example](./examples/main.go) you can see it is compared to 
 - [golang-lru](github.com/hashicorp/golang-lru)
 - [golang-fifo](github.com/scalalang2/golang-fifo)
```
Miss count sieve:               338193
Miss count golang-fifo:         328766
Miss count golang-lru:          424727
```

Running 1 cache at time (using commented code) the result of memory allocated are the following:

golang-lru
```
# before workload

Alloc = 179 KB
TotalAlloc = 179 KB
Sys = 6547 KB
NumGC = 0
------

Miss count golang-lru: 424727

# after workload

Alloc = 22740 KB
TotalAlloc = 129156 KB
Sys = 57299 KB
NumGC = 14
------
```

golang-fifo
```
# before workfload

Alloc = 180 KB
TotalAlloc = 180 KB
Sys = 6547 KB
NumGC = 0
------

Miss count golang-fifo:         328766

# after workload

Alloc = 28752 KB
TotalAlloc = 134146 KB
Sys = 62227 KB
NumGC = 14
------
```

sieve:
```
# before workload

Alloc = 179 KB
TotalAlloc = 179 KB
Sys = 6291 KB
NumGC = 0
------

Miss count sieve: 338193

# after workload

Alloc = 27875 KB
TotalAlloc = 114269 KB
Sys = 61331 KB
NumGC = 11
------
```
