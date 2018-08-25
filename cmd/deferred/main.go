package main

import (
	"fmt"
	"os"

	"github.com/daohoangson/go-deferred/pkg/runner"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Printf("Usage: %s http://domain.com/xenforo/deferred.php [url2] [url3] ...\n", args[0])
		os.Exit(1)
	}

	urlCount := len(args) - 1
	exitCodes := make(chan int, urlCount)
	r := runner.New(nil, nil)

	for i := 0; i < urlCount; i++ {
		url := args[i+1]
		go func(workerID int, url string, exitCodes chan int) {
			_, err := runner.Loop(r, url)

			exitCode := 0
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error processing %s: %s\n", url, err)
				exitCode = 2
			}

			exitCodes <- exitCode
		}(i, url, exitCodes)
	}

	summaryExitCode := 0
	for i := 0; i < urlCount; i++ {
		exitCode := <-exitCodes
		if exitCode > summaryExitCode {
			summaryExitCode = exitCode
		}
	}

	os.Exit(summaryExitCode)
}
