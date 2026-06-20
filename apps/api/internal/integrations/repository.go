package integrations

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

var ErrGithubInstallationNotFound = errors.New("github installation not found")

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) UpsertGithubInstallation(ctx context.Context, params UpsertGithubInstallationParams) (*GithubInstallation, error) {
	query := `
	INSERT INTO github_installations (
		user_id,
		installation_id,
		account_login,
		github_account_id,
		account_type,
		repository_selection
	)
	VALUES ($1, $2, $3, $4, $5, $6)
	ON CONFLICT (user_id) DO UPDATE SET
		installation_id = EXCLUDED.installation_id,
		account_login = EXCLUDED.account_login,
		github_account_id = EXCLUDED.github_account_id,
		account_type = EXCLUDED.account_type,
		repository_selection = EXCLUDED.repository_selection,
		suspended_at = NULL,
		updated_at = now()
	RETURNING
		id,
		user_id,
		installation_id,
		account_login,
		github_account_id,
		account_type,
		repository_selection,
		suspended_at,
		created_at,
		updated_at
	`

	var installation GithubInstallation
	err := r.db.GetContext(
		ctx,
		&installation,
		query,
		params.UserID,
		params.InstallationID,
		params.AccountLogin,
		params.GithubAccountID,
		params.AccountType,
		params.RepositorySelection,
	)
	if err != nil {
		return nil, err
	}

	return &installation, nil
}

func (r *Repository) FindGithubInstallationByUserID(ctx context.Context, userID string) (*GithubInstallation, error) {
	query := `
	SELECT
		id,
		user_id,
		installation_id,
		account_login,
		github_account_id,
		account_type,
		repository_selection,
		suspended_at,
		created_at,
		updated_at
	FROM github_installations
	WHERE user_id = $1
		AND suspended_at IS NULL
	`

	var installation GithubInstallation
	err := r.db.GetContext(ctx, &installation, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrGithubInstallationNotFound
		}
		return nil, err
	}

	return &installation, nil
}
