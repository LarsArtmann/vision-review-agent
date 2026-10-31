package reviewed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// flippingEndpoint is an OpenAI-compatible fake that can be toggled healthy
// and unhealthy between passes. /v1/models backs the health probe;
// /v1/chat/completions backs the real reviews and counts every chat call so
// the test can prove the gated pass never reached the model.
type flippingEndpoint struct {
	mu      sync.Mutex
	healthy bool
	chats   int
}

func newFlippingEndpoint(healthy bool) *flippingEndpoint {
	return &flippingEndpoint{healthy: healthy}
}

func (e *flippingEndpoint) setHealthy(healthy bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.healthy = healthy
}

func (e *flippingEndpoint) chatCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.chats
}

func (e *flippingEndpoint) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch {
	case strings.HasSuffix(req.URL.Path, "/models"):
		if !e.isHealthy() {
			http.Error(w, "endpoint down", http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(
			map[string]any{"data": []map[string]any{{"id": "fake-vision-model"}}},
		); err != nil {
			http.Error(w, "encode listing: "+err.Error(), http.StatusInternalServerError)
		}
	case strings.HasSuffix(req.URL.Path, "/chat/completions"):
		e.mu.Lock()
		e.chats++
		e.mu.Unlock()

		if !e.isHealthy() {
			http.Error(w, "endpoint down", http.StatusInternalServerError)

			return
		}

		var request fakeChatRequest
		if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
			http.Error(w, "decode request: "+err.Error(), http.StatusBadRequest)

			return
		}

		completion := fakeCompletion{
			ID:      "chatcmpl-flipping",
			Object:  "chat.completion",
			Created: 1,
			Model:   request.Model,
			Choices: []fakeChoice{{
				Index:        0,
				Message:      fakeMessage{Role: "assistant", Content: "## Review\nLooks fine.\n\n**Score: 6/10**"},
				FinishReason: "stop",
			}},
			Usage: fakeUsage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(completion); err != nil {
			http.Error(w, "encode completion: "+err.Error(), http.StatusInternalServerError)
		}
	default:
		http.Error(w, fmt.Sprintf("unexpected %s %s", req.Method, req.URL.Path), http.StatusNotFound)
	}
}

func (e *flippingEndpoint) isHealthy() bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.healthy
}

// writePNGFile writes a minimal valid PNG (one pixel of the given shade) so
// ScanProject, the blob store, and the vision loader all accept it.
func writePNGFile(t *testing.T, path string, shade uint8) {
	t.Helper()

	image := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	image.Set(0, 0, color.NRGBA{R: shade, G: shade, B: shade, A: 0xff})

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}

	defer file.Close()

	if err := png.Encode(file, image); err != nil {
		t.Fatalf("encode %s: %v", path, err)
	}
}

// countMarkdownFiles walks root counting .md files (reviews + INDEX).
func countMarkdownFiles(t *testing.T, root string) int {
	t.Helper()

	count := 0

	walkErr := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !entry.IsDir() && filepath.Ext(path) == ".md" {
			count++
		}

		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, os.ErrNotExist) {
		t.Fatalf("walk %s: %v", root, walkErr)
	}

	return count
}

// countReviewedEvents decodes every view.reviewed event payload in the
// journal.
func countReviewedEvents(t *testing.T, store *Store) (int, []string) {
	t.Helper()

	events, err := store.AllEvents(t.Context())
	if err != nil {
		t.Fatalf("read events: %v", err)
	}

	shas := make([]string, 0, len(events))

	for _, evt := range events {
		if evt.Type() != EventViewReviewed {
			continue
		}

		review, decodeErr := event.DecodePayloadAuto[Reviewed](evt)
		if decodeErr != nil {
			t.Fatalf("decode review payload: %v", decodeErr)
		}

		shas = append(shas, review.SHA256)
	}

	return len(shas), shas
}

