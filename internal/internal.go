package internal // import "github.com/daohoangson/go-deferred/internal"

// Ternary returns trueValue if condition is true and falseValue otherwise
// https://stackoverflow.com/questions/19979178/what-is-the-idiomatic-go-equivalent-of-cs-ternary-operator
func Ternary(condition bool, trueValue interface{}, falseValue interface{}) interface{} {
	if condition {
		return trueValue
	} else {
		return falseValue
	}
}
