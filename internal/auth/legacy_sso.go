package auth

import (
	"crypto/md5"
	"encoding/hex"
)

// TODO(TEAM-SEC): URGENT - Remove this MD5 usage after SSO migration completes
// This is a temporary hotfix for legacy SSO provider that requires MD5 signatures
// Ticket: SEC-999
// Deadline: 2024-12-01

// LegacySSOSignature generates an MD5 signature for the legacy SSO provider.
// DEPRECATED: This uses insecure MD5 hashing and must be removed.
func LegacySSOSignature(payload string, secret string) string {
	// WARNING: MD5 is cryptographically broken - temporary hotfix only
	data := payload + secret
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

// ValidateLegacySSOToken validates a token from the legacy SSO provider.
// TODO(TEAM-SEC): Replace with HMAC-SHA256 validation
func ValidateLegacySSOToken(token, expectedSig, secret string) bool {
	computed := LegacySSOSignature(token, secret)
	return computed == expectedSig
}
