package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	reviewed "github.com/larsartmann/vision-review-agent/internal/reviewd"
)

// newFlagSet builds a ContinueOnError flag set that writes to the command's
// stderr, so parse failures return errors instead of exiting.
func newFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	flagSet := flag.NewFlagSet(name, flag.ContinueOnError)
	flagSet.SetOutput(stderr)

	return flagSet
}

// loadDaemonConfig expands and loads the config at path.
func loadDaemonConfig(path string) (reviewed.Config, error) {
	expanded, err := reviewed.ExpandTilde(path)
	if err != nil {
		return reviewed.Config{}, fmt.Errorf("expand config path: %w", err)
	}

	config, err := reviewed.LoadConfig(expanded)
	if err != nil {
		return reviewed.Config{}, fmt.Errorf("load daemon config: %w", err)
	}

	return config, nil
}

// openPipeline builds the full review pipeline from daemon configuration.
func openPipeline(ctx context.Context, config reviewed.Config) (*reviewed.Pipeline, *reviewed.Store, error) {
	reviewer, err := reviewed.NewReviewerFromConfig(ctx, config)
	if err != nil {
		return nil, nil, fmt.Errorf("build reviewer: %w", err)
	}

	store, err := reviewed.OpenStore(filepath.Join(config.DataDir, "events.db"), slog.Default())
	if err != nil {
		return nil, nil, fmt.Errorf("open event store: %w", err)
	}

	pipeline, err := reviewed.NewPipeline(
		reviewer,
		store,
		reviewed.NewBlobStore(config.DataDir),
		reviewed.NewWriter(config.ReviewsDir),
		slog.Default(),
	)
	if err != nil {
		_ = store.Close()

		return nil, nil, fmt.Errorf("build pipeline: %w", err)
	}

	return pipeline, store, nil
}

// runOnceCommand runs a single pass over all configured projects.
func runOnceCommand(args []string, stdout, stderr io.Writer) int {
	flagSet := newFlagSet("once", stderr)

	configPath := flagSet.String("config", reviewed.DefaultConfigPath, "path to the daemon config JSON")

	if err := flagSet.Parse(args); err != nil {
		return exitUsage
	}

	config, err := loadDaemonConfig(*configPath)
	if err != nil {
		fmt.Fprintf(stderr, "visionreviewd once: %v\n", err)

		return exitFailed
	}

	pipeline, store, err := openPipeline(context.Background(), config)
	if err != nil {
		fmt.Fprintf(stderr, "visionreviewd once: %v\n", err)

		return exitFailed
	}

	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			fmt.Fprintf(stderr, "visionreviewd once: close store: %v\n", closeErr)
		}
	}()

	result, err := pipeline.Pass(context.Background(), config.Projects)

	fmt.Fprintf(stdout,
		"pass complete: %d projects, %d views, %d captured, %d skipped, %d reviewed, %d compared\n",
		result.Projects, result.Views, result.Captured, result.Skipped, result.Reviewed, result.Compared)

	if err != nil {
		fmt.Fprintf(stderr, "visionreviewd once: %v\n", err)

		return exitFailed
	}

	return exitOK
}

// runDiscoverCommand walks root and prints a paste-ready config JSON.
func runDiscoverCommand(args []string, stdout, stderr io.Writer) int {
	flagSet := newFlagSet("discover", stderr)

	if err := flagSet.Parse(args); err != nil {
		return exitUsage
	}

	if flagSet.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: visionreviewd discover ROOT")

		return exitUsage
	}

	suggestions, err := reviewed.DiscoverProjects(flagSet.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "visionreviewd discover: %v\n", err)

		return exitFailed
	}

	suggested, err := reviewed.SuggestedConfigJSON(suggestions)
	if err != nil {
		fmt.Fprintf(stderr, "visionreviewd discover: %v\n", err)

		return exitFailed
	}

	fmt.Fprintln(stdout, suggested)

	return exitOK
}

