package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/tm-acme-shop/acme-shop-shared-go/errors"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-shared-go/models"
)

// PostgresUserStore implements the interfaces.UserStore interface
// using PostgreSQL as the backing store.
type PostgresUserStore struct {
	db     *sql.DB
	logger *logging.LoggerV2
}

// NewPostgresUserStore creates a new PostgreSQL-backed user store.
func NewPostgresUserStore(db *sql.DB, logger *logging.LoggerV2) *PostgresUserStore {
	return &PostgresUserStore{
		db:     db,
		logger: logger.WithField("component", "postgres-user-store"),
	}
}

// GetByID retrieves a user by their unique identifier.
func (s *PostgresUserStore) GetByID(ctx context.Context, id string) (*models.User, error) {
	s.logger.Debug("fetching user by ID", logging.Fields{"user_id": id})

	query := `
		SELECT id, email, first_name, last_name, role, active,
		       created_at, updated_at, last_login_at,
		       notifications_enabled, theme, locale, timezone
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`

	user := &models.User{}
	var lastLoginAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.Role,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
		&lastLoginAt,
		&user.Preferences.NotificationsEnabled,
		&user.Preferences.Theme,
		&user.Preferences.Locale,
		&user.Preferences.Timezone,
	)

	if err == sql.ErrNoRows {
		return nil, errors.ErrNotFound
	}
	if err != nil {
		s.logger.Error("failed to fetch user by ID", logging.Fields{
			"user_id": id,
			"error":   err.Error(),
		})
		return nil, err
	}

	if lastLoginAt.Valid {
		user.LastLoginAt = lastLoginAt.Time
	}

	s.logger.Info("user fetched successfully", logging.Fields{"user_id": id})
	return user, nil
}

// GetByEmail retrieves a user by their email address.
func (s *PostgresUserStore) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	s.logger.Debug("fetching user by email", logging.Fields{"email": email})

	query := `
		SELECT id, email, first_name, last_name, role, active,
		       created_at, updated_at, last_login_at,
		       notifications_enabled, theme, locale, timezone
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`

	user := &models.User{}
	var lastLoginAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.Role,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
		&lastLoginAt,
		&user.Preferences.NotificationsEnabled,
		&user.Preferences.Theme,
		&user.Preferences.Locale,
		&user.Preferences.Timezone,
	)

	if err == sql.ErrNoRows {
		return nil, errors.ErrNotFound
	}
	if err != nil {
		// TODO(TEAM-PLATFORM): Use structured logging consistently
		logging.Errorf("failed to fetch user by email: %s, error: %v", email, err)
		return nil, err
	}

	if lastLoginAt.Valid {
		user.LastLoginAt = lastLoginAt.Time
	}

	return user, nil
}

// Create creates a new user in the store.
func (s *PostgresUserStore) Create(ctx context.Context, req *models.CreateUserRequest) (*models.User, error) {
	s.logger.Info("creating new user", logging.Fields{"email": req.Email})

	now := time.Now().UTC()
	id := generateUserID()

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
		req.Email,
		req.FirstName,
		req.LastName,
		req.Password, // Already hashed by the service layer
		req.Role,
		true, // Active by default
		now,
		now,
		true,     // Notifications enabled by default
		"system", // Default theme
		"en-US",  // Default locale
		"UTC",    // Default timezone
	).Scan(&returnedID)

	if err != nil {
		s.logger.Error("failed to create user", logging.Fields{
			"email": req.Email,
			"error": err.Error(),
		})
		return nil, err
	}

	return s.GetByID(ctx, returnedID)
}

// Update updates an existing user.
func (s *PostgresUserStore) Update(ctx context.Context, id string, req *models.UpdateUserRequest) (*models.User, error) {
	s.logger.Info("updating user", logging.Fields{"user_id": id})

	// Build dynamic update query
	updates := []string{"updated_at = $1"}
	args := []interface{}{time.Now().UTC()}
	argNum := 2

	if req.FirstName != nil {
		updates = append(updates, "first_name = $"+string(rune('0'+argNum)))
		args = append(args, *req.FirstName)
		argNum++
	}
	if req.LastName != nil {
		updates = append(updates, "last_name = $"+string(rune('0'+argNum)))
		args = append(args, *req.LastName)
		argNum++
	}
	if req.Active != nil {
		updates = append(updates, "active = $"+string(rune('0'+argNum)))
		args = append(args, *req.Active)
		argNum++
	}
	if req.Preferences != nil {
		updates = append(updates, "notifications_enabled = $"+string(rune('0'+argNum)))
		args = append(args, req.Preferences.NotificationsEnabled)
		argNum++

		updates = append(updates, "theme = $"+string(rune('0'+argNum)))
		args = append(args, req.Preferences.Theme)
		argNum++

		updates = append(updates, "locale = $"+string(rune('0'+argNum)))
		args = append(args, req.Preferences.Locale)
		argNum++

		updates = append(updates, "timezone = $"+string(rune('0'+argNum)))
		args = append(args, req.Preferences.Timezone)
		argNum++
	}

	args = append(args, id)

	query := "UPDATE users SET " + joinStrings(updates, ", ") +
		" WHERE id = $" + string(rune('0'+argNum)) + " AND deleted_at IS NULL"

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		s.logger.Error("failed to update user", logging.Fields{
			"user_id": id,
			"error":   err.Error(),
		})
		return nil, err
	}

	return s.GetByID(ctx, id)
}

