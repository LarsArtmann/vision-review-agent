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

// Re-export the typed model-error classification system so consumers can
// inspect AI model errors without importing pkg/errors or charm.land/fantasy
// directly.
type (
	// ErrorKind classifies the category of a model invocation failure.
	// See apperrors.ErrorKind for the full list of kinds.
	ErrorKind = apperrors.ErrorKind

	// ModelError is a classified error wrapping an underlying provider or
	// context error with a domain-level ErrorKind. Extract it via
	// errors.AsType to inspect Kind, StatusCode, and IsRetryable().
	ModelError = apperrors.ModelError
)

// Re-export ErrorKind constants for consumer convenience.
const (
	KindRateLimited       = apperrors.KindRateLimited
	KindTimeout           = apperrors.KindTimeout
	KindServerError       = apperrors.KindServerError
	KindNotImplemented    = apperrors.KindNotImplemented
	KindServiceUnavailable = apperrors.KindServiceUnavailable
	KindNetwork           = apperrors.KindNetwork
	KindAuthentication    = apperrors.KindAuthentication
	KindNotFound          = apperrors.KindNotFound
	KindBadRequest        = apperrors.KindBadRequest
	KindContentFilter     = apperrors.KindContentFilter
	KindContextTooLarge   = apperrors.KindContextTooLarge
	KindCancelled         = apperrors.KindCancelled
	KindStructuredParse   = apperrors.KindStructuredParse
	KindUnknown           = apperrors.KindUnknown
)

// Classify inspects an error from a model invocation and returns a classified
// ModelError. See apperrors.Classify for details.
func Classify(err error) *apperrors.ModelError {
	return apperrors.Classify(err)
}

// IsRetryable reports whether an error represents a transient failure worth
// retrying. See apperrors.IsRetryable for details.
func IsRetryable(err error) bool {
	return apperrors.IsRetryable(err)
}

// classifyModelErr wraps a model-invocation error into a classified ModelError
// annotated with the operation name and prompt. It is used at every model
// call-site, preserving the original cause via Unwrap while adding a
// domain-level ErrorKind for consumer decision-making.
func classifyModelErr(op, prompt string, err error) error {
	modelErr := apperrors.Classify(err)
	modelErr.Op = op
	modelErr.Prompt = prompt

	return modelErr
}
