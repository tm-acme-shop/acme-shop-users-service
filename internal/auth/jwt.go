package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-shared-go/models"
)

var (
	ErrInvalidToken    = errors.New("invalid token")
	ErrExpiredToken    = errors.New("token has expired")
	ErrInvalidClaims   = errors.New("invalid claims")
	ErrTokenNotYetValid = errors.New("token not yet valid")
)

// JWTClaims represents the claims in a JWT token.
type JWTClaims struct {
	jwt.RegisteredClaims
	UserID    string          `json:"user_id"`
	Email     string          `json:"email"`
	Role      models.UserRole `json:"role"`
	SessionID string          `json:"session_id,omitempty"`
}

// JWTClaimsV1 represents the legacy JWT claims format.
// Deprecated: Use JWTClaims instead.
type JWTClaimsV1 struct {
	jwt.RegisteredClaims
	UserID string `json:"uid"`
	Email  string `json:"email"`
	// TODO(TEAM-SEC): Remove legacy claims after migration
}

// JWTService handles JWT token generation and validation.
type JWTService struct {
	secret     []byte
	expiration time.Duration
	issuer     string
	logger     *logging.LoggerV2
}

// NewJWTService creates a new JWT service.
func NewJWTService(secret string, expiration time.Duration) *JWTService {
	return &JWTService{
		secret:     []byte(secret),
		expiration: expiration,
		issuer:     "acme-users-service",
		logger:     logging.NewLoggerV2("jwt-service"),
	}
}

// GenerateToken generates a new JWT token for a user.
func (s *JWTService) GenerateToken(user *models.User, sessionID string) (string, error) {
	s.logger.Debug("generating JWT token", logging.Fields{
		"user_id":    user.ID,
		"session_id": sessionID,
	})

	now := time.Now()
	claims := &JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expiration)),
			NotBefore: jwt.NewNumericDate(now),
		},
		UserID:    user.ID,
		Email:     user.Email,
		Role:      user.Role,
		SessionID: sessionID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.secret)
	if err != nil {
		s.logger.Error("failed to sign JWT token", logging.Fields{
			"error": err.Error(),
		})
		return "", err
	}

	s.logger.Info("JWT token generated", logging.Fields{
		"user_id":    user.ID,
		"expires_at": claims.ExpiresAt.Time,
	})

	return signedToken, nil
}

// ValidateToken validates a JWT token and returns the claims.
func (s *JWTService) ValidateToken(tokenString string) (*JWTClaims, error) {
	s.logger.Debug("validating JWT token")

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		s.logger.Warn("token validation failed", logging.Fields{
			"error": err.Error(),
		})
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidClaims
	}

	s.logger.Debug("JWT token validated", logging.Fields{
		"user_id": claims.UserID,
	})

	return claims, nil
}

// GenerateTokenV1 generates a legacy JWT token.
// Deprecated: Use GenerateToken instead.
// TODO(TEAM-API): Remove after v1 API deprecation
func (s *JWTService) GenerateTokenV1(userID, email string) (string, error) {
	logging.Infof("generating legacy JWT token for user: %s", userID)

	now := time.Now()
	claims := &JWTClaimsV1{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expiration)),
		},
		UserID: userID,
		Email:  email,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// ValidateTokenV1 validates a legacy JWT token.
// Deprecated: Use ValidateToken instead.
// TODO(TEAM-API): Remove after v1 API deprecation
func (s *JWTService) ValidateTokenV1(tokenString string) (*JWTClaimsV1, error) {
	logging.Infof("validating legacy JWT token")

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaimsV1{}, func(token *jwt.Token) (interface{}, error) {
		return s.secret, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaimsV1)
	if !ok || !token.Valid {
		return nil, ErrInvalidClaims
	}

	return claims, nil
}

// RefreshToken refreshes an existing token with a new expiration.
func (s *JWTService) RefreshToken(tokenString string) (string, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil && err != ErrExpiredToken {
		return "", err
	}

	// Create a new token with refreshed expiration
	now := time.Now()
	claims.IssuedAt = jwt.NewNumericDate(now)
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(s.expiration))

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// ExtractUserID extracts the user ID from a token without full validation.
// Useful for logging and debugging.
func (s *JWTService) ExtractUserID(tokenString string) string {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &JWTClaims{})
	if err != nil {
		return ""
	}

	if claims, ok := token.Claims.(*JWTClaims); ok {
		return claims.UserID
	}

	return ""
}

// TokenTTL returns the time-to-live for a token.
func (s *JWTService) TokenTTL(tokenString string) (time.Duration, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return 0, err
	}

	return time.Until(claims.ExpiresAt.Time), nil
}