// runCompareCommand manually compares a BEFORE and AFTER screenshot.
func runCompareCommand(args []string, stdout, stderr io.Writer) int {
	flagSet := newFlagSet("compare", stderr)

	configPath := flagSet.String("config", reviewed.DefaultConfigPath, "path to the daemon config JSON")
	project := flagSet.String("project", "", "project the compared view belongs to")

	if err := flagSet.Parse(args); err != nil {
		return exitUsage
	}

	if *project == "" {
		fmt.Fprintln(stderr, "visionreviewd compare: -project is required")

		return exitUsage
	}

	if flagSet.NArg() != compareArgCount {
		fmt.Fprintln(stderr, "usage: visionreviewd compare -project NAME BEFORE.png AFTER.png")

		return exitUsage
	}

	before, after := flagSet.Arg(0), flagSet.Arg(1)

	config, err := loadDaemonConfig(*configPath)
	if err != nil {
		fmt.Fprintf(stderr, "visionreviewd compare: %v\n", err)

		return exitFailed
	}

	pipeline, store, err := openPipeline(context.Background(), config)
	if err != nil {
		fmt.Fprintf(stderr, "visionreviewd compare: %v\n", err)

		return exitFailed
	}

	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			fmt.Fprintf(stderr, "visionreviewd compare: close store: %v\n", closeErr)
		}
	}()

	if err := pipeline.CompareManually(context.Background(), *project, before, after); err != nil {
		fmt.Fprintf(stderr, "visionreviewd compare: %v\n", err)

		return exitFailed
	}

	fmt.Fprintf(stdout, "comparison written under %s\n", config.ReviewsDir)

	return exitOK
}

// compareArgCount is the positional argument count compare expects.
const compareArgCount = 2

// runDaemonCommand runs the interval loop until SIGINT/SIGTERM.
func runDaemonCommand(args []string, stdout, stderr io.Writer) int {
	flagSet := newFlagSet("run", stderr)

	configPath := flagSet.String("config", reviewed.DefaultConfigPath, "path to the daemon config JSON")

	if err := flagSet.Parse(args); err != nil {
		return exitUsage
	}

	config, err := loadDaemonConfig(*configPath)
	if err != nil {
		fmt.Fprintf(stderr, "visionreviewd run: %v\n", err)

		return exitFailed
	}

	pipeline, store, err := openPipeline(context.Background(), config)
	if err != nil {
		fmt.Fprintf(stderr, "visionreviewd run: %v\n", err)

		return exitFailed
	}

	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			fmt.Fprintf(stderr, "visionreviewd run: close store: %v\n", closeErr)
		}
	}()

	daemon, err := reviewed.NewDaemon(pipeline, config.Projects, config.Interval, slog.Default())
	if err != nil {
		fmt.Fprintf(stderr, "visionreviewd run: %v\n", err)

		return exitFailed
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Fprintf(stdout, "visionreviewd: reviewing every %s, Ctrl-C to stop\n", config.Interval)

	if err := daemon.Run(ctx); err != nil {
		fmt.Fprintf(stderr, "visionreviewd run: %v\n", err)

		return exitFailed
	}

	fmt.Fprintln(stdout, "visionreviewd: stopped")

	return exitOK
}

// runEventsCommand prints the recorded event journal with filters.
func runEventsCommand(args []string, stdout, stderr io.Writer) int {
	flagSet := newFlagSet("events", stderr)

	configPath := flagSet.String("config", reviewed.DefaultConfigPath, "path to the daemon config JSON")
	project := flagSet.String("project", "", "only events of this project")
	view := flagSet.String("view", "", "only events of this view key (e.g. Home--dark--desktop)")
	eventType := flagSet.String("type", "", "only events of this type (view.captured, view.reviewed, view.compared)")
	last := flagSet.Int("last", 0, "print only the last N events (0 prints all)")

	if err := flagSet.Parse(args); err != nil {
		return exitUsage
	}

	config, err := loadDaemonConfig(*configPath)
	if err != nil {
		fmt.Fprintf(stderr, "visionreviewd events: %v\n", err)

		return exitFailed
	}

	store, err := reviewed.OpenStore(filepath.Join(config.DataDir, "events.db"), slog.Default())
	if err != nil {
		fmt.Fprintf(stderr, "visionreviewd events: %v\n", err)

		return exitFailed
	}

	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			fmt.Fprintf(stderr, "visionreviewd events: close store: %v\n", closeErr)
		}
	}()

	events, err := store.AllEvents(context.Background())
	if err != nil {
		fmt.Fprintf(stderr, "visionreviewd events: %v\n", err)

		return exitFailed
	}

	summaries := filterEventSummaries(reviewed.SummarizeEvents(events), *project, *view, *eventType)
	if *last > 0 && len(summaries) > *last {
		summaries = summaries[len(summaries)-*last:]
	}

	for _, summary := range summaries {
		fmt.Fprintf(stdout, "%s  %-15s  %s:%s  v%d  %s\n",
			summary.OccurredAt.UTC().Format(eventsTimeFormat),
			summary.Type,
			summary.Project,
			summary.ViewKey,
			summary.Version,
			summary.Detail)
	}

	fmt.Fprintf(stdout, "%d events\n", len(summaries))

	return exitOK
}

