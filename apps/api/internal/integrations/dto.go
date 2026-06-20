package integrations

type GithubInstallStartResponse struct {
	InstallURL string `json:"installUrl"`
	State      string `json:"state"`
}

type CompleteGithubInstallationInput struct {
	InstallationID int64  `json:"installationId" validate:"required,gt=0"`
	SetupAction    string `json:"setupAction" validate:"required,oneof=install update"`
	State          string `json:"state" validate:"required"`
}

type UpsertGithubInstallationParams struct {
	UserID              string
	InstallationID      int64
	AccountLogin        string
	GithubAccountID     int64
	AccountType         GithubAccountType
	RepositorySelection GithubRepositorySelection
}

type GithubInstallationResponse struct {
	Connected           bool   `json:"connected"`
	CanImportProjects   bool   `json:"canImportProjects"`
	InstallationID      *int64 `json:"installationId,omitempty"`
	AccountLogin        string `json:"accountLogin,omitempty"`
	AccountType         string `json:"accountType,omitempty"`
	RepositorySelection string `json:"repositorySelection,omitempty"`
}
