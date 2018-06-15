package internal // import "github.com/daohoangson/go-deferred/internal"

import (
	"crypto/tls"
	"net/http"
	"time"
)

var _httpClient *http.Client

// GetHttpClient setups a sensible http client to be used
// https://medium.com/@nate510/don-t-use-go-s-default-http-client-4804cb19f779
func GetHttpClient() *http.Client {
	if _httpClient == nil {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		_httpClient = &http.Client{
			Transport: tr,
			Timeout:   time.Second * 3,
		}
	}

	return _httpClient
}
