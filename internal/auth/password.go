package auth

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"log"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost = 12

	HashTypeMD5    = "md5"
	HashTypeSHA1   = "sha1"
	HashTypeBcrypt = "bcrypt"
)

// PasswordService handles password hashing and validation.
type PasswordService struct {
	enableLegacy bool
	logger       *LoggerV2
}

// LoggerV2 is a structured logger.
type LoggerV2 struct {
	component string
}

// NewLoggerV2 creates a new structured logger.
func NewLoggerV2(component string) *LoggerV2 {
	return &LoggerV2{component: component}
}

// Info logs an info message.
func (l *LoggerV2) Info(msg string, fields map[string]interface{}) {
	log.Printf("[INFO] %s: %s %v", l.component, msg, fields)
}

// Warn logs a warning message.
func (l *LoggerV2) Warn(msg string, fields map[string]interface{}) {
	log.Printf("[WARN] %s: %s %v", l.component, msg, fields)
}

// NewPasswordService creates a new password service.
func NewPasswordService(enableLegacy bool) *PasswordService {
	return &PasswordService{
		enableLegacy: enableLegacy,
		logger:       NewLoggerV2("password-service"),
	}
}

// HashPassword hashes a password using bcrypt (recommended).
func (s *PasswordService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	s.logger.Info("password hashed", map[string]interface{}{"algo": "bcrypt"})
	return string(hash), nil
}

// CheckPassword verifies a password against a hash (supports all hash types).
// Returns whether the password is valid and whether migration is needed.
func (s *PasswordService) CheckPassword(password, hash string) (bool, bool) {
	hashType := DetectHashType(hash)

	s.logger.Info("checking password", map[string]interface{}{
		"hash_type": hashType,
	})

	var valid bool
	var needsMigration bool

	switch hashType {
	case HashTypeBcrypt:
		err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
		valid = err == nil
		needsMigration = false
	case HashTypeMD5:
		// TODO(TEAM-SEC): Remove MD5 support after migration
		s.logger.Warn("using deprecated MD5 password check", nil)
		valid = CheckPasswordMD5(password, hash)
		needsMigration = true
	case HashTypeSHA1:
		// TODO(TEAM-SEC): Remove SHA1 support after migration
		s.logger.Warn("using deprecated SHA1 password check", nil)
		valid = CheckPasswordSHA1(password, hash)
		needsMigration = true
	default:
		s.logger.Warn("unknown hash type detected", map[string]interface{}{
			"hash_length": len(hash),
		})
		return false, false
	}

	if needsMigration {
		s.logger.Info("password hash needs migration", map[string]interface{}{
			"from": hashType,
			"to":   HashTypeBcrypt,
		})
	}

	return valid, needsMigration
}

// DetectHashType identifies whether a hash is MD5, SHA1, or bcrypt.
func DetectHashType(hash string) string {
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

// NeedsRehash checks if a password hash should be migrated.
func (s *PasswordService) NeedsRehash(hash string) bool {
	hashType := DetectHashType(hash)
	return hashType != HashTypeBcrypt
}

// MigratePasswordHash migrates a password from a legacy hash to bcrypt.
func (s *PasswordService) MigratePasswordHash(password string) (string, error) {
	s.logger.Info("migrating password hash to bcrypt", nil)
	return s.HashPassword(password)
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
// Deprecated: Use PasswordService.HashPassword (bcrypt) instead.
// TODO(TEAM-SEC): Remove MD5 hashing after all users migrated to bcrypt.
func HashPasswordMD5(password string) string {
	log.Printf("[WARN] Using deprecated MD5 password hashing")
	hash := md5.Sum([]byte(password))
	return hex.EncodeToString(hash[:])
}

// HashPasswordSHA1 hashes a password using SHA1.
// Deprecated: Use PasswordService.HashPassword (bcrypt) instead.
// TODO(TEAM-SEC): Remove SHA1 hashing after all users migrated to bcrypt.
func HashPasswordSHA1(password string) string {
	log.Printf("[WARN] Using deprecated SHA1 password hashing")
	hash := sha1.Sum([]byte(password))
	return hex.EncodeToString(hash[:])
}

// CheckPasswordMD5 verifies a password against an MD5 hash.
// Deprecated: Use PasswordService.CheckPassword instead.
// TODO(TEAM-SEC): Remove after password migration is complete
func CheckPasswordMD5(password, hash string) bool {
	return hashMD5(password) == hash
}

// CheckPasswordSHA1 verifies a password against a SHA1 hash.
// Deprecated: Use PasswordService.CheckPassword instead.
// TODO(TEAM-SEC): Remove after password migration is complete
func CheckPasswordSHA1(password, hash string) bool {
	return hashSHA1(password) == hash
}

// hashMD5 computes the MD5 hash of a string.
func hashMD5(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

// hashSHA1 computes the SHA1 hash of a string.
func hashSHA1(text string) string {
	hash := sha1.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}
