package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"rx-rz/rectangle-api/internal/apperror"
	"rx-rz/rectangle-api/internal/helpers"
	"rx-rz/rectangle-api/internal/user"
	"strconv"
	"strings"
	"time"
)

type Handler struct {
	service *AuthService
	logger  *slog.Logger
}

type HandlerOptions struct {
	Service *AuthService
	Logger  *slog.Logger
}

func NewHandler(opts HandlerOptions) *Handler {
	return &Handler{
		service: opts.Service,
		logger:  opts.Logger,
	}
}

func (h *Handler) SignupWithEmail(w http.ResponseWriter, r *http.Request) {
	var input EmailSignupInput
	if err := helpers.ReadJSON(w, r, &input); err != nil {
		helpers.WriteError(w, apperror.BadRequest(err.Error()))
		return
	}

	if err := helpers.ValidateStruct(input); err != nil {
		helpers.WriteError(w, err)
		return
	}

	createdUser, err := h.service.SignupWithEmail(r.Context(), input)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	err = helpers.WriteData(w, http.StatusCreated, AuthResponse{
		User: user.ToUserResponse(*createdUser),
	}, nil)
	if err != nil {
		h.logger.Error("failed to write signup response", "error", err)
	}
}

func (h *Handler) LoginWithEmail(w http.ResponseWriter, r *http.Request) {
	var input EmailLoginInput
	if err := helpers.ReadJSON(w, r, &input); err != nil {
		helpers.WriteError(w, apperror.BadRequest(err.Error()))
		return
	}

	if err := helpers.ValidateStruct(input); err != nil {
		helpers.WriteError(w, err)
		return
	}

	input.UserAgent = r.UserAgent()
	input.IPAddress = clientIP(r)

	result, err := h.service.LoginWithEmail(r.Context(), input)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	setSessionCookie(w, result.Token, result.Session.ExpiresAt, h.service.cfg.AppEnv == "production")
	err = helpers.WriteData(w, http.StatusOK, AuthResponse{
		User:    user.ToUserResponse(result.User),
		Session: toAuthSessionResponse(result.Session),
	}, nil)
	if err != nil {
		h.logger.Error("failed to write login response", "error", err)
	}
}

func (h *Handler) SendOTP(w http.ResponseWriter, r *http.Request) {
	var input SendOTPInput
	if err := helpers.ReadJSON(w, r, &input); err != nil {
		helpers.WriteError(w, apperror.BadRequest(err.Error()))
		return
	}
	if err := helpers.ValidateStruct(input); err != nil {
		helpers.WriteError(w, err)
		return
	}
	err := h.service.SendOTP(r.Context(), SendOTPParams{
		Email:     input.Email,
		Device:    r.UserAgent(),
		IPAddress: clientIP(r),
		Region:    "Unavailable",
	})
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	err = helpers.WriteMessage(w, http.StatusCreated, "OTP sent successfully", nil)
	if err != nil {
		h.logger.Error("failed to write response", "error", err)
	}
}

func (h *Handler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var input VerifyOTPInput
	if err := helpers.ReadJSON(w, r, &input); err != nil {
		helpers.WriteError(w, apperror.BadRequest(err.Error()))
		return
	}
	if err := helpers.ValidateStruct(input); err != nil {
		helpers.WriteError(w, err)
		return
	}
	result, err := h.service.VerifyOTP(r.Context(), VerifyOTPParams{
		Email:     input.Email,
		Code:      input.Code,
		UserAgent: r.UserAgent(),
		IPAddress: clientIP(r),
	})
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	setSessionCookie(w, result.Token, result.Session.ExpiresAt, h.service.cfg.AppEnv == "production")
	err = helpers.WriteData(w, http.StatusOK, AuthResponse{
		User:    user.ToUserResponse(result.User),
		Session: toAuthSessionResponse(result.Session),
	}, nil)
	if err != nil {
		h.logger.Error("failed to write response", "error", err)
	}
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	token, err := helpers.ReadCookie(r, "session_token")
	if err != nil {
		helpers.WriteError(w, apperror.Unauthorized("not authenticated"))
		return
	}

	result, err := h.service.Me(r.Context(), token)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	err = helpers.WriteData(w, http.StatusOK, toMeResponse(*result), nil)
	if err != nil {
		h.logger.Error("failed to write me response", "error", err)
	}
}

