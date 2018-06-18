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

	delayMin int64
	delayMax int64
	secret   string

	queued sync.Map

	stats      map[string]*Stats
	statsMutex sync.Mutex

	timer           *time.Timer
	timerCounterSet uint64
	timerCounterRun uint64
	timerMutex      sync.Mutex
	timerTimestamp  int64
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
		code, err := d.serve(w, r)
		logger := d.logger.WithField("uri", r.RequestURI)

		if err != nil {
			logger = logger.WithError(err)
			code = http.StatusInternalServerError
		}
		if code != http.StatusOK {
			internal.RespondCode(w, code)
		}

		logger = logger.WithField("code", code)
		if code >= 500 {
			logger.Error("Responded with 5xx")
		} else if code >= 400 {
			logger.Warn("Responded with 4xx")
		} else if code != http.StatusOK {
			logger.Info("Responded")
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

	// at the earliest, schedule for the next second to avoid weird loops
	d.delayMin = 1
	// do not schedule for further than 5 minute
	d.delayMax = 300

	d.stats = make(map[string]*Stats)

	d.timerTimestamp = math.MaxInt64

	logger.Debug("Initialized daemon")
}

func (d *daemon) loadStats(url string) *Stats {
	var stats *Stats
	if statsValue, ok := d.stats[url]; ok {
		stats = statsValue
	}
	if stats == nil {
		stats = &Stats{}
	}

	return stats
}

func (d *daemon) serve(w http.ResponseWriter, r *http.Request) (int, error) {
	u, err := url.Parse(r.RequestURI)
	if err != nil {
		return 0, err
	}

	switch u.Path {
	case "/favicon.ico":
		return d.serveFavicon(w, u)
	case "/queue":
		return d.serveQueue(w, u)
	case "/queued":
		return d.serveQueued(w, u)
	case "/stats":
		return d.serveStats(w, u)
	}

	return http.StatusNotFound, nil
}

func (d *daemon) serveFavicon(w http.ResponseWriter, u *url.URL) (int, error) {
	// https://github.com/mathiasbynens/small
	ico, err := internal.Base64Decode("AAABAAEAAQEAAAEAGAAwAAAAFgAAACgAAAABAAAAAgAAAAEAGAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAP8AAAAAAA==")
	if err != nil {
		return 0, err
	}

	w.Header().Set("Cache-Control", "max-age=84600, public")
	w.Write(ico)

	return http.StatusOK, nil
}

func (d *daemon) serveQueue(w http.ResponseWriter, u *url.URL) (int, error) {
	query := u.Query()
	hash := query.Get("hash")
	target := query.Get("target")
	delayValue := query.Get("delay")
	if len(target) == 0 || len(hash) == 0 {
		return http.StatusBadRequest, nil
	}

	md5 := internal.GetMD5(target, d.secret)
	if md5 != hash {
		return http.StatusForbidden, nil
	}

	delay, _ := strconv.ParseInt(delayValue, 10, 64)
	go d.step1Enqueue(target, delay)

	return http.StatusAccepted, nil
}

func (d *daemon) serveQueued(w http.ResponseWriter, u *url.URL) (int, error) {
	queued := make(map[string]int64)
	now := time.Now().Unix()

	d.queued.Range(func(key, value interface{}) bool {
		if url, ok := key.(string); ok {
			if timestamp, ok := value.(int64); ok {
				queued[url] = timestamp - now
			}
		}

		return true
	})

	json, err := json.Marshal(queued)
	if err != nil {
		return 0, err
	}

	w.Write(json)
	return http.StatusOK, nil
}

func (d *daemon) serveStats(w http.ResponseWriter, u *url.URL) (int, error) {
	d.statsMutex.Lock()
	json, err := json.Marshal(d.stats)
	d.statsMutex.Unlock()

	if err != nil {
		return 0, err
	}

	w.Write(json)
	return http.StatusOK, nil
}

func (d *daemon) step1Enqueue(url string, delay int64) {
	if delay < d.delayMin {
		delay = d.delayMin
	}
	if delay > d.delayMax {
		delay = d.delayMax
	}
	timestamp := time.Now().Unix() + delay

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

	if now < existing && existing <= timestamp {
		logger.Debug("Skipped")
		return
	}

	d.queued.Store(url, timestamp)
	logger.Debug("Stored")

	d.statsMutex.Lock()
	stats := d.loadStats(url)
	stats.CounterEnqueues++
	d.stats[url] = stats
	d.statsMutex.Unlock()

	d.step2Schedule()
}

func (d *daemon) step2Schedule() {
	var initialNext int64 = math.MaxInt64
	next := initialNext
	now := time.Now().Unix()

	d.queued.Range(func(key, value interface{}) bool {
		if timestamp, ok := value.(int64); ok {
			if now <= timestamp && timestamp < next {
				next = timestamp
			}
		} else {
			d.queued.Delete(key)
		}

		return true
	})

	var timerNew *time.Timer
	var timerOld *time.Timer
	var timerTimestamp int64

	if now <= next && next < initialNext {
		d.timerMutex.Lock()
		timerTimestamp = d.timerTimestamp
		if timerTimestamp <= now || next < timerTimestamp {
			timerNew = time.NewTimer(time.Second * time.Duration(next-now))
			timerOld = d.timer
			d.timer = timerNew
			d.timerCounterSet++
			d.timerTimestamp = next
		}
		d.timerMutex.Unlock()
	}

	logger := d.logger.WithFields(logrus.Fields{
		"!":     "Sche",
		"next":  next - now,
		"timer": timerTimestamp - now,
		"new?":  internal.Ternary(timerNew != nil, 1, 0),
		"old?":  internal.Ternary(timerOld != nil, 1, 0),
	})

	if timerNew == nil {
		logger.Debug("Skipped")
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

	d.timerMutex.Lock()
	logger.Debug("Running...")
	d.timer = nil
	d.timerCounterRun++
	d.timerMutex.Unlock()

	var wg sync.WaitGroup

	d.queued.Range(func(key, value interface{}) bool {
		if timestamp, ok := value.(int64); ok {
			if timestamp <= now {
				wg.Add(1)
				go func(key interface{}, timestamp int64) {
					d.step4Hit(key, timestamp)
					wg.Done()
				}(key, timestamp)
			} else {
				logger.WithFields(logrus.Fields{
					"_":         key,
					"timestamp": timestamp,
				}).Debug("Skipped hitting")
			}
		}

		return true
	})

	time.Sleep(time.Second)
	wg.Wait()
	d.step2Schedule()
}

func (d *daemon) step4Hit(key interface{}, timestamp int64) {
	logger := d.logger.WithFields(logrus.Fields{
		"!": "Hitt",
		"_": key,
	})

	url, ok := key.(string)
	if !ok {
		logger.Error("Failed type assertion")
		return
	}

	prevStats := d.loadStats(url)
	if timestamp <= prevStats.LatestTimestamp {
		logger.Debug("Skipped (already hit)")
		return
	}
	logger.Debug("Starting...")

	loops, _, err := runner.Loop(d.runner, url)
	logger = logger.WithField("loops", loops)

	d.statsMutex.Lock()
	stats := d.loadStats(url)
	stats.CounterOnTimers++
	stats.CounterLoops += loops
	stats.LatestTimestamp = time.Now().Unix()
	if err != nil {
		stats.CounterErrors++
		logger.WithError(err)
	}
	d.stats[url] = stats
	d.statsMutex.Unlock()

	logger.Info("Done")
}
