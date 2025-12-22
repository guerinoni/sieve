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

s.Set(1, "one")
s.Set(2, "two")

v, ok := s.Get(1)
if !ok {
    // do something
}

_ = v // use value
```

## Single thread version

```go
s := sieve.NewSingleThread[int, string](2)

s.Set(1, "one")
s.Set(2, "two")

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

s.Set(1, "one")
s.Set(2, "two")

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

Running the [example](./examples/main.go) you can see it is compared to:
 - [golang-lru](https://github.com/hashicorp/golang-lru)
 - [golang-fifo (s3-fifo)](https://github.com/scalalang2/golang-fifo)
 - [golang-fifo (sieve)](https://github.com/scalalang2/golang-fifo)

### Cache Miss Count (lower is better)

| Algorithm | Miss Count |
|-----------|------------|
| golang-sieve | 328,766 |
| **sieve** | 338,193 |
| **sieve-single-thread** | 338,193 |
| s3-fifo | 345,081 |
| golang-lru | 424,727 |

Both sieve variants achieve better hit rate than s3-fifo (~6,888 fewer misses) and golang-lru (~86,534 fewer misses).

## Benchmarks

```
goos: darwin
goarch: arm64
cpu: Apple M4 Pro

BenchmarkSimple-14                     21,206,392      48.46 ns/op      80 B/op       1 allocs/op
BenchmarkSimpleSingleThread-14         22,182,102      48.22 ns/op      80 B/op       1 allocs/op
BenchmarkSimpleLRU-14                  27,166,786      41.48 ns/op      80 B/op       1 allocs/op
BenchmarkSimpleS3FIFO-14                6,465,110     157.6  ns/op     192 B/op       4 allocs/op
BenchmarkSimpleGolangSieve-14          11,521,432      97.60 ns/op     136 B/op       3 allocs/op

BenchmarkBigInput-14                1,000,000,000     0.03507 ns/op      0 B/op       0 allocs/op
BenchmarkBigInputSingleThread-14    1,000,000,000     0.03468 ns/op      0 B/op       0 allocs/op
BenchmarkBigInputLRU-14             1,000,000,000     0.03241 ns/op      0 B/op       0 allocs/op
BenchmarkBigInputS3FIFO-14          1,000,000,000     0.04485 ns/op      0 B/op       0 allocs/op
BenchmarkBigInputGolangSieve-14     1,000,000,000     0.02771 ns/op      0 B/op       0 allocs/op

BenchmarkSimpleWithTTL-14              25,725,212      47.24 ns/op      80 B/op       1 allocs/op
BenchmarkSimpleConcurrent-14        1,000,000,000   0.0000209 ns/op      0 B/op       0 allocs/op
BenchmarkSimpleConcurrentWithTTL-14 1,000,000,000   0.0000333 ns/op      0 B/op       0 allocs/op
```

### Summary

| Metric | sieve | sieve-single-thread | golang-lru | s3-fifo | golang-sieve |
|--------|-------|---------------------|------------|---------|--------------|
| Hit Rate | Good | Good | Worst | Good | Best |
| Speed (simple) | 48.46 ns | 48.22 ns | 41.48 ns | 157.6 ns | 97.60 ns |
| Memory | 80 B/op | 80 B/op | 80 B/op | 192 B/op | 136 B/op |
| Allocations | 1 | 1 | 1 | 4 | 3 |