// eventsTimeFormat stamps event lines in the events command output.
const eventsTimeFormat = "2006-01-02 15:04:05"

// filterEventSummaries keeps only the summaries matching every non-empty
// filter.
func filterEventSummaries(
	summaries []reviewed.EventSummary,
	project string,
	view string,
	eventType string,
) []reviewed.EventSummary {
	filtered := make([]reviewed.EventSummary, 0, len(summaries))

	for _, summary := range summaries {
		if project != "" && summary.Project != project {
			continue
		}

		if view != "" && summary.ViewKey.String() != view {
			continue
		}

		if eventType != "" && summary.Type != eventType {
			continue
		}

		filtered = append(filtered, summary)
	}

	return filtered
}

// runReplayCommand rebuilds the reviews directory from the event journal.
func runReplayCommand(args []string, stdout, stderr io.Writer) int {
	flagSet := newFlagSet("replay", stderr)

	configPath := flagSet.String("config", reviewed.DefaultConfigPath, "path to the daemon config JSON")

	if err := flagSet.Parse(args); err != nil {
		return exitUsage
	}

	config, err := loadDaemonConfig(*configPath)
	if err != nil {
		fmt.Fprintf(stderr, "visionreviewd replay: %v\n", err)

		return exitFailed
	}

	store, err := reviewed.OpenStore(filepath.Join(config.DataDir, "events.db"), slog.Default())
	if err != nil {
		fmt.Fprintf(stderr, "visionreviewd replay: %v\n", err)

		return exitFailed
	}

	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			fmt.Fprintf(stderr, "visionreviewd replay: close store: %v\n", closeErr)
		}
	}()

	result, err := reviewed.Replay(context.Background(), store, reviewed.NewWriter(config.ReviewsDir))

	fmt.Fprintf(stdout, "replay complete: %d projects, %d views, %d reviews, %d comparisons\n",
		result.Projects, result.Views, result.Reviews, result.Comparisons)

	if err != nil {
		fmt.Fprintf(stderr, "visionreviewd replay: %v\n", err)

		return exitFailed
	}

	return exitOK
}

// runDoctorCommand checks config, directories, globs, and the model endpoint.
func runDoctorCommand(args []string, stdout, stderr io.Writer) int {
	flagSet := newFlagSet("doctor", stderr)

	configPath := flagSet.String("config", reviewed.DefaultConfigPath, "path to the daemon config JSON")

	if err := flagSet.Parse(args); err != nil {
		return exitUsage
	}

	config, err := loadDaemonConfig(*configPath)
	if err != nil {
		fmt.Fprintf(stderr, "visionreviewd doctor: %v\n", err)

		return exitFailed
	}

	checks := doctorChecks(context.Background(), config)

	failed := 0

	for _, check := range checks {
		status := "ok"

		if !check.ok {
			status = "FAIL"
			failed++
		}

		fmt.Fprintf(stdout, "%-4s %s: %s\n", status, check.name, check.detail)
	}

	fmt.Fprintf(stdout, "%d checks, %d failed\n", len(checks), failed)

	if failed > 0 {
		return exitFailed
	}

	return exitOK
}

// doctorEndpointTimeout bounds the model endpoint probe.
const doctorEndpointTimeout = 10 * time.Second

// doctorEndpointName labels the model endpoint check in doctor output.
const doctorEndpointName = "model endpoint"

// doctorCheck is the outcome of one health probe.
type doctorCheck struct {
	name   string
	ok     bool
	detail string
}

// doctorChecks runs every probe: writability of both directories, glob match
// counts per project, and the model endpoint's model list.
func doctorChecks(ctx context.Context, config reviewed.Config) []doctorCheck {
	checks := make([]doctorCheck, 0, len(config.Projects))
	checks = append(checks, checkWritableDir("dataDir", config.DataDir))
	checks = append(checks, checkWritableDir("reviewsDir", config.ReviewsDir))
	checks = append(checks, checkProjectGlobs(config)...)
	checks = append(checks, checkModelEndpoint(ctx, config))

	return checks
}

