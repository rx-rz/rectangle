package auth

import (
	"context"
	"database/sql"
	"errors"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateOTP(ctx context.Context, email string, otpHash []byte) error {
	query := `
	INSERT INTO otps (email, otp_hash, expires_at)
	VALUES ($1, $2, now() + interval '30 minutes')
	`
	_, err := r.db.ExecContext(ctx, query, email, otpHash)
	return err
}

func (r *Repository) GetOTPByEmail(ctx context.Context, email string, otpHash []byte, purpose OTPPurpose) (*OTP, error) {
	query := `
	SELECT id, user_id, email, otp_hash, purpose, expires_at, created_at, consumed_at, attempts
	FROM otps
	WHERE email = $1 
		AND otp_hash = $2
		AND purpose = $3
	ORDER BY created_at DESC
	LIMIT 1
	`
	var otp OTP
	args := []any{email, otpHash, purpose}
	err := r.db.GetContext(ctx, &otp, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOTPNotFound
		}
		return nil, err
	}
	return &otp, nil
}
