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

func (r *Repository) CreateOTP(ctx context.Context, params CreateOTPParams) error {
	query := `
	INSERT INTO otps (user_id, email, purpose, otp_hash, expires_at)
	VALUES ($1, $2, $3, $4, now() + interval '30 minutes')
	`
	args := []any{params.UserID, params.Email, params.Purpose, params.OtpHash}
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *Repository) GetOTPByEmail(ctx context.Context, email string, purpose OTPPurpose) (*OTP, error) {
	query := `
	SELECT id, user_id, email, otp_hash, purpose, expires_at, created_at, attempts
	FROM otps
	WHERE lower(email) = lower($1)
		AND purpose = $2
	ORDER BY created_at DESC
	LIMIT 1
	`
	var otp OTP
	args := []any{email, purpose}
	err := r.db.GetContext(ctx, &otp, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOTPNotFound
		}
		return nil, err
	}
	return &otp, nil
}

func (r *Repository) IncrementOTPAttempts(ctx context.Context, otpID int64) error {
	query := `
	UPDATE otps
	SET attempts = attempts + 1
	WHERE id = $1
		AND attempts < 5
	`
	_, err := r.db.ExecContext(ctx, query, otpID)
	return err
}

func (r *Repository) VerifyEmailWithOTP(ctx context.Context, userID string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		UPDATE users
		SET email_verified_at = now(),
			updated_at = now()
		WHERE id = $1
		AND email_verified_at IS NULL
	`, userID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		DELETE FROM otps
		WHERE user_id = $1
		AND purpose = 'email_verification'
	`, userID)
	if err != nil {
		return err
	}
	return tx.Commit()
}
