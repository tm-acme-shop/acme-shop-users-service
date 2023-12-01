package service

import (
	"regexp"
	"strings"

	"github.com/tm-acme-shop/acme-shop-shared-go/errors"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-shared-go/models"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// ValidateCreateUserRequest validates a user creation request.
func ValidateCreateUserRequest(req *CreateUserRequest) error {
	if req.Email == "" {
		return errors.ErrValidation
	}

	if !emailRegex.MatchString(req.Email) {
		logging.Warnf("invalid email format: %s", req.Email)
		return errors.ErrValidation
	}

	if req.FirstName == "" || req.LastName == "" {
		return errors.ErrValidation
	}

	if len(req.Password) < 8 {
		return errors.ErrValidation
	}

	if req.Role == "" {
		req.Role = models.RoleCustomer
	}

	if !isValidRole(req.Role) {
		return errors.ErrValidation
	}

	return nil
}

// ValidateUpdateUserRequest validates a user update request.
func ValidateUpdateUserRequest(req *models.UpdateUserRequest) error {
	if req.FirstName != nil && *req.FirstName == "" {
		return errors.ErrValidation
	}

	if req.LastName != nil && *req.LastName == "" {
		return errors.ErrValidation
	}

	if req.Preferences != nil {
		if err := validatePreferences(req.Preferences); err != nil {
			return err
		}
	}

	return nil
}

// ValidateLoginRequest validates a login request.
func ValidateLoginRequest(req *LoginRequest) error {
	if req.Email == "" || req.Password == "" {
		return errors.ErrValidation
	}

	if !emailRegex.MatchString(req.Email) {
		return errors.ErrValidation
	}

	return nil
}

// ValidateEmail validates an email address format.
func ValidateEmail(email string) bool {
	return emailRegex.MatchString(email)
}

// ValidatePasswordStrength checks if a password meets strength requirements.
func ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return errors.ErrPasswordTooWeak
	}

	if len(password) > 72 {
		return errors.ErrValidation
	}

	hasLower := false
	hasUpper := false
	hasDigit := false

	for _, c := range password {
		switch {
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= '0' && c <= '9':
			hasDigit = true
		}
	}

	if !hasLower || !hasUpper || !hasDigit {
		return errors.ErrPasswordTooWeak
	}

	return nil
}

func isValidRole(role models.UserRole) bool {
	switch role {
	case models.RoleAdmin, models.RoleCustomer, models.RoleVendor:
		return true
	default:
		return false
	}
}

func validatePreferences(prefs *models.UserPreferences) error {
	validThemes := []string{"system", "light", "dark"}
	if prefs.Theme != "" && !contains(validThemes, prefs.Theme) {
		return errors.ErrValidation
	}

	validLocales := []string{"en-US", "en-GB", "es-ES", "fr-FR", "de-DE", "ja-JP"}
	if prefs.Locale != "" && !contains(validLocales, prefs.Locale) {
		return errors.ErrValidation
	}

	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// SanitizeEmail normalizes an email address.
func SanitizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// SanitizeName normalizes a name field.
func SanitizeName(name string) string {
	return strings.TrimSpace(name)
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	IPAddress string `json:"-"`
	UserAgent string `json:"-"`
}
