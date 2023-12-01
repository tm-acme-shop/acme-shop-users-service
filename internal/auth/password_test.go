package auth

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	svc := NewPasswordService(false)

	t.Run("valid password", func(t *testing.T) {
		hash, err := svc.HashPassword("securePassword123")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if hash == "" {
			t.Fatal("expected non-empty hash")
		}
		if len(hash) < 50 {
			t.Fatalf("expected bcrypt hash length > 50, got %d", len(hash))
		}
	})

	t.Run("password too short", func(t *testing.T) {
		_, err := svc.HashPassword("short")
		if err != ErrPasswordTooShort {
			t.Fatalf("expected ErrPasswordTooShort, got %v", err)
		}
	})

	t.Run("empty password", func(t *testing.T) {
		_, err := svc.HashPassword("")
		if err != ErrPasswordEmpty {
			t.Fatalf("expected ErrPasswordEmpty, got %v", err)
		}
	})
}

func TestCheckPassword(t *testing.T) {
	svc := NewPasswordService(true)

	t.Run("bcrypt hash", func(t *testing.T) {
		password := "testPassword123"
		hash, _ := svc.HashPassword(password)

		valid, needsMigration := svc.CheckPassword(password, hash)
		if !valid {
			t.Fatal("expected password to be valid")
		}
		if needsMigration {
			t.Fatal("bcrypt hash should not need migration")
		}
	})

	t.Run("md5 hash", func(t *testing.T) {
		password := "testPassword123"
		hash := md5Hash(password) // MD5 hash

		valid, needsMigration := svc.CheckPassword(password, hash)
		if !valid {
			t.Fatal("expected password to be valid")
		}
		if !needsMigration {
			t.Fatal("MD5 hash should need migration")
		}
	})

	t.Run("sha1 hash", func(t *testing.T) {
		password := "testPassword123"
		hash := sha1Hash(password) // SHA1 hash

		valid, needsMigration := svc.CheckPassword(password, hash)
		if !valid {
			t.Fatal("expected password to be valid")
		}
		if !needsMigration {
			t.Fatal("SHA1 hash should need migration")
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		hash, _ := svc.HashPassword("correctPassword")

		valid, _ := svc.CheckPassword("wrongPassword", hash)
		if valid {
			t.Fatal("expected password to be invalid")
		}
	})
}

func TestDetectHashType(t *testing.T) {
	svc := NewPasswordService(false)

	tests := []struct {
		name     string
		hash     string
		expected string
	}{
		{
			name:     "bcrypt hash",
			hash:     "$2a$12$abc123def456ghi789jkl.mnopqrstuvwxyz0123456789ABCDEF",
			expected: HashTypeBcrypt,
		},
		{
			name:     "md5 hash",
			hash:     "5f4dcc3b5aa765d61d8327deb882cf99", // 32 chars
			expected: HashTypeMD5,
		},
		{
			name:     "sha1 hash",
			hash:     "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d", // 40 chars
			expected: HashTypeSHA1,
		},
		{
			name:     "unknown hash",
			hash:     "somethingelse",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.DetectHashType(tt.hash)
			if result != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestPasswordStrength(t *testing.T) {
	tests := []struct {
		password string
		minScore int
	}{
		{"short", 0},          // < 8 chars
		{"longenough", 1},     // >= 8 chars
		{"LongerPassword", 2}, // >= 12 chars with mixed case
		{"LongerP@ss1", 3},    // mixed case + digits
		{"LongerP@ss123!", 4}, // all requirements
	}

	for _, tt := range tests {
		t.Run(tt.password, func(t *testing.T) {
			score := PasswordStrength(tt.password)
			if score < tt.minScore {
				t.Fatalf("expected score >= %d, got %d", tt.minScore, score)
			}
		})
	}
}

func TestMD5Hash(t *testing.T) {
	// Test that MD5 hash produces expected output
	// TODO(TEAM-SEC): Remove after migration complete
	result := md5Hash("test")
	expected := "098f6bcd4621d373cade4e832627b4f6"

	if result != expected {
		t.Fatalf("expected %s, got %s", expected, result)
	}
}

func TestSHA1Hash(t *testing.T) {
	// Test that SHA1 hash produces expected output
	// TODO(TEAM-SEC): Remove after migration complete
	result := sha1Hash("test")
	expected := "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3"

	if result != expected {
		t.Fatalf("expected %s, got %s", expected, result)
	}
}
