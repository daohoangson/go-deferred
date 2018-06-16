package internal // import "github.com/daohoangson/go-deferred/internal"

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
)

// Base64Decode decodes a base64-encoded string
func Base64Decode(data string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(data)
}

// GetMD5 returns the md5 hash of data + secret
func GetMD5(data string, secret string) string {
	hasher := md5.New()
	hasher.Write([]byte(data + secret))
	return hex.EncodeToString(hasher.Sum(nil))
}
