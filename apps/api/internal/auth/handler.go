package auth

import (
	"log/slog"
	"net/http"
	"rx-rz/rectangle-api/internal/apperror"
	"rx-rz/rectangle-api/internal/helpers"
	"rx-rz/rectangle-api/internal/user"
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

	err = helpers.WriteJSON(w, http.StatusCreated, helpers.Envelope{
		"data": AuthResponse{
			User: user.ToUserResponse(*createdUser),
		},
	}, nil)
	if err != nil {
		h.logger.Error("failed to write signup response", "error", err)
	}
}

func (h *Handler) SendOTP(w http.ResponseWriter, r *http.Request) {
	var input SendOTPInput
	if err := helpers.ReadJSON(w, r, &input); err != nil {
		helpers.WriteError(w, apperror.BadRequest(err.Error()))
		return
	}
	err := h.service.SendOTP(r.Context(), SendOTPParams{Email: input.Email})
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	err = helpers.WriteJSON(w, http.StatusCreated, helpers.Envelope{
		"message": "OTP sent successfully",
	}, nil)
	if err != nil {
		h.logger.Error("failed to write response", "error", err)
	}
}
