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

// NewPasswordService creates a new password service.
func NewPasswordService(enableLegacy bool) *PasswordService {
	return &PasswordService{
		enableLegacy: enableLegacy,
		logger:       NewLoggerV2("password-service"),
	}
}

// HashPassword hashes a password using bcrypt (modern approach).
func (s *PasswordService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	s.logger.Info("password hashed", map[string]interface{}{"algo": "bcrypt"})
	return string(hash), nil
}

// CheckPassword verifies a password against a hash (supports all hash types).
func (s *PasswordService) CheckPassword(password, hash string) (bool, bool) {
	hashType := DetectHashType(hash)

	var valid bool
	var needsMigration bool

	switch hashType {
	case HashTypeBcrypt:
		err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
		valid = err == nil
		needsMigration = false
	case HashTypeMD5:
		valid = CheckPasswordMD5(password, hash)
		needsMigration = true
	case HashTypeSHA1:
		valid = CheckPasswordSHA1(password, hash)
		needsMigration = true
	default:
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
	switch len(hash) {
	case 32:
		return HashTypeMD5
	case 40:
		return HashTypeSHA1
	default:
		if len(hash) == 60 && strings.HasPrefix(hash, "$2") {
			return HashTypeBcrypt
		}
		return "unknown"
	}
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