// TestHealthGateEndToEndDieSkipRecover runs the REAL pipeline behind the
// REAL gate against one fake OpenAI-compatible endpoint that dies under a
// changed view and recovers. It proves the shipped behavior: the unhealthy
// pass is skipped whole (no model call, no writes, one warning), nothing is
// lost, and the recovered pass reviews exactly what was deferred while
// unchanged views stay skip-seen.
func TestHealthGateEndToEndDieSkipRecover(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	dataDir := filepath.Join(base, "data")
	reviewsDir := filepath.Join(base, "reviews")
	shots := filepath.Join(base, "shots")

	if err := os.MkdirAll(shots, 0o700); err != nil {
		t.Fatalf("mkdir shots: %v", err)
	}

	view1 := filepath.Join(shots, "Home--dark--desktop.png")
	writePNGFile(t, view1, 0x40)

	endpoint := newFlippingEndpoint(true)
	server := httptest.NewServer(endpoint)
	t.Cleanup(server.Close)

	config := fakeModelConfig(server.URL)

	ctx := context.Background()

	reviewer, err := NewReviewerFromConfig(ctx, config)
	if err != nil {
		t.Fatalf("build reviewer: %v", err)
	}

	store, err := OpenStore(JournalPath(dataDir), slog.Default())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	defer closeStoreQuietly(store)

	pipeline, err := NewPipeline(reviewer, store, NewBlobStore(dataDir), NewWriter(reviewsDir), slog.Default())
	if err != nil {
		t.Fatalf("build pipeline: %v", err)
	}

	var logs bytes.Buffer

	gate, err := NewHealthGate(pipeline, func(ctx context.Context) error {
		return ProbeModelEndpoint(ctx, config, EndpointProbeTimeout)
	}, slog.New(slog.NewTextHandler(&logs, nil)))
	if err != nil {
		t.Fatalf("build gate: %v", err)
	}

	projects := map[string][]string{"app": {shots + "/*.png"}}

	// Pass 1: healthy endpoint — view1 captured and reviewed.
	result, err := gate.Pass(ctx, projects)
	if err != nil {
		t.Fatalf("pass 1: %v", err)
	}

	if result.Captured != 1 || result.Reviewed != 1 || result.Skipped != 0 {
		t.Fatalf("pass 1 result = %+v, want captured 1, reviewed 1, skipped 0", result)
	}

	afterPass1 := countMarkdownFiles(t, reviewsDir)

	reviewCount, _ := countReviewedEvents(t, store)
	if reviewCount != 1 {
		t.Fatalf("pass 1: %d reviewed events, want 1", reviewCount)
	}

	// Endpoint dies and a second view appears; its bytes must survive the
	// skipped pass untouched.
	endpoint.setHealthy(false)

	view2 := filepath.Join(shots, "Settings--light--desktop.png")
	writePNGFile(t, view2, 0xc0)

	wantSHA, err := SHA256File(view2)
	if err != nil {
		t.Fatalf("hash view2: %v", err)
	}

	// Pass 2: the gate skips the whole pass.
	result2, err2 := gate.Pass(ctx, projects)
	if !errors.Is(err2, ErrEndpointUnhealthy) {
		t.Fatalf("pass 2 error = %v, want ErrEndpointUnhealthy", err2)
	}

	if result2 != (PassResult{}) {
		t.Fatalf("pass 2 result = %+v, want zero (nothing ran)", result2)
	}

	if endpoint.chatCount() != 1 {
		t.Fatalf("chat calls after pass 2 = %d, want 1 (skipped pass must not reach the model)", endpoint.chatCount())
	}

	if got := countMarkdownFiles(t, reviewsDir); got != afterPass1 {
		t.Fatalf("markdown files after pass 2 = %d, want %d (skipped pass must not write)", got, afterPass1)
	}

	if warnings := strings.Count(logs.String(), "pass skipped, model endpoint unhealthy"); warnings != 1 {
		t.Fatalf("skip warnings = %d, want 1 (logs: %s)", warnings, logs.String())
	}

	// Recovery.
	endpoint.setHealthy(true)

	// Pass 3: view1 unchanged stays skip-seen; deferred view2 is captured and
	// reviewed with exactly the bytes that existed during the skipped pass.
	result3, err3 := gate.Pass(ctx, projects)
	if err3 != nil {
		t.Fatalf("pass 3: %v", err3)
	}

	if result3.Skipped != 1 || result3.Captured != 1 || result3.Reviewed != 1 {
		t.Fatalf("pass 3 result = %+v, want skipped 1, captured 1, reviewed 1", result3)
	}

	if got := countMarkdownFiles(t, reviewsDir); got != afterPass1+1 {
		t.Fatalf("markdown files after pass 3 = %d, want %d", got, afterPass1+1)
	}

	reviewCount, shas := countReviewedEvents(t, store)
	if reviewCount != 2 {
		t.Fatalf("reviewed events after recovery = %d, want 2 (sha list: %v)", reviewCount, shas)
	}

	if len(shas) != 2 || shas[1] != wantSHA {
		t.Fatalf("deferred view reviewed with sha %v, want [%s %s]", shas, view1sha(t, view1), wantSHA)
	}

	if endpoint.chatCount() != 2 {
		t.Fatalf("chat calls after pass 3 = %d, want 2 (view1 must stay skip-seen)", endpoint.chatCount())
	}
}

func view1sha(t *testing.T, path string) string {
	t.Helper()

	sha, err := SHA256File(path)
	if err != nil {
		t.Fatalf("hash %s: %v", path, err)
	}

	return sha
}

// closeStoreQuietly closes a store in tests that already reported the
// failure path; close errors in cleanup are not assertion material.
func closeStoreQuietly(store *Store) {
	_ = store.Close()
}
