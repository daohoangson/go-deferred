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

func (r *runner) HitOnce(url string) (*Hit, error) {
	hit := new(Hit)
	hit.Data = new(Data)
	hit.TimeStart = time.Now()
	logger := r.logger.WithField("url", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.WithError(err).Error("Could not prepare request")
		return nil, err
	}
	req.Close = true

	logger.Debug("Sending request...")
	resp, err := r.client.Do(req)
	if err != nil {
		logger.WithError(err).Error("Could not send request")
		return nil, err
	}

	hit.TimeElapsed = time.Since(hit.TimeStart)
	logger.WithField("elapsed", hit.TimeElapsed).Debug("Received response")
	err = json.NewDecoder(resp.Body).Decode(&hit.Data)
	if err != nil {
		logger.WithError(err).Error("Could not parse response")
		return nil, err
	}

	logger.WithField("more deferred?", hit.Data.MoreDeferred).Info("Hit OK")
	return hit, nil
}

func (r *runner) Loop(url string) (int, *Hit, error) {
	loops := 0
	for {
		logger := r.logger.WithField("loops", loops)
		loops++

		logger.Debug("Starting loop...")
		result, err := r.HitOnce(url)
		if err != nil {
			logger.Info("Stopping loop because of an error...")
			return loops, nil, err
		}

		data := result.Data
		if len(data.Message) > 0 {
			logger.Info(data.Message)
		}

		if !data.MoreDeferred {
			logger.Info("No more deferred, stopping loop...")
			return loops, result, nil
		}
	}
}

func (r *runner) init(client *http.Client, logger *logrus.Logger) {
	if client == nil {
		client = internal.GetHttpClient()
	}
	r.client = client

	if logger == nil {
		logger = logrus.New()
	}
	r.logger = logger

	logger.Debug("Initialized runner")
}
