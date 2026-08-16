package reviewed

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// ReplayResult counts what a replay rebuilt from the journal.
type ReplayResult struct {
	// Projects is the number of project INDEX files rewritten.
	Projects int

	// Views is the number of distinct view streams folded.
	Views int

	// Reviews is the number of review events re-rendered.
	Reviews int

	// Comparisons is the number of comparison events re-rendered.
	Comparisons int
}

// Replay sentinel errors.
var (
	// ErrMalformedStreamID is returned when a View stream ID cannot be
	// split back into "<project>:<viewKey>".
	ErrMalformedStreamID = errors.New("replay: malformed stream id")

	// ErrReviewWithoutCapture is returned when a view.reviewed event has no
	// preceding view.captured to render its capture metadata from.
	ErrReviewWithoutCapture = errors.New("replay: review without a preceding capture")
)

// streamAddressParts is the segment count of a View stream ID:
// "<project>:<viewKey>".
const streamAddressParts = 2

// streamAddress identifies one view stream by its parsed parts.
type streamAddress struct {
	project string
	viewKey ViewKey
}

// parseStreamAddress splits a View stream ID ("<project>:<viewKey>") back
// into its parts. The view key segment is canonical, so ParseViewKey
// round-trips it even when the page itself contains "--".
func parseStreamAddress(streamID string) (streamAddress, error) {
	parts := strings.SplitN(streamID, ":", streamAddressParts)
	if len(parts) != streamAddressParts {
		return streamAddress{}, fmt.Errorf("%w %q: want <project>:<viewKey>", ErrMalformedStreamID, streamID)
	}

	viewKey, err := ParseViewKey(parts[1])
	if err != nil {
		return streamAddress{}, fmt.Errorf("parse view key in stream id %q: %w", streamID, err)
	}

	return streamAddress{project: parts[0], viewKey: viewKey}, nil
}

// replayStream is the fold state of one view stream during replay.
type replayStream struct {
	address      streamAddress
	state        ViewState
	lastCaptured Captured
}

// Replay rebuilds the reviews directory from the event journal: every
// recorded review and comparison is re-rendered, and every project INDEX is
// regenerated from the folded stream states. Output is deterministic — the
// same journal always replays to byte-identical files, so a wiped reviews
// directory is fully recoverable.
//
// Per-event failures (a decode or write error) are collected and joined
// after everything replayable was written, mirroring the pipeline's
// error-tolerance. Streams that only ever saw a manual comparison also gain
// an INDEX row here, which a pass-time INDEX (built from the filesystem
// scan) would not list; the journal is the source of truth.
func Replay(ctx context.Context, store *Store, writer *Writer) (ReplayResult, error) {
	events, err := store.AllEvents(ctx)
	if err != nil {
		return ReplayResult{}, fmt.Errorf("replay: %w", err)
	}

	result := ReplayResult{Projects: 0, Views: 0, Reviews: 0, Comparisons: 0}

	streams := make(map[string]*replayStream)

	var errs []error

	for _, evt := range events {
		if string(evt.StreamType()) != StreamTypeView {
			continue
		}

		stream, streamErr := replayStreamFor(streams, evt.StreamID().String())
		if streamErr != nil {
			errs = append(errs, streamErr)

			continue
		}

		written, writeErr := replayEvent(writer, stream, evt)
		if writeErr != nil {
			errs = append(errs, writeErr)
		}

		result.Reviews += written.Reviews
		result.Comparisons += written.Comparisons
	}

	writtenProjects, indexErr := replayIndexes(writer, streams)
	if indexErr != nil {
		errs = append(errs, indexErr)
	}

	result.Projects = len(writtenProjects)
	result.Views = len(streams)

	return result, errors.Join(errs...)
}

// replayStreamFor returns the fold state of a stream, creating it on first
// sight.
func replayStreamFor(streams map[string]*replayStream, streamID string) (*replayStream, error) {
	if stream, ok := streams[streamID]; ok {
		return stream, nil
	}

	address, err := parseStreamAddress(streamID)
	if err != nil {
		return nil, err
	}

	stream := &replayStream{
		address: address,
		state:   initialViewState(),
		lastCaptured: Captured{
			SourcePath: "",
			BlobPath:   "",
			SHA256:     "",
			CapturedAt: time.Time{},
		},
	}
	streams[streamID] = stream

	return stream, nil
}

// replayCounts reports what one replayed event wrote.
type replayCounts struct {
	Reviews     int
	Comparisons int
}

