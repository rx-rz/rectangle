package integrations

import (
	"log/slog"
	"net/http"
	"net/url"
	"rx-rz/rectangle-api/internal/apperror"
	"rx-rz/rectangle-api/internal/helpers"
)

type Handler struct {
	service *Service
	logger  *slog.Logger
}

type HandlerOptions struct {
	Service *Service
	Logger  *slog.Logger
}

func NewHandler(opts HandlerOptions) *Handler {
	return &Handler{
		service: opts.Service,
		logger:  opts.Logger,
	}
}

func (h *Handler) StartGithubInstallation(w http.ResponseWriter, r *http.Request) {
	if _, err := h.currentUserID(r); err != nil {
		helpers.WriteError(w, err)
		return
	}

	result, err := h.service.StartGithubInstallation(r.Context())
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "github_install_state",
		Value:    result.State,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.service.cfg.AppEnv == "production",
		MaxAge:   10 * 60,
	})

	if err := helpers.WriteData(w, http.StatusOK, result, nil); err != nil {
		h.logger.Error("failed to write github installation start response", "error", err)
	}
}

func (h *Handler) HandleGithubInstallationCallback(w http.ResponseWriter, r *http.Request) {
	redirectURL, err := buildSetupRedirectURL(h.service.cfg.GithubAppIntegrationRedirectSetupURL, r.URL.Query())
	if err != nil {
		helpers.WriteError(w, apperror.BadRequest("invalid github setup redirect url"))
		return
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *Handler) CompleteGithubInstallation(w http.ResponseWriter, r *http.Request) {
	userID, err := h.currentUserID(r)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	var input CompleteGithubInstallationInput
	if err := helpers.ReadJSON(w, r, &input); err != nil {
		helpers.WriteError(w, apperror.BadRequest(err.Error()))
		return
	}
	if err := helpers.ValidateStruct(input); err != nil {
		helpers.WriteError(w, err)
		return
	}

	state, err := helpers.ReadCookie(r, "github_install_state")
	if err != nil {
		helpers.WriteError(w, apperror.Unauthorized("missing github installation state"))
		return
	}
	if input.State != state {
		helpers.WriteError(w, apperror.Unauthorized("invalid github installation state"))
		return
	}

	result, err := h.service.CompleteGithubInstallation(r.Context(), userID, input)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	clearCookie(w, "github_install_state")
	if err := helpers.WriteData(w, http.StatusOK, result, nil); err != nil {
		h.logger.Error("failed to write github installation response", "error", err)
	}
}

func (h *Handler) GetGithubInstallation(w http.ResponseWriter, r *http.Request) {
	userID, err := h.currentUserID(r)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	result, err := h.service.GetGithubInstallation(r.Context(), userID)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	if err := helpers.WriteData(w, http.StatusOK, result, nil); err != nil {
		h.logger.Error("failed to write github installation status response", "error", err)
	}
}

func (h *Handler) GithubWebhook(w http.ResponseWriter, _ *http.Request) {
	// TODO: Verify X-Hub-Signature-256 with cfg.GithubAppIntegrationSecret and handle installation events.
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) currentUserID(r *http.Request) (string, error) {
	token, err := helpers.ReadCookie(r, "session_token")
	if err != nil {
		return "", apperror.Unauthorized("not authenticated")
	}
	return h.service.CurrentUserID(r.Context(), token)
}

func clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

func buildSetupRedirectURL(baseURL string, params url.Values) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	q := u.Query()
	for key, values := range params {
		for _, value := range values {
			q.Add(key, value)
		}
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}
