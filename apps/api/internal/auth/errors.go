package auth

import "errors"

var (
	ErrOTPNotFound    = errors.New("otp not found")
	ErrOTPAlreadyUsed = errors.New("otp already used")
	ErrOTPExpired     = errors.New("otp expired")
)
