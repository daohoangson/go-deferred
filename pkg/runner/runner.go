package runner // import "github.com/daohoangson/go-deferred/pkg/runner"

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/daohoangson/go-deferred/internal"
)

type runner struct {
	client *http.Client
	logger *logrus.Logger

	maxHitsPerLoop uint64
}

// New returns a new Runner instance
func New(client *http.Client, logger *logrus.Logger) Runner {
	r := &runner{}
	r.init(client, logger)
	return r
}

// Loop keeps hitting the specified URL until there is no more jobs
func Loop(r Runner, url string) (Hits, error) {
	hits := Hits{}
	hits.TimeStart = time.Now()
	maxHitsPerLoop := r.GetMaxHitsPerLoop()

	var someError error
	outerLogger := r.GetLogger().WithFields(logrus.Fields{
		"!": "Loop",
		"_": url,
	})

	for {
		innerLogger := outerLogger.WithField("seq", len(hits.List))

		innerLogger.Debug("Looping...")
		hit, err := r.Hit(url)
		hits.List = append(hits.List, hit)
		if err != nil {
			innerLogger = innerLogger.WithError(err)
			someError = err
			break
		}

		data := hit.Data
		if len(data.Message) > 0 {
			innerLogger.Warn(data.Message)
		}

		if !data.MoreDeferred && !data.More {
			break
		}

		if maxHitsPerLoop > 0 && uint64(len(hits.List)) == maxHitsPerLoop {
			innerLogger.Warn("Reached max hits per loop")
			break
		}
	}

	hits.TimeElapsed = time.Since(hits.TimeStart)
	outerLogger = outerLogger.WithFields(logrus.Fields{
		"elapsed": hits.TimeElapsed,
		"len":     len(hits.List),
	})

	if someError != nil {
		outerLogger.Error("Stopped")
		return hits, someError
	}

	outerLogger.Info("Stopped")
	return hits, nil
}

func (r *runner) GetLogger() *logrus.Logger {
	return r.logger
}

func (r *runner) GetMaxHitsPerLoop() uint64 {
	return r.maxHitsPerLoop
}

func (r *runner) Hit(url string) (Hit, error) {
	hit := Hit{}
	hit.TimeStart = time.Now()
	logger := r.logger.WithFields(logrus.Fields{
		"!": "Once",
		"_": url,
	})

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		logger.WithError(err).Error("Could not prepare request")
		return hit, err
	}
	req.Close = true
	req.Header.Set(internal.GetProtocolVersionHeaderKey(), internal.GetProtocolVersion())

	logger.Debug("Sending...")
	resp, err := r.client.Do(req)
	hit.TimeElapsed = time.Since(hit.TimeStart)
	logger.WithField("elapsed", hit.TimeElapsed).Debug("Received")

	if err != nil {
		logger.WithError(err).Error("Could not send request")
		return hit, err
	}

	err = json.NewDecoder(resp.Body).Decode(&hit.Data)
	if err != nil {
		logger.WithError(err).Error("Could not parse response")
		return hit, err
	}

	enqueueValue := resp.Header.Get(internal.GetProtocolEnqueueHeaderKey())
	if len(enqueueValue) > 0 {
		if enqueue, err := strconv.ParseInt(enqueueValue, 10, 64); err == nil {
			hit.HasEnqueue = true
			hit.Enqueue = enqueue
			logger = logger.WithField("enqueue", enqueue)
		}
	}

	logger.WithField("more?", internal.Ternary(hit.Data.MoreDeferred || hit.Data.More, 1, 0)).Debug("Parsed")
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

	r.maxHitsPerLoop = 5
	maxHitsPerLoopValue := os.Getenv("DEFERRED_MAX_HITS_PER_LOOP")
	if len(maxHitsPerLoopValue) > 0 {
		if maxHitsPerLoop, err := strconv.ParseUint(maxHitsPerLoopValue, 10, 64); err == nil {
			r.maxHitsPerLoop = maxHitsPerLoop
			logger.WithField("maxHitsPerLoop", maxHitsPerLoop).Info("Updated max hits per loop")
		}
	}

	logger.Debug("Initialized runner")
}
