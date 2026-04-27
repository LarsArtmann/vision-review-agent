package apperrors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrors(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{name: "ErrNoModel", err: ErrNoModel, wantMsg: "vision agent: no model configured"},
		{name: "ErrEmptyPrompt", err: ErrEmptyPrompt, wantMsg: "vision agent: prompt cannot be empty"},
		{name: "ErrNoImages", err: ErrNoImages, wantMsg: "vision agent: at least one image is required"},
		{name: "ErrInvalidTemperature", err: ErrInvalidTemperature, wantMsg: "vision agent: temperature must be between 0.0 and 2.0"},
		{name: "ErrInvalidMaxTokens", err: ErrInvalidMaxTokens, wantMsg: "vision agent: max output tokens cannot be negative"},
		{name: "ErrInvalidImage", err: ErrInvalidImage, wantMsg: "vision agent: data does not appear to be a valid image"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err)
			assert.EqualError(t, tt.err, tt.wantMsg)
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
		{name: "matches itself", err: ErrNoModel, target: ErrNoModel, want: true},
		{name: "different errors", err: ErrNoModel, target: ErrNoImages, want: false},
		{name: "literal string match", err: errors.New("vision agent: no model configured"), target: ErrNoModel, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, errors.Is(tt.err, tt.target))
		})
	}
}
