package internal // import "github.com/daohoangson/go-deferred/internal"

// GetProtocolVersion returns the version string for go-deferred protocol
func GetProtocolVersion() string {
	return "2018061901"
}

// GetProtocolVersionHeaderKey returns the header key for go-deferred protocol version
func GetProtocolVersionHeaderKey() string {
	return "X-Go-Deferred-Version"
}

// GetProtocolEnqueueHeaderKey returns the header key for go-deferred protocol enqueue
func GetProtocolEnqueueHeaderKey() string {
	return "X-Go-Deferred-Enqueue"
}

// Ternary returns trueValue if condition is true and falseValue otherwise
// https://stackoverflow.com/questions/19979178/what-is-the-idiomatic-go-equivalent-of-cs-ternary-operator
func Ternary(condition bool, trueValue interface{}, falseValue interface{}) interface{} {
	if condition {
		return trueValue
	} else {
		return falseValue
	}
}
