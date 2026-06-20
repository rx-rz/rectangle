package integrations

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/url"
	"rx-rz/rectangle-api/internal/apperror"
	"rx-rz/rectangle-api/internal/auth"
	"rx-rz/rectangle-api/internal/config"
)

type InstallationRepository interface {
	UpsertGithubInstallation(ctx context.Context, params UpsertGithubInstallationParams) (*GithubInstallation, error)
	FindGithubInstallationByUserID(ctx context.Context, userID string) (*GithubInstallation, error)
}

type SessionRepository interface {
	FindCurrentSession(ctx context.Context, tokenHash []byte) (*auth.CurrentSessionResult, error)
}

type GithubInstallationClient interface {
	FetchInstallation(ctx context.Context, installationID int64) (*GithubInstallationDetails, error)
}

type Service struct {
	installationRepo InstallationRepository
	sessionRepo      SessionRepository
	githubClient     GithubInstallationClient
	cfg              config.Config
	logger           *slog.Logger
}

type ServiceOptions struct {
	InstallationRepository InstallationRepository
	SessionRepository      SessionRepository
	GithubClient           GithubInstallationClient
	Config                 config.Config
	Logger                 *slog.Logger
}

func NewService(opts ServiceOptions) *Service {
	return &Service{
		installationRepo: opts.InstallationRepository,
		sessionRepo:      opts.SessionRepository,
		githubClient:     opts.GithubClient,
		cfg:              opts.Config,
		logger:           opts.Logger,
	}
}

func (s *Service) StartGithubInstallation(_ context.Context) (*GithubInstallStartResponse, error) {
	state, err := newState()
	if err != nil {
		return nil, err
	}

	installURL, err := buildInstallURL(s.cfg.GithubAppIntegrationInstallURL, state)
	if err != nil {
		return nil, err
	}

	return &GithubInstallStartResponse{
		InstallURL: installURL,
		State:      state,
	}, nil
}

func (s *Service) CompleteGithubInstallation(ctx context.Context, userID string, input CompleteGithubInstallationInput) (*GithubInstallationResponse, error) {
	if s.githubClient == nil {
		return nil, apperror.Internal()
	}

	details, err := s.githubClient.FetchInstallation(ctx, input.InstallationID)
	if err != nil {
		s.logger.Error("failed to fetch github installation", "error", err)
		return nil, apperror.BadRequest("github installation could not be verified")
	}
	if details.SuspendedAt != nil {
		return nil, apperror.BadRequest("github installation is suspended")
	}

	installation, err := s.installationRepo.UpsertGithubInstallation(ctx, UpsertGithubInstallationParams{
		UserID:              userID,
		InstallationID:      details.InstallationID,
		AccountLogin:        details.AccountLogin,
		GithubAccountID:     details.GithubAccountID,
		AccountType:         details.AccountType,
		RepositorySelection: details.RepositorySelection,
	})
	if err != nil {
		return nil, err
	}

	return toGithubInstallationResponse(installation), nil
}

func (s *Service) GetGithubInstallation(ctx context.Context, userID string) (*GithubInstallationResponse, error) {
	installation, err := s.installationRepo.FindGithubInstallationByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrGithubInstallationNotFound) {
			return &GithubInstallationResponse{
				Connected:         false,
				CanImportProjects: false,
			}, nil
		}
		return nil, err
	}

	return toGithubInstallationResponse(installation), nil
}

func (s *Service) CurrentUserID(ctx context.Context, sessionToken string) (string, error) {
	if sessionToken == "" {
		return "", apperror.Unauthorized("not authenticated")
	}
	if s.sessionRepo == nil {
		return "", apperror.Internal()
	}

	result, err := s.sessionRepo.FindCurrentSession(ctx, hashSessionToken(sessionToken))
	if err != nil {
		if errors.Is(err, auth.ErrSessionNotFound) {
			return "", apperror.Unauthorized("not authenticated")
		}
		return "", err
	}

	return result.User.ID, nil
}

func buildInstallURL(baseURL string, state string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("state", state)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func newState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func hashSessionToken(token string) []byte {
	hash := sha256.Sum256([]byte(token))
	return hash[:]
}

func toGithubInstallationResponse(installation *GithubInstallation) *GithubInstallationResponse {
	installationID := installation.InstallationID
	return &GithubInstallationResponse{
		Connected:           true,
		CanImportProjects:   true,
		InstallationID:      &installationID,
		AccountLogin:        installation.AccountLogin,
		AccountType:         string(installation.AccountType),
		RepositorySelection: string(installation.RepositorySelection),
	}
}
