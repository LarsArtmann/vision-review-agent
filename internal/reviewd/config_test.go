package reviewed

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDefaultConfigValidate(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()
	config.DataDir = "/tmp/vra"
	config.ReviewsDir = "/tmp/vra/reviews"
	config.Projects = map[string][]string{"proj": {"/abs/*.png"}}

	if err := config.Validate(); err != nil {
		t.Fatalf("default config with one project should validate: %v", err)
	}
}

func TestDefaultConfigNormalizeExpandsTilde(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("user home dir: %v", err)
	}

	config := DefaultConfig()
	config.Projects = map[string][]string{"proj": {"~/shots/*.png"}}

	normalized, err := config.Normalize()
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}

	if want := filepath.Join(home, ".local", "share", "vision-review-agent"); normalized.DataDir != want {
		t.Fatalf("dataDir = %q, want %q", normalized.DataDir, want)
	}

	if want := filepath.Join(home, ".local", "share", "vision-review-agent", "reviews"); normalized.ReviewsDir != want {
		t.Fatalf("reviewsDir = %q, want %q", normalized.ReviewsDir, want)
	}

	if want := filepath.Join(home, "shots", "*.png"); normalized.Projects["proj"][0] != want {
		t.Fatalf("glob = %q, want %q", normalized.Projects["proj"][0], want)
	}

	// Original untouched.
	if config.DataDir != DefaultDataDir {
		t.Fatalf("normalize mutated receiver: %q", config.DataDir)
	}
}

func TestValidateErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mutate    func(*Config)
		wantInMsg string
	}{
		{
			name:      "empty model",
			mutate:    func(c *Config) { c.Model = "" },
			wantInMsg: "model is empty",
		},
		{
			name:      "relative base url",
			mutate:    func(c *Config) { c.BaseURL = "localhost:8390/v1" },
			wantInMsg: `baseUrl must be an absolute http(s) URL, got "localhost:8390/v1"`,
		},
		{
			name:      "non-http scheme",
			mutate:    func(c *Config) { c.BaseURL = "ftp://127.0.0.1:8390/v1" },
			wantInMsg: `scheme must be http or https, got "ftp://127.0.0.1:8390/v1"`,
		},
		{
			name:      "zero interval",
			mutate:    func(c *Config) { c.Interval = 0 },
			wantInMsg: "interval must be positive, got 0s",
		},
		{
			name:      "negative timeout",
			mutate:    func(c *Config) { c.Timeout = -1 * time.Second },
			wantInMsg: "timeout must be positive, got -1s",
		},
		{
			name:      "empty data dir",
			mutate:    func(c *Config) { c.DataDir = "" },
			wantInMsg: "dataDir is empty",
		},
		{
			name:      "empty reviews dir",
			mutate:    func(c *Config) { c.ReviewsDir = "" },
			wantInMsg: "reviewsDir is empty",
		},
		{
			name:      "no projects",
			mutate:    func(c *Config) { c.Projects = map[string][]string{} },
			wantInMsg: "no projects configured",
		},
		{
			name:      "project without globs",
			mutate:    func(c *Config) { c.Projects = map[string][]string{"proj": {}} },
			wantInMsg: `project "proj" has no screenshot globs`,
		},
		{
			name:      "relative glob",
			mutate:    func(c *Config) { c.Projects = map[string][]string{"proj": {"shots/*.png"}} },
			wantInMsg: `glob must be absolute`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := DefaultConfig()
			config.DataDir = "/tmp/vra"
			config.ReviewsDir = "/tmp/vra/reviews"
			config.Projects = map[string][]string{"proj": {"/abs/*.png"}}
			tt.mutate(&config)

			err := config.Validate()
			if err == nil {
				t.Fatalf("Validate should fail for %s", tt.name)
			}

			if !errors.Is(err, ErrInvalidConfig) {
				t.Fatalf("error should wrap ErrInvalidConfig, got %v", err)
			}

			if !strings.Contains(err.Error(), tt.wantInMsg) {
				t.Fatalf("error %q should contain %q", err.Error(), tt.wantInMsg)
			}
		})
	}
}

func writeConfigFile(t *testing.T, dir, content string) string {
	t.Helper()

	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	return path
}

func TestLoadConfigOverridesDefaults(t *testing.T) {
	t.Parallel()

	path := writeConfigFile(t, t.TempDir(), `{
		"model": "other-model:Q4_K_M",
		"baseUrl": "http://10.0.0.5:9000/v1",
		"interval": "1h30m",
		"dataDir": "/var/lib/vra",
		"projects": {
			"discordsync": ["/home/lars/projects/DiscordSync/internal/web/testdata/visual/*.png"]
		}
	}`)

	config, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if config.Model != "other-model:Q4_K_M" {
		t.Fatalf("model = %q", config.Model)
	}

	if config.BaseURL != "http://10.0.0.5:9000/v1" {
		t.Fatalf("baseUrl = %q", config.BaseURL)
	}

	if config.Interval != 90*time.Minute {
		t.Fatalf("interval = %s, want 1h30m", config.Interval)
	}

	if config.Timeout != DefaultTimeout {
		t.Fatalf("timeout = %s, want default (absent key keeps default)", config.Timeout)
	}

	if config.DataDir != "/var/lib/vra" {
		t.Fatalf("dataDir = %q", config.DataDir)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("user home dir: %v", err)
	}

	if want := filepath.Join(home, ".local", "share", "vision-review-agent", "reviews"); config.ReviewsDir != want {
		t.Fatalf("reviewsDir = %q, want %q (default ~-expanded)", config.ReviewsDir, want)
	}

	if len(config.Projects["discordsync"]) != 1 {
		t.Fatalf("projects = %v", config.Projects)
	}
}

func TestLoadConfigEmptyFileKeepsDefaults(t *testing.T) {
	t.Parallel()

	path := writeConfigFile(t, t.TempDir(), `{"projects": {"proj": ["/abs/*.png"]}}`)

	config, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if config.Model != DefaultModel {
		t.Fatalf("model = %q, want default", config.Model)
	}

	if config.Interval != DefaultInterval {
		t.Fatalf("interval = %s, want default", config.Interval)
	}
}

func TestLoadConfigRejectsBadDuration(t *testing.T) {
	t.Parallel()

	path := writeConfigFile(t, t.TempDir(), `{"interval": "not-a-duration"}`)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("bad duration should fail")
	}

	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("error should wrap ErrInvalidConfig, got %v", err)
	}

	if !strings.Contains(err.Error(), `"not-a-duration"`) {
		t.Fatalf("error should include offending value: %v", err)
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	t.Parallel()

	_, err := LoadConfig(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil {
		t.Fatal("missing config file should fail")
	}
}

func TestConfigMarshalRoundtrip(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()
	config.Projects = map[string][]string{"proj": {"/abs/*.png"}}

	encoded, err := config.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if !strings.Contains(string(encoded), `"interval":"10m0s"`) {
		t.Fatalf("durations should marshal as strings: %s", encoded)
	}

	decoded := DefaultConfig()
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !reflect.DeepEqual(decoded, config) {
		t.Fatalf("roundtrip mismatch:\n got %+v\nwant %+v", decoded, config)
	}
}