// replayEvent folds one event into its stream state and re-renders the
// markdown it originally produced.
func replayEvent(writer *Writer, stream *replayStream, evt event.Event) (replayCounts, error) {
	counts := replayCounts{Reviews: 0, Comparisons: 0}

	state, err := ApplyViewState(stream.state, evt)
	if err != nil {
		return counts, fmt.Errorf("stream %s: %w", evt.StreamID(), err)
	}

	stream.state = state

	switch evt.Type() {
	case EventViewCaptured:
		captured, decodeErr := event.DecodePayloadAuto[Captured](evt)
		if decodeErr != nil {
			return counts, fmt.Errorf("stream %s: decode %s: %w", evt.StreamID(), EventViewCaptured, decodeErr)
		}

		stream.lastCaptured = captured
	case EventViewReviewed:
		review, decodeErr := event.DecodePayloadAuto[Reviewed](evt)
		if decodeErr != nil {
			return counts, fmt.Errorf("stream %s: decode %s: %w", evt.StreamID(), EventViewReviewed, decodeErr)
		}

		if stream.state.Captures == 0 {
			return counts, fmt.Errorf("%w: stream %s", ErrReviewWithoutCapture, evt.StreamID())
		}

		content := RenderViewReview(stream.address.project, stream.address.viewKey, stream.lastCaptured, review)

		writeErr := writer.WriteViewReview(stream.address.project, stream.address.viewKey, content)
		if writeErr != nil {
			return counts, fmt.Errorf("stream %s: write review: %w", evt.StreamID(), writeErr)
		}

		counts.Reviews++
	case EventViewCompared:
		compared, decodeErr := event.DecodePayloadAuto[Compared](evt)
		if decodeErr != nil {
			return counts, fmt.Errorf("stream %s: decode %s: %w", evt.StreamID(), EventViewCompared, decodeErr)
		}

		content := RenderComparison(stream.address.project, stream.address.viewKey, compared)

		writeErr := writer.WriteComparison(
			stream.address.project,
			stream.address.viewKey,
			compared.ComparedAt,
			content,
		)
		if writeErr != nil {
			return counts, fmt.Errorf("stream %s: write comparison: %w", evt.StreamID(), writeErr)
		}

		counts.Comparisons++
	}

	return counts, nil
}

// replayIndexes rewrites every project's INDEX from the folded streams and
// returns the project names written.
func replayIndexes(writer *Writer, streams map[string]*replayStream) ([]string, error) {
	rowsByProject := make(map[string][]IndexRow)

	for _, streamID := range sortedStreamIDs(streams) {
		stream := streams[streamID]
		rowsByProject[stream.address.project] = append(rowsByProject[stream.address.project], IndexRow{
			ViewKey:   stream.address.viewKey,
			Score:     stream.state.LastScore,
			Previous:  stream.state.PrevScore,
			UpdatedAt: stream.state.UpdatedAt(),
		})
	}

	projects := make([]string, 0, len(rowsByProject))

	for project := range rowsByProject {
		projects = append(projects, project)
	}

	sort.Strings(projects)

	var errs []error

	for _, project := range projects {
		rows := rowsByProject[project]

		if err := writer.WriteIndex(project, RenderIndex(project, indexStamp(rows), rows)); err != nil {
			errs = append(errs, fmt.Errorf("project %s: write index: %w", project, err))
		}
	}

	return projects, errors.Join(errs...)
}

// sortedStreamIDs returns stream IDs in deterministic order.
func sortedStreamIDs(streams map[string]*replayStream) []string {
	ids := make([]string, 0, len(streams))

	for streamID := range streams {
		ids = append(ids, streamID)
	}

	sort.Strings(ids)

	return ids
}

// indexStamp picks the deterministic INDEX timestamp: the newest row update
// time. Both the pipeline's INDEX refresh and Replay use it, so a pass and a
// replay of the same journal render byte-identical INDEX files.
func indexStamp(rows []IndexRow) time.Time {
	stamp := time.Time{}

	for _, row := range rows {
		if row.UpdatedAt.After(stamp) {
			stamp = row.UpdatedAt
		}
	}

	return stamp
}

// EventSummary is one journal entry prepared for display: the stream it
// belongs to, what happened, and the hashes and scores involved.
type EventSummary struct {
	Version    event.Version
	OccurredAt time.Time
	Project    string
	ViewKey    ViewKey
	Type       string
	Detail     string
}

// SummarizeEvents turns the raw journal into display summaries. Non-View
// streams are skipped; malformed stream IDs and undecodable payloads
// degrade to a Detail note instead of failing the whole listing.
func SummarizeEvents(events []event.Event) []EventSummary {
	summaries := make([]EventSummary, 0, len(events))

	for _, evt := range events {
		if string(evt.StreamType()) != StreamTypeView {
			continue
		}

		address, err := parseStreamAddress(evt.StreamID().String())
		if err != nil {
			summaries = append(summaries, EventSummary{
				Version:    evt.Version(),
				OccurredAt: evt.OccurredAt(),
				Project:    "",
				ViewKey:    ViewKey{Page: "", Theme: "", Viewport: ""},
				Type:       string(evt.Type()),
				Detail:     fmt.Sprintf("malformed stream: %v", err),
			})

			continue
		}

		summaries = append(summaries, EventSummary{
			Version:    evt.Version(),
			OccurredAt: evt.OccurredAt(),
			Project:    address.project,
			ViewKey:    address.viewKey,
			Type:       string(evt.Type()),
			Detail:     eventDetail(evt),
		})
	}

	return summaries
}

// eventDetail renders the payload-dependent part of an event summary.
func eventDetail(evt event.Event) string {
	switch evt.Type() {
	case EventViewCaptured:
		captured, err := event.DecodePayloadAuto[Captured](evt)
		if err != nil {
			return decodeFailedDetail(err)
		}

		return "sha=" + ShortSHA(captured.SHA256)
	case EventViewReviewed:
		review, err := event.DecodePayloadAuto[Reviewed](evt)
		if err != nil {
			return decodeFailedDetail(err)
		}

		return "sha=" + ShortSHA(review.SHA256) + " score=" + FormatScore(review.Score)
	case EventViewCompared:
		compared, err := event.DecodePayloadAuto[Compared](evt)
		if err != nil {
			return decodeFailedDetail(err)
		}

		return "before=" + ShortSHA(compared.BeforeSHA256) + " after=" + ShortSHA(compared.AfterSHA256)
	default:
		return ""
	}
}

// decodeFailedDetail renders a payload decode failure for display.
func decodeFailedDetail(err error) string {
	return "undecodable payload: " + err.Error()
}
