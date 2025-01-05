package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/guerinoni/sieve"
)

func main() {
	const fileName = "trace"
	const capacity = 30
	cache := sieve.New[string, string](capacity)

	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	missCount := 0

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	for read := scanner.Scan(); read; read = scanner.Scan() {
		d := scanner.Text()
		if _, ok := cache.Get(d); !ok {
			missCount += 1
			cache.Insert(d, d)
		}
	}

	fmt.Printf("Miss count: %d\n", missCount)
}
