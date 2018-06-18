package runner // import "github.com/daohoangson/go-deferred/pkg/runner"

import (
	"time"

	"github.com/Sirupsen/logrus"
)

// Data represents response from hit target
type Data struct {
	Message      string
	MoreDeferred bool
}

// Hit represents a successful hit
type Hit struct {
	Data        *Data
	TimeStart   time.Time
	TimeElapsed time.Duration
}

// Runner represents an object that can hit deferred.php targets
type Runner interface {
	GetLogger() *logrus.Logger
	Hit(url string) (*Hit, error)
}
