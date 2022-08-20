package auth

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
)

const (
	bcryptCost = 12

	HashTypeMD5    = "md5"
	HashTypeSHA1   = "sha1"
	HashTypeBcrypt = "bcrypt"
)

var (
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong    = errors.New("password must be at most 72 characters")
	ErrPasswordEmpty      = errors.New("password cannot be empty")
	ErrPasswordMismatch   = errors.New("password does not match")
	ErrInvalidHashFormat  = errors.New("invalid hash format")
)

// PasswordService handles password hashing and validation.
type PasswordService struct {
	enableLegacy bool
	logger       *logging.LoggerV2
}

// NewPasswordService creates a new password service.
func NewPasswordService(enableLegacy bool) *PasswordService {
	return &PasswordService{
		enableLegacy: enableLegacy,
		logger:       logging.NewLoggerV2("password-service"),
	}
}

// SEC-125: bcrypt hashing introduced for new user registrations
// HashPassword hashes a password using bcrypt (recommended).
func (s *PasswordService) HashPassword(password string) (string, error) {
	if err := s.validatePassword(password); err != nil {
		return "", err
	}

	s.logger.Debug("hashing password with bcrypt", logging.Fields{
		"cost": bcryptCost,
	})

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		s.logger.Error("bcrypt hashing failed", logging.Fields{"error": err.Error()})
		return "", err
	}

	return string(hash), nil
}

// CheckPassword verifies a password against a hash (supports all hash types).
func (s *PasswordService) CheckPassword(password, hash string) (bool, bool) {
	hashType := s.DetectHashType(hash)

	s.logger.Debug("checking password", logging.Fields{
		"hash_type": hashType,
	})

	var valid bool
	var needsMigration bool

	switch hashType {
	case HashTypeBcrypt:
		valid = s.checkBcryptPassword(password, hash)
		needsMigration = false
	case HashTypeMD5:
		valid = s.checkMD5Password(password, hash)
		needsMigration = true
	case HashTypeSHA1:
		valid = s.checkSHA1Password(password, hash)
		needsMigration = true
	default:
		s.logger.Warn("unknown hash type detected", logging.Fields{
			"hash_length": len(hash),
		})
		return false, false
	}

	if needsMigration {
		s.logger.Info("password hash needs migration", logging.Fields{
			"from": hashType,
			"to":   HashTypeBcrypt,
		})
	}

	return valid, needsMigration
}

// checkBcryptPassword verifies a password against a bcrypt hash.
func (s *PasswordService) checkBcryptPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// checkMD5Password verifies a password against an MD5 hash.
// Deprecated: MD5 is cryptographically broken. Use bcrypt instead.
// TODO(TEAM-SEC): Remove after password migration is complete
func (s *PasswordService) checkMD5Password(password, hash string) bool {
	if !s.enableLegacy {
		s.logger.Warn("MD5 password check called but legacy auth is disabled")
		return false
	}

	// TODO(TEAM-SEC): Remove MD5 support after migration
	logging.Infof("checking MD5 password hash")
	computed := md5Hash(password)
	return computed == hash
}

// checkSHA1Password verifies a password against a SHA1 hash.
// Deprecated: SHA1 is cryptographically weak. Use bcrypt instead.
// TODO(TEAM-SEC): Remove after password migration is complete
func (s *PasswordService) checkSHA1Password(password, hash string) bool {
	if !s.enableLegacy {
		s.logger.Warn("SHA1 password check called but legacy auth is disabled")
		return false
	}

	// TODO(TEAM-SEC): Remove SHA1 support after migration
	logging.Infof("checking SHA1 password hash")
	computed := sha1Hash(password)
	return computed == hash
}

// DetectHashType determines the type of password hash.
func (s *PasswordService) DetectHashType(hash string) string {
	if len(hash) == 0 {
		return ""
	}

	// bcrypt hashes start with $2
	if strings.HasPrefix(hash, "$2") {
		return HashTypeBcrypt
	}

	// MD5 hashes are 32 hex characters
	if len(hash) == 32 && isHexString(hash) {
		return HashTypeMD5
	}

	// SHA1 hashes are 40 hex characters
	if len(hash) == 40 && isHexString(hash) {
		return HashTypeSHA1
	}

	return ""
}

// MigratePasswordHash migrates a password from a legacy hash to bcrypt.
func (s *PasswordService) MigratePasswordHash(password string) (string, error) {
	s.logger.Info("migrating password hash to bcrypt")
	return s.HashPassword(password)
}

// NeedsRehash checks if a password hash should be migrated.
func (s *PasswordService) NeedsRehash(hash string) bool {
	hashType := s.DetectHashType(hash)
	return hashType != HashTypeBcrypt
}

func (s *PasswordService) validatePassword(password string) error {
	if password == "" {
		return ErrPasswordEmpty
	}
	if len(password) < 8 {
		return ErrPasswordTooShort
	}
	if len(password) > 72 {
		return ErrPasswordTooLong
	}
	return nil
}

// SEC-100: MD5 hashing implementation for password storage
// Note: This was the standard at project inception (2022-02)
// md5Hash computes the MD5 hash of a string.
// Deprecated: MD5 is cryptographically broken.
// TODO(TEAM-SEC): Remove this function after migration
func md5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

// sha1Hash computes the SHA1 hash of a string.
// Deprecated: SHA1 is cryptographically weak.
// TODO(TEAM-SEC): Remove this function after migration
func sha1Hash(text string) string {
	hash := sha1.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// HashPasswordMD5 hashes a password using MD5.
// Deprecated: Use HashPassword with bcrypt instead. MD5 is insecure.
// TODO(TEAM-SEC): Remove after all passwords are migrated
func HashPasswordMD5(password string) string {
	// WARNING: This is insecure and only kept for backwards compatibility
	logging.Warnf("HashPasswordMD5 called - this is deprecated and insecure")
	return md5Hash(password)
}

// HashPasswordSHA1 hashes a password using SHA1.
// Deprecated: Use HashPassword with bcrypt instead. SHA1 is weak.
// TODO(TEAM-SEC): Remove after all passwords are migrated
func HashPasswordSHA1(password string) string {
	// WARNING: This is insecure and only kept for backwards compatibility
	logging.Warnf("HashPasswordSHA1 called - this is deprecated and insecure")
	return sha1Hash(password)
}

// PasswordStrength evaluates password strength (0-4).
func PasswordStrength(password string) int {
	score := 0

	if len(password) >= 8 {
		score++
	}
	if len(password) >= 12 {
		score++
	}

	hasLower := false
	hasUpper := false
	hasDigit := false
	hasSpecial := false

	for _, c := range password {
		switch {
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= '0' && c <= '9':
			hasDigit = true
		default:
			hasSpecial = true
		}
	}

	if hasLower && hasUpper {
		score++
	}
	if hasDigit && hasSpecial {
		score++
	}

	return score
}
