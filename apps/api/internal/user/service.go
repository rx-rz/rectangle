package user

import (
	"context"
	"errors"
	"log/slog"
	"rx-rz/rectangle-api/internal/apperror"
	"rx-rz/rectangle-api/internal/config"
	"rx-rz/rectangle-api/internal/helpers"
)

type UserRepository interface {
	Create(ctx context.Context, params CreateUserParams) (*User, error)
	Update(ctx context.Context, params UpdateUserParams) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
}

type UserService struct {
	userRepo UserRepository
	cfg      config.Config
	logger   *slog.Logger
}

type ServiceOptions struct {
	UserRepository UserRepository
	Config         config.Config
	Logger         *slog.Logger
}

func (s *UserService) FindOrCreateUser(ctx context.Context, input OauthSignupInput) (*User, error) {
	existingUser, err := s.userRepo.FindByEmail(ctx, input.Email)

	if err == nil && existingUser == nil {
		id := helpers.NewUserID()
		hashedPassword, err := helpers.NewHasher().Hash(input.Password)
		if err != nil {
			switch {
			case errors.Is(err, helpers.ErrInvalidPassword):
				return nil, apperror.BadRequest("invalid password")
			case errors.Is(err, helpers.ErrInvalidHash):
				return nil, apperror.Internal()
			default:
				return nil, apperror.Internal()
			}
		}
		user, err := s.userRepo.Create(ctx, CreateUserParams{
			ID:           id,
			Name:         input.Name,
			Email:        input.Email,
			PasswordHash: hashedPassword,
			AvatarURL:    input.AvatarURL,
		})

		if err != nil {
			return nil, err
		}
		return user, nil
	}
	if err != nil {
		return nil, err
	}
	return existingUser, nil
}
