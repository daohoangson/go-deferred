package runner // import "github.com/daohoangson/go-deferred/pkg/runner"

import (
	"errors"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/daohoangson/go-deferred/internal"
)

type mockedRunner struct {
	hits      []MockedHit
	hitsMutex sync.Mutex

	maxHitsPerLoop uint64
}

// MockedHit represents a hit for mocked runner
type MockedHit struct {
	Duration     time.Duration
	Enqueue      int64
	Error        error
	HasEnqueue   bool
	MoreDeferred bool
}

// NewMocked returns a mocked Runner instance
func NewMocked(hits []MockedHit, maxHitsPerLoop uint64) Runner {
	m := &mockedRunner{}
	m.hits = hits
	m.maxHitsPerLoop = maxHitsPerLoop
	return m
}

func (m *mockedRunner) GetLogger() *logrus.Logger {
	return internal.GetLogger()
}

func (m *mockedRunner) GetMaxHitsPerLoop() uint64 {
	return m.maxHitsPerLoop
}

func (m *mockedRunner) Hit(url string) (Hit, error) {
	var mockedHit *MockedHit
	hit := Hit{}

	m.hitsMutex.Lock()
	if len(m.hits) > 0 {
		mockedHit, m.hits = &m.hits[0], m.hits[1:]
	}
	m.hitsMutex.Unlock()

	if mockedHit == nil {
		return hit, errors.New("No hit")
	}

	if mockedHit.Duration > 0 {
		time.Sleep(mockedHit.Duration)
	}

	if mockedHit.Error != nil {
		return hit, mockedHit.Error
	}

	hit.Data.MoreDeferred = mockedHit.MoreDeferred
	hit.Enqueue = mockedHit.Enqueue
	hit.HasEnqueue = mockedHit.HasEnqueue
	return hit, nil
}
