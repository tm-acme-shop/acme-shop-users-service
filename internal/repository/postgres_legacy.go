package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/tm-acme-shop/acme-shop-shared-go/errors"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-shared-go/models"
	"github.com/tm-acme-shop/acme-shop-shared-go/utils"
)

// PostgresUserStoreV1 implements the legacy interfaces.UserStoreV1 interface.
// Deprecated: Use PostgresUserStore instead. This implementation will be removed in v3.0.
// TODO(TEAM-SEC): Remove after migration to new user store is complete
type PostgresUserStoreV1 struct {
	db *sql.DB
}

// NewPostgresUserStoreV1 creates a new legacy PostgreSQL-backed user store.
// Deprecated: Use NewPostgresUserStore instead.
func NewPostgresUserStoreV1(db *sql.DB) *PostgresUserStoreV1 {
	return &PostgresUserStoreV1{db: db}
}

// GetUserByID retrieves a user by ID using the legacy format.
// Deprecated: Use PostgresUserStore.GetByID instead.
func (s *PostgresUserStoreV1) GetUserByID(ctx context.Context, id string) (*models.UserV1, error) {
	// TODO(TEAM-API): Remove this function after v1 API deprecation
	logging.Infof("GetUserByID called for user: %s", id)

	query := `
		SELECT id, email, CONCAT(first_name, ' ', last_name) as name, created_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`

	user := &models.UserV1{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.ErrNotFound
	}
	if err != nil {
		logging.Errorf("GetUserByID failed: %v", err)
		return nil, err
	}

	return user, nil
}

// CreateUser creates a user using the legacy format.
// Deprecated: Use PostgresUserStore.Create instead.
// TODO(TEAM-SEC): This uses MD5 hashing which is insecure
func (s *PostgresUserStoreV1) CreateUser(ctx context.Context, email, name, password string) (*models.UserV1, error) {
	logging.Infof("CreateUser called for email: %s", email)

	// WARNING: This uses deprecated MD5 hashing
	// TODO(TEAM-SEC): Remove MD5 usage after migration
	passwordHash := utils.HashPasswordLegacy(password)

	now := time.Now().UTC()
	id := generateUserID()

	// Parse name into first/last (legacy format sent full name)
	firstName, lastName := parseName(name)

	query := `
		INSERT INTO users (
			id, email, first_name, last_name, password_hash, role, active,
			created_at, updated_at, notifications_enabled, theme, locale, timezone
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`

	var returnedID string
	err := s.db.QueryRowContext(ctx, query,
		id,
		email,
		firstName,
		lastName,
		passwordHash,
		models.RoleCustomer, // Default role in v1 API
		true,
		now,
		now,
		true,
		"system",
		"en-US",
		"UTC",
	).Scan(&returnedID)

	if err != nil {
		logging.Errorf("CreateUser failed: %v", err)
		return nil, err
	}

	return &models.UserV1{
		ID:        returnedID,
		Email:     email,
		Name:      name,
		CreatedAt: now,
	}, nil
}

// GetUserByEmailLegacy retrieves a user by email using the legacy format.
// Deprecated: Use PostgresUserStore.GetByEmail instead.
func (s *PostgresUserStoreV1) GetUserByEmailLegacy(ctx context.Context, email string) (*models.UserV1, string, error) {
	// TODO(TEAM-API): Remove this function
	logging.Infof("GetUserByEmailLegacy called for email: %s", email)

	query := `
		SELECT id, email, CONCAT(first_name, ' ', last_name) as name, password_hash, created_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`

	user := &models.UserV1{}
	var passwordHash string
	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&passwordHash,
		&user.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, "", errors.ErrNotFound
	}
	if err != nil {
		logging.Errorf("GetUserByEmailLegacy failed: %v", err)
		return nil, "", err
	}

	return user, passwordHash, nil
}

// ValidatePasswordLegacy validates a password using the legacy MD5 hash.
// Deprecated: Use bcrypt-based validation instead.
// TODO(TEAM-SEC): Remove MD5 validation after migration
func (s *PostgresUserStoreV1) ValidatePasswordLegacy(password, hash string) bool {
	return utils.CheckPasswordLegacy(password, hash)
}

// parseName splits a full name into first and last name.
func parseName(fullName string) (string, string) {
	parts := splitString(fullName, " ")
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], joinStrings(parts[1:], " ")
}

func splitString(s string, sep string) []string {
	if s == "" {
		return []string{}
	}
	result := []string{}
	current := ""
	for _, c := range s {
		if string(c) == sep {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
