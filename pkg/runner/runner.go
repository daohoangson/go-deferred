package runner // import "github.com/daohoangson/go-deferred/pkg/runner"

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/daohoangson/go-deferred/internal"
)

type runner struct {
	client *http.Client
	logger *logrus.Logger
}

// New returns a new Runner instance
func New(client *http.Client, logger *logrus.Logger) Runner {
	r := &runner{}
	r.init(client, logger)
	return r
}

// Loop keeps hitting the specified URL until there is no more jobs
func Loop(r Runner, url string) (uint64, *Hit, error) {
	var loops uint64

	logger := r.GetLogger()

	for {
		loops++
		logger := logger.WithFields(logrus.Fields{
			"!":     "Loop",
			"_":     url,
			"loops": loops,
		})

		logger.Debug("Looping...")
		result, err := r.Hit(url)
		if err != nil {
			logger.WithError(err).Error("Stopped")
			return loops, nil, err
		}

		data := result.Data
		if len(data.Message) > 0 {
			logger.Warn(data.Message)
		}

		if !data.MoreDeferred {
			logger.Info("Stopped")
			return loops, result, nil
		}
	}
}

func (r *runner) GetLogger() *logrus.Logger {
	return r.logger
}

func (r *runner) Hit(url string) (*Hit, error) {
	hit := new(Hit)
	hit.Data = new(Data)
	hit.TimeStart = time.Now()
	logger := r.logger.WithFields(logrus.Fields{
		"!": "Once",
		"_": url,
	})

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		logger.WithError(err).Error("Could not prepare request")
		return nil, err
	}
	req.Close = true

	logger.Debug("Sending...")
	resp, err := r.client.Do(req)
	if err != nil {
		logger.WithError(err).Error("Could not send request")
		return nil, err
	}

	hit.TimeElapsed = time.Since(hit.TimeStart)
	logger.WithField("elapsed", hit.TimeElapsed).Debug("Received")
	err = json.NewDecoder(resp.Body).Decode(&hit.Data)
	if err != nil {
		logger.WithError(err).Error("Could not parse response")
		return nil, err
	}

	logger.WithField("more?", internal.Ternary(hit.Data.MoreDeferred, 1, 0)).Debug("Parsed")
	return hit, nil
}

func (r *runner) init(client *http.Client, logger *logrus.Logger) {
	if client == nil {
		client = internal.GetHTTPClient()
	}
	r.client = client

	if logger == nil {
		logger = internal.GetLogger()
	}
	r.logger = logger

	logger.Debug("Initialized runner")
}
