package daemon // import "github.com/daohoangson/go-deferred/pkg/daemon"

// Daemon represents a server that can hit deferred.php targets
type Daemon interface {
	ListenAndServe(uint64) error
	SetSecret(string)
}

// Stats represents metrics for an URL
type Stats struct {
	CounterEnqueues uint64 `json:"counter_enqueues"`
	CounterErrors   uint64 `json:"counter_errors"`
	CounterLoops    uint64 `json:"counter_loops"`
	CounterOnTimers uint64 `json:"counter_on_timers"`
	LatestTimestamp int64  `json:"latest_timestamp"`
}
