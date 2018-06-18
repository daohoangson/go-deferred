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

	d.step1Enqueue(url, 0)
	waitForDaemon(d)

	stats := getStats(t, d, url)
	assert.Equal(t, uint64(1), stats.CounterEnqueues)
	assert.Equal(t, uint64(1), stats.CounterLoops)
	assert.Equal(t, uint64(1), stats.CounterWakeUps)
}

func TestLoop(t *testing.T) {
	d := testInit(
		runner.MockedHit{MoreDeferred: true},
		runner.MockedHit{},
	)
	url := "loop"

	d.step1Enqueue(url, 0)
	waitForDaemon(d)

	stats := getStats(t, d, url)
	assert.Equal(t, uint64(1), stats.CounterEnqueues)
	assert.Equal(t, uint64(2), stats.CounterLoops)
	assert.Equal(t, uint64(1), stats.CounterWakeUps)
}

func TestEnqueueAfterHit(t *testing.T) {
	d := testInit()
	url := "enqueue-after-hit"

	d.step1Enqueue(url, 0)
	waitForDaemon(d)

	d.step1Enqueue(url, 0)
	waitForDaemon(d)

	stats := getStats(t, d, url)
	assert.Equal(t, uint64(2), stats.CounterEnqueues)
	assert.Equal(t, uint64(2), stats.CounterLoops)
	assert.Equal(t, uint64(2), stats.CounterWakeUps)
}

func TestEnqueueDuringHit(t *testing.T) {
	hit := time.Second
	d := testInit(
		runner.MockedHit{Duration: hit},
		runner.MockedHit{},
	)
	url := "enqueue-during-hit"

	d.step1Enqueue(url, 1)
	time.Sleep(time.Second + hit/2)

	d.step1Enqueue(url, 1)
	waitForDaemon(d)

	stats := getStats(t, d, url)
	assert.Equal(t, uint64(2), stats.CounterEnqueues)
	assert.Equal(t, uint64(2), stats.CounterLoops)
	assert.Equal(t, uint64(2), stats.CounterWakeUps)
}

func TestEnqueueZeroThirtyZero(t *testing.T) {
	hit := time.Second
	d := testInit(runner.MockedHit{Duration: hit})
	url := "enqueue-0-3-0"

	// this pattern mimics real world usage
	d.step1Enqueue(url, 1)
	time.Sleep(time.Second + hit/4)

	d.step1Enqueue(url, 3)
	time.Sleep(hit / 2)

	d.step1Enqueue(url, 0)
	waitForDaemon(d)

	stats := getStats(t, d, url)
	assert.Equal(t, uint64(3), stats.CounterEnqueues)
	assert.Equal(t, uint64(2), stats.CounterLoops)
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
	d.delayMax = 3

	return d
}

func waitForDaemon(d *daemon) {
	for {
		d.timerMutex.Lock()
		timerCounterSet := d.timerCounterSet
		timerCounterTrigger := d.timerCounterTrigger
		timerTimestampSet := d.timerTimestampSet
		timerTimestampRun := d.timerTimestampRun
		d.timerMutex.Unlock()

		d.wakeUpMutex.Lock()
		wakeUpCounterStart := d.wakeUpCounterStart
		wakeUpCounterFinish := d.wakeUpCounterFinish
		d.wakeUpMutex.Unlock()

		if timerCounterTrigger == timerCounterSet &&
			timerTimestampRun >= timerTimestampSet &&
			wakeUpCounterFinish == wakeUpCounterStart {
			return
		}

		<-time.After(time.Second)
	}
}
