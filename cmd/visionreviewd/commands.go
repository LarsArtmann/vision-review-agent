package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"

	reviewed "github.com/larsartmann/vision-review-agent/internal/reviewd"
)

// errNotImplemented marks subcommands whose backing feature lands in a later
// task; it keeps dispatch complete while the bodies catch up.
var errNotImplemented = errors.New("not implemented yet")

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

// runDaemonCommand runs the interval loop until the context is cancelled.
// stdout gains output (per-pass summaries) when the daemon loop lands.
//
//nolint:unparam // stdout writes output once the command body lands
func runDaemonCommand(args []string, stdout, stderr io.Writer) int {
	_ = args

	fmt.Fprintf(stderr, "visionreviewd run: %v (landing with the daemon loop task)\n", errNotImplemented)

	return exitFailed
}

// runEventsCommand prints the recorded event journal.
//
//nolint:unparam // stdout writes output once the command body lands
func runEventsCommand(args []string, stdout, stderr io.Writer) int {
	_ = args

	fmt.Fprintf(stderr, "visionreviewd events: %v\n", errNotImplemented)

	return exitFailed
}

// runReplayCommand rebuilds the reviews directory from the event journal.
//
//nolint:unparam // stdout writes output once the command body lands
func runReplayCommand(args []string, stdout, stderr io.Writer) int {
	_ = args

	fmt.Fprintf(stderr, "visionreviewd replay: %v\n", errNotImplemented)

	return exitFailed
}

// runDoctorCommand checks config, directories, globs, and the model endpoint.
//
//nolint:unparam // stdout writes output once the command body lands
func runDoctorCommand(args []string, stdout, stderr io.Writer) int {
	_ = args

	fmt.Fprintf(stderr, "visionreviewd doctor: %v\n", errNotImplemented)

	return exitFailed
}
