package vision

import (
	"context"
	"runtime"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

// BatchResult holds the result of analyzing a single image in a batch operation.
type BatchResult struct {
	// Index is the position of the image in the original input slice.
	Index int

	// Result is the analysis result, or nil if Err is non-nil.
	Result *AnalyzeResult

	// Err is the error encountered for this image, or nil on success.
	Err error

	// Duration is the wall-clock time spent analyzing this image.
	// Zero when the image was skipped (nil input).
	Duration time.Duration
}

// AnalyzeBatch analyzes multiple images concurrently with the same prompt.
// Each image is analyzed independently. Results are returned in the same order
// as the input images, with errors captured per-image rather than failing fast.
//
// Nil images are silently skipped (their BatchResult will have a nil Result and nil Err).
//
// Concurrency is limited to maxConcurrency simultaneous requests.
// If maxConcurrency is 0 or negative, it defaults to runtime.NumCPU().
func (va *Agent) AnalyzeBatch(
	ctx context.Context,
	prompt string,
	maxConcurrency int,
	images ...*ImageSource,
) []BatchResult {
	if maxConcurrency <= 0 {
		maxConcurrency = runtime.NumCPU()
	}

	results := make([]BatchResult, len(images))
	sem := semaphore.NewWeighted(int64(maxConcurrency))

	var wg sync.WaitGroup

	for i, img := range images {
		if img == nil {
			results[i] = BatchResult{Index: i}

			continue
		}

		wg.Add(1)

		go func(index int, image *ImageSource) {
			defer wg.Done()

			_ = sem.Acquire(ctx, 1)
			defer sem.Release(1)

			start := time.Now()
			result, err := va.Analyze(ctx, prompt, image)

			results[index] = BatchResult{
				Index:    index,
				Result:   result,
				Err:      err,
				Duration: time.Since(start),
			}
		}(i, img)
	}

	wg.Wait()

	return results
}
