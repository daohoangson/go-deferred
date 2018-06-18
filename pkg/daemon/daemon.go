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

	coolDown time.Duration
	delayMin int64
	delayMax int64
	secret   string

	queued sync.Map

	stats      map[string]*Stats
	statsMutex sync.Mutex

	timerCounterSet     uint64
	timerCounterTrigger uint64
	timerCounterRun     uint64
	timerMutex          sync.Mutex
	timerTimestampSet   int64
	timerTimestampRun   int64

	wakeUpCounterStart  uint64
	wakeUpCounterFinish uint64
	wakeUpMutex         sync.Mutex
	wakeUpSignal        chan int64
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

	d.coolDown = time.Second

	// at the earliest, schedule for the next second to avoid weird loops
	d.delayMin = 1
	// do not schedule for further than 5 minute
	d.delayMax = 300

	d.stats = make(map[string]*Stats)

	d.timerTimestampSet = math.MaxInt64
	d.wakeUpSignal = make(chan int64, 42)
	go func(c chan int64) {
		for {
			<-c
			d.step3WakeUp()
		}
	}(d.wakeUpSignal)

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

	d.step2Schedule("step1")
}

func (d *daemon) step2Schedule(from string) {
	var initialNext int64 = math.MaxInt64
	next := initialNext
	now := time.Now().Unix()
	cutOff := now - d.delayMax

	d.queued.Range(func(key, value interface{}) bool {
		if timestamp, ok := value.(int64); ok {
			if timestamp > cutOff {
				if timestamp < next {
					next = timestamp
				}
			} else {
				d.queued.Delete(key)
			}
		} else {
			d.queued.Delete(key)
		}

		return true
	})

	logger := d.logger.WithFields(logrus.Fields{
		"!":    "Sche",
		"from": from,
	})
	timerNeeded := false

	if next < initialNext {
		d.timerMutex.Lock()
		timerOthersRunning := d.timerCounterTrigger < d.timerCounterSet
		if !timerOthersRunning || next <= now {
			timerNeeded = true
			d.timerCounterSet++
			d.timerTimestampSet = now
		}
		d.timerMutex.Unlock()

		logger = logger.WithFields(logrus.Fields{
			"next":   next - now,
			"others": internal.Ternary(timerOthersRunning, 1, 0),
		})
	}

	if !timerNeeded {
		logger.Debug("Skipped")
		return
	}

	go func(next, now int64) {
		if next > now {
			<-time.After(time.Second * time.Duration(next-now))
		}

		d.timerMutex.Lock()
		d.timerCounterTrigger++
		d.timerMutex.Unlock()

		d.wakeUpSignal <- next
	}(next, now)
	logger.Info("Set timer")
}

func (d *daemon) step3WakeUp() {
	now := time.Now().Unix()
	logger := d.logger.WithFields(logrus.Fields{
		"!":   "WkUp",
		"now": now,
	})

	d.wakeUpMutex.Lock()
	logger.Debug("Running...")
	d.wakeUpCounterStart++
	d.wakeUpMutex.Unlock()

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

	wg.Wait()

	d.timerMutex.Lock()
	d.timerCounterRun++
	d.timerTimestampRun = now
	d.timerMutex.Unlock()

	d.wakeUpMutex.Lock()
	d.wakeUpCounterFinish++
	d.wakeUpMutex.Unlock()

	time.Sleep(d.coolDown)

	d.step2Schedule("step3")
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
	prev := prevStats.LastHit - timestamp
	logger = logger.WithField("prev", prev)
	if prev >= 0 {
		logger.Debug("Skipped (already hit)")
		return
	}
	logger.Debug("Starting...")

	loops, _, err := runner.Loop(d.runner, url)
	logger = logger.WithField("loops", loops)

	d.statsMutex.Lock()
	stats := d.loadStats(url)
	stats.CounterWakeUps++
	stats.CounterLoops += loops
	stats.LastHit = time.Now().Unix()
	if err != nil {
		stats.CounterErrors++
		logger.WithError(err)
	}
	d.stats[url] = stats
	d.statsMutex.Unlock()

	logger.Info("Done")
}
