package reviewed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
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
// URL and the offending detail so the message is self-diagnosing.
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
// once the endpoint is back.
type HealthGate struct {
	inner  PassRunner
	probe  func(ctx context.Context) error
	logger *slog.Logger
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

	return &HealthGate{inner: inner, probe: probe, logger: logger}, nil
}

// Pass probes the model endpoint and delegates to the wrapped runner only
// when it is healthy. A dead context bypasses the probe so shutdown keeps
// the inner runner's accurate "skipped, pass context done" errors.
func (g *HealthGate) Pass(ctx context.Context, projects map[string][]string) (PassResult, error) {
	if ctx.Err() != nil {
		return g.inner.Pass(ctx, projects)
	}

	if err := g.probe(ctx); err != nil {
		g.logger.Warn("pass skipped, model endpoint unhealthy", "err", err)

		return PassResult{}, fmt.Errorf("pass skipped, %w: %w", ErrEndpointUnhealthy, err)
	}

	return g.inner.Pass(ctx, projects)
}
