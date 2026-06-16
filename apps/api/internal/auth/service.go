package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"log/slog"
	"rx-rz/rectangle-api/internal/apperror"
	"rx-rz/rectangle-api/internal/config"
	"rx-rz/rectangle-api/internal/helpers"
	"rx-rz/rectangle-api/internal/user"
	"rx-rz/rectangle-api/platform/mail"
	"time"
)

type UserRepository interface {
	Create(ctx context.Context, params user.CreateUserParams) (*user.User, error)
	Update(ctx context.Context, params user.UpdateUserParams) (*user.User, error)
	FindByEmail(ctx context.Context, email string) (*user.User, error)
	GetPasswordHashByEmail(ctx context.Context, email string) (string, error)
}

type OTPRepository interface {
	CreateOTP(ctx context.Context, params CreateOTPParams) error
	GetOTPByEmail(ctx context.Context, email string, purpose OTPPurpose) (*OTP, error)
	IncrementOTPAttempts(ctx context.Context, otpID int64) error
	VerifyEmailWithOTP(ctx context.Context, userID string) (*user.User, error)
}

type OAuthRepository interface {
	FindOrCreateOAuthUserWithSession(ctx context.Context, params CreateOAuthSessionParams) (*SessionResult, error)
}

type SessionRepository interface {
	CreateSession(ctx context.Context, params CreateSessionParams) (*Session, error)
}

type AuthService struct {
	userRepo    UserRepository
	otpRepo     OTPRepository
	oauthRepo   OAuthRepository
	sessionRepo SessionRepository
	mailer      OTPMailer
	cfg         config.Config
	logger      *slog.Logger
}

type OTPMailer interface {
	SendOTPMail(ctx context.Context, data mail.OTPEmailParams, to string) error
	SplitOTP(code string) mail.OTPDigits
}

type ServiceOptions struct {
	UserRepository    UserRepository
	OTPRepository     OTPRepository
	OAuthRepository   OAuthRepository
	SessionRepository SessionRepository
	Mailer            OTPMailer
	Config            config.Config
	Logger            *slog.Logger
}

func NewService(opts ServiceOptions) *AuthService {
	return &AuthService{
		userRepo:    opts.UserRepository,
		otpRepo:     opts.OTPRepository,
		oauthRepo:   opts.OAuthRepository,
		sessionRepo: opts.SessionRepository,
		mailer:      opts.Mailer,
		cfg:         opts.Config,
		logger:      opts.Logger,
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

func (s *AuthService) LoginWithEmail(ctx context.Context, input EmailLoginInput) (*SessionResult, error) {
	existingUser, err := s.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		return nil, err
	}
	if existingUser == nil {
		return nil, apperror.Unauthorized("invalid email or password")
	}

	passwordHash, err := s.userRepo.GetPasswordHashByEmail(ctx, input.Email)
	if err != nil {
		return nil, err
	}
	err = helpers.NewHasher().Compare(passwordHash, input.Password)
	if err != nil {
		switch {
		case errors.Is(err, helpers.ErrInvalidPassword):
			return nil, apperror.Unauthorized("invalid email or password")
		case errors.Is(err, helpers.ErrInvalidHash):
			return nil, apperror.Internal()
		default:
			return nil, apperror.Internal()
		}
	}
	if !existingUser.EmailVerifiedAt.Valid {
		return nil, apperror.Unauthorized("email is not verified")
	}

	return s.createSessionResult(ctx, existingUser, input.UserAgent, input.IPAddress)
}

func (s *AuthService) SendOTP(ctx context.Context, input SendOTPParams) error {
	otp, err := helpers.GenerateOTP()
	if err != nil {
		return err
	}

	existingUser, err := s.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		return err
	}
	if existingUser == nil {
		return apperror.NotFound("otp not found")
	}

	otpHash := helpers.HashOTPCode(otp, s.cfg.OtpSecret)
	err = s.otpRepo.CreateOTP(ctx, CreateOTPParams{
		UserID:  existingUser.ID,
		Email:   existingUser.Email,
		Purpose: OTPPurposeEmailVerification,
		OtpHash: otpHash,
	})
	if err != nil {
		s.logger.Log(ctx, slog.LevelError, err.Error())
		return err
	}
	if s.mailer == nil {
		return apperror.Internal()
	}
	return s.mailer.SendOTPMail(ctx, mail.OTPEmailParams{
		Digits:       s.mailer.SplitOTP(otp),
		Device:       fallback(input.Device, "Unknown device"),
		RequestedAt:  time.Now().Format("Jan 2, 2006 at 15:04 MST"),
		IPAddress:    fallback(input.IPAddress, "Unavailable"),
		Region:       fallback(input.Region, "Unavailable"),
		DashboardURL: s.cfg.MailerDashboardURL,
		DocsURL:      s.cfg.MailerDocsURL,
		SupportURL:   s.cfg.MailerSupportURL,
	}, existingUser.Email)
}

