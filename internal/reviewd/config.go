package reviewed

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Defaults for the visionreviewd daemon. The llama-server port (8390) is
// registered in SystemNix lib/ports.nix as vision-review-agent-llama; the
// llama-server default port 8080 is taken on this host.
const (
	DefaultModel      = "GitMylo/nsfwcaption-qwen3-vl-8b-v3-gguf:Q8_0"
	DefaultBaseURL    = "http://127.0.0.1:8390/v1"
	DefaultDataDir    = "~/.local/share/vision-review-agent"
	DefaultReviewsDir = "~/.local/share/vision-review-agent/reviews"
	DefaultConfigPath = "~/.config/visionreviewd/config.json"
	DefaultInterval   = 10 * time.Minute
	DefaultTimeout    = 5 * time.Minute
)

// ErrInvalidConfig is the sentinel for every Config validation failure.
// Wrapped errors include the offending value.
var ErrInvalidConfig = errors.New("invalid visionreviewd config")

// Config is the full daemon configuration. JSON field semantics: durations
// are strings like "10m"; paths may start with "~" and are expanded on load.
type Config struct {
	Model      string              `json:"model"`
	BaseURL    string              `json:"baseUrl"`
	APIKey     string              `json:"apiKey,omitempty"`
	Interval   time.Duration       `json:"-"`
	Timeout    time.Duration       `json:"-"`
	DataDir    string              `json:"dataDir"`
	ReviewsDir string              `json:"reviewsDir"`
	Projects   map[string][]string `json:"projects"`

	// Retry is the model-call retry policy. A present stanza replaces the
	// default policy wholesale; zero fields fall back to the vision SDK
	// defaults (3 attempts, 500ms initial, 30s cap, 2.0x, jitter). Omit the
	// stanza for the daemon default (enabled, 3 attempts, 2s initial);
	// {"enabled": false} disables retry entirely.
	Retry *RetryConfig `json:"retry,omitempty"`

	// HealthGate is the pre-pass endpoint gating policy. The zero stanza
	// means defaults: gate enabled with the package probe timeout. Omit the
	// stanza for the same defaults; {"enabled": false} turns the gate off
	// (every pass runs and fails per-view against a dead endpoint).
	HealthGate *HealthGateConfig `json:"healthGate,omitempty"`

	// SourceURLs optionally maps a project name to the page its
	// screenshots were taken from (website captures). Rendered in review
	// markdown so every review points back at the live page.
	SourceURLs map[string]string `json:"sourceURLs,omitempty"` //nolint:tagliatelle // wire key: existing user configs (~/.config/visionreviewd) use sourceURLs; URL is not a word
}

// Daemon retry defaults: attempts, cap, and growth factor for the
// out-of-the-box policy (initial backoff is 2s, the llama-server
// cold-start window).
const (
	defaultRetryAttempts   = 3
	defaultRetryMaxBackoff = 30 * time.Second
	defaultRetryMultiplier = 2.0
)

// DefaultRetryPolicy is the daemon's out-of-the-box retry policy: three
// attempts starting at 2s (the llama-server cold-start window), capped at
// 30s, doubling between attempts with jitter. Transient endpoint blips stop
// failing whole views.
func DefaultRetryPolicy() RetryConfig {
	return RetryConfig{
		Enabled:        true,
		MaxAttempts:    defaultRetryAttempts,
		InitialBackoff: 2 * time.Second,
		MaxBackoff:     defaultRetryMaxBackoff,
		Multiplier:     defaultRetryMultiplier,
	}
}

// RetryConfig is the daemon's model-retry policy: enabled flag plus the
// vision SDK retry knobs. Jitter is always on (retry storms against one
// llama-server instance are pointless).
type RetryConfig struct {
	// Enabled turns retry off entirely when false.
	Enabled bool `json:"enabled"`

	// MaxAttempts is the total number of model attempts including the
	// first; zero falls back to the SDK default.
	MaxAttempts int `json:"maxAttempts,omitempty"`

	// InitialBackoff is the delay before the first retry; zero falls back
	// to the SDK default.
	InitialBackoff time.Duration `json:"-"`

	// MaxBackoff caps the delay between retries; zero falls back to the
	// SDK default.
	MaxBackoff time.Duration `json:"-"`

	// Multiplier grows the backoff between retries; zero falls back to
	// the SDK default.
	Multiplier float64 `json:"multiplier,omitempty"`
}

// HealthGateConfig is the daemon's pre-pass endpoint gating policy.
type HealthGateConfig struct {
	// Enabled toggles the gate; nil means enabled (on by default).
	Enabled *bool `json:"enabled,omitempty"`

	// ProbeTimeout bounds the pre-pass endpoint probe; zero falls back to
	// EndpointProbeTimeout.
	ProbeTimeout time.Duration `json:"-"`
}

