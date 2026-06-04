package auth

import "errors"

var (
	ErrOTPNotFound        = errors.New("otp not found")
	ErrOTPExpired         = errors.New("otp expired")
	ErrOTPInvalid         = errors.New("otp invalid")
	ErrOTPTooManyAttempts = errors.New("too many otp attempts")
)
