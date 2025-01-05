package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/guerinoni/sieve"
)

func main() {
	const fileName = "trace"
	const capacity = 50
	sieveChan := make(chan int, 1)
	sieveCache := sieve.New[string, string](capacity)
	go traceRunner(fileName, sieveChan, sieveCache)

	missCount := <-sieveChan
	fmt.Printf("Miss count: %d\n", missCount)
}

func traceRunner(fileName string, ch chan<- int, cache sieve.Cache[string, string]) {
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	missCount := 0

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	// Returns a boolean based on whether there's a next instance of `\n`
	// character in the IO stream. This step also advances the internal pointer
	// to the next position (after '\n') if it did find that token.

	for read := scanner.Scan(); read; read = scanner.Scan() {
		d := scanner.Text()
		if _, ok := cache.Get(d); !ok {
			missCount += 1
			cache.Insert(d, d)
		}
	}

	ch <- missCount
}