// healthGateJSON is the wire shape of HealthGateConfig: the timeout as a
// human string.
type healthGateJSON struct {
	Enabled      *bool  `json:"enabled,omitempty"`
	ProbeTimeout string `json:"probeTimeout,omitempty"`
}

// retryJSON is the wire shape of RetryConfig: durations as human strings.
type retryJSON struct {
	Enabled        *bool   `json:"enabled,omitempty"`
	MaxAttempts    int     `json:"maxAttempts,omitempty"`
	InitialBackoff string  `json:"initialBackoff,omitempty"`
	MaxBackoff     string  `json:"maxBackoff,omitempty"`
	Multiplier     float64 `json:"multiplier,omitempty"`
}

// configJSON is the wire shape of Config: durations as human strings.
type configJSON struct {
	Model      string              `json:"model"`
	BaseURL    string              `json:"baseUrl"`
	APIKey     string              `json:"apiKey,omitempty"`
	Interval   string              `json:"interval"`
	Timeout    string              `json:"timeout"`
	DataDir    string              `json:"dataDir"`
	ReviewsDir string              `json:"reviewsDir"`
	Projects   map[string][]string `json:"projects"`
	Retry      *retryJSON          `json:"retry,omitempty"`
	HealthGate *healthGateJSON     `json:"healthGate,omitempty"`
	SourceURLs map[string]string   `json:"sourceURLs,omitempty"` //nolint:tagliatelle // wire key: existing user configs use sourceURLs; URL is not a word
}

// DefaultConfig returns the daemon defaults: the caption-tuned vision model,
// llama-server on its registered port, a 10-minute scan interval, a
// 5-minute per-request timeout, and the default model retry policy.
func DefaultConfig() Config {
	policy := DefaultRetryPolicy()

	return Config{
		Model:      DefaultModel,
		BaseURL:    DefaultBaseURL,
		APIKey:     "",
		Interval:   DefaultInterval,
		Timeout:    DefaultTimeout,
		DataDir:    DefaultDataDir,
		ReviewsDir: DefaultReviewsDir,
		Projects:   map[string][]string{},
		Retry:      &policy,
		HealthGate: &HealthGateConfig{},
	}
}

// UnmarshalJSON decodes the wire shape over the existing values, so fields
// absent from the document keep their current (default) values. Durations are
// duration strings such as "10m" or "1h30m".
func (c *Config) UnmarshalJSON(data []byte) error {
	raw := configJSON{
		Model:      c.Model,
		BaseURL:    c.BaseURL,
		APIKey:     c.APIKey,
		Interval:   c.Interval.String(),
		Timeout:    c.Timeout.String(),
		DataDir:    c.DataDir,
		ReviewsDir: c.ReviewsDir,
		Projects:   c.Projects,
		Retry:      nil, // decoded separately below: an absent stanza keeps the loaded default
		HealthGate: nil, // decoded separately below: an absent stanza keeps the loaded default
		SourceURLs: c.SourceURLs,
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("decode config: %w", err)
	}

	interval, err := parseDurationField("interval", raw.Interval)
	if err != nil {
		return err
	}

	timeout, err := parseDurationField("timeout", raw.Timeout)
	if err != nil {
		return err
	}

	c.Model = raw.Model
	c.BaseURL = raw.BaseURL
	c.APIKey = raw.APIKey
	c.Interval = interval
	c.Timeout = timeout
	c.DataDir = raw.DataDir
	c.ReviewsDir = raw.ReviewsDir

	if raw.Projects != nil {
		c.Projects = raw.Projects
	}

	// A present retry stanza replaces the whole policy (all-or-nothing);
	// an absent stanza keeps the loaded default.
	if raw.Retry != nil {
		policy, decodeErr := decodeRetryPolicy(raw.Retry)
		if decodeErr != nil {
			return decodeErr
		}

		c.Retry = policy
	}

	// A present healthGate stanza replaces the whole policy; an absent
	// stanza keeps the loaded default.
	if raw.HealthGate != nil {
		gate, decodeErr := decodeHealthGatePolicy(raw.HealthGate)
		if decodeErr != nil {
			return decodeErr
		}

		c.HealthGate = gate
	}

	if raw.SourceURLs != nil {
		c.SourceURLs = raw.SourceURLs
	}

	return nil
}

