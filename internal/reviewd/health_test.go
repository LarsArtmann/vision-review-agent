package reviewed

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestProbeModelEndpoint(t *testing.T) {
	t.Parallel()

	listed, err := json.Marshal(map[string]any{"data": []map[string]any{{"id": "stub-model"}, {"id": "other"}}})
	if err != nil {
		t.Fatalf("marshal listing: %v", err)
	}

	cases := []struct {
		name       string
		status     int
		body       string
		wantErr    bool
		wantSubstr string
	}{
		{name: "healthy model listed", status: http.StatusOK, body: string(listed)},
		{
			name:       "model not listed",
			status:     http.StatusOK,
			body:       `{"data":[{"id":"other"}]}`,
			wantErr:    true,
			wantSubstr: "not listed",
		},
		{
			name:       "server error status",
			status:     http.StatusInternalServerError,
			body:       `{"data":[]}`,
			wantErr:    true,
			wantSubstr: "status",
		},
		{
			name:       "malformed body",
			status:     http.StatusOK,
			body:       "not json",
			wantErr:    true,
			wantSubstr: "decode",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := serveModels(t, tc.status, tc.body, nil)

			config := DefaultConfig()
			config.Model = "stub-model"
			config.BaseURL = server.URL + "/v1"

			err := ProbeModelEndpoint(t.Context(), config, EndpointProbeTimeout)
			assertProbeOutcome(t, err, tc.wantErr, tc.wantSubstr)
		})
	}
}

// serveModels starts an httptest server answering /v1/models with the given
// status and body; sawAuth, when non-nil, records the Authorization header.
func serveModels(t *testing.T, status int, body string, sawAuth *string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/v1/models" {
			t.Errorf("probe hit %s, want /v1/models", req.URL.Path)
		}

		if sawAuth != nil {
			*sawAuth = req.Header.Get("Authorization")
		}

		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return server
}

// assertProbeOutcome checks the probe error against the table expectation:
// every failure must carry the ErrEndpointUnhealthy sentinel and the wanted
// diagnostic substring.
func assertProbeOutcome(t *testing.T, err error, wantErr bool, wantSubstr string) {
	t.Helper()

	if !wantErr {
		if err != nil {
			t.Fatalf("probe = %v, want nil", err)
		}

		return
	}

	if !errors.Is(err, ErrEndpointUnhealthy) {
		t.Fatalf("probe = %v, want ErrEndpointUnhealthy", err)
	}

	if wantSubstr != "" && !strings.Contains(err.Error(), wantSubstr) {
		t.Fatalf("probe error %q missing %q", err, wantSubstr)
	}
}

func TestProbeModelEndpointSendsAPIKey(t *testing.T) {
	t.Parallel()

	sawAuth := ""

	server := serveModels(t, http.StatusOK, `{"data":[{"id":"stub-model"}]}`, &sawAuth)

	config := DefaultConfig()
	config.Model = "stub-model"
	config.BaseURL = server.URL + "/v1"
	config.APIKey = "secret"

	assertProbeOutcome(t, ProbeModelEndpoint(t.Context(), config, EndpointProbeTimeout), false, "")

	if sawAuth != "Bearer "+config.APIKey {
		t.Fatalf("authorization header = %q, want Bearer token", sawAuth)
	}
}

func TestProbeModelEndpointUnreachable(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	server.Close()

	config := DefaultConfig()
	config.Model = "stub-model"
	config.BaseURL = server.URL + "/v1"

	err := ProbeModelEndpoint(t.Context(), config, EndpointProbeTimeout)
	if !errors.Is(err, ErrEndpointUnhealthy) {
		t.Fatalf("probe = %v, want ErrEndpointUnhealthy", err)
	}

	if !strings.Contains(err.Error(), "/models") {
		t.Fatalf("probe error %q should name the models URL", err)
	}
}

// gateRecorder is a stub PassRunner that records whether it ran and what it
// was handed.
type gateRecorder struct {
	mu       sync.Mutex
	calls    int
	projects map[string][]string
	result   PassResult
	err      error
}

func (g *gateRecorder) Pass(ctx context.Context, projects map[string][]string) (PassResult, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.calls++
	g.projects = projects

	if ctx.Err() != nil {
		return PassResult{}, ctx.Err()
	}

	return g.result, g.err
}

func (g *gateRecorder) passCalls() int {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.calls
}

func TestHealthGateSkipsPassWhenUnhealthy(t *testing.T) {
	t.Parallel()

	inner := &gateRecorder{}

	gate, err := NewHealthGate(inner, func(_ context.Context) error {
		return errors.New("dial refused")
	}, nil)
	if err != nil {
		t.Fatalf("build gate: %v", err)
	}

	projects := map[string][]string{"app": {"/shots/*.png"}}

	result, err := gate.Pass(t.Context(), projects)
	if !errors.Is(err, ErrEndpointUnhealthy) {
		t.Fatalf("gate pass = %v, want ErrEndpointUnhealthy", err)
	}

	if inner.passCalls() != 0 {
		t.Fatalf("inner ran %d times, want 0", inner.passCalls())
	}

	if result != (PassResult{}) {
		t.Fatalf("result = %+v, want zero", result)
	}
}

func TestHealthGateDelegatesWhenHealthy(t *testing.T) {
	t.Parallel()

	want := PassResult{Views: 3, Captured: 2, Reviewed: 2}
	inner := &gateRecorder{result: want}

	gate, err := NewHealthGate(inner, func(_ context.Context) error { return nil }, nil)
	if err != nil {
		t.Fatalf("build gate: %v", err)
	}

	projects := map[string][]string{"app": {"/shots/*.png"}}

	result, err := gate.Pass(t.Context(), projects)
	if err != nil {
		t.Fatalf("gate pass: %v", err)
	}

	if inner.passCalls() != 1 {
		t.Fatalf("inner ran %d times, want 1", inner.passCalls())
	}

	if inner.projects["app"] == nil {
		t.Fatal("inner did not receive the projects")
	}

	if result != want {
		t.Fatalf("result = %+v, want %+v", result, want)
	}
}

func TestHealthGateDeadContextBypassesProbe(t *testing.T) {
	t.Parallel()

	probeRan := false

	inner := &gateRecorder{}

	gate, err := NewHealthGate(inner, func(_ context.Context) error {
		probeRan = true

		return errors.New("probe should not run")
	}, nil)
	if err != nil {
		t.Fatalf("build gate: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// The inner runner surfaces the dead context itself; the probe must not
	// mask shutdown as endpoint downtime.
	if _, err := gate.Pass(ctx, map[string][]string{"app": nil}); !errors.Is(err, context.Canceled) {
		t.Fatalf("gate pass = %v, want context.Canceled", err)
	}

	if probeRan {
		t.Fatal("probe ran on a dead context")
	}

	if inner.passCalls() != 1 {
		t.Fatalf("inner ran %d times, want 1", inner.passCalls())
	}
}

func TestNewHealthGateRejectsInvalidWiring(t *testing.T) {
	t.Parallel()

	runner := &gateRecorder{}
	probe := func(_ context.Context) error { return nil }

	if _, err := NewHealthGate(nil, probe, nil); !errors.Is(err, ErrNoGatedRunner) {
		t.Fatalf("nil runner = %v, want ErrNoGatedRunner", err)
	}

	if _, err := NewHealthGate(runner, nil, nil); !errors.Is(err, ErrNoProbe) {
		t.Fatalf("nil probe = %v, want ErrNoProbe", err)
	}
}
