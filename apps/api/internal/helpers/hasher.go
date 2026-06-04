package helpers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	memory      uint32 = 64 * 1024 //64mb
	iterations  uint32 = 3
	parallelism uint8  = 2
	saltLength  uint32 = 16
	keyLength   uint32 = 32
)

var (
	ErrInvalidPassword = errors.New("invalid password")
	ErrInvalidHash     = errors.New("invalid hash")
)

type Hasher struct{}

func NewHasher() *Hasher {
	return &Hasher{}
}

func (h *Hasher) Hash(plain string) (string, error) {
	salt, err := randomBytes(saltLength)
	if err != nil {
		return "", err
	}
	hash := argon2.IDKey(
		[]byte(plain),
		salt,
		iterations,
		memory,
		parallelism,
		keyLength,
	)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)

	return encodedSalt + "." + encodedHash, nil
}

func (h *Hasher) Compare(encodedHash string, plain string) error {
	salt, hash, err := decodeHash(encodedHash)
	if err != nil {
		return err
	}

	otherHash := argon2.IDKey(
		[]byte(plain),
		salt,
		iterations,
		memory,
		parallelism,
		keyLength,
	)

	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		return nil
	}
	return ErrInvalidPassword
}

func HashOTPCode(code string, secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(code))
	return mac.Sum(nil)
}

func VerifyOTPCode(code string, hash []byte, secret string) bool {
	expected := HashOTPCode(code, secret)
	return hmac.Equal(expected, hash)
}

func randomBytes(length uint32) ([]byte, error) {
	bytes := make([]byte, length)

	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	return bytes, nil
}

func decodeHash(encodedHash string) ([]byte, []byte, error) {
	parts := strings.Split(encodedHash, ".")
	if len(parts) != 2 {
		return nil, nil, ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, nil, ErrInvalidHash
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, nil, ErrInvalidHash
	}

	if len(salt) != int(saltLength) {
		return nil, nil, ErrInvalidHash
	}

	if len(hash) != int(keyLength) {
		return nil, nil, ErrInvalidHash
	}

	return salt, hash, nil
}
