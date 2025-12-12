package migrations

import (
	"context"
	"database/sql"
	"time"

	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
)

// Migration represents a database migration.
type Migration struct {
	ID        int
	Name      string
	SQL       string
	Rollback  string
	AppliedAt time.Time
}

// Migrator handles database migrations.
type Migrator struct {
	db     *sql.DB
	logger *logging.LoggerV2
}

// NewMigrator creates a new migrator instance.
func NewMigrator(db *sql.DB) *Migrator {
	return &Migrator{
		db:     db,
		logger: logging.NewLoggerV2("migrator"),
	}
}

// Run executes all pending migrations.
func (m *Migrator) Run(ctx context.Context) error {
	m.logger.Info("running migrations")

	// Create migrations table if not exists
	if err := m.createMigrationsTable(ctx); err != nil {
		return err
	}

	// Get list of applied migrations
	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	// Run pending migrations
	for _, migration := range allMigrations {
		if _, ok := applied[migration.ID]; ok {
			continue
		}

		m.logger.Info("applying migration", logging.Fields{
			"id":   migration.ID,
			"name": migration.Name,
		})

		if err := m.applyMigration(ctx, migration); err != nil {
			return err
		}
	}

	m.logger.Info("migrations completed")
	return nil
}

func (m *Migrator) createMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id INTEGER PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`
	_, err := m.db.ExecContext(ctx, query)
	return err
}

func (m *Migrator) getAppliedMigrations(ctx context.Context) (map[int]bool, error) {
	query := `SELECT id FROM schema_migrations`
	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		applied[id] = true
	}
	return applied, nil
}

func (m *Migrator) applyMigration(ctx context.Context, migration Migration) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
		logging.Errorf("migration %d failed: %v", migration.ID, err)
		return err
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (id, name) VALUES ($1, $2)`,
		migration.ID, migration.Name); err != nil {
		return err
	}

	return tx.Commit()
}

// allMigrations contains all database migrations.
var allMigrations = []Migration{
	{
		ID:   1,
		Name: "create_users_table",
		SQL: `
			CREATE TABLE IF NOT EXISTS users (
				id VARCHAR(50) PRIMARY KEY,
				email VARCHAR(255) UNIQUE NOT NULL,
				first_name VARCHAR(100) NOT NULL,
				last_name VARCHAR(100) NOT NULL,
				password_hash VARCHAR(255) NOT NULL,
				role VARCHAR(50) NOT NULL DEFAULT 'customer',
				active BOOLEAN NOT NULL DEFAULT true,
				created_at TIMESTAMP NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
				deleted_at TIMESTAMP,
				last_login_at TIMESTAMP,
				notifications_enabled BOOLEAN NOT NULL DEFAULT true,
				theme VARCHAR(50) NOT NULL DEFAULT 'system',
				locale VARCHAR(10) NOT NULL DEFAULT 'en-US',
				timezone VARCHAR(50) NOT NULL DEFAULT 'UTC'
			);
			CREATE INDEX idx_users_email ON users(email);
			CREATE INDEX idx_users_role ON users(role);
			CREATE INDEX idx_users_active ON users(active);
		`,
		Rollback: `DROP TABLE IF EXISTS users;`,
	},
	{
		ID:   2,
		Name: "create_sessions_table",
		SQL: `
			CREATE TABLE IF NOT EXISTS sessions (
				id VARCHAR(50) PRIMARY KEY,
				user_id VARCHAR(50) NOT NULL REFERENCES users(id),
				created_at TIMESTAMP NOT NULL DEFAULT NOW(),
				expires_at TIMESTAMP NOT NULL,
				ip_address VARCHAR(50),
				user_agent TEXT,
				active BOOLEAN NOT NULL DEFAULT true
			);
			CREATE INDEX idx_sessions_user_id ON sessions(user_id);
			CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
		`,
		Rollback: `DROP TABLE IF EXISTS sessions;`,
	},
	{
		ID:   3,
		Name: "create_audit_log_table",
		SQL: `
			CREATE TABLE IF NOT EXISTS audit_log (
				id SERIAL PRIMARY KEY,
				user_id VARCHAR(50) REFERENCES users(id),
				action VARCHAR(50) NOT NULL,
				resource_type VARCHAR(50) NOT NULL,
				resource_id VARCHAR(50),
				old_value JSONB,
				new_value JSONB,
				ip_address VARCHAR(50),
				created_at TIMESTAMP NOT NULL DEFAULT NOW()
			);
			CREATE INDEX idx_audit_log_user_id ON audit_log(user_id);
			CREATE INDEX idx_audit_log_created_at ON audit_log(created_at);
		`,
		Rollback: `DROP TABLE IF EXISTS audit_log;`,
	},
	{
		ID:   4,
		Name: "add_password_hash_type_column",
		SQL: `
			-- Add column to track password hash type for migration tracking
			-- TODO(TEAM-SEC): Remove after all passwords migrated to bcrypt
			ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash_type VARCHAR(20) DEFAULT 'unknown';
			
			-- Update existing records based on hash length
			UPDATE users SET password_hash_type = 
				CASE 
					WHEN password_hash LIKE '$2%' THEN 'bcrypt'
					WHEN LENGTH(password_hash) = 32 THEN 'md5'
					WHEN LENGTH(password_hash) = 40 THEN 'sha1'
					ELSE 'unknown'
				END;
		`,
		Rollback: `ALTER TABLE users DROP COLUMN IF EXISTS password_hash_type;`,
	},
	{
		ID:   5,
		Name: "add_api_keys_table",
		SQL: `
			-- Legacy API keys table for backwards compatibility
			-- Deprecated: TODO(TEAM-SEC): Remove after migration to JWT
			CREATE TABLE IF NOT EXISTS api_keys (
				id VARCHAR(50) PRIMARY KEY,
				user_id VARCHAR(50) NOT NULL REFERENCES users(id),
				key_hash VARCHAR(255) NOT NULL,
				name VARCHAR(100),
				last_used_at TIMESTAMP,
				created_at TIMESTAMP NOT NULL DEFAULT NOW(),
				expires_at TIMESTAMP,
				active BOOLEAN NOT NULL DEFAULT true
			);
			CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
			CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
		`,
		Rollback: `DROP TABLE IF EXISTS api_keys;`,
	},
}