// decodeRetryPolicy converts the wire stanza; a duration parse failure is a
// config error naming the offending field.
func decodeRetryPolicy(raw *retryJSON) (*RetryConfig, error) {
	policy := RetryConfig{
		Enabled:        true,
		MaxAttempts:    0,
		InitialBackoff: 0,
		MaxBackoff:     0,
		Multiplier:     0,
	}

	if raw.Enabled != nil {
		policy.Enabled = *raw.Enabled
	}

	policy.MaxAttempts = raw.MaxAttempts
	policy.Multiplier = raw.Multiplier

	initial, err := parseOptionalDuration("retry.initialBackoff", raw.InitialBackoff)
	if err != nil {
		return nil, err
	}

	policy.InitialBackoff = initial

	maxBackoff, err := parseOptionalDuration("retry.maxBackoff", raw.MaxBackoff)
	if err != nil {
		return nil, err
	}

	policy.MaxBackoff = maxBackoff

	return &policy, nil
}

// parseOptionalDuration parses a duration string that may be empty (zero
// value means "fall back to the SDK default").
func parseOptionalDuration(field, value string) (time.Duration, error) {
	if value == "" {
		return 0, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%w: %s must be a duration string like \"2s\": got %q",
			ErrInvalidConfig, field, value)
	}

	return parsed, nil
}

// MarshalJSON encodes durations as human-readable strings.
func (c Config) MarshalJSON() ([]byte, error) {
	raw := configJSON{
		Model:      c.Model,
		BaseURL:    c.BaseURL,
		APIKey:     c.APIKey,
		Interval:   c.Interval.String(),
		Timeout:    c.Timeout.String(),
		DataDir:    c.DataDir,
		ReviewsDir: c.ReviewsDir,
		Projects:   c.Projects,
		Retry:      encodeRetryPolicy(c.Retry),
		HealthGate: encodeHealthGatePolicy(c.HealthGate),
		SourceURLs: c.SourceURLs,
	}

	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("encode config: %w", err)
	}

	return encoded, nil
}

// encodeRetryPolicy converts the policy to its wire shape; nil omits the
// stanza.
func encodeRetryPolicy(policy *RetryConfig) *retryJSON {
	if policy == nil {
		return nil
	}

	enabled := policy.Enabled
	initial := ""

	if policy.InitialBackoff > 0 {
		initial = policy.InitialBackoff.String()
	}

	maxBackoff := ""

	if policy.MaxBackoff > 0 {
		maxBackoff = policy.MaxBackoff.String()
	}

	return &retryJSON{
		Enabled:        &enabled,
		MaxAttempts:    policy.MaxAttempts,
		InitialBackoff: initial,
		MaxBackoff:     maxBackoff,
		Multiplier:     policy.Multiplier,
	}
}

// decodeHealthGatePolicy converts the wire stanza; a duration parse failure
// is a config error naming the offending field. A nil Enabled stays nil
// (enabled by default).
func decodeHealthGatePolicy(raw *healthGateJSON) (*HealthGateConfig, error) {
	gate := HealthGateConfig{
		Enabled:      nil,
		ProbeTimeout: 0,
	}

	if raw.Enabled != nil {
		gate.Enabled = raw.Enabled
	}

	timeout, err := parseOptionalDuration("healthGate.probeTimeout", raw.ProbeTimeout)
	if err != nil {
		return nil, err
	}

	gate.ProbeTimeout = timeout

	return &gate, nil
}

// encodeHealthGatePolicy converts the policy to its wire shape; nil omits
// the stanza.
func encodeHealthGatePolicy(policy *HealthGateConfig) *healthGateJSON {
	if policy == nil {
		return nil
	}

	probeTimeout := ""

	if policy.ProbeTimeout > 0 {
		probeTimeout = policy.ProbeTimeout.String()
	}

	return &healthGateJSON{
		Enabled:      policy.Enabled,
		ProbeTimeout: probeTimeout,
	}
}

func parseDurationField(field, value string) (time.Duration, error) {
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%w: %s must be a duration string like \"10m\": got %q",
			ErrInvalidConfig, field, value)
	}

	return parsed, nil
}

// Normalize expands "~" in DataDir, ReviewsDir, and every project glob so all
// downstream paths are absolute. Idempotent.
func (c Config) Normalize() (Config, error) {
	normalized := c

	dataDir, err := expandTilde(c.DataDir)
	if err != nil {
		return Config{}, err
	}

	normalized.DataDir = dataDir

	reviewsDir, err := expandTilde(c.ReviewsDir)
	if err != nil {
		return Config{}, err
	}

	normalized.ReviewsDir = reviewsDir

	projects, err := expandProjectGlobs(c.Projects)
	if err != nil {
		return Config{}, err
	}

	normalized.Projects = projects

	return normalized, nil
}

func expandProjectGlobs(projects map[string][]string) (map[string][]string, error) {
	expanded := make(map[string][]string, len(projects))

	for project, globs := range projects {
		patterns := make([]string, 0, len(globs))

		for _, pattern := range globs {
			path, err := expandTilde(pattern)
			if err != nil {
				return nil, err
			}

			patterns = append(patterns, path)
		}

		expanded[project] = patterns
	}

	return expanded, nil
}

