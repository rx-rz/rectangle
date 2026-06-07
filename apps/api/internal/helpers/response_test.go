package helpers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"rx-rz/rectangle-api/internal/apperror"
)

func TestWriteDataUsesSuccessEnvelope(t *testing.T) {
	recorder := httptest.NewRecorder()

	err := WriteData(recorder, http.StatusCreated, Envelope{"id": "usr_123"}, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON, got %v", err)
	}

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}
	if body["status"] != "success" {
		t.Fatalf("expected success status, got %v", body["status"])
	}
	if body["data"] == nil {
		t.Fatal("expected data in response")
	}
}

func TestWriteErrorClassifiesClientAndServerErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus string
	}{
		{
			name:       "client error",
			err:        apperror.BadRequest("bad input"),
			wantStatus: "fail",
		},
		{
			name:       "server error",
			err:        apperror.Internal(),
			wantStatus: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()

			if err := WriteError(recorder, tt.err); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			var body map[string]any
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("expected valid JSON, got %v", err)
			}
			if body["status"] != tt.wantStatus {
				t.Fatalf("expected status %q, got %v", tt.wantStatus, body["status"])
			}
			if body["error"] == nil {
				t.Fatal("expected error in response")
			}
		})
	}
}