func (h *Handler) StartGithubOauth(w http.ResponseWriter, r *http.Request) {
	state := rand.Text()
	http.SetCookie(w, &http.Cookie{
		Name:     "github_oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.service.cfg.AppEnv == "production",
		MaxAge:   10 * 60,
	})
	authUrl := buildGithubAuthURL(OauthConfig{
		ClientID:    h.service.cfg.GithubClientID,
		RedirectURI: h.service.cfg.GithubRedirectURI,
	}, state)
	if err := helpers.WriteData(w, http.StatusOK, OauthLinkOutput{
		AuthURL: authUrl,
	}, nil); err != nil {
		h.logger.Error("failed to write response", "error", err)
	}
}

func (h *Handler) StartGoogleOauth(w http.ResponseWriter, r *http.Request) {
	state := rand.Text()
	codeVerifier, codeChallenge := generatePKCE()
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   10 * 60,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_code_verifier",
		Value:    codeVerifier,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   10 * 60,
	})

	authUrl := buildGoogleAuthURL(OauthConfig{
		ClientID:    h.service.cfg.GoogleClientID,
		RedirectURI: h.service.cfg.GoogleRedirectURI,
	}, state, codeChallenge)

	if err := helpers.WriteData(w, http.StatusOK, OauthLinkOutput{
		AuthURL: authUrl,
	}, nil); err != nil {
		h.logger.Error("failed to write response", "error", err)
	}
}

func (h *Handler) HandleGoogleOauth(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		redirectOAuthError(w, r, h.service.cfg.WebAppURL, "missing_oauth_params")
		return
	}
	savedState, err := helpers.ReadCookie(r, "oauth_state")
	if err != nil {
		redirectOAuthError(w, r, h.service.cfg.WebAppURL, "missing_oauth_state")
		return
	}
	codeVerifier, err := helpers.ReadCookie(r, "oauth_code_verifier")
	if err != nil {
		redirectOAuthError(w, r, h.service.cfg.WebAppURL, "missing_code_verifier")
		return
	}
	if state != savedState {
		redirectOAuthError(w, r, h.service.cfg.WebAppURL, "invalid_oauth_state")
		return
	}
	tokenResp, err := exchangeGoogleCode(r.Context(), OauthConfig{
		ClientID:     h.service.cfg.GoogleClientID,
		ClientSecret: h.service.cfg.GoogleClientSecret,
		RedirectURI:  h.service.cfg.GoogleRedirectURI,
	}, code, codeVerifier)
	if err != nil {
		h.logger.Error("failed to exchange google oauth code", "error", err)
		redirectOAuthError(w, r, h.service.cfg.WebAppURL, "google_token_exchange_failed")
		return
	}
	googleUser, err := fetchGoogleUser(r.Context(), tokenResp.AccessToken)
	if err != nil {
		h.logger.Error("failed to fetch google user", "error", err)
		redirectOAuthError(w, r, h.service.cfg.WebAppURL, "google_user_fetch_failed")
		return
	}

	result, err := h.service.LoginWithOAuth(r.Context(), OAuthInput{
		ProviderUserID: googleUser.ID,
		Email:          googleUser.Email,
		EmailVerified:  googleUser.EmailVerified,
		Name:           optionalString(googleUser.Name),
		AvatarURL:      optionalString(googleUser.Picture),
		UserAgent:      r.UserAgent(),
		IPAddress:      clientIP(r),
	}, OAuthProviderGoogle)
	if err != nil {
		h.logger.Error("failed to complete google login", "error", err)
		redirectOAuthError(w, r, h.service.cfg.WebAppURL, oauthRedirectErrorCode(err, "google_login_failed"))
		return
	}

	setSessionCookie(w, result.Token, result.Session.ExpiresAt, h.service.cfg.AppEnv == "production")
	clearCookie(w, "oauth_state")
	clearCookie(w, "oauth_code_verifier")
	http.Redirect(w, r, h.service.cfg.WebAppURL, http.StatusFound)
}

