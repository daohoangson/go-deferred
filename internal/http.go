package internal // import "github.com/daohoangson/go-deferred/internal"

import (
	"crypto/tls"
	"net/http"
	"time"
)

var _httpClient *http.Client

// GetClient setups a sensible http client to be used
// https://medium.com/@nate510/don-t-use-go-s-default-http-client-4804cb19f779
func GetClient() *http.Client {
	if _httpClient == nil {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		_httpClient = &http.Client{
			Transport: tr,
			Timeout:   time.Minute * 10,
		}
	}

	return _httpClient
}

// RespondCode writes a http status code as response with the default status text
func RespondCode(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}
