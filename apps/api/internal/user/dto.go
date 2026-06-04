package user

import (
	"time"
)

type CreateUserParams struct {
	ID           string
	Name         *string
	Email        string
	PasswordHash string
}

type UpdateUserParams struct {
	ID        string
	Name      *string
	AvatarURL *string
	Email     *string
}

type UserResponse struct {
	ID              string     `json:"id"`
	Name            *string    `json:"name,omitempty"`
	Email           string     `json:"email"`
	AvatarURL       *string    `json:"avatar_url,omitempty"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

func ToUserResponse(u User) UserResponse {
	var name *string
	if u.Name.Valid {
		name = &u.Name.String
	}

	var avatarURL *string
	if u.AvatarURL.Valid {
		avatarURL = &u.AvatarURL.String
	}

	var emailVerifiedAt *time.Time
	if u.EmailVerifiedAt.Valid {
		emailVerifiedAt = &u.EmailVerifiedAt.Time
	}

	return UserResponse{
		ID:              u.ID,
		Name:            name,
		Email:           u.Email,
		AvatarURL:       avatarURL,
		EmailVerifiedAt: emailVerifiedAt,
		CreatedAt:       u.CreatedAt,
	}
}
