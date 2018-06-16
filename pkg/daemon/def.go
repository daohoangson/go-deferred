package daemon // import "github.com/daohoangson/go-deferred/pkg/daemon"

// Daemon represents a server that can hit deferred.php targets
type Daemon interface {
	ListenAndServe(uint64) error
	SetSecret(string)
}

// Stats represents metrics for an URL
type Stats struct {
	CounterErrors   uint64 `json:"counter_errors"`
	CounterHits     uint64 `json:"counter_hits"`
	CounterLoops    uint64 `json:"counter_loops"`
	CounterEnqueues uint64 `json:"counter_enqueues"`
	LatestTimestamp int64  `json:"latest_timestamp"`
}
