package auth

import "time"

type OTPPurpose string

const (
	OTPPurposeEmailVerification OTPPurpose = "email_verification"
	OTPPurposePasswordReset     OTPPurpose = "password_reset"
)

type OTP struct {
	ID        int64      `db:"id"`
	UserID    string     `db:"user_id"`
	Email     string     `db:"email"`
	OtpHash   []byte     `db:"otp_hash"`
	Purpose   OTPPurpose `db:"purpose"`
	ExpiresAt time.Time  `db:"expires_at"`
	CreatedAt time.Time  `db:"created_at"`
	Attempts  int        `db:"attempts"`
}
