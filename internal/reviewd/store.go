package reviewed

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsbbolt "github.com/larsartmann/go-cqrs-lite/storage/bbolt/v4"
)

// StreamTypeView is the single stream type visionreviewd records: one View
// stream per project:Page--theme--viewport.
const StreamTypeView = "View"

// ViewState is the folded state of one view stream: what the latest capture
// is, whether the model reviewed it, and the score trend.
type ViewState struct {
	SHA256     string
	BlobPath   string
	CapturedAt time.Time
	Captures   int

	// ReviewedSHA is the hash LastReview belongs to; empty right after a new
	// capture arrived.
	ReviewedSHA string

	LastReview *Reviewed
	LastScore  int
	PrevScore  int
	Reviews    int

	Comparisons int
}

// NeedsReview reports whether the current capture has no review yet.
func (v ViewState) NeedsReview() bool {
	return v.Captures > 0 && v.ReviewedSHA != v.SHA256
}

// UpdatedAt reports when the view's INDEX row last changed: the review time
// when the current capture was reviewed, else its capture time. Both the
// pipeline's INDEX refresh and Replay derive from it, so replay stays
// byte-identical.
func (v ViewState) UpdatedAt() time.Time {
	if v.LastReview != nil && v.ReviewedSHA == v.SHA256 && v.LastReview.ReviewedAt.After(v.CapturedAt) {
		return v.LastReview.ReviewedAt
	}

	return v.CapturedAt
}

// ApplyViewState folds one event into a view state. Unknown event types are
// ignored so old logs stay replayable after new event kinds appear.
func ApplyViewState(state ViewState, evt event.Event) (ViewState, error) {
	switch evt.Type() {
	case EventViewCaptured:
		captured, err := event.DecodePayloadAuto[Captured](evt)
		if err != nil {
			return state, fmt.Errorf("decode %s: %w", EventViewCaptured, err)
		}

		state.SHA256 = captured.SHA256
		state.BlobPath = captured.BlobPath
		state.CapturedAt = captured.CapturedAt
		state.Captures++
		state.ReviewedSHA = ""
	case EventViewReviewed:
		reviewed, err := event.DecodePayloadAuto[Reviewed](evt)
		if err != nil {
			return state, fmt.Errorf("decode %s: %w", EventViewReviewed, err)
		}

		state.ReviewedSHA = reviewed.SHA256
		state.PrevScore = state.LastScore
		state.LastScore = reviewed.Score
		review := reviewed
		state.LastReview = &review
		state.Reviews++
	case EventViewCompared:
		_, err := event.DecodePayloadAuto[Compared](evt)
		if err != nil {
			return state, fmt.Errorf("decode %s: %w", EventViewCompared, err)
		}

		state.Comparisons++
	default:
		return state, nil
	}

	return state, nil
}

// initialViewState returns the explicit zero state every view stream starts
// from: no capture yet and both score slots unknown (not zero).
func initialViewState() ViewState {
	return ViewState{
		SHA256:      "",
		BlobPath:    "",
		CapturedAt:  time.Time{},
		Captures:    0,
		ReviewedSHA: "",
		LastReview:  nil,
		LastScore:   ScoreUnknown,
		PrevScore:   ScoreUnknown,
		Reviews:     0,
		Comparisons: 0,
	}
}

// Store is the event-sourced view history on bbolt.
type Store struct {
	backend *cqrsbbolt.Backend
	repo    *decider.Repository[ViewState]
}

// OpenStore opens (creating if needed) the event store at path.
func OpenStore(path string, logger *slog.Logger) (*Store, error) {
	if err := ensureParentDir(path); err != nil {
		return nil, err
	}

	backend, err := cqrsbbolt.Open(path, logger)
	if err != nil {
		return nil, fmt.Errorf("open event store %s: %w", path, err)
	}

	repo, err := decider.NewRepository(backend.EventStore(), nil, decider.Decider[ViewState]{
		Initial: initialViewState(),
		Apply:   ApplyViewState,
	})
	if err != nil {
		if closeErr := backend.Close(); closeErr != nil {
			return nil, errors.Join(fmt.Errorf("build view repository: %w", err), closeErr)
		}

		return nil, fmt.Errorf("build view repository: %w", err)
	}

	return &Store{backend: backend, repo: repo}, nil
}

