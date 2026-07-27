package apperrors

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"charm.land/fantasy"
)

// HTTP status codes used for error classification.
// Defined as named constants rather than net/http imports to keep the
// classification logic self-documenting and free of magic numbers.
const (
	statusUnauthorized         = 401
	statusForbidden            = 403
	statusNotFound             = 404
	statusRequestTimeout       = 408
	statusTooManyRequests      = 429
	statusServerError          = 500
	statusNotImplemented       = 501
	statusServiceUnavailable   = 503
	statusBadRequest           = 400
	statusContentFilterSignals = 400 // some providers use 400 for content-filter
)

// maxPromptDisplayLen is the maximum number of characters of the prompt
// included in a ModelError's Error() output. Longer prompts are truncated so
// error messages stay readable in logs and terminals.
const maxPromptDisplayLen = 100

// ErrorKind classifies the category of an error produced while invoking an AI
// model. It lets consumers make decisions (retry vs. fix-input vs. give up)
// without reaching into the underlying provider SDK.
type ErrorKind string

const (
	// KindRateLimited: the provider returned 429 Too Many Requests.
	// Transient — retry with backoff.
	KindRateLimited ErrorKind = "rate_limited"

	// KindTimeout: the request exceeded a deadline (local timeout or 408).
	// Transient — retry, possibly with a longer timeout.
	KindTimeout ErrorKind = "timeout"

	// KindServerError: the provider returned a 5xx response (excluding 501/503
	// which have their own kinds). Transient — retry with backoff.
	KindServerError ErrorKind = "server_error"

	// KindNotImplemented: the provider returned 501 Not Implemented. The model
	// or feature does not exist on this endpoint. Not retryable — use a
	// different model or provider.
	KindNotImplemented ErrorKind = "not_implemented"

	// KindServiceUnavailable: the provider returned 503 Service Unavailable.
	// The service is temporarily overloaded or down. Transient — retry with
	// backoff (usually longer than for generic 5xx).
	KindServiceUnavailable ErrorKind = "service_unavailable"

	// KindNetwork: a transport-level failure with no HTTP status code
	// (e.g. connection reset, unexpected EOF mid-stream).
	// Transient — retry.
	KindNetwork ErrorKind = "network_error"

	// KindAuthentication: the provider rejected credentials (401/403).
	// Not retryable — fix the API key or permissions.
	KindAuthentication ErrorKind = "authentication"

	// KindNotFound: the requested model or resource does not exist (404).
	// Not retryable — check the model name.
	KindNotFound ErrorKind = "not_found"

	// KindBadRequest: the provider rejected the request payload (400).
	// Not retryable — fix the request parameters.
	KindBadRequest ErrorKind = "bad_request"

	// KindContentFilter: the provider's content policy rejected the request or
	// response. Some providers return 400 with a content-filter message; others
	// use 200 with a content_filter finish reason. Not retryable without
	// changing the prompt or image.
	KindContentFilter ErrorKind = "content_filter"

	// KindContextTooLarge: the input exceeded the model's context window.
	// Not retryable without reducing input size.
	KindContextTooLarge ErrorKind = "context_too_large"

	// KindCancelled: the caller cancelled the context.
	// Not retryable unless the caller chooses to re-issue.
	KindCancelled ErrorKind = "cancelled"

	// KindStructuredParse: structured (JSON object) generation failed to
	// produce or parse a valid result.
	// Not retryable without changing the prompt or schema.
	KindStructuredParse ErrorKind = "structured_parse_error"

	// KindUnknown: the error could not be classified into any of the above.
	KindUnknown ErrorKind = "unknown"
)

// ModelError is a classified error returned by model-invoking operations
// (Analyze, AnalyzeStream, AnalyzeStructured). It preserves the original
// cause via Unwrap so errors.Is and errors.AsType continue to work, while
// exposing a domain-level ErrorKind for consumer decision-making.
//
// Consumers can extract it with:
//
//	var me *apperrors.ModelError
//	if errors.AsType(err, &me) {
//	    if me.IsRetryable() { ... back off and retry ... }
//	}
type ModelError struct {
	// Kind is the classified category of the error.
	Kind ErrorKind

	// Op is the high-level operation that failed
	// (e.g. "analyze", "stream", "structured generate").
	Op string

	// Prompt is the user prompt that triggered the failed call,
	// included for debugging. Truncated in Error() output.
	Prompt string

	// StatusCode is the HTTP status code if the error originated from a
	// provider response. Zero when not applicable.
	StatusCode int

	// Cause is the original underlying error.
	Cause error
}

// Error implements the error interface.
func (e *ModelError) Error() string {
	cause := ""
	if e.Cause != nil {
		cause = e.Cause.Error()
	}

	prompt := truncatePrompt(e.Prompt)

	if e.Op != "" {
		return fmt.Sprintf(
			"%s failed [%s] (prompt=%q): %s",
			e.Op,
			e.Kind,
			prompt,
			cause,
		)
	}

	return fmt.Sprintf("[%s] (prompt=%q): %s", e.Kind, prompt, cause)
}

// Unwrap returns the underlying cause, enabling errors.Is and errors.AsType
// to traverse through a ModelError to the original provider or context error.
func (e *ModelError) Unwrap() error {
	return e.Cause
}

