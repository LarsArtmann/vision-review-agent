package reviewed

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsbbolt "github.com/larsartmann/go-cqrs-lite/storage/bbolt/v4"
)

type spikeCaptured struct {
	Path string `json:"path"`
	SHA  string `json:"sha"`
}

type spikeState struct {
	Captures int    `json:"captures"`
	LastSHA  string `json:"lastSha"`
}

func applySpike(state spikeState, evt event.Event) (spikeState, error) {
	switch evt.Type() {
	case "view.captured":
		p, err := event.DecodePayloadAuto[spikeCaptured](evt)
		if err != nil {
			return state, err
		}

		state.Captures++
		state.LastSHA = p.SHA
	}

	return state, nil
}

// TestSpikeEventStore proves the bbolt event store + decider repository round
// trip works end to end before the real domain is built on top of it.
func TestSpikeEventStore(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "events.bbolt")

	backend, err := cqrsbbolt.Open(path, slog.Default())
	if err != nil {
		_ = t.Fatalf("open bbolt backend: %v", err)
	}
	defer backend.Close()

	repo, err := decider.NewRepository(backend.EventStore(), nil, decider.Decider[spikeState]{
		Initial: spikeState{},
		Apply:   applySpike,
	})
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	streamID, err := id.ParseStreamID("discordsync:Settings--dark--desktop")
	if err != nil {
		t.Fatalf("parse stream id: %v", err)
	}

	streamType, err := id.ParseStreamType("View")
	if err != nil {
		t.Fatalf("parse stream type: %v", err)
	}

	err = repo.Execute(ctx, streamID, streamType, func(_ spikeState, version event.Version) ([]event.Event, error) {
		return event.Single("view.captured", streamID, streamType, version.Increment(),
			spikeCaptured{Path: "/tmp/a.png", SHA: "abc123"})
	})
	if err != nil {
		t.Fatalf("execute capture: %v", err)
	}

	err = repo.Execute(ctx, streamID, streamType, func(_ spikeState, version event.Version) ([]event.Event, error) {
		return event.Single("view.captured", streamID, streamType, version.Increment(),
			spikeCaptured{Path: "/tmp/b.png", SHA: "def456"})
	})
	if err != nil {
		t.Fatalf("execute second capture: %v", err)
	}

	state, version, err := repo.Load(ctx, streamID, streamType)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if state.Captures != 2 {
		t.Fatalf("captures: got %d, want 2", state.Captures)
	}

	if state.LastSHA != "def456" {
		t.Fatalf("last sha: got %q, want %q", state.LastSHA, "def456")
	}

	if version != 2 {
		t.Fatalf("version: got %d, want 2", version)
	}

	events, err := backend.EventStore().Load(ctx, id.NewStreamRef(streamType, streamID))
	if err != nil {
		t.Fatalf("load raw events: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("raw events: got %d, want 2", len(events))
	}

	if events[0].Type() != "view.captured" {
		t.Fatalf("event type: got %q", events[0].Type())
	}
}
