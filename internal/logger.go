package internal // import "github.com/daohoangson/go-deferred/internal"

import (
	"os"

	"github.com/Sirupsen/logrus"
)

var _logger *logrus.Logger

// GetLogger prepares a default logger instance
func GetLogger() *logrus.Logger {
	if _logger == nil {
		_logger = logrus.New()

		levelValue := os.Getenv("DEFERRED_LOG_LEVEL")
		if len(levelValue) > 0 {
			if level, err := logrus.ParseLevel(levelValue); err == nil {
				_logger.SetLevel(level)
				_logger.WithField("level", level).Info("Updated logger level")
			}
		}
	}

	return _logger
}
