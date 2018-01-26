package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type Data struct {
	Message      string
	MoreDeferred bool
}

var _httpClient *http.Client

func httpClient() *http.Client {
	if _httpClient == nil {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		_httpClient = &http.Client{Transport: tr}
	}

	return _httpClient
}

func httpGet(url string) (*Data, error) {
	client := httpClient()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Close = true

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	data := new(Data)
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func loop(url string) error {
	for {
		start := time.Now()
		data, err := httpGet(url)
		if err != nil {
			return err
		}

		elapsed := time.Since(start)
		if data == nil {
			fmt.Printf("%s = N/A (elapsed=%s)\n", url, elapsed)
			return nil
		}

		if len(data.Message) > 0 {
			fmt.Println(data.Message)
		}

		if data.MoreDeferred {
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
