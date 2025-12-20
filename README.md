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

### Cache Miss Count (lower is better)

| Algorithm | Miss Count |
|-----------|------------|
| **sieve** | 338,193 |
| **sieve-single-thread** | 338,193 |
| s3-fifo | 345,081 |
| golang-lru | 424,727 |

Both sieve variants achieve the best hit rate with ~6,888 fewer misses than s3-fifo and ~86,534 fewer than LRU.

## Benchmarks

```
goos: darwin
goarch: arm64
cpu: Apple M4 Pro

BenchmarkSimple-14                     21,145,606      52.68 ns/op      80 B/op       1 allocs/op
BenchmarkSimpleSingleThread-14         22,654,136      47.99 ns/op      80 B/op       1 allocs/op
BenchmarkSimpleLRU-14                  29,453,671      41.98 ns/op      80 B/op       1 allocs/op
BenchmarkSimpleS3FIFO-14                7,925,346     156.9  ns/op     192 B/op       4 allocs/op

BenchmarkBigInput-14                1,000,000,000     0.03371 ns/op      0 B/op       0 allocs/op
BenchmarkBigInputSingleThread-14    1,000,000,000     0.03297 ns/op      0 B/op       0 allocs/op
BenchmarkBigInputLRU-14             1,000,000,000     0.03177 ns/op      0 B/op       0 allocs/op
BenchmarkBigInputS3FIFO-14          1,000,000,000     0.04467 ns/op      0 B/op       0 allocs/op

BenchmarkSimpleWithTTL-14              25,785,574      45.94 ns/op      80 B/op       1 allocs/op
BenchmarkSimpleConcurrent-14        1,000,000,000   0.0000345 ns/op      0 B/op       0 allocs/op
BenchmarkSimpleConcurrentWithTTL-14 1,000,000,000   0.0000303 ns/op      0 B/op       0 allocs/op
```

### Summary

| Metric | sieve | sieve-single-thread | golang-lru | s3-fifo |
|--------|-------|---------------------|------------|---------|
| Hit Rate | Best | Best | Worst | Good |
| Speed (simple) | 52.68 ns | 47.99 ns | 41.98 ns | 156.9 ns |
| Memory | 80 B/op | 80 B/op | 80 B/op | 192 B/op |
| Allocations | 1 | 1 | 1 | 4 |