// Validate checks the normalized configuration and reports every offending
// value in the error message.
func (c Config) Validate() error {
	if err := c.validateEndpoints(); err != nil {
		return err
	}

	if err := c.validateTimings(); err != nil {
		return err
	}

	if err := c.validateDirs(); err != nil {
		return err
	}

	return c.validateProjects()
}

func (c Config) validateEndpoints() error {
	if c.Model == "" {
		return fmt.Errorf("%w: model is empty, want e.g. %q", ErrInvalidConfig, DefaultModel)
	}

	parsedURL, err := url.Parse(c.BaseURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return fmt.Errorf("%w: baseUrl must be an absolute http(s) URL, got %q", ErrInvalidConfig, c.BaseURL)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("%w: baseUrl scheme must be http or https, got %q", ErrInvalidConfig, c.BaseURL)
	}

	return nil
}

func (c Config) validateTimings() error {
	if c.Interval <= 0 {
		return fmt.Errorf("%w: interval must be positive, got %s", ErrInvalidConfig, c.Interval)
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("%w: timeout must be positive, got %s", ErrInvalidConfig, c.Timeout)
	}

	return c.validateRetry()
}

// validateRetry rejects negative retry knobs; zero means "SDK default" and
// is legal.
func (c Config) validateRetry() error {
	if c.Retry == nil {
		return nil
	}

	if c.Retry.MaxAttempts < 0 {
		return fmt.Errorf("%w: retry.maxAttempts must not be negative, got %d", ErrInvalidConfig, c.Retry.MaxAttempts)
	}

	if c.Retry.InitialBackoff < 0 {
		return fmt.Errorf(
			"%w: retry.initialBackoff must not be negative, got %s",
			ErrInvalidConfig,
			c.Retry.InitialBackoff,
		)
	}

	if c.Retry.MaxBackoff < 0 {
		return fmt.Errorf("%w: retry.maxBackoff must not be negative, got %s", ErrInvalidConfig, c.Retry.MaxBackoff)
	}

	if c.Retry.Multiplier < 0 {
		return fmt.Errorf("%w: retry.multiplier must not be negative, got %g", ErrInvalidConfig, c.Retry.Multiplier)
	}

	return nil
}

func (c Config) validateDirs() error {
	if c.DataDir == "" {
		return fmt.Errorf("%w: dataDir is empty, want e.g. %q", ErrInvalidConfig, DefaultDataDir)
	}

	if c.ReviewsDir == "" {
		return fmt.Errorf("%w: reviewsDir is empty, want e.g. %q", ErrInvalidConfig, DefaultReviewsDir)
	}

	return nil
}

func (c Config) validateProjects() error {
	if len(c.Projects) == 0 {
		return fmt.Errorf(
			"%w: no projects configured, add at least one project with screenshot globs",
			ErrInvalidConfig,
		)
	}

	for project, globs := range c.Projects {
		if project == "" {
			return fmt.Errorf("%w: project name is empty", ErrInvalidConfig)
		}

		if len(globs) == 0 {
			return fmt.Errorf("%w: project %q has no screenshot globs", ErrInvalidConfig, project)
		}

		if err := validateGlobs(project, globs); err != nil {
			return err
		}
	}

	return nil
}

func validateGlobs(project string, globs []string) error {
	for _, pattern := range globs {
		if pattern == "" {
			return fmt.Errorf("%w: project %q has an empty glob", ErrInvalidConfig, project)
		}

		if !filepath.IsAbs(pattern) {
			return fmt.Errorf("%w: project %q glob must be absolute (after ~ expansion), got %q",
				ErrInvalidConfig, project, pattern)
		}
	}

	return nil
}

// LoadConfig reads a JSON config file over the defaults, expands "~" paths,
// and validates the result.
func LoadConfig(path string) (Config, error) {
	config := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}

	normalized, err := config.Normalize()
	if err != nil {
		return Config{}, err
	}

	if err := normalized.Validate(); err != nil {
		return Config{}, fmt.Errorf("config %s: %w", path, err)
	}

	return normalized, nil
}

// ExpandTilde replaces a leading "~" with the user's home directory. It is
// the exported form of the path expansion Config.Normalize applies, so
// callers (e.g. the CLI resolving the config path itself) share one rule.
func ExpandTilde(path string) (string, error) {
	return expandTilde(path)
}

// expandTilde replaces a leading "~" with the user's home directory.
func expandTilde(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expand ~ in %s: %w", path, err)
		}

		return filepath.Join(home, strings.TrimPrefix(path, "~")), nil
	}

	return path, nil
}
