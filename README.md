# sieve

In memory cache with sieve eviction algorithm in pure Go.

- [x] thread-safe, with cache hits sharing a read lock
- [x] opt-out safety to use in single thread with more performance
- [x] no allocation per insert once the cache is full
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

Note that expired entries are removed lazily: `Len` counts them until a `Get`
or `Set` touches them, so with a TTL set `Len` is not a liveness metric.


## How it works

[This is the paper](https://yazhuozhang.com/assets/publication/nsdi24-sieve.pdf)

#### TL;DR

Web cache workloads commonly exhibit Power-law (generalized Zipfian) distributions [20, 26, 27, 34, 49, 52, 55, 81, 82, 97], where a small subset of objects accounts for the majority of requests. This skew in access patterns heavily influences cache management strategies.

Promotion and demotion are internal cache operations designed to maintain an efficient logical ordering of cached objects based on their access frequency or recency:

1.	Lazy promotion refers to deferring the promotion of cached objects until eviction time, minimizing the effort required to manage cache state. For instance, adding a reinsertion mechanism to a FIFO (First-In-First-Out) policy introduces lazy promotion. Unlike FIFO, which lacks promotion entirely, or LRU (Least Recently Used), which performs eager promotion by moving objects to the head of the cache on every hit, lazy promotion balances computational efficiency with better-informed eviction decisions. By deferring promotion, it can improve:
	•	Throughput, as it reduces computational overhead during hits.
	•	Efficiency, as decisions are made with more data about an object’s popularity.

2.	Quick demotion involves rapidly removing objects soon after insertion, particularly if they exhibit low popularity. This strategy is especially effective in handling workloads where objects are frequently scanned but rarely reused, as discussed in prior studies [16, 60, 67, 70, 75, 77]. Recent research [94] extends this concept to web cache workloads, demonstrating that quick demotion is beneficial because these workloads also follow Power-law distributions. With most objects being unpopular, quick demotion helps optimize cache usage by prioritizing valuable storage for high-demand content.

#### Notes from the paper

- A hit only sets the visited bit and never moves the node, so `Get` runs under
  a shared read lock and concurrent hits do not serialize (§3.1 and §4.3). A
  node that is already visited is not written again, so hot keys do not bounce
  their cache line between the cores.
- SIEVE is not scan-resistant (§7.2). A scan can push popular keys out, and
  without a ghost history the cache cannot recognize them when they come back.
  For block or page workloads with scans, an algorithm with a ghost queue such
  as ARC or S3-FIFO can do better.

## Comparison

Running the [example](./examples/main.go) you can see it is compared to:
 - [golang-lru](https://github.com/hashicorp/golang-lru)
 - [golang-fifo (s3-fifo)](https://github.com/scalalang2/golang-fifo)
 - [golang-fifo (sieve)](https://github.com/scalalang2/golang-fifo)

### Cache Miss Count (lower is better)

| Algorithm | Miss Count |
|-----------|------------|
| **sieve** | 328,766 |
| **sieve-single-thread** | 328,766 |
| golang-sieve | 328,766 |
| s3-fifo | 345,081 |
| golang-lru | 424,727 |

All sieve variants achieve the best hit rate with ~16,315 fewer misses than s3-fifo and ~95,961 fewer than LRU.

## Benchmarks

`BenchmarkSimple` inserts only unique keys, so the cache is always full and every
call pays an eviction. `BenchmarkBigInput` replays the sample input as a mixed
get and set workload, which is closer to real usage. `BenchmarkSimpleConcurrent`
runs through `b.RunParallel`, so its cost includes the lock contention of all the
cores.

`BenchmarkZipf` and `BenchmarkZipfParallel` are the large scale ones: a cache of
a million entries in front of 16M requests over about 1.5M distinct keys with a
Zipf (power-law) popularity, the workload the paper targets. They start on a
warm cache and also report the miss ratio, about 6%. The parallel ratio differs
because the goroutines interleave different parts of the trace.

The benchmarks of the other caches live in the `examples` module, so that this
one keeps an empty require list. `make bench` runs both.

```
goos: darwin
goarch: arm64
cpu: Apple M4 Pro
pkg: github.com/guerinoni/sieve

BenchmarkSimple-14                    169,879,341      35.27 ns/op       0 B/op      0 allocs/op
BenchmarkSimpleSingleThread-14        178,226,271      33.73 ns/op       0 B/op      0 allocs/op
BenchmarkBigInput-14                  401,950,822      15.02 ns/op       0 B/op      0 allocs/op
BenchmarkBigInputSingleThread-14      410,973,892      14.60 ns/op       0 B/op      0 allocs/op

BenchmarkSimpleWithTTL-14             133,781,913      44.91 ns/op       0 B/op      0 allocs/op
BenchmarkBigInputWithTTL-14           235,094,086      25.59 ns/op       0 B/op      0 allocs/op
BenchmarkSimpleConcurrent-14           57,786,535     101.1  ns/op       0 B/op      0 allocs/op
BenchmarkSimpleConcurrentWithTTL-14    46,597,582     136.8  ns/op       0 B/op      0 allocs/op

BenchmarkZipf/New-14                  133,118,554      45.58 ns/op   0.06262 misses/op   0 B/op   0 allocs/op
BenchmarkZipf/NewSingleThread-14      132,810,908      44.76 ns/op   0.06262 misses/op   0 B/op   0 allocs/op
BenchmarkZipf/WithTTL-14               96,530,530      64.47 ns/op   0.06262 misses/op   0 B/op   0 allocs/op
BenchmarkZipfParallel/New-14           63,294,152      97.16 ns/op   0.04258 misses/op   0 B/op   0 allocs/op
BenchmarkZipfParallel/WithTTL-14       45,444,558     129.9  ns/op   0.04113 misses/op   0 B/op   0 allocs/op

pkg: github.com/guerinoni/sieve/examples

BenchmarkSimpleLRU-14                 122,381,400      48.88 ns/op      80 B/op      1 allocs/op
BenchmarkSimpleS3FIFO-14               36,193,890     165.4  ns/op     192 B/op      5 allocs/op
BenchmarkSimpleGolangSieve-14          55,114,594     108.8  ns/op     136 B/op      3 allocs/op

BenchmarkBigInputLRU-14               303,105,064      19.63 ns/op       0 B/op      0 allocs/op
BenchmarkBigInputS3FIFO-14            181,910,962      33.20 ns/op       0 B/op      0 allocs/op
BenchmarkBigInputGolangSieve-14       407,846,848      14.70 ns/op       0 B/op      0 allocs/op
```

Note: once the cache is full, `Set` reuses the node it evicts, so an insert
allocates nothing and the insert-only `BenchmarkSimple` rows read `0 B/op`. The
only allocations left are the nodes created while the cache fills up, and the
growth of the map. For the other caches, `-benchmem` reports per-operation
averages rounded to integers, which hides the allocations of the mixed
workloads: the rows that show their node cost are the insert-only ones.

### Summary

| Metric | sieve | sieve-single-thread | golang-lru | s3-fifo | golang-sieve |
|--------|-------|---------------------|------------|---------|--------------|
| Hit Rate | Best | Best | Worst | Good | Best |
| Speed (insert only) | 35.27 ns | 33.73 ns | 48.88 ns | 165.4 ns | 108.8 ns |
| Speed (mixed workload) | 15.02 ns | 14.60 ns | 19.63 ns | 33.20 ns | 14.70 ns |
| Memory (insert only) | 0 B/op | 0 B/op | 80 B/op | 192 B/op | 136 B/op |
| Allocations (insert only) | 0 | 0 | 1 | 5 | 3 |
