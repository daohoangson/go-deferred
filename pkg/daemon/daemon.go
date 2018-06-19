package daemon // import "github.com/daohoangson/go-deferred/pkg/daemon"

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/daohoangson/go-deferred/internal"
	"github.com/daohoangson/go-deferred/pkg/runner"
)

type daemon struct {
	runner runner.Runner
	logger *logrus.Logger

	coolDown        time.Duration
	cutOff          time.Duration
	defaultSchedule time.Duration
	secret          string

	queued sync.Map

	stats      map[string]*Stats
	statsMutex sync.Mutex

	timerCounter uint64
	timers       sync.Map

	wakeUpCounterStart  uint64
	wakeUpCounterFinish uint64
	wakeUpMutex         sync.Mutex
	wakeUpSignal        chan uint64
}

// New returns a new Deamon instance
func New(runner runner.Runner, logger *logrus.Logger) Daemon {
	d := &daemon{}
	d.init(runner, logger)
	return d
}

func (d *daemon) ListenAndServe(port uint64) error {
	addr := fmt.Sprintf(":%d", port)
	d.logger.WithField("addr", addr).Warn("Going to listen and serve now...")

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

func (d *daemon) enqueueNow(url string) {
	d.step1Enqueue(url, 0)
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

	d.defaultSchedule = 30 * time.Second
	d.cutOff = 300 * time.Second

	d.stats = make(map[string]*Stats)

	d.wakeUpSignal = make(chan uint64, 42)
	go func(c chan uint64) {
		for {
			counter := <-c

			if counter == 0 {
				// our test script sends counter zero to stop this goroutine
				// TODO: implement a better way to do this
				break
			}

			d.step3WakeUp(counter)
		}
	}(d.wakeUpSignal)

	logger.Debug("Initialized daemon")
}

func (d *daemon) getTimerSoon() *time.Time {
	var soon *time.Time
	now := time.Now()

	d.timers.Range(func(key, value interface{}) bool {
		if t, ok := value.(time.Time); ok {
			if t.After(now) {
				if soon == nil || t.Before(*soon) {
					soon = &t
				}
			}
		}

		return true
	})

	return soon
}

func (d *daemon) hasTimers() bool {
	result := false

	d.timers.Range(func(key, value interface{}) bool {
		if _, ok := value.(time.Time); ok {
			result = true
			return false
		}

		return true
	})

	return result
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
	go d.step1Enqueue(target, time.Duration(delay)*time.Second)

	return http.StatusAccepted, nil
}

func (d *daemon) serveQueued(w http.ResponseWriter, u *url.URL) (int, error) {
	queued := make(map[string]float64)
	now := time.Now()

	d.queued.Range(func(key, value interface{}) bool {
		if url, ok := key.(string); ok {
			if t, ok := value.(time.Time); ok {
				queued[url] = t.Sub(now).Seconds()
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

func (d *daemon) step1Enqueue(url string, delay time.Duration) {
	now := time.Now()
	t := now.Add(delay)
	logger := d.logger.WithFields(logrus.Fields{
		"!": "Enqu",
		"_": url,
		"t": t.Sub(now).Seconds(),
	})

	var existing time.Time
	if existingValue, ok := d.queued.Load(url); ok {
		if existingTime, ok := existingValue.(time.Time); ok {
			existing = existingTime
			logger = logger.WithField("existing", existing.Sub(now).Seconds())
		}
	}

	if now.Before(existing) && existing.Before(t) {
		logger.Debug("Skipped")
		return
	}

	d.queued.Store(url, t)
	logger.Debug("Stored")

	d.statsMutex.Lock()
	stats := d.loadStats(url)
	stats.CounterEnqueues++
	d.stats[url] = stats
	d.statsMutex.Unlock()

	d.step2Schedule("step1")
}

func (d *daemon) step2Schedule(from string) {
	now := time.Now()
	initialNext := now.Add(24 * time.Hour)
	next := initialNext
	cutOff := now.Add(-d.cutOff)

	if d.defaultSchedule > 0 {
		next = now.Add(d.defaultSchedule)
	}

	d.queued.Range(func(key, value interface{}) bool {
		if t, ok := value.(time.Time); ok {
			if cutOff.Before(t) && t.Before(next) {
				next = t
			}
		} else {
			d.queued.Delete(key)
		}

		return true
	})

	logger := d.logger.WithFields(logrus.Fields{
		"!":    "Sche",
		"from": from,
		"next": next.Sub(now).Seconds(),
	})

	var newCounter uint64
	if next.Before(initialNext) {
		timerNeeded := false

		oldNext := d.getTimerSoon()
		if oldNext != nil {
			logger = logger.WithField("oldNext", oldNext.Sub(now).Seconds())
			if next.Before(*oldNext) {
				timerNeeded = true
			}
		} else {
			timerNeeded = true
		}
		if timerNeeded {
			newCounter = atomic.AddUint64(&d.timerCounter, 1)
			logger = logger.WithField("newCounter", newCounter)
		}
	}

	if newCounter == 0 {
		logger.Debug("Skipped")
		return
	}

	go func(next, now time.Time, counter uint64) {
		d.timers.Store(counter, next)

		if now.Before(next) {
			<-time.After(next.Sub(now))
		}

		d.wakeUpSignal <- counter
	}(next, now, newCounter)
	logger.Info("Set timer")
}

func (d *daemon) step3WakeUp(counter uint64) {
	now := time.Now()
	logger := d.logger.WithFields(logrus.Fields{
		"!":       "WkUp",
		"counter": counter,
	})

	d.wakeUpMutex.Lock()
	logger.Info("Running...")
	d.wakeUpCounterStart++
	d.wakeUpMutex.Unlock()

	var wg sync.WaitGroup

	d.queued.Range(func(key, value interface{}) bool {
		if t, ok := value.(time.Time); ok {
			if t.Before(now) {
				wg.Add(1)
				go func(key interface{}, t time.Time) {
					d.step4Hit(key, t)
					wg.Done()
				}(key, t)
			} else {
				logger.WithFields(logrus.Fields{
					"_": key,
					"t": t.Unix(),
				}).Debug("Skipped")
			}
		}

		return true
	})

	wg.Wait()

	d.timers.Delete(counter)
	if !d.hasTimers() {
		time.Sleep(d.coolDown)
		d.step2Schedule("step3")
	}

	d.wakeUpMutex.Lock()
	d.wakeUpCounterFinish++
	d.wakeUpMutex.Unlock()
}

func (d *daemon) step4Hit(key interface{}, t time.Time) {
	logger := d.logger.WithFields(logrus.Fields{
		"!": "Hitt",
		"_": key,
	})

	url, ok := key.(string)
	if !ok {
		logger.Error("Failed type assertion")
		return
	}

	d.statsMutex.Lock()
	prevStats := d.loadStats(url)
	prevStats.CounterWakeUps++
	d.stats[url] = prevStats
	isURLFirstHit := prevStats.CounterWakeUps == 1
	d.statsMutex.Unlock()

	skip := false
	if !isURLFirstHit {
		lastHitSubT := prevStats.LastHit.Sub(t)
		logger = logger.WithField("lastHitSubT", lastHitSubT)
		if lastHitSubT > 0 {
			skip = true
		}
	}
	if skip {
		logger.Debug("Skipped")
		return
	}

	loops, _, err := runner.Loop(d.runner, url)
	logger = logger.WithField("loops", loops)

	d.statsMutex.Lock()
	stats := d.loadStats(url)
	stats.CounterLoops += loops
	if err == nil {
		stats.LastHit = t.Add(time.Nanosecond)
	} else {
		stats.CounterErrors++
		logger.WithError(err)
	}
	d.stats[url] = stats
	d.statsMutex.Unlock()

	if err == nil {
		logger.Debug("Succeeded")
	} else {
		logger.Error("Failed")
		time.Sleep(d.coolDown)
	}
}
