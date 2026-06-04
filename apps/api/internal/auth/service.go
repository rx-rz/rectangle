package auth

import (
	"context"
	"errors"
	"log/slog"
	"rx-rz/rectangle-api/internal/apperror"
	"rx-rz/rectangle-api/internal/config"
	"rx-rz/rectangle-api/internal/helpers"
	"rx-rz/rectangle-api/internal/user"
	"time"
)

type UserRepository interface {
	Create(ctx context.Context, params user.CreateUserParams) (*user.User, error)
	Update(ctx context.Context, params user.UpdateUserParams) (*user.User, error)
	FindByEmail(ctx context.Context, email string) (*user.User, error)
	GetPasswordHashByEmail(ctx context.Context, email string) (string, error)
}

type OTPRepository interface {
	CreateOTP(ctx context.Context, email string, otpHash []byte) error
	GetOTPByEmail(ctx context.Context, email string, otpHash []byte, purpose OTPPurpose) (*OTP, error)
}

type AuthService struct {
	userRepo UserRepository
	otpRepo  OTPRepository
	cfg      config.Config
	logger   *slog.Logger
}

type ServiceOptions struct {
	UserRepository UserRepository
	OTPRepository  OTPRepository
	Logger         *slog.Logger
}

func NewService(opts ServiceOptions) *AuthService {
	return &AuthService{
		userRepo: opts.UserRepository,
		otpRepo:  opts.OTPRepository,
		logger:   opts.Logger,
	}
}

func (s *AuthService) SignupWithEmail(ctx context.Context, input EmailSignupInput) (*user.User, error) {
	existingUser, err := s.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, apperror.Conflict("user already exists.")
	}
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
	dto := user.CreateUserParams{
		ID:           id,
		Name:         input.Name,
		Email:        input.Email,
		PasswordHash: hashedPassword,
	}

	user, err := s.userRepo.Create(ctx, dto)
	if err != nil {
		return nil, err
	}
	return user, nil

}

func (s *AuthService) SendOTP(ctx context.Context, input SendOTPParams) error {
	otp, err := helpers.GenerateOTP()
	s.logger.Log(ctx, slog.LevelInfo, otp)
	if err != nil {
		return err
	}
	otpHash := helpers.HashOTPCode(otp, s.cfg.OtpSecret)
	err = s.otpRepo.CreateOTP(ctx, input.Email, (otpHash))
	if err != nil {
		return err
	}
	return nil
}

func (s *AuthService) VerifyOTP(ctx context.Context, input VerifyOTPParams) error {
	otpHash := helpers.HashOTPCode(input.Code, s.cfg.OtpSecret)
	dbOtp, err := s.otpRepo.GetOTPByEmail(ctx, input.Email, otpHash, OTPPurposeEmailVerification)
	if err != nil {
		return err
	}
	if dbOtp.ConsumedAt.Valid {
		return ErrOTPAlreadyUsed
	}
	if time.Now().After(dbOtp.ExpiresAt) {

	}
}
