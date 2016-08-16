package main

import "encoding/json"
import "fmt"
import "net/http"
import "os"
import "time"

func httpGet(url string) (bool, error) {
	type Data struct {
		MoreDeferred bool
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}
	req.Close = true

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}

	data := new(Data)
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return false, err
	}

	return data.MoreDeferred, nil
}

func loop(url string) error {
	for {
		start := time.Now()
		hasMoreDeferred, err := httpGet(url)
		if err != nil {
			return err
		}

		elapsed := time.Since(start)
		if hasMoreDeferred {
			fmt.Printf("%s = true (elapsed=%s)\n", url, elapsed)
		} else {
			fmt.Printf("%s = false (elapsed=%s)\n", url, elapsed)
			return nil
		}
	}
}

func worker(workerId int, url string, workers chan int) {
	fmt.Printf("Worker #%d: %s\n", workerId, url)
	err := loop(url)
	if err != nil {
		fmt.Printf("Worker #%d encountered an error:s (%s)\n", workerId, err)
	} else {
		fmt.Printf("Worker #%d is shutting down...\n", workerId)
	}
	workers <- workerId
}

func master(workerCount int, workers chan int) {
	start := time.Now()

	for i := 0; i < workerCount; i++ {
		workerId := <-workers
		fmt.Printf("Worker #%d has been shutdown\n", workerId)
	}

	elapsed := time.Since(start)
	fmt.Printf("Workers: %d. Total Elapsed: %s\n", workerCount, elapsed)
}

func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Printf("Usage: %s http://domain.com/xenforo/deferred.php [url2] [url3] ...\n", args[0])
		os.Exit(1)
	}

	workers := make(chan int)
	workerCount := 0

	for i := 1; i < len(args); i++ {
		url := args[i]
		go worker(i, url, workers)
		workerCount++
	}

	master(workerCount, workers)
}