// Delete removes a user from the store (soft delete).
func (s *PostgresUserStore) Delete(ctx context.Context, id string) error {
	s.logger.Info("deleting user", logging.Fields{"user_id": id})

	query := `
		UPDATE users
		SET deleted_at = $1, active = false
		WHERE id = $2 AND deleted_at IS NULL
	`

	result, err := s.db.ExecContext(ctx, query, time.Now().UTC(), id)
	if err != nil {
		s.logger.Error("failed to delete user", logging.Fields{
			"user_id": id,
			"error":   err.Error(),
		})
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.ErrNotFound
	}

	return nil
}

// List retrieves users based on filter criteria.
func (s *PostgresUserStore) List(ctx context.Context, filter *models.UserListFilter) ([]*models.User, int, error) {
	s.logger.Debug("listing users", logging.Fields{
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})

	// Base query
	baseQuery := `
		FROM users
		WHERE deleted_at IS NULL
	`
	args := []interface{}{}
	argNum := 1

	if filter.Role != nil {
		baseQuery += " AND role = $" + string(rune('0'+argNum))
		args = append(args, *filter.Role)
		argNum++
	}
	if filter.Active != nil {
		baseQuery += " AND active = $" + string(rune('0'+argNum))
		args = append(args, *filter.Active)
		argNum++
	}
	if filter.Search != "" {
		baseQuery += " AND (first_name ILIKE $" + string(rune('0'+argNum)) +
			" OR last_name ILIKE $" + string(rune('0'+argNum)) +
			" OR email ILIKE $" + string(rune('0'+argNum)) + ")"
		args = append(args, "%"+filter.Search+"%")
		argNum++
	}

	// Count query
	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Data query
	selectQuery := `
		SELECT id, email, first_name, last_name, role, active,
		       created_at, updated_at, last_login_at,
		       notifications_enabled, theme, locale, timezone
	` + baseQuery + ` ORDER BY created_at DESC LIMIT $` + string(rune('0'+argNum)) +
		` OFFSET $` + string(rune('0'+argNum+1))

	args = append(args, filter.Limit, filter.Offset)

	rows, err := s.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	users := []*models.User{}
	for rows.Next() {
		user := &models.User{}
		var lastLoginAt sql.NullTime

		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.FirstName,
			&user.LastName,
			&user.Role,
			&user.Active,
			&user.CreatedAt,
			&user.UpdatedAt,
			&lastLoginAt,
			&user.Preferences.NotificationsEnabled,
			&user.Preferences.Theme,
			&user.Preferences.Locale,
			&user.Preferences.Timezone,
		)
		if err != nil {
			return nil, 0, err
		}

		if lastLoginAt.Valid {
			user.LastLoginAt = lastLoginAt.Time
		}

		users = append(users, user)
	}

	return users, total, nil
}

// UpdateLastLogin updates the user's last login timestamp.
func (s *PostgresUserStore) UpdateLastLogin(ctx context.Context, id string) error {
	query := `UPDATE users SET last_login_at = $1 WHERE id = $2`
	_, err := s.db.ExecContext(ctx, query, time.Now().UTC(), id)
	if err != nil {
		s.logger.Error("failed to update last login", logging.Fields{
			"user_id": id,
			"error":   err.Error(),
		})
	}
	return err
}

// GetPasswordHash retrieves the password hash for authentication.
func (s *PostgresUserStore) GetPasswordHash(ctx context.Context, id string) (string, error) {
	var hash string
	query := `SELECT password_hash FROM users WHERE id = $1 AND deleted_at IS NULL`
	err := s.db.QueryRowContext(ctx, query, id).Scan(&hash)
	if err == sql.ErrNoRows {
		return "", errors.ErrNotFound
	}
	return hash, err
}

// UpdatePasswordHash updates the user's password hash.
func (s *PostgresUserStore) UpdatePasswordHash(ctx context.Context, id, hash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3`
	_, err := s.db.ExecContext(ctx, query, hash, time.Now().UTC(), id)
	return err
}

func generateUserID() string {
	// Simple ID generation for demo
	return "user-" + randomString(12)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[i%len(letters)]
	}
	return string(b)
}

func joinStrings(s []string, sep string) string {
	if len(s) == 0 {
		return ""
	}
	result := s[0]
	for i := 1; i < len(s); i++ {
		result += sep + s[i]
	}
	return result
}