func (h *Handler) HandleGithubOauth(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		redirectOAuthError(w, r, h.service.cfg.WebAppURL, "missing_oauth_params")
		return
	}
	savedState, err := helpers.ReadCookie(r, "github_oauth_state")
	if err != nil {
		redirectOAuthError(w, r, h.service.cfg.WebAppURL, "missing_oauth_params")
		return
	}
	if state != savedState {
		redirectOAuthError(w, r, h.service.cfg.WebAppURL, "invalid_oauth_state")
		return
	}
	tokenResp, err := exchangeGithubCode(r.Context(), OauthConfig{
		ClientID:     h.service.cfg.GithubClientID,
		ClientSecret: h.service.cfg.GithubClientSecret,
		RedirectURI:  h.service.cfg.GithubRedirectURI,
	}, code)
	if err != nil {
		h.logger.Error("failed to exchange github oauth code", "error", err)
		redirectOAuthError(w, r, h.service.cfg.WebAppURL, "github_token_exchange_failed")
		return
	}
	githubUser, err := fetchGithubUser(r.Context(), tokenResp.AccessToken)
	if err != nil {
		h.logger.Error("failed to fetch github user", "error", err)
		redirectOAuthError(w, r, h.service.cfg.WebAppURL, "github_user_fetch_failed")
		return
	}

	if githubUser.Email == "" || !githubUser.EmailVerified {
		email, err := fetchGithubVerifiedEmail(r.Context(), tokenResp.AccessToken)
		if err != nil {
			h.logger.Error("failed to fetch github email", "error", err)
			redirectOAuthError(w, r, h.service.cfg.WebAppURL, "github_email_fetch_failed")
			return
		}
		githubUser.Email = email
		githubUser.EmailVerified = true
	}

	name := githubUser.Name
	if name == "" {
		name = githubUser.Login
	}

	result, err := h.service.LoginWithOAuth(r.Context(), OAuthInput{
		ProviderUserID: strconv.FormatInt(githubUser.ID, 10),
		Email:          githubUser.Email,
		EmailVerified:  githubUser.EmailVerified,
		Name:           optionalString(name),
		AvatarURL:      optionalString(githubUser.AvatarURL),
		UserAgent:      r.UserAgent(),
		IPAddress:      clientIP(r),
	}, OAuthProviderGithub)
	if err != nil {
		h.logger.Error("failed to complete github login", "error", err)
		redirectOAuthError(w, r, h.service.cfg.WebAppURL, oauthRedirectErrorCode(err, "github_login_failed"))
		return
	}

	setSessionCookie(w, result.Token, result.Session.ExpiresAt, h.service.cfg.AppEnv == "production")
	clearCookie(w, "github_oauth_state")
	http.Redirect(w, r, h.service.cfg.WebAppURL, http.StatusFound)
}

type OauthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

func generatePKCE() (verifier string, challenge string) {
	verifier = rand.Text()
	hash := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(hash[:])
	return
}

func buildGithubAuthURL(cfg OauthConfig, state string) string {
	u, _ := url.Parse("https://github.com/login/oauth/authorize")
	q := u.Query()
	q.Set("client_id", cfg.ClientID)
	q.Set("redirect_uri", cfg.RedirectURI)
	q.Set("scope", "read:user user:email")
	q.Set("state", state)
	u.RawQuery = q.Encode()
	return u.String()
}

func buildGoogleAuthURL(cfg OauthConfig, state string, codeChallenge string) string {
	u, _ := url.Parse("https://accounts.google.com/o/oauth2/v2/auth")

	q := u.Query()
	q.Set("client_id", cfg.ClientID)
	q.Set("redirect_uri", cfg.RedirectURI)
	q.Set("response_type", "code")
	q.Set("scope", "openid email profile")
	q.Set("state", state)
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", "S256")
	q.Set("prompt", "select_account")

	u.RawQuery = q.Encode()

	return u.String()
}

type GoogleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
	IDToken      string `json:"id_token,omitempty"`
}

type GitHubTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error,omitempty"`
	ErrorDesc   string `json:"error_description,omitempty"`
}

