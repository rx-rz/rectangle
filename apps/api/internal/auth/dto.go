package auth

import (
	"rx-rz/rectangle-api/internal/user"
	"time"
)

type EmailSignupInput struct {
	Name     *string `json:"name,omitempty" validate:"omitempty,min=2,max=255"`
	Email    string  `json:"email" validate:"required,email,max=255"`
	Password string  `json:"password" validate:"required,min=8,max=72"`
}

type EmailLoginInput struct {
	Email     string `json:"email" validate:"required,email,max=255"`
	Password  string `json:"password" validate:"required"`
	UserAgent string `json:"-"`
	IPAddress string `json:"-"`
}

type VerifyOTPInput struct {
	Email string `json:"email" validate:"required,email,max=255"`
	Code  string `json:"code" validate:"required,len=6,numeric"`
}

type SendOTPInput struct {
	Email string `json:"email" validate:"required,email,max=255"`
}

type GoogleOauthLinkOutput struct {
	AuthURL string `json:"authUrl"`
}

type SendOTPParams struct {
	Email     string
	Device    string
	IPAddress string
	Region    string
}

type VerifyOTPParams struct {
	Email     string
	Code      string
	UserAgent string
	IPAddress string
}

type GoogleOAuthInput struct {
	ProviderUserID string
	Email          string
	EmailVerified  bool
	Name           *string
	AvatarURL      *string
	UserAgent      string
	IPAddress      string
}

type CreateOTPParams struct {
	UserID  string
	Email   string
	Purpose OTPPurpose
	OtpHash []byte
}

type CreateOAuthSessionParams struct {
	UserID         string
	SessionID      string
	Provider       OAuthProvider
	ProviderUserID string
	Email          string
	Name           *string
	AvatarURL      *string
	TokenHash      []byte
	UserAgent      string
	IPAddress      string
	ExpiresAt      time.Time
}

type CreateSessionParams struct {
	SessionID string
	UserID    string
	TokenHash []byte
	UserAgent string
	IPAddress string
	ExpiresAt time.Time
}

type SessionResult struct {
	User    user.User
	Session Session
	Token   string
}

type AuthSessionResponse struct {
	ID        string    `json:"id"`
	ExpiresAt time.Time `json:"expires_at"`
}

type AuthResponse struct {
	User    user.UserResponse    `json:"user"`
	Session *AuthSessionResponse `json:"session,omitempty"`
}
