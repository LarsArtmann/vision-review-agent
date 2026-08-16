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
//
//nolint:recvcheck // UnmarshalJSON must take a pointer, MarshalJSON a value (standard JSON pattern).
type Config struct {
	Model      string              `json:"model"`
	BaseURL    string              `json:"baseUrl"`
	APIKey     string              `json:"apiKey,omitempty"`
	Interval   time.Duration       `json:"-"`
	Timeout    time.Duration       `json:"-"`
	DataDir    string              `json:"dataDir"`
	ReviewsDir string              `json:"reviewsDir"`
	Projects   map[string][]string `json:"projects"`
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
}

// DefaultConfig returns the daemon defaults: the caption-tuned vision model,
// llama-server on its registered port, a 10-minute scan interval, and a
// 5-minute per-request timeout.
func DefaultConfig() Config {
	return Config{
		Model:      DefaultModel,
		BaseURL:    DefaultBaseURL,
		APIKey:     "",
		Interval:   DefaultInterval,
		Timeout:    DefaultTimeout,
		DataDir:    DefaultDataDir,
		ReviewsDir: DefaultReviewsDir,
		Projects:   map[string][]string{},
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

	return nil
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
	}

	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("encode config: %w", err)
	}

	return encoded, nil
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