// IsRetryable reports whether a retry of the same operation might succeed.
// Rate limits, timeouts, server errors, service-unavailability, and network
// failures are retryable; not-implemented, authentication, bad-request,
// content-filter, context-too-large, cancellation, and parse errors are not.
func (e *ModelError) IsRetryable() bool {
	switch e.Kind {
	case KindRateLimited, KindTimeout, KindServerError, KindServiceUnavailable, KindNetwork:
		return true
	case KindNotImplemented, KindAuthentication, KindNotFound, KindBadRequest,
		KindContentFilter, KindContextTooLarge, KindCancelled, KindStructuredParse, KindUnknown:
		return false
	}

	return false
}

// truncatePrompt shortens a prompt to maxPromptDisplayLen characters for
// display in error messages.
func truncatePrompt(prompt string) string {
	if len(prompt) <= maxPromptDisplayLen {
		return prompt
	}

	return prompt[:maxPromptDisplayLen] + "..."
}

// Wrap creates a ModelError with an explicit kind, for cases where the
// classification is known at the call site (e.g. a JSON unmarshal failure
// is always KindStructuredParse).
func Wrap(kind ErrorKind, op, prompt string, cause error) *ModelError {
	return &ModelError{
		Kind:       kind,
		Op:         op,
		Prompt:     prompt,
		StatusCode: 0,
		Cause:      cause,
	}
}

// classified creates a ModelError with a kind and cause, leaving Op, Prompt,
// and StatusCode at their zero values for call sites to populate.
func classified(kind ErrorKind, cause error) *ModelError {
	return &ModelError{
		Kind:       kind,
		Op:         "",
		Prompt:     "",
		StatusCode: 0,
		Cause:      cause,
	}
}

// Classify inspects an error returned by a model invocation and returns a
// ModelError whose Kind reflects the category of failure. It understands the
// error types produced by charm.land/fantasy (ProviderError,
// NoObjectGeneratedError, RetryError) as well as context cancellation and
// deadline errors.
//
// The returned ModelError is not yet annotated with Op or Prompt; call sites
// should set those fields (or use the vision package's internal classify
// helper) before returning the error to the consumer.
func Classify(err error) *ModelError {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.Canceled) {
		return classified(KindCancelled, err)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return classified(KindTimeout, err)
	}

	if _, ok := errors.AsType[*fantasy.NoObjectGeneratedError](err); ok {
		return classified(KindStructuredParse, err)
	}

	if providerErr, ok := errors.AsType[*fantasy.ProviderError](err); ok {
		return classifyProviderError(providerErr, err)
	}

	return classified(KindUnknown, err)
}

// classifyProviderError maps a fantasy.ProviderError to a domain ErrorKind
// based on its HTTP status code and context-too-large flag.
func classifyProviderError(providerErr *fantasy.ProviderError, err error) *ModelError {
	modelErr := &ModelError{
		Kind:       KindUnknown,
		Op:         "",
		Prompt:     "",
		StatusCode: providerErr.StatusCode,
		Cause:      err,
	}

	if providerErr.IsContextTooLarge() {
		modelErr.Kind = KindContextTooLarge

		return modelErr
	}

	switch providerErr.StatusCode {
	case statusUnauthorized, statusForbidden:
		modelErr.Kind = KindAuthentication
	case statusNotFound:
		modelErr.Kind = KindNotFound
	case statusTooManyRequests:
		modelErr.Kind = KindRateLimited
	case statusRequestTimeout:
		modelErr.Kind = KindTimeout
	case statusNotImplemented:
		modelErr.Kind = KindNotImplemented
	case statusServiceUnavailable:
		modelErr.Kind = KindServiceUnavailable
	case statusBadRequest:
		if isContentFilterRejection(providerErr) {
			modelErr.Kind = KindContentFilter
		} else {
			modelErr.Kind = KindBadRequest
		}
	default:
		modelErr.Kind = kindFromStatusOrRetryability(providerErr)
	}

	return modelErr
}

// kindFromStatusOrRetryability handles status codes outside the explicit
// switch above: 5xx (except 501/503 which have their own kinds) maps to
// KindServerError, a missing status code with a retryable transport failure
// maps to KindNetwork, and everything else falls back to KindUnknown.
func kindFromStatusOrRetryability(providerErr *fantasy.ProviderError) ErrorKind {
	switch {
	case providerErr.StatusCode >= statusServerError:
		return KindServerError
	case providerErr.StatusCode == 0 && providerErr.IsRetryable():
		return KindNetwork
	default:
		return KindUnknown
	}
}

// isContentFilterRejection checks whether a 400 ProviderError is actually a
// content-policy rejection by scanning its message for known signal phrases.
func isContentFilterRejection(providerErr *fantasy.ProviderError) bool {
	signals := []string{
		"content filter",
		"content policy",
		"content_filter",
		"safety",
	}

	msg := strings.ToLower(providerErr.Message)
	for _, signal := range signals {
		if strings.Contains(msg, signal) {
			return true
		}
	}

	return false
}

// IsRetryable reports whether an error — possibly wrapped in one or more
// ModelErrors — represents a transient failure worth retrying. It first
// checks for a classified ModelError, then falls back to the underlying
// provider's own retryability signal.
func IsRetryable(err error) bool {
	if me, ok := errors.AsType[*ModelError](err); ok {
		return me.IsRetryable()
	}

	if providerErr, ok := errors.AsType[*fantasy.ProviderError](err); ok {
		return providerErr.IsRetryable()
	}

	return false
}
