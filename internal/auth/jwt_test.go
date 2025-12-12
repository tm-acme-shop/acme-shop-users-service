package auth

import (
	"testing"
	"time"

	"github.com/tm-acme-shop/acme-shop-shared-go/models"
)

func TestJWTService(t *testing.T) {
	svc := NewJWTService("test-secret-key", 24*time.Hour)

	testUser := &models.User{
		ID:    "user-123",
		Email: "test@example.com",
		Role:  models.RoleCustomer,
	}

	t.Run("generate and validate token", func(t *testing.T) {
		token, err := svc.GenerateToken(testUser, "session-123")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if token == "" {
			t.Fatal("expected non-empty token")
		}

		claims, err := svc.ValidateToken(token)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if claims.UserID != testUser.ID {
			t.Fatalf("expected user ID %s, got %s", testUser.ID, claims.UserID)
		}
		if claims.Email != testUser.Email {
			t.Fatalf("expected email %s, got %s", testUser.Email, claims.Email)
		}
		if claims.SessionID != "session-123" {
			t.Fatalf("expected session ID session-123, got %s", claims.SessionID)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := svc.ValidateToken("invalid-token")
		if err == nil {
			t.Fatal("expected error for invalid token")
		}
	})

	t.Run("extract user ID", func(t *testing.T) {
		token, _ := svc.GenerateToken(testUser, "session-123")

		userID := svc.ExtractUserID(token)
		if userID != testUser.ID {
			t.Fatalf("expected user ID %s, got %s", testUser.ID, userID)
		}
	})

	t.Run("token TTL", func(t *testing.T) {
		token, _ := svc.GenerateToken(testUser, "session-123")

		ttl, err := svc.TokenTTL(token)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if ttl < 23*time.Hour || ttl > 24*time.Hour {
			t.Fatalf("expected TTL around 24h, got %v", ttl)
		}
	})
}

func TestJWTServiceV1(t *testing.T) {
	// Test legacy JWT methods
	// TODO(TEAM-API): Remove after v1 API deprecation
	svc := NewJWTService("test-secret-key", 24*time.Hour)

	t.Run("generate and validate v1 token", func(t *testing.T) {
		token, err := svc.GenerateTokenV1("user-123", "test@example.com")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		claims, err := svc.ValidateTokenV1(token)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if claims.UserID != "user-123" {
			t.Fatalf("expected user ID user-123, got %s", claims.UserID)
		}
	})
}

func TestRefreshToken(t *testing.T) {
	svc := NewJWTService("test-secret-key", 24*time.Hour)

	testUser := &models.User{
		ID:    "user-123",
		Email: "test@example.com",
		Role:  models.RoleCustomer,
	}

	t.Run("refresh valid token", func(t *testing.T) {
		originalToken, _ := svc.GenerateToken(testUser, "session-123")

		// Wait a tiny bit so the new token has a different issued time
		time.Sleep(10 * time.Millisecond)

		newToken, err := svc.RefreshToken(originalToken)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if newToken == originalToken {
			t.Fatal("expected new token to be different from original")
		}

		claims, err := svc.ValidateToken(newToken)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if claims.UserID != testUser.ID {
			t.Fatalf("expected user ID %s, got %s", testUser.ID, claims.UserID)
		}
	})
}