// Close releases the underlying bbolt database.
func (s *Store) Close() error {
	if err := s.backend.Close(); err != nil {
		return fmt.Errorf("close event store: %w", err)
	}

	return nil
}

// RecordCapture appends a view.captured event.
func (s *Store) RecordCapture(ctx context.Context, project string, viewKey ViewKey, payload Captured) error {
	return s.record(ctx, project, viewKey, EventViewCaptured, payload)
}

// RecordReview appends a view.reviewed event.
func (s *Store) RecordReview(ctx context.Context, project string, viewKey ViewKey, payload Reviewed) error {
	return s.record(ctx, project, viewKey, EventViewReviewed, payload)
}

// RecordComparison appends a view.compared event.
func (s *Store) RecordComparison(ctx context.Context, project string, viewKey ViewKey, payload Compared) error {
	return s.record(ctx, project, viewKey, EventViewCompared, payload)
}

// LoadView folds a view's stream into its current state.
func (s *Store) LoadView(ctx context.Context, project string, viewKey ViewKey) (ViewState, event.Version, error) {
	streamID, streamType, err := viewStream(project, viewKey)
	if err != nil {
		return ViewState{}, 0, err
	}

	state, version, err := s.repo.Load(ctx, streamID, streamType)
	if err != nil {
		return ViewState{}, 0, fmt.Errorf("load view %s: %w", streamID, err)
	}

	return state, version, nil
}

// ViewEvents returns the raw event history of one view.
func (s *Store) ViewEvents(ctx context.Context, project string, viewKey ViewKey) ([]event.Event, error) {
	streamID, streamType, err := viewStream(project, viewKey)
	if err != nil {
		return nil, err
	}

	events, err := s.backend.EventStore().Load(ctx, id.NewStreamRef(streamType, streamID))
	if err != nil {
		return nil, fmt.Errorf("load events for %s: %w", streamID, err)
	}

	return events, nil
}

// AllEvents reads the complete journal across all streams, ordered by
// occurrence.
func (s *Store) AllEvents(ctx context.Context) ([]event.Event, error) {
	events, err := s.backend.EventStore().ReadAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("read journal: %w", err)
	}

	return events, nil
}

func (s *Store) record(
	ctx context.Context,
	project string,
	viewKey ViewKey,
	eventType string,
	payload any,
) error {
	streamID, streamType, err := viewStream(project, viewKey)
	if err != nil {
		return err
	}

	err = s.repo.Execute(ctx, streamID, streamType, func(_ ViewState, version event.Version) ([]event.Event, error) {
		return event.Single(event.Type(eventType), streamID, streamType, version.Increment(), payload)
	})
	if err != nil {
		return fmt.Errorf("record %s on %s: %w", eventType, streamID, err)
	}

	return nil
}

func viewStream(project string, viewKey ViewKey) (id.StreamID, id.StreamType, error) {
	streamID, err := ViewStreamID(project, viewKey)
	if err != nil {
		var zeroType id.StreamType

		return id.StreamID{}, zeroType, err
	}

	streamType, err := id.ParseStreamType(StreamTypeView)
	if err != nil {
		var zeroType id.StreamType

		return id.StreamID{}, zeroType, fmt.Errorf("parse stream type: %w", err)
	}

	return streamID, streamType, nil
}

func ensureParentDir(path string) error {
	parent := filepath.Dir(path)

	if err := os.MkdirAll(parent, blobDirPermission); err != nil {
		return fmt.Errorf("create event store dir %s: %w", parent, err)
	}

	return nil
}
