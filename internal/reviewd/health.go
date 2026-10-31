package reviewed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Health gate dependency sentinels: NewHealthGate rejects invalid wiring
// with these.
var (
	ErrNoGatedRunner     = errors.New("health gate: nil runner")
	ErrNoProbe           = errors.New("health gate: nil probe")
	ErrEndpointUnhealthy = errors.New("model endpoint unhealthy")
)

// EndpointProbeTimeout bounds the daemon's pre-pass endpoint probe. It is
// shorter than doctor's bound: a pass is skipped, not diagnosed, so the
// probe should fail fast against a dead endpoint.
const EndpointProbeTimeout = 5 * time.Second

// ProbeModelEndpoint verifies the OpenAI-compatible endpoint answers
// {baseUrl}/models and lists the configured model. A nil return means the
// endpoint can serve the daemon's model right now; every failure carries the
// URL and the offending detail so the message is self-diagnosing. Each call
// builds its own http.Client (bounded by the timeout argument) but
// connections pool through the shared http.DefaultTransport, so no
// package-level client is needed.
func ProbeModelEndpoint(ctx context.Context, config Config, timeout time.Duration) error {
	modelsURL := strings.TrimSuffix(config.BaseURL, "/") + "/models"

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsURL, nil)
	if err != nil {
		return fmt.Errorf("%w: build request: %w", ErrEndpointUnhealthy, err)
	}

	if config.APIKey != "" {
		request.Header.Set("Authorization", "Bearer "+config.APIKey)
	}

	client := &http.Client{Timeout: timeout}

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrEndpointUnhealthy, modelsURL, err)
	}

	// Read and close eagerly so every failure below carries the URL in the
	// returned error instead of relying on a stderr side channel.
	body, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()

	if readErr != nil {
		return fmt.Errorf("%w: read %s: %w", ErrEndpointUnhealthy, modelsURL, readErr)
	}

	if closeErr != nil {
		return fmt.Errorf("%w: close %s: %w", ErrEndpointUnhealthy, modelsURL, closeErr)
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %s: status %s", ErrEndpointUnhealthy, modelsURL, response.Status)
	}

	var listed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &listed); err != nil {
		return fmt.Errorf("%w: decode %s: %w", ErrEndpointUnhealthy, modelsURL, err)
	}

	for _, model := range listed.Data {
		if model.ID == config.Model {
			return nil
		}
	}

	return fmt.Errorf("%w: model %q not listed at %s (%d models offered)",
		ErrEndpointUnhealthy, config.Model, modelsURL, len(listed.Data))
}

// HealthGate is a PassRunner decorator that skips a whole pass when the
// model endpoint is unhealthy. Without it, a dead endpoint turns every pass
// into one failed model call per changed view; with it, the pass is skipped
// with a single warning and the daemon heals itself on the next interval
// once the endpoint is back. The gate tracks its probe outcomes so every
// log line carries the operator-visible state: consecutive skip count,
// time since the endpoint was last healthy, and how many captures the
// skipped pass deferred.
type HealthGate struct {
	inner  PassRunner
	probe  func(ctx context.Context) error
	logger *slog.Logger

	mu               sync.Mutex
	consecutiveSkips int
	lastHealthy      time.Time // zero: the probe has never succeeded
}

// NewHealthGate wires the decorator. The probe must be non-nil and cheap
// relative to a pass; a nil logger falls back to the default logger.
func NewHealthGate(
	inner PassRunner,
	probe func(ctx context.Context) error,
	logger *slog.Logger,
) (*HealthGate, error) {
	if inner == nil {
		return nil, fmt.Errorf("new health gate: %w", ErrNoGatedRunner)
	}

	if probe == nil {
		return nil, fmt.Errorf("new health gate: %w", ErrNoProbe)
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &HealthGate{
		inner:            inner,
		probe:            probe,
		logger:           logger,
		mu:               sync.Mutex{},
		consecutiveSkips: 0,
		lastHealthy:      time.Time{},
	}, nil
}

// Pass probes the model endpoint and delegates to the wrapped runner only
// when it is healthy. A dead context bypasses the probe so shutdown keeps
// the inner runner's accurate "skipped, pass context done" errors.
func (g *HealthGate) Pass(ctx context.Context, projects map[string][]string) (PassResult, error) {
	if ctx.Err() != nil {
		result, err := g.inner.Pass(ctx, projects)
		if err != nil {
			return result, fmt.Errorf("gated pass: %w", err)
		}

		return result, nil
	}

	if err := g.probe(ctx); err != nil {
		g.recordSkip(projects, err)

		return PassResult{}, fmt.Errorf("pass skipped, %w: %w", ErrEndpointUnhealthy, err)
	}

	g.recordHealthy()

	result, err := g.inner.Pass(ctx, projects)
	if err != nil {
		return result, fmt.Errorf("gated pass: %w", err)
	}

	return result, nil
}

// recordSkip bumps the skip counter and warns with the operator state:
// which consecutive skip this is, how long the endpoint has been down, and
// roughly how many captures the deferred pass would have processed.
func (g *HealthGate) recordSkip(projects map[string][]string, probeErr error) {
	g.mu.Lock()
	g.consecutiveSkips++
	skips := g.consecutiveSkips
	lastHealthy := g.lastHealthy
	g.mu.Unlock()

	attrs := []any{
		"err", probeErr,
		"consecutive_skips", skips,
		"deferred_captures", deferredCaptureEstimate(projects),
	}

	if !lastHealthy.IsZero() {
		attrs = append(attrs, "down_since", time.Since(lastHealthy).Round(time.Second).String())
	}

	g.logger.Warn("pass skipped, model endpoint unhealthy", attrs...)
}

// recordHealthy resets the counters; a recovery after at least one skip
// logs the outage length so state transitions are visible in the journal.
func (g *HealthGate) recordHealthy() {
	g.mu.Lock()
	skips := g.consecutiveSkips
	lastHealthy := g.lastHealthy
	g.consecutiveSkips = 0
	g.lastHealthy = time.Now()
	g.mu.Unlock()

	if skips == 0 {
		return
	}

	outage := "unknown"

	if !lastHealthy.IsZero() {
		outage = time.Since(lastHealthy).Round(time.Second).String()
	}

	g.logger.Info("model endpoint recovered",
		"skipped_passes", skips,
		"outage", outage)
}

// deferredCaptureEstimate approximates how many captures the deferred pass
// would have processed by counting glob matches. It ignores glob errors
// (patterns are validated at config load) and counts files, not views, so
// the number is an upper bound — good enough to size the backlog at 3am.
func deferredCaptureEstimate(projects map[string][]string) int {
	total := 0

	for _, globs := range projects {
		for _, pattern := range globs {
			matches, err := filepath.Glob(pattern)
			if err != nil {
				continue
			}

			total += len(matches)
		}
	}

	return total
}
