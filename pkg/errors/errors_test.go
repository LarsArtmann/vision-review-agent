package apperrors

import (
	"errors"
	"testing"
)

func TestErrors(t *testing.T) {

	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{
			name:    "ErrNoModel",
			err:     ErrNoModel,
			wantMsg: "vision agent: no model configured",
		},
		{
			name:    "ErrEmptyPrompt",
			err:     ErrEmptyPrompt,
			wantMsg: "vision agent: prompt cannot be empty",
		},
		{
			name:    "ErrNoImages",
			err:     ErrNoImages,
			wantMsg: "vision agent: at least one image is required",
		},
		{
			name:    "ErrInvalidTemperature",
			err:     ErrInvalidTemperature,
			wantMsg: "vision agent: temperature must be between 0.0 and 2.0",
		},
		{
			name:    "ErrInvalidMaxTokens",
			err:     ErrInvalidMaxTokens,
			wantMsg: "vision agent: max output tokens cannot be negative",
		},
		{
			name:    "ErrInvalidImage",
			err:     ErrInvalidImage,
			wantMsg: "vision agent: data does not appear to be a valid image",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.err == nil {
				t.Error("expected non-nil error")
			}
			if tt.err.Error() != tt.wantMsg {
				t.Errorf("expected %q, got %q", tt.wantMsg, tt.err.Error())
			}
		})
	}
}

func TestErrorsIs(t *testing.T) {

	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{
			name:   "ErrNoModel matches itself",
			err:    ErrNoModel,
			target: ErrNoModel,
			want:   true,
		},
		{
			name:   "wrapped ErrEmptyPrompt matches",
			err:    errors.New("wrap: " + ErrEmptyPrompt.Error()),
			target: ErrEmptyPrompt,
			want:   false,
		},
		{
			name:   "different errors don't match",
			err:    ErrNoModel,
			target: ErrNoImages,
			want:   false,
		},
		{
			name:   "fmt.Errorf wrap with %w matches",
			err:    errors.New("vision agent: no model configured"),
			target: ErrNoModel,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := errors.Is(tt.err, tt.target); got != tt.want {
				t.Errorf("errors.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}
