package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"sync"

	"github.com/guerinoni/sieve"
	lru "github.com/hashicorp/golang-lru/v2"
	golangfifo "github.com/scalalang2/golang-fifo/sieve"
)

const fileName = "input"
const capacity = 100

func main() {
	// printMemoryUsage()

	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	data := make([]string, 0)
	for scanner.Scan() {
		data = append(data, scanner.Text())
	}

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		missCountSieve := doSieve(data)
		fmt.Printf("Miss count sieve:		%d\n", missCountSieve)
		wg.Done()
	}()

	go func() {
		missCountLRU := doLRU(data)
		fmt.Printf("Miss count golang-lru:		%d\n", missCountLRU)
		wg.Done()
	}()

	go func() {
		missCountGolangFifo := doGolangFifo(data)
		fmt.Printf("Miss count golang-fifo:		%d\n", missCountGolangFifo)
		wg.Done()
	}()

	wg.Wait()

	// printMemoryUsage()
}

func doSieve(intput []string) int {
	mc := 0
	cache := sieve.New[string, string](capacity)

	for _, d := range intput {
		if _, ok := cache.Get(d); !ok {
			mc += 1
			cache.Set(d, d)
		}
	}

	return mc
}

func doLRU(input []string) int {
	mc := 0
	cache, err := lru.New[string, string](capacity)
	if err != nil {
		fmt.Println(err)
		return mc
	}

	for _, d := range input {
		if _, ok := cache.Get(d); !ok {
			mc += 1
			cache.Add(d, d)
		}
	}

	return mc
}

func doGolangFifo(input []string) int {
	mc := 0
	cache := golangfifo.New[string, string](capacity, 0)

	for _, d := range input {
		if _, ok := cache.Get(d); !ok {
			mc += 1
			cache.Set(d, d)
		}
	}

	return mc
}

func printMemoryUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("Alloc = %v KB\n", m.Alloc/1024)
	fmt.Printf("TotalAlloc = %v KB\n", m.TotalAlloc/1024)
	fmt.Printf("Sys = %v KB\n", m.Sys/1024)
	fmt.Printf("NumGC = %v\n", m.NumGC)
	fmt.Println("------")
}
