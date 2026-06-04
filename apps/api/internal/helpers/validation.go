package helpers

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"rx-rz/rectangle-api/internal/apperror"
)

var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New()
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "-" {
			return ""
		}
		return name
	})

	return v
}

// ValidateStruct validates a request DTO using its validate tags.
func ValidateStruct(input any) error {
	if err := validate.Struct(input); err != nil {
		var invalidErr *validator.InvalidValidationError
		if errors.As(err, &invalidErr) {
			return apperror.Internal()
		}

		if validationErrors, ok := err.(validator.ValidationErrors); ok && len(validationErrors) > 0 {
			return apperror.BadRequest(validationErrorMessage(validationErrors[0]))
		}

		return apperror.BadRequest("request body is invalid")
	}

	return nil
}

func validationErrorMessage(err validator.FieldError) string {
	field := err.Field()

	switch err.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be valid", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, err.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", field, err.Param())
	case "len":
		return fmt.Sprintf("%s must be %s characters", field, err.Param())
	case "numeric":
		return fmt.Sprintf("%s must be numeric", field)
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}
