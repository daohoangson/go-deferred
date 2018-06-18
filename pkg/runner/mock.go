package runner // import "github.com/daohoangson/go-deferred/pkg/runner"

import (
	"errors"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/daohoangson/go-deferred/internal"
)

type mockedRunner struct {
	hits []MockedHit
}

// MockedHit represents a hit for mocked runner
type MockedHit struct {
	Duration     time.Duration
	Error        error
	MoreDeferred bool
}

// NewMocked returns a mocked Runner instance
func NewMocked(hits []MockedHit) Runner {
	m := &mockedRunner{}
	m.hits = hits
	return m
}

func (m *mockedRunner) GetLogger() *logrus.Logger {
	return internal.GetLogger()
}

func (m *mockedRunner) Hit(url string) (*Hit, error) {
	if len(m.hits) == 0 {
		return nil, errors.New("No hit")
	}

	var mHit MockedHit
	mHit, m.hits = m.hits[0], m.hits[1:]

	if mHit.Duration > 0 {
		time.Sleep(mHit.Duration)
	}

	if mHit.Error != nil {
		return nil, mHit.Error
	}

	hit := new(Hit)
	hit.Data = new(Data)
	hit.Data.MoreDeferred = mHit.MoreDeferred
	return hit, nil
}
