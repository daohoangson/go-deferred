package daemon // import "github.com/daohoangson/go-deferred/pkg/daemon"

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/daohoangson/go-deferred/internal"
	"github.com/daohoangson/go-deferred/pkg/runner"
)

type daemon struct {
	runner runner.Runner
	logger *logrus.Logger

	secret string
	queued sync.Map
	stats  sync.Map

	timer          *time.Timer
	timerMutex     sync.Mutex
	timerTimestamp int64
}

// New returns a new Deamon instance
func New(runner runner.Runner, logger *logrus.Logger) Daemon {
	d := &daemon{}
	d.init(runner, logger)
	return d
}

func (d *daemon) ListenAndServe(port uint64) error {
	addr := fmt.Sprintf(":%d", port)
	d.logger.WithField("addr", addr).Info("Going to listen and serve now...")

	var f http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		err := d.serve(w, r)
		if err != nil {
			d.logger.WithField("requestURI", r.RequestURI).WithError(err).Error("Responding HTTP 500...")
			internal.RespondCode(w, http.StatusInternalServerError)
		}
	}

	return http.ListenAndServe(addr, f)
}

func (d *daemon) SetSecret(secret string) {
	d.secret = secret
}

func (d *daemon) init(r runner.Runner, logger *logrus.Logger) {
	if logger == nil {
		logger = internal.GetLogger()
	}
	d.logger = logger

	if r == nil {
		r = runner.New(nil, logger)
	}
	d.runner = r

	d.timerTimestamp = math.MaxInt64

	logger.Debug("Initialized daemon")
}

func (d *daemon) loadStats(url string) *Stats {
	var stats *Stats
	if statsValue, ok := d.stats.Load(url); ok {
		if statsPtr, ok := statsValue.(*Stats); ok {
			stats = statsPtr
		}
	}
	if stats == nil {
		stats = &Stats{}
	}

	return stats
}

func (d *daemon) serve(w http.ResponseWriter, r *http.Request) error {
	u, err := url.Parse(r.RequestURI)
	if err != nil {
		return err
	}

	switch u.Path {
	case "/queue":
		return d.serveQueue(w, u)
	case "/stats":
		return d.serveStats(w, u)
	}

	internal.RespondCode(w, http.StatusBadRequest)
	return nil
}

func (d *daemon) serveQueue(w http.ResponseWriter, u *url.URL) error {
	query := u.Query()
	hash := query.Get("hash")
	target := query.Get("target")
	delayValue := query.Get("delay")
	if len(target) == 0 || len(hash) == 0 {
		internal.RespondCode(w, http.StatusBadRequest)
		return nil
	}

	md5 := internal.GetMD5(target, d.secret)
	if md5 != hash {
		internal.RespondCode(w, http.StatusForbidden)
		return nil
	}

	delay, _ := strconv.ParseInt(delayValue, 10, 64)
	timestamp := time.Now().Unix()
	if delay > 0 {
		timestamp += delay
	}

	go d.step1Enqueue(target, timestamp)

	internal.RespondCode(w, http.StatusAccepted)
	return nil
}

func (d *daemon) serveStats(w http.ResponseWriter, u *url.URL) error {
	list := make([]*Stats, 0)

	d.stats.Range(func(key, value interface{}) bool {
		if stats, ok := value.(*Stats); ok {
			list = append(list, stats)
		}

		return true
	})

	if json, err := json.Marshal(list); err == nil {
		w.Write(json)
	}

	return nil
}

func (d *daemon) step1Enqueue(url string, timestamp int64) {
	var existing int64
	if existingValue, ok := d.queued.Load(url); ok {
		if existingInt64, ok := existingValue.(int64); ok {
			existing = existingInt64
		}
	}

	now := time.Now().Unix()
	logger := d.logger.WithFields(logrus.Fields{
		"!":         "Enqu",
		"_":         url,
		"existing":  existing - now,
		"timestamp": timestamp - now,
	})

	if existing >= now && timestamp >= existing {
		logger.Debug("Skipped")
		return
	}

	d.queued.Store(url, timestamp)
	logger.Debug("Stored")

	stats := d.loadStats(url)
	stats.CounterEnqueues++
	d.stats.Store(url, stats)

	d.step2Schedule()
}

func (d *daemon) step2Schedule() {
	var next int64 = math.MaxInt64

	d.queued.Range(func(key, value interface{}) bool {
		if timestamp, ok := value.(int64); ok {
			if timestamp < next {
				next = timestamp
			}
		} else {
			d.queued.Delete(key)
		}

		return true
	})

	now := time.Now().Unix()
	if next < now {
		next = now
	}

	var timerNew *time.Timer
	var timerOld *time.Timer

	d.timerMutex.Lock()
	timerTimestamp := d.timerTimestamp
	if timerTimestamp < now || next < timerTimestamp {
		timerNew = time.NewTimer(time.Second * time.Duration(next-now))
		timerOld = d.timer
		d.timer = timerNew
		d.timerTimestamp = next
	}
	d.timerMutex.Unlock()

	logger := d.logger.WithFields(logrus.Fields{
		"!":     "Sche",
		"next":  next - now,
		"timer": timerTimestamp - now,
		"new?":  internal.Ternary(timerNew != nil, 1, 0),
		"old?":  internal.Ternary(timerOld != nil, 1, 0),
	})

	if timerNew == nil {
		logger.Info("Skipped")
		return
	}

	if timerOld != nil {
		timerOld.Stop()
	}

	go d.step3OnTimer(timerNew)
	logger.Info("Set timer")
}

func (d *daemon) step3OnTimer(timer *time.Timer) {
	<-timer.C

	now := time.Now().Unix()
	logger := d.logger.WithFields(logrus.Fields{
		"!":   "Timr",
		"now": now,
	})
	logger.Debug("Running...")

	d.queued.Range(func(key, value interface{}) bool {
		if timestamp, ok := value.(int64); ok {
			if timestamp <= now {
				d.step4Hit(key, now)
			} else {
				logger.WithFields(logrus.Fields{
					"_":         key,
					"timestamp": timestamp,
				}).Debug("Skipped hitting")
			}
		}

		return true
	})
}

func (d *daemon) step4Hit(key interface{}, timestamp int64) {
	logger := d.logger.WithFields(logrus.Fields{
		"!": "Hitt",
		"_": key,
	})

	logger.Debug("Starting...")
	url, ok := key.(string)
	if !ok {
		logger.Error("Failed type assertion")
		return
	}

	loops, _, err := d.runner.Loop(url)
	logger = logger.WithField("loops", loops)

	stats := d.loadStats(url)
	stats.CounterHits++
	stats.CounterLoops += loops
	stats.LatestTimestamp = timestamp
	stats.URL = url

	if err != nil {
		stats.CounterErrors++
		logger.WithError(err)
	}

	d.stats.Store(url, stats)
	logger.Info("Done")
}
