package auth

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"rx-rz/rectangle-api/internal/user"

	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

var ErrOAuthAccountLinked = errors.New("oauth account already linked to another user")

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
	WHERE email = $1
		AND purpose = $2
	ORDER BY created_at DESC
	LIMIT 1
	`

	var otp OTP

	err := r.db.GetContext(ctx, &otp, query, email, purpose)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get otp by email",
			"email", email,
			"purpose", purpose,
			"error", err,
		)

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

func (r *Repository) VerifyEmailWithOTP(ctx context.Context, userID string) (*user.User, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var dbUser user.User
	err = tx.QueryRowxContext(ctx, `
		UPDATE users
		SET email_verified_at = COALESCE(email_verified_at, now()),
			updated_at = now()
		WHERE id = $1
		RETURNING id, name, email, avatar_url, email_verified_at, created_at
	`, userID).StructScan(&dbUser)
	if err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx, `
		DELETE FROM otps
		WHERE user_id = $1
		AND purpose = 'email_verification'
	`, userID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &dbUser, nil
}

func (r *Repository) FindOrCreateOAuthUserWithSession(ctx context.Context, params CreateOAuthSessionParams) (*SessionResult, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	dbUser, err := findOAuthUser(ctx, tx, params.Provider, params.ProviderUserID)
	if err != nil {
		return nil, err
	}

	if dbUser == nil {
		dbUser, err = createOAuthUser(ctx, tx, params)
		if err != nil {
			return nil, err
		}
		if err := linkOAuthAccount(ctx, tx, params.Provider, params.ProviderUserID, dbUser.ID); err != nil {
			return nil, err
		}
	}

	session, err := createSession(ctx, tx, params, dbUser.ID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &SessionResult{
		User:    *dbUser,
		Session: *session,
	}, nil
}

func findOAuthUser(ctx context.Context, tx *sqlx.Tx, provider OAuthProvider, providerUserID string) (*user.User, error) {
	query := `
	SELECT users.id, users.name, users.email, users.avatar_url, users.email_verified_at, users.created_at
	FROM oauth_accounts
	JOIN users ON users.id = oauth_accounts.user_id
	WHERE oauth_accounts.provider = $1
		AND oauth_accounts.provider_user_id = $2
	`

	var dbUser user.User
	if err := tx.GetContext(ctx, &dbUser, query, provider, providerUserID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &dbUser, nil
}

func createOAuthUser(ctx context.Context, tx *sqlx.Tx, params CreateOAuthSessionParams) (*user.User, error) {
	query := `
	INSERT INTO users (id, name, email, avatar_url, email_verified_at)
	VALUES ($1, $2, $3, $4, now())
	RETURNING id, name, email, avatar_url, email_verified_at, created_at
	`

	var dbUser user.User
	if err := tx.QueryRowxContext(
		ctx,
		query,
		params.UserID,
		params.Name,
		params.Email,
		params.AvatarURL,
	).StructScan(&dbUser); err != nil {
		return nil, err
	}
	return &dbUser, nil
}

func linkOAuthAccount(ctx context.Context, tx *sqlx.Tx, provider OAuthProvider, providerUserID, userID string) error {
	query := `
	INSERT INTO oauth_accounts (provider, provider_user_id, user_id)
	VALUES ($1, $2, $3)
	ON CONFLICT (provider, provider_user_id) DO NOTHING
	`

	result, err := tx.ExecContext(ctx, query, provider, providerUserID, userID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrOAuthAccountLinked
	}
	return nil
}

func createSession(ctx context.Context, tx *sqlx.Tx, params CreateOAuthSessionParams, userID string) (*Session, error) {
	query := `
	INSERT INTO sessions (id, user_id, user_agent, token_hash, ip_address, expires_at)
	VALUES ($1, $2, NULLIF($3, ''), $4, NULLIF($5, '')::inet, $6)
	RETURNING id, user_id, user_agent, ip_address::text AS ip_address, expires_at, created_at
	`

	var session Session
	if err := tx.QueryRowxContext(
		ctx,
		query,
		params.SessionID,
		userID,
		params.UserAgent,
		params.TokenHash,
		params.IPAddress,
		params.ExpiresAt,
	).StructScan(&session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *Repository) CreateSession(ctx context.Context, params CreateSessionParams) (*Session, error) {
	query := `
	INSERT INTO sessions (id, user_id, user_agent, token_hash, ip_address, expires_at)
	VALUES ($1, $2, NULLIF($3, ''), $4, NULLIF($5, '')::inet, $6)
	RETURNING id, user_id, user_agent, ip_address::text AS ip_address, expires_at, created_at
	`

	var session Session
	if err := r.db.QueryRowxContext(
		ctx,
		query,
		params.SessionID,
		params.UserID,
		params.UserAgent,
		params.TokenHash,
		params.IPAddress,
		params.ExpiresAt,
	).StructScan(&session); err != nil {
		return nil, err
	}
	return &session, nil
}
