package apperrors_test

import (
	"errors"
	"fmt"

	apperrors "github.com/larsartmann/vision-review-agent/pkg/errors"
)

// Wrap classifies a model failure into a *ModelError whose kind drives
// consumer retry decisions. The generic errors.AsType extracts the
// classified error even through additional wrapping.
func ExampleWrap() {
	err := apperrors.Wrap(
		apperrors.KindRateLimited,
		"Analyze",
		"describe this UI",
		errors.New("429 too many requests"),
	)

	fmt.Println(err)

	modelErr, ok := errors.AsType[*apperrors.ModelError](err)
	fmt.Println("extracted:", ok)
	fmt.Println("kind:", modelErr.Kind)
	fmt.Println("retryable:", modelErr.IsRetryable())
	// Output:
	// Analyze failed [rate_limited] (prompt="describe this UI"): 429 too many requests
	// extracted: true
	// kind: rate_limited
	// retryable: true
}

// IsRetryable answers "should I retry this?" without extracting the
// ModelError first; non-classified errors report false.
func ExampleIsRetryable() {
	classified := apperrors.Wrap(apperrors.KindTimeout, "Analyze", "p", errors.New("deadline exceeded"))
	plain := errors.New("not a model error")

	fmt.Println(apperrors.IsRetryable(classified))
	fmt.Println(apperrors.IsRetryable(plain))
	// Output:
	// true
	// false
}
