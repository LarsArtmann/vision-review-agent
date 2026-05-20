// Package apperrors provides centralized domain-specific errors for the vision review agent.
// All errors use Go 1.13+ stdlib error handling.
package apperrors

import stderrors "errors"

var (
	// ErrNoModel is returned when no language model is configured.
	ErrNoModel = stderrors.New("vision agent: no model configured")

	// ErrEmptyPrompt is returned when the prompt is empty.
	ErrEmptyPrompt = stderrors.New("vision agent: prompt cannot be empty")

	// ErrNoImages is returned when no images are provided for analysis.
	ErrNoImages = stderrors.New("vision agent: at least one image is required")

	// ErrInvalidTemperature is returned when temperature is out of range.
	ErrInvalidTemperature = stderrors.New("vision agent: temperature must be between 0.0 and 2.0")

	// ErrInvalidMaxTokens is returned when max tokens is negative.
	ErrInvalidMaxTokens = stderrors.New("vision agent: max output tokens cannot be negative")

	// ErrInvalidImage is returned when the data does not match any known image format.
	ErrInvalidImage = stderrors.New("vision agent: data does not appear to be a valid image")

	// ErrEmptyImageData is returned when image data is empty.
	ErrEmptyImageData = stderrors.New("vision agent: image data is empty")

	// ErrImageTooLarge is returned when image data exceeds the maximum allowed size.
	ErrImageTooLarge = stderrors.New("vision agent: image data exceeds maximum size")
)
