package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/daohoangson/go-deferred/pkg/runner"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Printf("Usage: %s http://domain.com/xenforo/deferred.php [url2] [url3] ...\n", args[0])
		os.Exit(1)
	}

	var wg sync.WaitGroup
	r := runner.New(nil, nil)

	for i := 1; i < len(args); i++ {
		wg.Add(1)

		url := args[i]
		go func(workerID int, url string) {
			defer wg.Done()
			runner.Loop(r, url)
		}(i, url)
	}

	wg.Wait()
}
