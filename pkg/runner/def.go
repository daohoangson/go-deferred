package runner // import "github.com/daohoangson/go-deferred/pkg/runner"

import (
	"time"

	"github.com/Sirupsen/logrus"
)

// Data represents response from hit target
type Data struct {
	Message      string
	MoreDeferred bool

	// XenForo 2 job.php
	More bool
}

// Hit represents a successful hit
type Hit struct {
	Data        Data
	Enqueue     int64
	HasEnqueue  bool
	TimeStart   time.Time
	TimeElapsed time.Duration
}

// Hits represents a series of hits (a loop)
type Hits struct {
	List        []Hit
	TimeStart   time.Time
	TimeElapsed time.Duration
}

// Runner represents an object that can hit deferred.php targets
type Runner interface {
	GetCooldownDuration() time.Duration
	GetDumpResponseOnParseError() bool
	GetErrorsBeforeQuitting() uint64
	GetLogger() *logrus.Logger
	GetMaxHitsPerLoop() uint64
	Hit(url string) (Hit, error)
}
