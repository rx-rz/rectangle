package auth

import "rx-rz/rectangle-api/internal/user"

type EmailSignupInput struct {
	Name     *string `json:"name,omitempty" validate:"omitempty,min=2,max=255"`
	Email    string  `json:"email" validate:"required,email,max=255"`
	Password string  `json:"password" validate:"required,min=8,max=72"`
}

type EmailLoginInput struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required"`
}

type VerifyOTPInput struct {
	Email string `json:"email" validate:"required,email,max=255"`
	Code  string `json:"code" validate:"required,len=6,numeric"`
}

type SendOTPInput struct {
	Email string `json:"email" validate:"required,email,max=255"`
}

type SendOTPParams struct {
	Email string
}

type VerifyOTPParams struct {
	Email string
	Code  string
}

type CreateOTPParams struct {
	UserID  string
	Email   string
	Purpose OTPPurpose
	OtpHash []byte
}

type AuthResponse struct {
	User user.UserResponse `json:"user"`
	// Session AuthSessionResponse `json:"session,omitempty"`
}
