package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"rx-rz/rectangle-api/internal/apperror"
	"strings"
)

// MAX_JSON_BODY_SIZE is the maximum request body size accepted by ReadJSON.
const MAX_JSON_BODY_SIZE = 1_048_576

// Envelope is the standard top-level shape for JSON responses.
type Envelope map[string]any

// WriteJSON marshals data as an indented JSON response with the given status and headers.
func WriteJSON(w http.ResponseWriter, status int, data Envelope, headers http.Header) error {
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	js = append(js, '\n')

	maps.Copy(w.Header(), headers)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_, err = w.Write(js)
	return err
}

// ReadJSON decodes a single JSON value from the request body into dst.
//
// It rejects unknown fields, empty bodies, bodies larger than MAX_JSON_BODY_SIZE,
// and requests containing more than one JSON value. Invalid destination values
// panic because they indicate a programming error rather than a client error.
func ReadJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, MAX_JSON_BODY_SIZE)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON at character %d", syntaxError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")

		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}

			return fmt.Errorf("body contains incorrect JSON type at character %d", unmarshalTypeError.Offset)

		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s", fieldName)

		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must not be larger than %d bytes", maxBytesError.Limit)

		case errors.As(err, &invalidUnmarshalError):
			panic(err)

		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}

func WriteError(w http.ResponseWriter, err error) error {
	appErr := apperror.Convert(err)
	return WriteJSON(w, appErr.Status, Envelope{
		"error": Envelope{
			"code":    appErr.Code,
			"message": appErr.Message,
		},
	}, nil)
}
