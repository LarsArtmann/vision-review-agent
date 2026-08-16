package reviewed

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"
)

// Pipeline dependency sentinels: NewPipeline rejects nil wiring with these.
var (
	ErrNoReviewer  = errors.New("pipeline: nil reviewer")
	ErrNoStore     = errors.New("pipeline: nil store")
	ErrNoBlobStore = errors.New("pipeline: nil blob store")
	ErrNoWriter    = errors.New("pipeline: nil writer")
)

// PassResult counts what one pass over all configured projects did.
type PassResult struct { // Projects is the number of projects scanned.
	Projects int

	// Views is the number of distinct views seen across all projects.
	Views int

	// Captured is the number of new captures archived and recorded.
	Captured int

	// Skipped is the number of views whose current file hash already matched
	// their last capture.
	Skipped int

	// Reviewed is the number of model reviews completed.
	Reviewed int

	// Compared is the number of BEFORE→AFTER comparisons completed.
	Compared int
}

// Pipeline performs review passes over configured projects: scan for
// screenshots, archive new captures, compare them against their predecessor,
// review them with the model, and write the markdown output.
type Pipeline struct {
	reviewer *Reviewer
	store    *Store
	blobs    *BlobStore
	writer   *Writer
	logger   *slog.Logger
}

// NewPipeline wires a pass runner. All review dependencies must be non-nil; a
// nil logger falls back to the default logger.
func NewPipeline(
	reviewer *Reviewer,
	store *Store,
	blobs *BlobStore,
	writer *Writer,
	logger *slog.Logger,
) (*Pipeline, error) {
	if reviewer == nil {
		return nil, fmt.Errorf("new pipeline: %w", ErrNoReviewer)
	}

	if store == nil {
		return nil, fmt.Errorf("new pipeline: %w", ErrNoStore)
	}

	if blobs == nil {
		return nil, fmt.Errorf("new pipeline: %w", ErrNoBlobStore)
	}

	if writer == nil {
		return nil, fmt.Errorf("new pipeline: %w", ErrNoWriter)
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &Pipeline{reviewer: reviewer, store: store, blobs: blobs, writer: writer, logger: logger}, nil
}

// Pass scans every project and processes each changed view once. Per-view
// failures (model errors, write errors) are collected and joined into the
// returned error after every project was processed, so one broken view never
// blocks the others.
func (p *Pipeline) Pass(ctx context.Context, projects map[string][]string) (PassResult, error) {
	result := PassResult{
		Projects: 0,
		Views:    0,
		Captured: 0,
		Skipped:  0,
		Reviewed: 0,
		Compared: 0,
	}

	var errs []error

	for _, project := range sortedProjectNames(projects) {
		if err := ctx.Err(); err != nil {
			errs = append(errs, fmt.Errorf("project %s: skipped, pass context done: %w", project, err))

			break
		}

		result.Projects++

		captures, err := ScanProject(projects[project])
		if err != nil {
			errs = append(errs, fmt.Errorf("project %s: %w", project, err))

			continue
		}

		projectResult, err := p.passProject(ctx, project, captures)
		result.Views += projectResult.Views
		result.Captured += projectResult.Captured
		result.Skipped += projectResult.Skipped
		result.Reviewed += projectResult.Reviewed
		result.Compared += projectResult.Compared

		if err != nil {
			errs = append(errs, fmt.Errorf("project %s: %w", project, err))
		}
	}

	return result, errors.Join(errs...)
}

// passProject processes one project's captures and refreshes its INDEX.
func (p *Pipeline) passProject(ctx context.Context, project string, captures []Capture) (PassResult, error) {
	result := PassResult{
		Projects: 0,
		Views:    len(captures),
		Captured: 0,
		Skipped:  0,
		Reviewed: 0,
		Compared: 0,
	}

	var errs []error

	for _, capture := range captures {
		if err := ctx.Err(); err != nil {
			errs = append(errs, fmt.Errorf("view %s: skipped, pass context done: %w", capture.ViewKey, err))

			break
		}

		state, _, err := p.store.LoadView(ctx, project, capture.ViewKey)
		if err != nil {
			errs = append(errs, fmt.Errorf("view %s: load: %w", capture.ViewKey, err))

			continue
		}

		if state.Captures > 0 && state.SHA256 == capture.SHA256 {
			result.Skipped++

			continue
		}

		captured, err := p.ingest(ctx, project, capture)
		if err != nil {
			errs = append(errs, err)

			continue
		}

		result.Captured++

		if state.BlobPath != "" && state.SHA256 != captured.SHA256 {
			if compareErr := p.autoCompare(ctx, project, capture.ViewKey, state, captured); compareErr != nil {
				p.logger.Warn("comparison failed; review continues",
					"view", capture.ViewKey.String(), "err", compareErr)
				errs = append(errs, compareErr)
			} else {
				result.Compared++
			}
		}

		if reviewErr := p.review(ctx, project, capture.ViewKey, captured); reviewErr != nil {
			p.logger.Warn("review failed", "view", capture.ViewKey.String(), "err", reviewErr)
			errs = append(errs, reviewErr)

			continue
		}

		result.Reviewed++
	}

	// A cancelled pass skips the INDEX refresh on purpose: folding the store
	// under a dead context would only append noise to the error list.
	if ctx.Err() == nil {
		if err := p.refreshIndex(ctx, project, captures); err != nil {
			errs = append(errs, err)
		}
	}

	return result, errors.Join(errs...)
}

// ingest archives the capture's file in the blob store and records the
// view.captured event, using the hash the blob store actually stored.
func (p *Pipeline) ingest(ctx context.Context, project string, capture Capture) (Captured, error) {
	sha, blobPath, err := p.blobs.Store(capture.Path)
	if err != nil {
		return Captured{}, fmt.Errorf("view %s: store blob: %w", capture.ViewKey, err)
	}

	captured := Captured{
		SourcePath: capture.Path,
		BlobPath:   blobPath,
		SHA256:     sha,
		CapturedAt: capture.ModifiedAt.UTC(),
	}

	if err := p.store.RecordCapture(ctx, project, capture.ViewKey, captured); err != nil {
		return Captured{}, fmt.Errorf("view %s: record capture: %w", capture.ViewKey, err)
	}

	return captured, nil
}

// autoCompare judges the BEFORE (previous capture) against the AFTER (just
// ingested capture) and records plus writes the comparison.
func (p *Pipeline) autoCompare(
	ctx context.Context,
	project string,
	viewKey ViewKey,
	before ViewState,
	after Captured,
) error {
	outcome, err := p.reviewer.Compare(ctx, viewKey, before.BlobPath, after.BlobPath)
	if err != nil {
		return fmt.Errorf("view %s: compare: %w", viewKey, err)
	}

	compared := Compared{
		BeforeSHA256:   before.SHA256,
		BeforeBlobPath: before.BlobPath,
		AfterSHA256:    after.SHA256,
		AfterBlobPath:  after.BlobPath,
		Model:          p.reviewer.Model(),
		Markdown:       outcome.Markdown,
		ComparedAt:     time.Now().UTC(),
	}

	if err := p.store.RecordComparison(ctx, project, viewKey, compared); err != nil {
		return fmt.Errorf("view %s: record comparison: %w", viewKey, err)
	}

	content := RenderComparison(project, viewKey, compared)

	if err := p.writer.WriteComparison(project, viewKey, compared.ComparedAt, content); err != nil {
		return fmt.Errorf("view %s: write comparison: %w", viewKey, err)
	}

	return nil
}

// review asks the model to judge the archived capture, records the
// view.reviewed event, and writes the view's review markdown.
func (p *Pipeline) review(ctx context.Context, project string, viewKey ViewKey, captured Captured) error {
	outcome, err := p.reviewer.Review(ctx, viewKey, captured.BlobPath)
	if err != nil {
		return fmt.Errorf("view %s: review: %w", viewKey, err)
	}

	reviewed := Reviewed{
		SHA256:     captured.SHA256,
		Model:      p.reviewer.Model(),
		Markdown:   outcome.Markdown,
		Score:      outcome.Score,
		ReviewedAt: time.Now().UTC(),
	}

	if err := p.store.RecordReview(ctx, project, viewKey, reviewed); err != nil {
		return fmt.Errorf("view %s: record review: %w", viewKey, err)
	}

	content := RenderViewReview(project, viewKey, captured, reviewed)

	if err := p.writer.WriteViewReview(project, viewKey, content); err != nil {
		return fmt.Errorf("view %s: write review: %w", viewKey, err)
	}

	return nil
}

// refreshIndex rewrites the project's INDEX.md from the folded state of every
// scanned view.
func (p *Pipeline) refreshIndex(ctx context.Context, project string, captures []Capture) error {
	rows := make([]IndexRow, 0, len(captures))

	for _, capture := range captures {
		state, _, err := p.store.LoadView(ctx, project, capture.ViewKey)
		if err != nil {
			return fmt.Errorf("view %s: load for index: %w", capture.ViewKey, err)
		}

		rows = append(rows, IndexRow{
			ViewKey:   capture.ViewKey,
			Score:     state.LastScore,
			Previous:  state.PrevScore,
			UpdatedAt: state.UpdatedAt(),
		})
	}

	if err := p.writer.WriteIndex(project, RenderIndex(project, indexStamp(rows), rows)); err != nil {
		return fmt.Errorf("write index: %w", err)
	}

	return nil
}

// sortedProjectNames returns project names in deterministic order.
func sortedProjectNames(projects map[string][]string) []string {
	names := make([]string, 0, len(projects))

	for name := range projects {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}
