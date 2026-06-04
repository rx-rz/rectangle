package user

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, params CreateUserParams) (*User, error) {
	query := `
	INSERT INTO users (id, name, email, password_hash)
	VALUES ($1, $2, $3, $4)
	RETURNING id, name, email, avatar_url, email_verified_at, created_at
	`
	var user User

	args := []any{params.ID, params.Name, params.Email, params.PasswordHash}
	err := r.db.QueryRowxContext(ctx, query, args...).StructScan(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) Update(ctx context.Context, params UpdateUserParams) (*User, error) {
	query := `
	UPDATE users
	SET name = COALESCE($1, name),
		avatar_url = COALESCE($2, avatar_url),
		email = COALESCE($3, email),
		updated_at = NOW()
	WHERE id = $4
	RETURNING id, name, email, avatar_url, email_verified_at, created_at
	`
	var user User
	args := []any{params.Name, params.AvatarURL, params.Email, params.ID}
	err := r.db.QueryRowxContext(ctx, query, args...).StructScan(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (*User, error) {
	query := `
	SELECT id, name, email, avatar_url, email_verified_at, created_at
	FROM users
	WHERE lower(email) = lower($1)
	`

	var user User
	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *Repository) GetPasswordHashByEmail(ctx context.Context, email string) (string, error) {
	query := `
	SELECT password_hash
	FROM users
	WHERE email = $1
	`

	var passwordHash string
	err := r.db.GetContext(ctx, &passwordHash, query, email)
	if err != nil {
		return "", err
	}
	return passwordHash, nil
}
