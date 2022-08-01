package repository

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/tm-acme-shop/acme-shop-users-service/internal/auth"
	"github.com/tm-acme-shop/acme-shop-users-service/internal/service"
)

// PostgresUserStore implements user storage using PostgreSQL.
type PostgresUserStore struct {
	db     *sql.DB
	logger *auth.LoggerV2
}

// NewPostgresUserStore creates a new PostgreSQL-backed user store.
func NewPostgresUserStore(db *sql.DB) *PostgresUserStore {
	return &PostgresUserStore{
		db:     db,
		logger: auth.NewLoggerV2("postgres-user-store"),
	}
}

// GetByID retrieves a user by their unique identifier.
func (s *PostgresUserStore) GetByID(ctx context.Context, id string) (*service.User, error) {
	s.logger.Info("fetching user by ID", map[string]interface{}{"user_id": id})

	query := `
		SELECT id, email, name
		FROM users
		WHERE id = $1
	`

	user := &service.User{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
	)

	if err == sql.ErrNoRows {
		return nil, err
	}
	if err != nil {
		log.Printf("Failed to fetch user by ID: %v", err)
		return nil, err
	}

	return user, nil
}

// Create creates a new user in the store.
func (s *PostgresUserStore) Create(ctx context.Context, email, name, passwordHash string) (*service.User, error) {
	s.logger.Info("creating user", map[string]interface{}{"email": email})

	now := time.Now().UTC()
	id := generateUserID()

	query := `
		INSERT INTO users (id, email, name, password_hash, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	var returnedID string
	err := s.db.QueryRowContext(ctx, query, id, email, name, passwordHash, now).Scan(&returnedID)
	if err != nil {
		log.Printf("Failed to create user: %v", err)
		return nil, err
	}

	return &service.User{
		ID:    returnedID,
		Email: email,
		Name:  name,
	}, nil
}

// List retrieves all users.
func (s *PostgresUserStore) List(ctx context.Context) ([]*service.User, error) {
	s.logger.Info("listing users", nil)

	query := `SELECT id, email, name FROM users ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []*service.User{}
	for rows.Next() {
		user := &service.User{}
		if err := rows.Scan(&user.ID, &user.Email, &user.Name); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

// GetPasswordHash retrieves the password hash for a user.
func (s *PostgresUserStore) GetPasswordHash(ctx context.Context, id string) (string, error) {
	var hash string
	query := `SELECT password_hash FROM users WHERE id = $1`
	err := s.db.QueryRowContext(ctx, query, id).Scan(&hash)
	return hash, err
}

// UpdatePasswordHash updates the password hash for a user.
func (s *PostgresUserStore) UpdatePasswordHash(ctx context.Context, id, hash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3`
	_, err := s.db.ExecContext(ctx, query, hash, time.Now().UTC(), id)
	return err
}

func generateUserID() string {
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
