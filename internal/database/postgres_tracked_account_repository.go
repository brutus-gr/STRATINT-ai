package database

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/STRATINT/stratint/internal/models"
)

type PostgresTrackedAccountRepository struct {
	db *sql.DB
}

func NewPostgresTrackedAccountRepository(db *sql.DB) *PostgresTrackedAccountRepository {
	return &PostgresTrackedAccountRepository{db: db}
}

func (r *PostgresTrackedAccountRepository) Store(account *models.TrackedAccount) error {
	metadataJSON, err := json.Marshal(account.Metadata)
	if err != nil {
		return err
	}

	if account.ID == "" {
		// New account - let DB generate ID
		query := `
			INSERT INTO tracked_accounts
			(platform, account_identifier, display_name, enabled,
			 last_fetched_id, last_fetched_at, fetch_interval_minutes, metadata)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (platform, account_identifier)
			DO UPDATE SET
				display_name = EXCLUDED.display_name,
				enabled = EXCLUDED.enabled,
				fetch_interval_minutes = EXCLUDED.fetch_interval_minutes,
				metadata = EXCLUDED.metadata,
				updated_at = NOW()
			RETURNING id, created_at, updated_at
		`

		err = r.db.QueryRow(query,
			account.Platform,
			account.AccountIdentifier,
			account.DisplayName,
			account.Enabled,
			account.LastFetchedID,
			account.LastFetchedAt,
			account.FetchIntervalMinutes,
			metadataJSON,
		).Scan(&account.ID, &account.CreatedAt, &account.UpdatedAt)
	} else {
		// Existing account - update
		query := `
			UPDATE tracked_accounts SET
				display_name = $2,
				enabled = $3,
				fetch_interval_minutes = $4,
				metadata = $5,
				updated_at = NOW()
			WHERE id = $1
			RETURNING id, created_at, updated_at
		`

		err = r.db.QueryRow(query,
			account.ID,
			account.DisplayName,
			account.Enabled,
			account.FetchIntervalMinutes,
			metadataJSON,
		).Scan(&account.ID, &account.CreatedAt, &account.UpdatedAt)
	}

	return err
}

func (r *PostgresTrackedAccountRepository) GetByID(id string) (*models.TrackedAccount, error) {
	query := `
		SELECT id, platform, account_identifier, display_name, enabled,
		       last_fetched_id, last_fetched_at, fetch_interval_minutes,
		       metadata, created_at, updated_at
		FROM tracked_accounts
		WHERE id = $1
	`

	var account models.TrackedAccount
	var metadataJSON []byte

	err := r.db.QueryRow(query, id).Scan(
		&account.ID,
		&account.Platform,
		&account.AccountIdentifier,
		&account.DisplayName,
		&account.Enabled,
		&account.LastFetchedID,
		&account.LastFetchedAt,
		&account.FetchIntervalMinutes,
		&metadataJSON,
		&account.CreatedAt,
		&account.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &account.Metadata); err != nil {
			return nil, err
		}
	}

	return &account, nil
}

func (r *PostgresTrackedAccountRepository) GetByPlatformAndIdentifier(platform, identifier string) (*models.TrackedAccount, error) {
	query := `
		SELECT id, platform, account_identifier, display_name, enabled,
		       last_fetched_id, last_fetched_at, fetch_interval_minutes,
		       metadata, created_at, updated_at
		FROM tracked_accounts
		WHERE platform = $1 AND account_identifier = $2
	`

	var account models.TrackedAccount
	var metadataJSON []byte

	err := r.db.QueryRow(query, platform, identifier).Scan(
		&account.ID,
		&account.Platform,
		&account.AccountIdentifier,
		&account.DisplayName,
		&account.Enabled,
		&account.LastFetchedID,
		&account.LastFetchedAt,
		&account.FetchIntervalMinutes,
		&metadataJSON,
		&account.CreatedAt,
		&account.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &account.Metadata); err != nil {
			return nil, err
		}
	}

	return &account, nil
}

func (r *PostgresTrackedAccountRepository) ListByPlatform(platform string, enabledOnly bool) ([]*models.TrackedAccount, error) {
	query := `
		SELECT id, platform, account_identifier, display_name, enabled,
		       last_fetched_id, last_fetched_at, fetch_interval_minutes,
		       metadata, created_at, updated_at
		FROM tracked_accounts
		WHERE platform = $1
	`

	if enabledOnly {
		query += " AND enabled = true"
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.Query(query, platform)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanAccounts(rows)
}

func (r *PostgresTrackedAccountRepository) ListAll(enabledOnly bool) ([]*models.TrackedAccount, error) {
	query := `
		SELECT id, platform, account_identifier, display_name, enabled,
		       last_fetched_id, last_fetched_at, fetch_interval_minutes,
		       metadata, created_at, updated_at
		FROM tracked_accounts
	`

	if enabledOnly {
		query += " WHERE enabled = true"
	}

	query += " ORDER BY platform, created_at DESC"

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanAccounts(rows)
}

func (r *PostgresTrackedAccountRepository) UpdateLastFetched(id, lastFetchedID string, lastFetchedAt time.Time) error {
	query := `
		UPDATE tracked_accounts
		SET last_fetched_id = $2,
		    last_fetched_at = $3,
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(query, id, lastFetchedID, lastFetchedAt)
	return err
}

func (r *PostgresTrackedAccountRepository) Delete(id string) error {
	query := `DELETE FROM tracked_accounts WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *PostgresTrackedAccountRepository) SetEnabled(id string, enabled bool) error {
	query := `
		UPDATE tracked_accounts
		SET enabled = $2, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(query, id, enabled)
	return err
}

func (r *PostgresTrackedAccountRepository) scanAccounts(rows *sql.Rows) ([]*models.TrackedAccount, error) {
	var accounts []*models.TrackedAccount

	for rows.Next() {
		var account models.TrackedAccount
		var metadataJSON []byte

		err := rows.Scan(
			&account.ID,
			&account.Platform,
			&account.AccountIdentifier,
			&account.DisplayName,
			&account.Enabled,
			&account.LastFetchedID,
			&account.LastFetchedAt,
			&account.FetchIntervalMinutes,
			&metadataJSON,
			&account.CreatedAt,
			&account.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &account.Metadata); err != nil {
				return nil, err
			}
		}

		accounts = append(accounts, &account)
	}

	return accounts, rows.Err()
}
