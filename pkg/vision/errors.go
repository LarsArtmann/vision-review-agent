package vision

import (
	apperrors "github.com/larsartmann/vision-review-agent/pkg/errors"
)

// Re-export domain errors for backwards compatibility.
var (
	// ErrNoModel is returned when no language model is configured.
	ErrNoModel = apperrors.ErrNoModel

	// ErrEmptyPrompt is returned when the prompt is empty.
	ErrEmptyPrompt = apperrors.ErrEmptyPrompt

	// ErrNoImages is returned when no images are provided for analysis.
	ErrNoImages = apperrors.ErrNoImages

	// ErrInvalidTemperature is returned when temperature is out of range.
	ErrInvalidTemperature = apperrors.ErrInvalidTemperature

	// ErrInvalidMaxTokens is returned when max tokens is negative.
	ErrInvalidMaxTokens = apperrors.ErrInvalidMaxTokens

	// ErrInvalidTopP is returned when top-p is out of range.
	ErrInvalidTopP = apperrors.ErrInvalidTopP

	// ErrInvalidTopK is returned when top-k is negative.
	ErrInvalidTopK = apperrors.ErrInvalidTopK

	// ErrInvalidPresencePenalty is returned when presence penalty is out of range.
	ErrInvalidPresencePenalty = apperrors.ErrInvalidPresencePenalty

	// ErrInvalidFrequencyPenalty is returned when frequency penalty is out of range.
	ErrInvalidFrequencyPenalty = apperrors.ErrInvalidFrequencyPenalty

	// ErrInvalidImage is returned when the data does not match any known image format.
	ErrInvalidImage = apperrors.ErrInvalidImage

	// ErrEmptyImageData is returned when image data is empty.
	ErrEmptyImageData = apperrors.ErrEmptyImageData

	// ErrImageTooLarge is returned when image data exceeds the maximum allowed size.
	ErrImageTooLarge = apperrors.ErrImageTooLarge
)