// checkWritableDir probes that dir exists (or can be created) and a file can
// be written and removed inside it.
func checkWritableDir(name string, dir string) doctorCheck {
	if err := os.MkdirAll(dir, reviewsDirPermission); err != nil {
		return doctorCheck{name: name, ok: false, detail: fmt.Sprintf("create %s: %v", dir, err)}
	}

	probe := filepath.Join(dir, ".visionreviewd-doctor-probe")

	if err := os.WriteFile(probe, []byte("probe"), reviewsFilePermission); err != nil {
		return doctorCheck{name: name, ok: false, detail: fmt.Sprintf("write %s: %v", probe, err)}
	}

	if err := os.Remove(probe); err != nil {
		return doctorCheck{name: name, ok: false, detail: fmt.Sprintf("remove %s: %v", probe, err)}
	}

	return doctorCheck{name: name, ok: true, detail: dir}
}

// reviewsDirPermission and reviewsFilePermission mirror the Writer's modes so
// doctor probes with the same access the daemon will use.
const (
	reviewsDirPermission  = 0o750
	reviewsFilePermission = 0o640
)

// checkProjectGlobs counts how many screenshots every project's globs match.
// A project with zero matches is reported as failing: it would never be
// reviewed.
func checkProjectGlobs(config reviewed.Config) []doctorCheck {
	checks := make([]doctorCheck, 0, len(config.Projects))

	for _, project := range slices.Sorted(maps.Keys(config.Projects)) {
		total := 0

		for _, pattern := range config.Projects[project] {
			matches, err := filepath.Glob(pattern)
			if err != nil {
				checks = append(checks, doctorCheck{
					name:   "globs " + project,
					ok:     false,
					detail: fmt.Sprintf("pattern %s: %v", pattern, err),
				})

				continue
			}

			total += len(matches)
		}

		checks = append(checks, doctorCheck{
			name:   "globs " + project,
			ok:     total > 0,
			detail: fmt.Sprintf("%d screenshots match", total),
		})
	}

	return checks
}

// checkModelEndpoint fetches {baseUrl}/models and verifies the configured
// model is offered.
func checkModelEndpoint(ctx context.Context, config reviewed.Config) doctorCheck {
	modelsURL := strings.TrimSuffix(config.BaseURL, "/") + "/models"

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsURL, nil)
	if err != nil {
		return doctorCheck{name: doctorEndpointName, ok: false, detail: fmt.Sprintf("build request: %v", err)}
	}

	if config.APIKey != "" {
		request.Header.Set("Authorization", "Bearer "+config.APIKey)
	}

	client := &http.Client{Timeout: doctorEndpointTimeout}

	response, err := client.Do(request)
	if err != nil {
		return doctorCheck{name: doctorEndpointName, ok: false, detail: fmt.Sprintf("%s: %v", modelsURL, err)}
	}

	// Read and close eagerly (instead of defer) so every failure below can
	// report through the check detail instead of os.Stderr, which would
	// bypass the command's injected writers.
	body, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()

	if readErr != nil {
		return doctorCheck{name: doctorEndpointName, ok: false, detail: fmt.Sprintf("read %s: %v", modelsURL, readErr)}
	}

	if closeErr != nil {
		return doctorCheck{
			name:   doctorEndpointName,
			ok:     false,
			detail: fmt.Sprintf("close %s: %v", modelsURL, closeErr),
		}
	}

	if response.StatusCode != http.StatusOK {
		return doctorCheck{
			name:   doctorEndpointName,
			ok:     false,
			detail: fmt.Sprintf("%s: status %s", modelsURL, response.Status),
		}
	}

	var listed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &listed); err != nil {
		return doctorCheck{name: doctorEndpointName, ok: false, detail: fmt.Sprintf("decode %s: %v", modelsURL, err)}
	}

	for _, model := range listed.Data {
		if model.ID == config.Model {
			return doctorCheck{name: doctorEndpointName, ok: true, detail: config.Model + " listed at " + modelsURL}
		}
	}

	return doctorCheck{
		name: doctorEndpointName,
		ok:   false,
		detail: fmt.Sprintf("model %q not listed at %s (%d models offered)",
			config.Model, modelsURL, len(listed.Data)),
	}
}
