package reviewed_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"charm.land/fantasy"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// stubLanguageModel is a mock fantasy.LanguageModel for the BDD specs.
// Generate returns the canned markdown; generateErr injects failures. Every
// call's prompt is captured for assertions.
type stubLanguageModel struct {
	mu          sync.Mutex
	markdown    string
	generateErr error
	prompts     []fantasy.Prompt
}

func (m *stubLanguageModel) Generate(_ context.Context, in fantasy.Call) (*fantasy.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.prompts = append(m.prompts, in.Prompt)

	if m.generateErr != nil {
		return nil, m.generateErr
	}

	return &fantasy.Response{
		Content: []fantasy.Content{
			fantasy.TextContent{Text: m.markdown},
		},
		FinishReason: fantasy.FinishReasonStop,
		Usage:        fantasy.Usage{TotalTokens: 12},
	}, nil
}

func (m *stubLanguageModel) Provider() string { return "stub" }

func (m *stubLanguageModel) Model() string { return "stub-review-model" }

func (m *stubLanguageModel) Stream(_ context.Context, _ fantasy.Call) (fantasy.StreamResponse, error) {
	return nil, errors.New("streaming unused in reviewd specs")
}

func (m *stubLanguageModel) GenerateObject(
	_ context.Context,
	_ fantasy.ObjectCall,
) (*fantasy.ObjectResponse, error) {
	return nil, errors.New("object mode unused in reviewd specs")
}

func (m *stubLanguageModel) StreamObject(
	_ context.Context,
	_ fantasy.ObjectCall,
) (fantasy.ObjectStreamResponse, error) {
	return nil, errors.New("object mode unused in reviewd specs")
}

// calls returns how many Generate invocations happened.
func (m *stubLanguageModel) calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.prompts)
}

// promptAt returns the i-th captured prompt (nil when out of range).
func (m *stubLanguageModel) promptAt(i int) fantasy.Prompt {
	m.mu.Lock()
	defer m.mu.Unlock()

	if i < 0 || i >= len(m.prompts) {
		return nil
	}

	return m.prompts[i]
}

// setMarkdown swaps the canned response between passes.
func (m *stubLanguageModel) setMarkdown(markdown string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.markdown = markdown
}

// setGenerateErr injects a failure for subsequent Generate calls.
func (m *stubLanguageModel) setGenerateErr(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.generateErr = err
}

// countFileParts counts image file parts in a prompt.
func countFileParts(prompt fantasy.Prompt) int {
	count := 0

	for _, message := range prompt {
		for _, part := range message.Content {
			if _, ok := part.(fantasy.FilePart); ok {
				count++
			}
		}
	}

	return count
}

// countEvents returns how many events of the given type were recorded.
func countEvents(events []event.Event, eventType string) int {
	count := 0

	for _, evt := range events {
		if string(evt.Type()) == eventType {
			count++
		}
	}

	return count
}

// writeShotPNG writes a minimal valid 1x1 PNG at path.
func writeShotPNG(path string) error {
	if err := os.WriteFile(path, shotPNG, 0o644); err != nil {
		return fmt.Errorf("write shot png %s: %w", path, err)
	}

	return nil
}

// writeChangedShotPNG writes a different valid PNG at path, so its content
// hash differs from writeShotPNG's.
func writeChangedShotPNG(path string) error {
	changed := make([]byte, len(shotPNG))
	copy(changed, shotPNG)

	const idatPixelOffset = 45

	changed[idatPixelOffset] ^= 0xFF

	if err := os.WriteFile(path, changed, 0o644); err != nil {
		return fmt.Errorf("write changed shot png %s: %w", path, err)
	}

	return nil
}

var shotPNG = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4, 0x89, 0x00, 0x00, 0x00,
	0x0D, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9C, 0x62, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00, 0x00, 0x00, 0x00, 0x49,
	0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
}

// readFileOrFail returns the file's content or fails the spec.
func readFileOrFail(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		panic("read " + filepath.Clean(path) + ": " + err.Error())
	}

	return string(data)
}
