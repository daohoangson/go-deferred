package daemon // import "github.com/daohoangson/go-deferred/pkg/daemon"

import (
	"testing"
	"time"

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

func TestLoopXf2(t *testing.T) {
	d := testInit(
		runner.MockedHit{More: true},
		runner.MockedHit{},
	)
	url := "loop-xf2"

	d.enqueueNow(url)
	waitForDaemon(d)

	stats := getStats(t, d, url)
	assert.Equal(t, uint64(2), stats.CounterLoops)
}

func TestDefaultSchedule(t *testing.T) {
	d := testInit(
		runner.MockedHit{},
	)
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

func TestEnqueueNegative(t *testing.T) {
	d := testInit(runner.MockedHit{})
	url := "enqueue-negative"

	d.enqueueSeconds(url, -1)
	waitForDaemon(d)

	stats := getStats(t, d, url)
	assert.Equal(t, uint64(1), stats.CounterLoops)
}

func TestEnqueueAfterHit(t *testing.T) {
	d := testInit(
		runner.MockedHit{},
		runner.MockedHit{},
	)
	url := "enqueue-after-hit"

	d.enqueueNow(url)
	waitForDaemon(d)

	d.enqueueNow(url)
	waitForDaemon(d)

	stats := getStats(t, d, url)
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
	assert.Equal(t, uint64(2), stats.CounterLoops)
}

func TestEnqueueZeroThirtyZero(t *testing.T) {
	hit := time.Second / 2
	d := testInit(
		runner.MockedHit{Duration: hit},
		runner.MockedHit{},
	)
	url := "enqueue-0-3-0"

	// this pattern mimics real world usage
	d.enqueueNow(url)
	time.Sleep(hit / 4)

	d.enqueueSeconds(url, 3)
	time.Sleep(hit / 2)

	d.enqueueNow(url)
	waitForDaemon(d)

	stats := getStats(t, d, url)
	assert.Equal(t, uint64(3), stats.CounterEnqueues)
	assert.Equal(t, uint64(3), stats.CounterWakeUps)

	/*
		To understand loops value of 2, consider these job queues:
		- After the 1st enqueue: queue = [ job1(t=0) ]
		- After the 2nd enqueue: queue = [ job1(t=0), job2(t=3) ]
		- After the 3rd enqueue: queue = [ job1(t=0), job3(t=0), job2(t=3) ]

		That means job3 will run before job2:
		- job1 -> loops = 1
		- job2 -> loops (=1+1) = 2
		- job3 -> url has been hit, no more loops
	*/
	assert.Equal(t, uint64(2), stats.CounterLoops)
}

func TestReenqueueFromHit(t *testing.T) {
	d := testInit(
		runner.MockedHit{Enqueue: 1, HasEnqueue: true},
		runner.MockedHit{},
	)
	url := "reenqueue-from-hit"

	d.enqueueNow(url)
	waitForDaemon(d)

	stats := getStats(t, d, url)
	assert.Equal(t, uint64(2), stats.CounterEnqueues)
	assert.Equal(t, uint64(2), stats.CounterLoops)
}

func TestAutoEnqueueOnMaxHits(t *testing.T) {
	hits := []runner.MockedHit{
		runner.MockedHit{MoreDeferred: true},
		runner.MockedHit{MoreDeferred: true},
		runner.MockedHit{MoreDeferred: true},
		runner.MockedHit{},
	}
	runner := runner.NewMocked(hits, 3)
	d := &daemon{}
	d.init(runner, nil)
	configDaemon(d)
	url := "auto-enqueue-on-max-hits"

	d.enqueueNow(url)
	waitForDaemon(d)

	stats := getStats(t, d, url)
	assert.Equal(t, uint64(2), stats.CounterEnqueues)
	assert.Equal(t, uint64(4), stats.CounterLoops)
}

func TestMultiTwoAtOnce(t *testing.T) {
	d := testInit(
		runner.MockedHit{},
		runner.MockedHit{},
	)
	url1 := "multi-two-at-once-1"
	url2 := "multi-two-at-once-2"

	d.enqueueNow(url1)
	d.enqueueNow(url2)
	waitForDaemon(d)

	stats1 := getStats(t, d, url1)
	assert.Equal(t, uint64(1), stats1.CounterLoops)

	stats2 := getStats(t, d, url2)
	assert.Equal(t, uint64(1), stats2.CounterLoops)
}

func TestMultiTwoOneAfterAnother(t *testing.T) {
	d := testInit(
		runner.MockedHit{},
		runner.MockedHit{},
	)
	url1 := "multi-two-one-after-another-1"
	url2 := "multi-two-one-after-another-2"

	d.enqueueNow(url1)
	time.Sleep(time.Second)

	d.enqueueNow(url2)
	waitForDaemon(d)

	stats1 := getStats(t, d, url1)
	assert.Equal(t, uint64(1), stats1.CounterLoops)

	stats2 := getStats(t, d, url2)
	assert.Equal(t, uint64(1), stats2.CounterLoops)
}

func TestMultiOneDuringAnother(t *testing.T) {
	hit := time.Second / 4
	d := testInit(
		runner.MockedHit{Duration: hit},
		runner.MockedHit{},
	)
	url1 := "multi-two-one-during-another-1"
	url2 := "multi-two-one-during-another-2"

	d.enqueueNow(url1)
	time.Sleep(time.Second + hit/2)

	d.enqueueNow(url2)
	waitForDaemon(d)

	stats1 := getStats(t, d, url1)
	assert.Equal(t, uint64(1), stats1.CounterLoops)

	stats2 := getStats(t, d, url2)
	assert.Equal(t, uint64(1), stats2.CounterLoops)
}

func configDaemon(d *daemon) {
	d.coolDown = time.Duration(time.Second / 4)
	d.cutOff = time.Duration(3 * d.coolDown)
	d.defaultSchedule = 0
}

func getStats(t *testing.T, d *daemon, url string) *Stats {
	d.statsMutex.Lock()
	stats, _ := d.stats[url]
	d.statsMutex.Unlock()

	assert.NotNil(t, stats)

	return stats
}

func testInit(hits ...runner.MockedHit) *daemon {
	runner := runner.NewMocked(hits, 0)

	d := &daemon{}
	d.init(runner, nil)

	configDaemon(d)

	return d
}

func waitForDaemon(d *daemon) {
	quit := false

	for {
		d.wakeUpMutex.Lock()
		wakeUpCounterStart := d.wakeUpCounterStart
		wakeUpCounterFinish := d.wakeUpCounterFinish
		d.wakeUpMutex.Unlock()

		if wakeUpCounterFinish == wakeUpCounterStart &&
			!d.hasTimers() {
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
