package auth

import (
	"log/slog"
	"net"
	"net/http"
	"rx-rz/rectangle-api/internal/apperror"
	"rx-rz/rectangle-api/internal/helpers"
	"rx-rz/rectangle-api/internal/user"
	"strings"
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

	loggedInUser, err := h.service.LoginWithEmail(r.Context(), input)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}

	err = helpers.WriteData(w, http.StatusOK, AuthResponse{
		User: user.ToUserResponse(*loggedInUser),
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
	err := h.service.VerifyOTP(r.Context(), VerifyOTPParams{Email: input.Email, Code: input.Code})
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	err = helpers.WriteMessage(w, http.StatusOK, "otp verified successfully. your email is now verified", nil)
	if err != nil {
		h.logger.Error("failed to write response", "error", err)
	}
}
