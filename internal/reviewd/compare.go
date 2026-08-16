package reviewed

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// CompareManually runs a one-off BEFORE→AFTER comparison of two arbitrary
// screenshots of the same view: both files are archived in the blob store,
// the model judges the pair, the view.compared event is recorded, and the
// comparison markdown is written. The view key is derived from the AFTER
// file name. The project INDEX is not touched; the next pass (or replay)
// refreshes it.
func (p *Pipeline) CompareManually(
	ctx context.Context,
	project string,
	beforePath string,
	afterPath string,
) error {
	viewKey, err := viewKeyOfPath(afterPath)
	if err != nil {
		return err
	}

	beforeSHA, beforeBlob, err := p.blobs.Store(beforePath)
	if err != nil {
		return fmt.Errorf("view %s: store before blob: %w", viewKey, err)
	}

	afterSHA, afterBlob, err := p.blobs.Store(afterPath)
	if err != nil {
		return fmt.Errorf("view %s: store after blob: %w", viewKey, err)
	}

	outcome, err := p.reviewer.Compare(ctx, viewKey, beforeBlob, afterBlob)
	if err != nil {
		return fmt.Errorf("view %s: compare: %w", viewKey, err)
	}

	compared := Compared{
		BeforeSHA256:   beforeSHA,
		BeforeBlobPath: beforeBlob,
		AfterSHA256:    afterSHA,
		AfterBlobPath:  afterBlob,
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

// viewKeyOfPath derives a view key from a screenshot file name.
func viewKeyOfPath(path string) (ViewKey, error) {
	viewKey, err := ParseViewKey(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	if err != nil {
		return ViewKey{}, fmt.Errorf("compare: derive view key from %s: %w", path, err)
	}

	return viewKey, nil
}
