# sieve

In memory cache with sieve eviction algorithm in pure Go.

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

Running the [example](./examples/main.go) you can see it is compared to [golang-lru](github.com/hashicorp/golang-lru/v2) using the same input and counting the cache miss.
```
Miss count sieve: 4051
Miss count golang-lru: 621835
```

Running 1 cache at time (using commented code) the result of memory allocated are the following:

golang-lru
```
# before workload

Alloc = 178 KB
TotalAlloc = 178 KB
Sys = 6547 KB
NumGC = 0
------

Miss count golang-lru: 621835

# after workload

Alloc = 2128 KB
TotalAlloc = 60348 KB
Sys = 11539 KB
NumGC = 16
------
```

sieve:
```
# before workload

Alloc = 178 KB
TotalAlloc = 178 KB
Sys = 6547 KB
NumGC = 0
------

Miss count sieve: 4051

# after workload

Alloc = 2221 KB
TotalAlloc = 2221 KB
Sys = 6803 KB
NumGC = 0
------
```