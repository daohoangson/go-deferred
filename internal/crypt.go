package internal // import "github.com/daohoangson/go-deferred/internal"

import (
	"crypto/md5"
	"encoding/hex"
)

// GetMD5 returns the md5 hash of data + secret
func GetMD5(data string, secret string) string {
	hasher := md5.New()
	hasher.Write([]byte(data + secret))
	return hex.EncodeToString(hasher.Sum(nil))
}
