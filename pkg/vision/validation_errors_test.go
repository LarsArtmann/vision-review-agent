package vision

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestValidationErrorsIncludeOffendingValues verifies that Config.Validate
// wraps each sentinel with the offending value so callers see WHAT was wrong,
// not just THAT it was wrong. Every wrapped error must still satisfy
// errors.Is for its sentinel so existing consumer code keeps working.
func TestValidationErrorsIncludeOffendingValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		config     Config
		sentinel   error
		wantSubstr string
	}{
		{
			name:       "temperature below range",
			config:     Config{Model: testModel(), Temperature: -0.5},
			sentinel:   ErrInvalidTemperature,
			wantSubstr: "got -0.50, want [0.0, 2.0]",
		},
		{
			name:       "temperature above range",
			config:     Config{Model: testModel(), Temperature: 3.0},
			sentinel:   ErrInvalidTemperature,
			wantSubstr: "got 3.00, want [0.0, 2.0]",
		},
		{
			name:       "max output tokens negative",
			config:     Config{Model: testModel(), MaxOutputTokens: -42},
			sentinel:   ErrInvalidMaxTokens,
			wantSubstr: "got -42, want >= 0",
		},
		{
			name:       "top-p above range",
			config:     Config{Model: testModel(), TopP: 1.5},
			sentinel:   ErrInvalidTopP,
			wantSubstr: "got 1.50, want [0.0, 1.0]",
		},
		{
			name:       "top-k negative",
			config:     Config{Model: testModel(), TopK: -5},
			sentinel:   ErrInvalidTopK,
			wantSubstr: "got -5, want >= 0",
		},
		{
			name:       "presence penalty above range",
			config:     Config{Model: testModel(), PresencePenalty: 2.5},
			sentinel:   ErrInvalidPresencePenalty,
			wantSubstr: "got 2.50, want [-2.0, 2.0]",
		},
		{
			name:       "frequency penalty below range",
			config:     Config{Model: testModel(), FrequencyPenalty: -3.0},
			sentinel:   ErrInvalidFrequencyPenalty,
			wantSubstr: "got -3.00, want [-2.0, 2.0]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()

			require.Error(t, err)
			require.True(t, errors.Is(err, tt.sentinel),
				"errors.Is must still match sentinel %v, got %v", tt.sentinel, err)
			require.Contains(t, err.Error(), tt.wantSubstr,
				"error message must include the offending value for diagnosis")
		})
	}
}

// TestNoModelReturnsBareSentinel verifies that the most common configuration
// error (missing model) returns the bare sentinel without wrapping, since
// there is no offending value to include.
func TestNoModelReturnsBareSentinel(t *testing.T) {
	t.Parallel()

	cfg := Config{}
	err := cfg.Validate()
	require.ErrorIs(t, err, ErrNoModel)
	require.Equal(t, ErrNoModel, err, "ErrNoModel should be returned without wrapping")
}
