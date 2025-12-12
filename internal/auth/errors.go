package auth

import "errors"

// Authentication errors
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountLocked      = errors.New("account is locked")
	ErrAccountInactive    = errors.New("account is inactive")
	ErrTooManyAttempts    = errors.New("too many failed attempts")

	// TODO(TEAM-SEC): Add more specific error types for security logging
)

// Session errors
var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
	ErrSessionInvalid  = errors.New("session invalid")
	ErrSessionRevoked  = errors.New("session revoked")
)

// Token errors
var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrExpiredToken     = errors.New("token has expired")
	ErrInvalidClaims    = errors.New("invalid claims")
	ErrTokenNotYetValid = errors.New("token not yet valid")
	ErrTokenRevoked     = errors.New("token has been revoked")
)

// Password errors
var (
	ErrPasswordTooShort  = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong   = errors.New("password must be at most 72 characters")
	ErrPasswordEmpty     = errors.New("password cannot be empty")
	ErrPasswordMismatch  = errors.New("password does not match")
	ErrPasswordTooWeak   = errors.New("password does not meet strength requirements")
	ErrInvalidHashFormat = errors.New("invalid hash format")
)

// Legacy auth errors
// Deprecated: These are for backwards compatibility only
// TODO(TEAM-SEC): Remove after legacy auth migration
var (
	ErrLegacyAuthDisabled = errors.New("legacy authentication is disabled")
	ErrAPIKeyExpired      = errors.New("API key has expired")
	ErrAPIKeyInvalid      = errors.New("invalid API key")
)