func (s *AuthService) VerifyOTP(ctx context.Context, input VerifyOTPParams) (*SessionResult, error) {
	dbOtp, err := s.otpRepo.GetOTPByEmail(ctx, input.Email, OTPPurposeEmailVerification)

	if err != nil {
		s.logger.Log(ctx, slog.LevelError, err.Error())
		if errors.Is(err, ErrOTPNotFound) {
			return nil, apperror.NotFound("otp not found")
		}
		return nil, err
	}
	if time.Now().After(dbOtp.ExpiresAt) {
		return nil, apperror.BadRequest("otp expired")
	}
	if dbOtp.Attempts >= 5 {
		return nil, apperror.New("TOO_MANY_OTP_ATTEMPTS", "too many otp attempts", 429)
	}
	if !helpers.VerifyOTPCode(input.Code, dbOtp.OtpHash, s.cfg.OtpSecret) {
		if err := s.otpRepo.IncrementOTPAttempts(ctx, dbOtp.ID); err != nil {
			return nil, err
		}
		return nil, apperror.BadRequest("otp invalid")
	}
	verifiedUser, err := s.otpRepo.VerifyEmailWithOTP(ctx, dbOtp.UserID)
	if err != nil {
		return nil, err
	}
	return s.createSessionResult(ctx, verifiedUser, input.UserAgent, input.IPAddress)
}

func (s *AuthService) LoginWithGoogle(ctx context.Context, input GoogleOAuthInput) (*SessionResult, error) {
	if !input.EmailVerified {
		return nil, apperror.Unauthorized("google email is not verified")
	}
	if s.oauthRepo == nil {
		return nil, apperror.Internal()
	}

	token, tokenHash, err := newSessionToken()
	if err != nil {
		return nil, err
	}

	result, err := s.oauthRepo.FindOrCreateOAuthUserWithSession(ctx, CreateOAuthSessionParams{
		UserID:         helpers.NewUserID(),
		SessionID:      helpers.NewSessionID(),
		Provider:       OAuthProviderGoogle,
		ProviderUserID: input.ProviderUserID,
		Email:          input.Email,
		Name:           input.Name,
		AvatarURL:      input.AvatarURL,
		TokenHash:      tokenHash,
		UserAgent:      input.UserAgent,
		IPAddress:      input.IPAddress,
		ExpiresAt:      time.Now().Add(30 * 24 * time.Hour),
	})
	if err != nil {
		if errors.Is(err, ErrOAuthAccountLinked) {
			return nil, apperror.Conflict("oauth account already linked")
		}
		return nil, err
	}
	result.Token = token
	return result, nil
}

func (s *AuthService) createSessionResult(ctx context.Context, dbUser *user.User, userAgent string, ipAddress string) (*SessionResult, error) {
	if s.sessionRepo == nil {
		return nil, apperror.Internal()
	}

	token, tokenHash, err := newSessionToken()
	if err != nil {
		return nil, err
	}

	session, err := s.sessionRepo.CreateSession(ctx, CreateSessionParams{
		SessionID: helpers.NewSessionID(),
		UserID:    dbUser.ID,
		TokenHash: tokenHash,
		UserAgent: userAgent,
		IPAddress: ipAddress,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	})
	if err != nil {
		return nil, err
	}

	return &SessionResult{
		User:    *dbUser,
		Session: *session,
		Token:   token,
	}, nil
}

func newSessionToken() (string, []byte, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", nil, err
	}

	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	hash := sha256.Sum256([]byte(token))
	return token, hash[:], nil
}

func fallback(value, fallbackValue string) string {
	if value == "" {
		return fallbackValue
	}
	return value
}
