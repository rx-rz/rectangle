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

func (h *Handler) StartGoogleOauth(w http.ResponseWriter, r *http.Request) {
	state := rand.Text()
	codeVerifier, codeChallenge := generatePKCE()
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_code_verifier",
		Value:    codeVerifier,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	authUrl := buildGoogleAuthURL(GoogleConfig{
		ClientID:    h.service.cfg.GoogleClientID,
		RedirectURI: h.service.cfg.GoogleRedirectURI,
	}, state, codeChallenge)

	if err := helpers.WriteData(w, http.StatusOK, GoogleOauthLinkOutput{
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
	tokenResp, err := exchangeGoogleCode(r.Context(), GoogleConfig{
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

	result, err := h.service.LoginWithGoogle(r.Context(), GoogleOAuthInput{
		ProviderUserID: googleUser.ID,
		Email:          googleUser.Email,
		EmailVerified:  googleUser.EmailVerified,
		Name:           optionalString(googleUser.Name),
		AvatarURL:      optionalString(googleUser.Picture),
		UserAgent:      r.UserAgent(),
		IPAddress:      clientIP(r),
	})
	if err != nil {
		h.logger.Error("failed to complete google login", "error", err)
		redirectOAuthError(w, r, h.service.cfg.WebAppURL, "google_login_failed")
		return
	}

	setSessionCookie(w, result.Token, result.Session.ExpiresAt, h.service.cfg.AppEnv == "production")
	clearCookie(w, "oauth_state")
	clearCookie(w, "oauth_code_verifier")
	http.Redirect(w, r, h.service.cfg.WebAppURL, http.StatusFound)
}

type GoogleConfig struct {
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

func buildGoogleAuthURL(cfg GoogleConfig, state string, codeChallenge string) string {
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

func exchangeGoogleCode(ctx context.Context, cfg GoogleConfig, code, codeVerifier string) (*GoogleTokenResponse, error) {
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

type GoogleUser struct {
	ID            string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
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
