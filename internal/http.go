package internal // import "github.com/daohoangson/go-deferred/internal"

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

var _httpClient *http.Client

// GetHTTPClient setups a sensible http client to be used
// https://medium.com/@nate510/don-t-use-go-s-default-http-client-4804cb19f779
func GetHTTPClient() *http.Client {
	if _httpClient == nil {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}

		timeout := time.Minute + time.Second
		timeoutInSecondsValue := os.Getenv("DEFERRED_HTTP_CLIENT_TIMEOUT_IN_SECONDS")
		if len(timeoutInSecondsValue) > 0 {
			if timeoutInSeconds, err := strconv.ParseInt(timeoutInSecondsValue, 10, 64); err == nil {
				timeout = time.Duration(timeoutInSeconds) * time.Second
				fmt.Printf("timeout=%s\n", timeout)
			}
		}

		_httpClient = &http.Client{
			Transport: tr,
			Timeout:   timeout,
		}
	}

	return _httpClient
}

// RespondCode writes a http status code as response with the default status text
func RespondCode(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}
