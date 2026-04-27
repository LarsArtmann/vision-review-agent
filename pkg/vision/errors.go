package vision

import "errors"

var (
	// ErrNoModel is returned when no language model is configured.
	ErrNoModel = errors.New("vision agent: no model configured")

	// ErrEmptyPrompt is returned when the prompt is empty.
	ErrEmptyPrompt = errors.New("vision agent: prompt cannot be empty")

	// ErrNoImages is returned when no images are provided for analysis.
	ErrNoImages = errors.New("vision agent: at least one image is required")

	// ErrInvalidTemperature is returned when temperature is out of range.
	ErrInvalidTemperature = errors.New("vision agent: temperature must be between 0.0 and 2.0")

	// ErrInvalidMaxTokens is returned when max tokens is negative.
	ErrInvalidMaxTokens = errors.New("vision agent: max output tokens cannot be negative")
)
