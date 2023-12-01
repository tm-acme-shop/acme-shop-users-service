package service

import (
	"context"

	"github.com/tm-acme-shop/acme-shop-shared-go/errors"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-shared-go/models"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/auth"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/config"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/repository"
)

// AuthService provides authentication operations.
type AuthService struct {
	repo            *repository.PostgresUserStore
	passwordService *auth.PasswordService
	jwtService      *auth.JWTService
	sessionService  *auth.SessionService
	config          *config.Config
	logger          *logging.LoggerV2
}

// NewAuthService creates a new authentication service.
func NewAuthService(
	repo *repository.PostgresUserStore,
	passwordService *auth.PasswordService,
	jwtService *auth.JWTService,
	sessionService *auth.SessionService,
	cfg *config.Config,
) *AuthService {
	return &AuthService{
		repo:            repo,
		passwordService: passwordService,
		jwtService:      jwtService,
		sessionService:  sessionService,
		config:          cfg,
		logger:          logging.NewLoggerV2("auth-service"),
	}
}

// Login authenticates a user and returns a token (v2 API).
func (s *AuthService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	s.logger.Info("login attempt", logging.Fields{
		"email": req.Email,
	})

	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		if err == errors.ErrNotFound {
			s.logger.Warn("login failed - user not found", logging.Fields{
				"email": req.Email,
			})
			return nil, errors.ErrInvalidCredentials
		}
		return nil, err
	}

	// Check if user is active
	if !user.Active {
		s.logger.Warn("login failed - user inactive", logging.Fields{
			"user_id": user.ID,
		})
		return nil, errors.ErrUserInactive
	}

	// Get password hash
	hash, err := s.repo.GetPasswordHash(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	// Verify password
	valid, needsMigration := s.passwordService.CheckPassword(req.Password, hash)
	if !valid {
		s.logger.Warn("login failed - invalid password", logging.Fields{
			"user_id": user.ID,
		})
		return nil, errors.ErrInvalidCredentials
	}

	// Migrate password hash if needed
	if needsMigration && s.config.Features.EnablePasswordMigration {
		s.logger.Info("migrating password hash", logging.Fields{
			"user_id": user.ID,
		})
		newHash, err := s.passwordService.MigratePasswordHash(req.Password)
		if err == nil {
			s.repo.UpdatePasswordHash(ctx, user.ID, newHash)
		}
	}

	// Create session
	session, err := s.sessionService.Create(
		ctx,
		user.ID,
		user.Email,
		string(user.Role),
		req.IPAddress,
		req.UserAgent,
	)
	if err != nil {
		return nil, err
	}

	// Generate JWT token
	token, err := s.jwtService.GenerateToken(user, session.ID)
	if err != nil {
		return nil, err
	}

	// Update last login
	s.repo.UpdateLastLogin(ctx, user.ID)

	s.logger.Info("login successful", logging.Fields{
		"user_id":    user.ID,
		"session_id": session.ID,
	})

	return &LoginResponse{
		Token:     token,
		User:      user,
		SessionID: session.ID,
		ExpiresAt: session.ExpiresAt,
	}, nil
}

// LoginV1 authenticates a user using the legacy API.
// Deprecated: Use Login instead.
// TODO(TEAM-API): Remove after v1 API deprecation
func (s *AuthService) LoginV1(ctx context.Context, email, password string) (*LoginResponseV1, error) {
	logging.Infof("LoginV1 called for email: %s", email)

	if !s.config.Features.EnableV1API {
		return nil, errors.ErrDeprecatedAPI
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, errors.ErrInvalidCredentials
	}

	hash, err := s.repo.GetPasswordHash(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	valid, _ := s.passwordService.CheckPassword(password, hash)
	if !valid {
		return nil, errors.ErrInvalidCredentials
	}

	// Generate legacy token
	token, err := s.jwtService.GenerateTokenV1(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	s.repo.UpdateLastLogin(ctx, user.ID)

	return &LoginResponseV1{
		Token: token,
		User:  user.ToV1(),
	}, nil
}

// Logout invalidates a user's session.
func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	s.logger.Info("logout", logging.Fields{"session_id": sessionID})

	return s.sessionService.Delete(ctx, sessionID)
}

// LogoutAll invalidates all sessions for a user.
func (s *AuthService) LogoutAll(ctx context.Context, userID string) error {
	s.logger.Info("logout all", logging.Fields{"user_id": userID})

	return s.sessionService.DeleteAllForUser(ctx, userID)
}

// ValidateToken validates a JWT token and returns the claims.
func (s *AuthService) ValidateToken(ctx context.Context, token string) (*auth.JWTClaims, error) {
	claims, err := s.jwtService.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	// Validate session is still active
	if claims.SessionID != "" {
		_, err := s.sessionService.Get(ctx, claims.SessionID)
		if err != nil {
			return nil, err
		}
	}

	return claims, nil
}

// ValidateTokenV1 validates a legacy JWT token.
// Deprecated: Use ValidateToken instead.
// TODO(TEAM-API): Remove after v1 API deprecation
func (s *AuthService) ValidateTokenV1(ctx context.Context, token string) (*auth.JWTClaimsV1, error) {
	logging.Infof("ValidateTokenV1 called")

	if !s.config.Features.EnableV1API {
		return nil, errors.ErrDeprecatedAPI
	}

	return s.jwtService.ValidateTokenV1(token)
}

// RefreshToken refreshes a JWT token.
func (s *AuthService) RefreshToken(ctx context.Context, token string) (*RefreshTokenResponse, error) {
	claims, err := s.jwtService.ValidateToken(token)
	if err != nil && err != auth.ErrExpiredToken {
		return nil, err
	}

	// Validate session
	session, err := s.sessionService.Get(ctx, claims.SessionID)
	if err != nil {
		return nil, err
	}

	// Refresh session
	if err := s.sessionService.Refresh(ctx, session.ID); err != nil {
		return nil, err
	}

	// Get user for new token
	user, err := s.repo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	// Generate new token
	newToken, err := s.jwtService.GenerateToken(user, session.ID)
	if err != nil {
		return nil, err
	}

	return &RefreshTokenResponse{
		Token:     newToken,
		ExpiresAt: session.ExpiresAt,
	}, nil
}

// GetSessions returns all active sessions for a user.
func (s *AuthService) GetSessions(ctx context.Context, userID string) ([]*auth.Session, error) {
	return s.sessionService.ListForUser(ctx, userID)
}

// RevokeSession revokes a specific session.
func (s *AuthService) RevokeSession(ctx context.Context, sessionID string) error {
	return s.sessionService.Revoke(ctx, sessionID)
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	IPAddress string `json:"-"`
	UserAgent string `json:"-"`
}

// LoginResponse represents a login response (v2 API).
type LoginResponse struct {
	Token     string       `json:"token"`
	User      *models.User `json:"user"`
	SessionID string       `json:"session_id"`
	ExpiresAt interface{}  `json:"expires_at"`
}

// LoginResponseV1 represents a login response (v1 API).
// Deprecated: Use LoginResponse instead.
type LoginResponseV1 struct {
	Token string         `json:"token"`
	User  *models.UserV1 `json:"user"`
}

// RefreshTokenResponse represents a token refresh response.
type RefreshTokenResponse struct {
	Token     string      `json:"token"`
	ExpiresAt interface{} `json:"expires_at"`
}
