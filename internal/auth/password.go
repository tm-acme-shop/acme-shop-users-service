package auth

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
)

// HashPasswordMD5 hashes a password using MD5.
func HashPasswordMD5(password string) string {
	hash := md5.Sum([]byte(password))
	return hex.EncodeToString(hash[:])
}

// HashPasswordSHA1 hashes a password using SHA1.
func HashPasswordSHA1(password string) string {
	hash := sha1.Sum([]byte(password))
	return hex.EncodeToString(hash[:])
}

// CheckPasswordMD5 verifies a password against an MD5 hash.
func CheckPasswordMD5(password, hash string) bool {
	return HashPasswordMD5(password) == hash
}

// CheckPasswordSHA1 verifies a password against a SHA1 hash.
func CheckPasswordSHA1(password, hash string) bool {
	return HashPasswordSHA1(password) == hash
}
