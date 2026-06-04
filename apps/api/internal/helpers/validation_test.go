package helpers

import (
	"errors"
	"testing"

	"rx-rz/rectangle-api/internal/apperror"
)

type validationTestInput struct {
	Email    string  `json:"email" validate:"required,email"`
	Name     *string `json:"name,omitempty" validate:"omitempty,min=2"`
	Password string  `json:"password" validate:"required,min=8,max=72"`
}

func TestValidateStructUsesJSONFieldNames(t *testing.T) {
	err := ValidateStruct(validationTestInput{
		Email:    "not-an-email",
		Password: "password123",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}

	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Message != "email must be valid" {
		t.Fatalf("expected JSON field name in error, got %q", appErr.Message)
	}
}

func TestValidateStructAllowsOmittedOptionalFields(t *testing.T) {
	err := ValidateStruct(validationTestInput{
		Email:    "dev@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("expected valid input, got %v", err)
	}
}
