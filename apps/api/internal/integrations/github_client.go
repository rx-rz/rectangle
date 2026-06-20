package integrations

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type GithubAppClient struct {
	appID      int64
	privateKey string
	httpClient *http.Client
}

type GithubAppClientOptions struct {
	AppID      int64
	PrivateKey string
	HTTPClient *http.Client
}

type GithubInstallationDetails struct {
	InstallationID      int64
	AccountLogin        string
	GithubAccountID     int64
	AccountType         GithubAccountType
	RepositorySelection GithubRepositorySelection
	SuspendedAt         *time.Time
}

type githubInstallationResponse struct {
	ID                  int64  `json:"id"`
	RepositorySelection string `json:"repository_selection"`
	SuspendedAt         string `json:"suspended_at"`
	Account             struct {
		Login string `json:"login"`
		ID    int64  `json:"id"`
		Type  string `json:"type"`
	} `json:"account"`
}

func NewGithubAppClient(opts GithubAppClientOptions) *GithubAppClient {
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &GithubAppClient{
		appID:      opts.AppID,
		privateKey: opts.PrivateKey,
		httpClient: httpClient,
	}
}

func (c *GithubAppClient) FetchInstallation(ctx context.Context, installationID int64) (*GithubInstallationDetails, error) {
	jwt, err := c.newJWT()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://api.github.com/app/installations/%d", installationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return nil, fmt.Errorf("github installation request failed: status=%d body=%s", res.StatusCode, string(body))
	}

	var payload githubInstallationResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return nil, err
	}

	suspendedAt, err := parseOptionalTime(payload.SuspendedAt)
	if err != nil {
		return nil, err
	}

	return &GithubInstallationDetails{
		InstallationID:      payload.ID,
		AccountLogin:        payload.Account.Login,
		GithubAccountID:     payload.Account.ID,
		AccountType:         normalizeGithubAccountType(payload.Account.Type),
		RepositorySelection: GithubRepositorySelection(payload.RepositorySelection),
		SuspendedAt:         suspendedAt,
	}, nil
}

func (c *GithubAppClient) newJWT() (string, error) {
	key, err := parsePrivateKey(c.privateKey)
	if err != nil {
		return "", err
	}

	now := time.Now()
	header := map[string]string{
		"alg": "RS256",
		"typ": "JWT",
	}
	claims := map[string]any{
		"iat": now.Add(-1 * time.Minute).Unix(),
		"exp": now.Add(9 * time.Minute).Unix(),
		"iss": strconv.FormatInt(c.appID, 10),
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	unsigned := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(claimsJSON)
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}

	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func parsePrivateKey(value string) (*rsa.PrivateKey, error) {
	normalized := strings.ReplaceAll(value, `\n`, "\n")
	block, _ := pem.Decode(bytes.TrimSpace([]byte(normalized)))
	if block == nil {
		return nil, fmt.Errorf("GITHUB_APP_INTEGRATION_PRIVATE_KEY must be PEM encoded")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	key, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("GITHUB_APP_INTEGRATION_PRIVATE_KEY must be an RSA private key")
	}
	return key, nil
}

func normalizeGithubAccountType(value string) GithubAccountType {
	if strings.EqualFold(value, "organization") {
		return GithubAccountTypeOrganization
	}
	return GithubAccountTypeUser
}

func parseOptionalTime(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
