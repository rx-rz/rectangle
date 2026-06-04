package user

import (
	"database/sql"
	"time"
)

type User struct {
	ID              string         `db:"id"`
	Name            sql.NullString `db:"name"`
	Email           string         `db:"email"`
	PasswordHash    sql.NullString `db:"password_hash"`
	AvatarURL       sql.NullString `db:"avatar_url"`
	EmailVerifiedAt sql.NullTime   `db:"email_verified_at"`
	CreatedAt       time.Time      `db:"created_at"`
}
