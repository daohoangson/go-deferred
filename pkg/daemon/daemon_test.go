package daemon // import "github.com/daohoangson/go-deferred/pkg/daemon"

import (
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/daohoangson/go-deferred/pkg/runner"
	"github.com/stretchr/testify/assert"
)

func TestOne(t *testing.T) {
	d := testInit(runner.MockedHit{})
	url := "one"

	d.enqueueNow(url)
	waitForDaemon(d)

	stats := getStats(t, d, url)
	assert.Equal(t, uint64(1), stats.CounterLoops)
}

func TestLoop(t *testing.T) {
	d := testInit(
		runner.MockedHit{MoreDeferred: true},
		runner.MockedHit{},
	)
	url := "loop"

	d.enqueueNow(url)
	waitForDaemon(d)

	stats := getStats(t, d, url)
	assert.Equal(t, uint64(2), stats.CounterLoops)
}

func TestDefaultSchedule(t *testing.T) {
	d := testInit()
	d.defaultSchedule = time.Second / 4
	url := "default-schedule"

	d.enqueueNow(url)

	time.Sleep(time.Second)
	d.statsMutex.Lock()
	stats1, _ := d.stats[url]
	assert.NotNil(t, stats1)
	counterLoops1 := stats1.CounterLoops
	counterWakeUps1 := stats1.CounterWakeUps
	d.statsMutex.Unlock()
	assert.Equal(t, uint64(1), counterLoops1)
	assert.True(t, counterWakeUps1 > 1)

	time.Sleep(time.Second)
	d.statsMutex.Lock()
	stats2, _ := d.stats[url]
	assert.NotNil(t, stats2)
	counterLoops2 := stats2.CounterLoops
	counterWakeUps2 := stats2.CounterWakeUps
	d.statsMutex.Unlock()
	assert.Equal(t, uint64(1), counterLoops2)
	assert.True(t, counterWakeUps2 > counterWakeUps1)

	d.wakeUpSignal <- 0
}

func TestEnqueueAfterHit(t *testing.T) {
	d := testInit()
	url := "enqueue-after-hit"

	d.enqueueNow(url)
	waitForDaemon(d)

	d.enqueueNow(url)
	waitForDaemon(d)

	stats := getStats(t, d, url)
	assert.Equal(t, uint64(2), stats.CounterEnqueues)
	assert.Equal(t, uint64(2), stats.CounterLoops)
}

func TestEnqueueDuringHit(t *testing.T) {
	hit := time.Second / 4
	d := testInit(
		runner.MockedHit{Duration: hit},
		runner.MockedHit{},
	)
	url := "enqueue-during-hit"

	d.enqueueNow(url)
	time.Sleep(time.Second + hit/2)

	d.enqueueNow(url)
	waitForDaemon(d)

	stats := getStats(t, d, url)
	assert.Equal(t, uint64(2), stats.CounterEnqueues)
	assert.Equal(t, uint64(2), stats.CounterLoops)
}

func TestEnqueueZeroThirtyZero(t *testing.T) {
	hit := time.Second / 2
	d := testInit(runner.MockedHit{Duration: hit})
	url := "enqueue-0-3-0"

	// this pattern mimics real world usage
	d.enqueueNow(url)
	time.Sleep(hit / 4)

	d.step1Enqueue(url, 3*time.Second)
	time.Sleep(hit / 2)

	d.enqueueNow(url)
	waitForDaemon(d)

	stats := getStats(t, d, url)
	assert.Equal(t, uint64(3), stats.CounterEnqueues)
	assert.Equal(t, uint64(1), stats.CounterLoops)
	assert.Equal(t, uint64(2), stats.CounterWakeUps)
}

func getStats(t *testing.T, d *daemon, url string) *Stats {
	d.statsMutex.Lock()
	stats, _ := d.stats[url]
	d.statsMutex.Unlock()

	assert.NotNil(t, stats)

	return stats
}

func testInit(hits ...runner.MockedHit) *daemon {
	runner := runner.NewMocked(hits)

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	d := &daemon{}
	d.init(runner, logger)

	d.coolDown = time.Duration(time.Second / 4)
	d.cutOff = time.Duration(3 * d.coolDown)
	d.defaultSchedule = 0

	return d
}

func waitForDaemon(d *daemon) {
	quit := false

	for {
		d.timerMutex.Lock()
		timerCounterSet := d.timerCounterSet
		timerCounterTrigger := d.timerCounterTrigger
		timerSet := d.timerSet
		timerRun := d.timerRun
		d.timerMutex.Unlock()

		d.wakeUpMutex.Lock()
		wakeUpCounterStart := d.wakeUpCounterStart
		wakeUpCounterFinish := d.wakeUpCounterFinish
		d.wakeUpMutex.Unlock()

		if timerCounterTrigger == timerCounterSet &&
			timerSet.Before(timerRun) &&
			wakeUpCounterFinish == wakeUpCounterStart {
			if !quit {
				quit = true
			} else {
				return
			}
		} else {
			quit = false
		}

		<-time.After(time.Second / 10)
	}
}