func exchangeGoogleCode(ctx context.Context, cfg OauthConfig, code, codeVerifier string) (*GoogleTokenResponse, error) {
	form := url.Values{}
	form.Set("code", code)
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)
	form.Set("redirect_uri", cfg.RedirectURI)
	form.Set("grant_type", "authorization_code")
	form.Set("code_verifier", codeVerifier)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://oauth2.googleapis.com/token",
		strings.NewReader(form.Encode()),
	)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return nil, fmt.Errorf("google token exchange failed: status=%d body=%s", res.StatusCode, string(body))
	}

	var tokenResp GoogleTokenResponse
	if err := json.NewDecoder(res.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	if tokenResp.AccessToken == "" {
		return nil, errors.New("google token response missing access_token")
	}

	return &tokenResp, nil
}

func exchangeGithubCode(ctx context.Context, cfg OauthConfig, code string) (*GitHubTokenResponse, error) {
	form := url.Values{}
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)
	form.Set("redirect_uri", cfg.RedirectURI)
	form.Set("code", code)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return nil, fmt.Errorf("github token exchange failed: status=%d body=%s", res.StatusCode, string(body))
	}

	var token GitHubTokenResponse
	if err := json.NewDecoder(res.Body).Decode(&token); err != nil {
		return nil, err
	}

	if token.Error != "" {
		return nil, fmt.Errorf("github token exchange failed: %s: %s", token.Error, token.ErrorDesc)
	}

	if token.AccessToken == "" {
		return nil, errors.New("github returned empty access token")
	}

	return &token, nil
}

type GoogleUser struct {
	ID            string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

type GitHubUser struct {
	ID            int64  `json:"id"`
	Login         string `json:"login"`
	Name          string `json:"name"`
	AvatarURL     string `json:"avatar_url"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"-"`
}

type GitHubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func fetchGoogleUser(ctx context.Context, accessToken string) (*GoogleUser, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"https://openidconnect.googleapis.com/v1/userinfo",
		nil,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return nil, fmt.Errorf("google userinfo failed: status=%d body=%s", res.StatusCode, string(body))
	}

	var user GoogleUser
	if err := json.NewDecoder(res.Body).Decode(&user); err != nil {
		return nil, err
	}

	if user.ID == "" || user.Email == "" {
		return nil, errors.New("google userinfo missing required fields")
	}

	return &user, nil
}

func fetchGithubUser(ctx context.Context, accessToken string) (*GitHubUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf(
			"github user request failed: status=%d body=%s",
			res.StatusCode,
			string(body),
		)
	}

	var user GitHubUser
	if err := json.NewDecoder(res.Body).Decode(&user); err != nil {
		return nil, err
	}
	if user.ID == 0 {
		return nil, errors.New("github user response missing id")
	}
	if user.Email != "" {
		user.EmailVerified = true
	}

	return &user, nil
}

func fetchGithubVerifiedEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return "", fmt.Errorf("github emails request failed: status=%d body=%s", res.StatusCode, string(body))
	}

	var emails []GitHubEmail
	if err := json.NewDecoder(res.Body).Decode(&emails); err != nil {
		return "", err
	}

	for _, email := range emails {
		if email.Primary && email.Verified && email.Email != "" {
			return email.Email, nil
		}
	}

	return "", errors.New("github account has no verified email")
}

func clientIP(r *http.Request) string {
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		return strings.TrimSpace(strings.Split(forwardedFor, ",")[0])
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func toAuthSessionResponse(session Session) *AuthSessionResponse {
	return &AuthSessionResponse{
		ID:        session.ID,
		ExpiresAt: session.ExpiresAt,
	}
}

func toMeResponse(result CurrentSessionResult) MeResponse {
	return MeResponse{
		User: user.ToUserResponse(result.User),
		Connections: ConnectionsResponse{
			Github: GithubConnectionResponse{
				Connected:         result.HasGithubConnection,
				CanImportProjects: result.HasGithubConnection,
			},
		},
	}
}

func setSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
	})
}

func clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	})
}

func redirectOAuthError(w http.ResponseWriter, r *http.Request, webAppURL string, code string) {
	redirectURL := webAppURL + "/auth/login?oauth_error=" + url.QueryEscape(code)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func oauthRedirectErrorCode(err error, fallback string) string {
	appErr := apperror.Convert(err)
	if appErr.Code == "USE_EXISTING_LOGIN_METHOD" {
		return "use_existing_login_method"
	}
	return fallback
}
