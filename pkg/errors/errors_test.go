package apperrors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{name: "ErrNoModel", err: ErrNoModel, wantMsg: "vision agent: no model configured"},
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
		{
			name:    "ErrEmptyImageData",
			err:     ErrEmptyImageData,
			wantMsg: "vision agent: image data is empty",
		},
		{
			name:    "ErrImageTooLarge",
			err:     ErrImageTooLarge,
			wantMsg: "vision agent: image data exceeds maximum size",
		},
	}

	for _, tt := range tests {
		// capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Error(t, tt.err)
			require.EqualError(t, tt.err, tt.wantMsg)
		})
	}
}

func TestErrorsIs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{name: "matches itself", err: ErrNoModel, target: ErrNoModel, want: true},
		{name: "different errors", err: ErrNoModel, target: ErrNoImages, want: false},
		{
			name:   "ErrEmptyImageData matches itself",
			err:    ErrEmptyImageData,
			target: ErrEmptyImageData,
			want:   true,
		},
		{
			name:   "ErrImageTooLarge matches itself",
			err:    ErrImageTooLarge,
			target: ErrImageTooLarge,
			want:   true,
		},
		{
			name:   "literal string match",
			err:    errors.New("vision agent: no model configured"),
			target: ErrNoModel,
			want:   false,
		},
	}

	for _, tt := range tests {
		// capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(
				t,
				tt.want,
				errors.Is(tt.err, tt.target),
			) // value match testing
		})
	}
}

// TestSentinelsSurviveErrorfWrapping verifies that every sentinel remains
// matchable through fmt.Errorf("%w") wrapping. Config.Validate() uses this
// pattern to enrich each validation sentinel with the offending value
// (e.g. "got 3.00, want [0.0, 2.0]"). If a sentinel lost errors.Is
// compatibility after wrapping, consumer code that switches on
// errors.Is(err, ErrInvalidTemperature) would silently break.
func TestSentinelsSurviveErrorfWrapping(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		sentinel error
	}{
		{"ErrNoModel", ErrNoModel},
		{"ErrEmptyPrompt", ErrEmptyPrompt},
		{"ErrNoImages", ErrNoImages},
		{"ErrInvalidTemperature", ErrInvalidTemperature},
		{"ErrInvalidMaxTokens", ErrInvalidMaxTokens},
		{"ErrInvalidTopP", ErrInvalidTopP},
		{"ErrInvalidTopK", ErrInvalidTopK},
		{"ErrInvalidPresencePenalty", ErrInvalidPresencePenalty},
		{"ErrInvalidFrequencyPenalty", ErrInvalidFrequencyPenalty},
		{"ErrInvalidImage", ErrInvalidImage},
		{"ErrEmptyImageData", ErrEmptyImageData},
		{"ErrImageTooLarge", ErrImageTooLarge},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			wrapped := fmt.Errorf("%w: got bad-value", tc.sentinel)
			require.ErrorIs(t, wrapped, tc.sentinel,
				"sentinel must survive %%w wrapping for enriched validation errors")
		})
	}
}
