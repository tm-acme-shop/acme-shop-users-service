package migrations

import (
	"context"
	"database/sql"
	"time"

	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-shared-go/utils"
)

// SEC-175: Password migration service for MD5→bcrypt upgrade
// PasswordMigrator handles migration of legacy password hashes.
// TODO(TEAM-SEC): Run this migration to upgrade all MD5/SHA1 hashes to bcrypt
type PasswordMigrator struct {
	db     *sql.DB
	logger *logging.LoggerV2
}

// NewPasswordMigrator creates a new password migrator.
func NewPasswordMigrator(db *sql.DB) *PasswordMigrator {
	return &PasswordMigrator{
		db:     db,
		logger: logging.NewLoggerV2("password-migrator"),
	}
}

// MigrationStats holds password migration statistics.
type MigrationStats struct {
	TotalUsers    int
	MD5Users      int
	SHA1Users     int
	BcryptUsers   int
	UnknownUsers  int
	MigratedCount int
	FailedCount   int
}

// GetStats returns password hash statistics.
func (m *PasswordMigrator) GetStats(ctx context.Context) (*MigrationStats, error) {
	m.logger.Info("getting password migration stats")

	stats := &MigrationStats{}

	query := `
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN password_hash_type = 'md5' THEN 1 ELSE 0 END) as md5_count,
			SUM(CASE WHEN password_hash_type = 'sha1' THEN 1 ELSE 0 END) as sha1_count,
			SUM(CASE WHEN password_hash_type = 'bcrypt' THEN 1 ELSE 0 END) as bcrypt_count,
			SUM(CASE WHEN password_hash_type = 'unknown' OR password_hash_type IS NULL THEN 1 ELSE 0 END) as unknown_count
		FROM users
		WHERE deleted_at IS NULL
	`

	err := m.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalUsers,
		&stats.MD5Users,
		&stats.SHA1Users,
		&stats.BcryptUsers,
		&stats.UnknownUsers,
	)

	if err != nil {
		return nil, err
	}

	m.logger.Info("password migration stats", logging.Fields{
		"total":   stats.TotalUsers,
		"md5":     stats.MD5Users,
		"sha1":    stats.SHA1Users,
		"bcrypt":  stats.BcryptUsers,
		"unknown": stats.UnknownUsers,
	})

	return stats, nil
}

// MigrateUserPassword migrates a single user's password on successful login.
// This is the preferred migration strategy - migrate on authentication.
func (m *PasswordMigrator) MigrateUserPassword(ctx context.Context, userID, password string) error {
	// TODO(TEAM-SEC): Call this function after successful legacy password validation

	m.logger.Info("migrating user password to bcrypt", logging.Fields{
		"user_id": userID,
	})

	// Hash with bcrypt
	newHash, err := utils.HashPassword(password)
	if err != nil {
		logging.Errorf("failed to hash password for user %s: %v", userID, err)
		return err
	}

	// Update in database
	query := `
		UPDATE users 
		SET password_hash = $1, password_hash_type = 'bcrypt', updated_at = $2
		WHERE id = $3
	`

	_, err = m.db.ExecContext(ctx, query, newHash, time.Now().UTC(), userID)
	if err != nil {
		return err
	}

	m.logger.Info("password migrated successfully", logging.Fields{
		"user_id": userID,
	})

	return nil
}

// IdentifyLegacyUsers returns users with legacy password hashes.
// Deprecated: Use GetStats instead for reporting.
func (m *PasswordMigrator) IdentifyLegacyUsers(ctx context.Context) ([]string, error) {
	logging.Infof("identifying users with legacy password hashes")

	query := `
		SELECT id FROM users 
		WHERE password_hash_type IN ('md5', 'sha1') 
		AND deleted_at IS NULL
		ORDER BY last_login_at DESC
	`

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		userIDs = append(userIDs, id)
	}

	logging.Infof("found %d users with legacy password hashes", len(userIDs))
	return userIDs, nil
}

// ForcePasswordReset forces password reset for users with legacy hashes.
// This is an alternative migration strategy for users who haven't logged in.
// TODO(TEAM-SEC): Consider sending password reset emails to affected users
func (m *PasswordMigrator) ForcePasswordReset(ctx context.Context, inactiveDays int) (int, error) {
	m.logger.Warn("forcing password reset for users with legacy hashes", logging.Fields{
		"inactive_days": inactiveDays,
	})

	cutoff := time.Now().AddDate(0, 0, -inactiveDays)

	query := `
		UPDATE users 
		SET password_hash = '', password_hash_type = 'reset_required', updated_at = $1
		WHERE password_hash_type IN ('md5', 'sha1')
		AND (last_login_at IS NULL OR last_login_at < $2)
		AND deleted_at IS NULL
	`

	result, err := m.db.ExecContext(ctx, query, time.Now().UTC(), cutoff)
	if err != nil {
		return 0, err
	}

	affected, _ := result.RowsAffected()

	logging.Warnf("forced password reset for %d users", affected)

	return int(affected), nil
}

// ValidateHashType validates and updates the hash type for a user's password.
func (m *PasswordMigrator) ValidateHashType(ctx context.Context, userID string) (string, error) {
	var hash string
	var hashType sql.NullString

	query := `SELECT password_hash, password_hash_type FROM users WHERE id = $1`
	err := m.db.QueryRowContext(ctx, query, userID).Scan(&hash, &hashType)
	if err != nil {
		return "", err
	}

	// Detect actual hash type
	detectedType := detectHashType(hash)

	// Update if different
	if !hashType.Valid || hashType.String != detectedType {
		updateQuery := `UPDATE users SET password_hash_type = $1 WHERE id = $2`
		m.db.ExecContext(ctx, updateQuery, detectedType, userID)
	}

	return detectedType, nil
}

func detectHashType(hash string) string {
	if len(hash) >= 4 && hash[:2] == "$2" {
		return "bcrypt"
	}
	if len(hash) == 32 {
		return "md5"
	}
	if len(hash) == 40 {
		return "sha1"
	}
	return "unknown"
}
