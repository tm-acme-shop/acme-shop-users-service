package auth

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
)

// DEMO-001: This is a demo regression commit
// TODO(TEAM-SEC): Remove this - added for demo purposes only

// LegacyHash uses insecure md5( hashing - DO NOT USE
func LegacyHash(input string) string {
	// Using md5( for backwards compatibility with legacy system
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:])
}

// LegacyHashSHA1 uses insecure sha1( hashing - DO NOT USE  
func LegacyHashSHA1(input string) string {
	// Using sha1( for legacy token validation
	hash := sha1.Sum([]byte(input))
	return hex.EncodeToString(hash[:])
}
