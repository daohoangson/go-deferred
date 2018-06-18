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
	delay := d.delayMin

	d.step1Enqueue(url, delay)
	sleep(delay)
	assertFinishedRunning(t, d)

	stats := getStats(t, d, url)
	assert.Equal(t, uint64(1), stats.CounterEnqueues)
	assert.Equal(t, uint64(1), stats.CounterLoops)
	assert.Equal(t, uint64(1), stats.CounterOnTimers)
}

func TestLoop(t *testing.T) {
	d := testInit(
		runner.MockedHit{MoreDeferred: true},
		runner.MockedHit{},
	)
	url := "loop"
	delay := d.delayMin

	d.step1Enqueue(url, delay)
	sleep(delay)
	assertFinishedRunning(t, d)

	stats := getStats(t, d, url)
	assert.Equal(t, uint64(1), stats.CounterEnqueues)
	assert.Equal(t, uint64(2), stats.CounterLoops)
	assert.Equal(t, uint64(1), stats.CounterOnTimers)
}

func TestEnqueueAfterHit(t *testing.T) {
	d := testInit(
		runner.MockedHit{},
		runner.MockedHit{},
	)
	url := "enqueue-after-hit"

	d.step1Enqueue(url, 1)
	time.Sleep(time.Second * 2)

	d.step1Enqueue(url, 1)
	time.Sleep(time.Second * 2)

	assertFinishedRunning(t, d)

	stats := getStats(t, d, url)
	assert.Equal(t, uint64(2), stats.CounterEnqueues)
	assert.Equal(t, uint64(2), stats.CounterLoops)
	assert.Equal(t, uint64(2), stats.CounterOnTimers)
}

func TestEnqueueDuringHit(t *testing.T) {
	d := testInit(
		runner.MockedHit{Duration: time.Second * 2},
		runner.MockedHit{},
	)
	url := "enqueue-during-hit"

	d.step1Enqueue(url, 1)
	time.Sleep(time.Second * 2)

	d.step1Enqueue(url, 2)
	time.Sleep(time.Second * 3)

	assertFinishedRunning(t, d)

	stats := getStats(t, d, url)
	assert.Equal(t, uint64(2), stats.CounterEnqueues)
	assert.Equal(t, uint64(2), stats.CounterLoops)
	assert.Equal(t, uint64(2), stats.CounterOnTimers)
}

func TestOnlyEnqueuingCancelPreviousTimer(t *testing.T) {
	d := testInit(
		runner.MockedHit{},
		runner.MockedHit{},
	)
	url := "enqueuing-cancel-previous-timer"

	d.step1Enqueue(url, 3)
	time.Sleep(time.Second)

	d.step1Enqueue(url, 1)
	time.Sleep(time.Second * 2)

	assertFinishedRunning(t, d)

	d.timerMutex.Lock()
	timerCounterSet := d.timerCounterSet
	timerCounterRun := d.timerCounterRun
	d.timerMutex.Unlock()
	assert.Equal(t, uint64(2), timerCounterSet)
	assert.Equal(t, uint64(1), timerCounterRun)

	stats := getStats(t, d, url)
	assert.Equal(t, uint64(2), stats.CounterEnqueues)
	assert.Equal(t, uint64(1), stats.CounterLoops)
	assert.Equal(t, uint64(1), stats.CounterOnTimers)
}

func assertFinishedRunning(t *testing.T, d *daemon) {
	d.timerMutex.Lock()
	timer := d.timer
	d.timerMutex.Unlock()

	assert.Nil(t, timer)
}

func getStats(t *testing.T, d *daemon, url string) *Stats {
	d.statsMutex.Lock()
	stats, _ := d.stats[url]
	d.statsMutex.Unlock()

	assert.NotNil(t, stats)

	return stats
}

func sleep(seconds int64) {
	time.Sleep(time.Second*time.Duration(seconds) + time.Millisecond*10)
}

func testInit(hits ...runner.MockedHit) *daemon {
	runner := runner.NewMocked(hits)

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	d := &daemon{}
	d.init(runner, logger)

	return d
}
