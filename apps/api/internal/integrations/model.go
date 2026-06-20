package integrations

import (
	"database/sql"
	"time"
)

type GithubAccountType string

const (
	GithubAccountTypeUser         GithubAccountType = "user"
	GithubAccountTypeOrganization GithubAccountType = "organization"
)

type GithubRepositorySelection string

const (
	GithubRepositorySelectionAll      GithubRepositorySelection = "all"
	GithubRepositorySelectionSelected GithubRepositorySelection = "selected"
)

type GithubInstallation struct {
	ID                  int64                     `db:"id"`
	UserID              string                    `db:"user_id"`
	InstallationID      int64                     `db:"installation_id"`
	AccountLogin        string                    `db:"account_login"`
	GithubAccountID     int64                     `db:"github_account_id"`
	AccountType         GithubAccountType         `db:"account_type"`
	RepositorySelection GithubRepositorySelection `db:"repository_selection"`
	SuspendedAt         sql.NullTime              `db:"suspended_at"`
	CreatedAt           time.Time                 `db:"created_at"`
	UpdatedAt           time.Time                 `db:"updated_at"`
}
