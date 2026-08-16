package cli_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"

	"charm.land/fantasy"
	"github.com/larsartmann/vision-review-agent/internal/cli"
	"github.com/larsartmann/vision-review-agent/pkg/vision"
	"github.com/stretchr/testify/require"
)

// stubModel is the smallest possible fantasy.LanguageModel: NewAgent never
// invokes it, it only checks non-nilness.
type stubModel struct{}

func (stubModel) Generate(_ context.Context, _ fantasy.Call) (*fantasy.Response, error) {
	return nil, errors.New("unused in cli tests")
}

func (stubModel) Provider() string { return "stub" }

func (stubModel) Model() string { return "stub-model" }

func (stubModel) Stream(_ context.Context, _ fantasy.Call) (fantasy.StreamResponse, error) {
	return nil, errors.New("unused in cli tests")
}

func (stubModel) GenerateObject(
	_ context.Context,
	_ fantasy.ObjectCall,
) (*fantasy.ObjectResponse, error) {
	return nil, errors.New("unused in cli tests")
}

func (stubModel) StreamObject(
	_ context.Context,
	_ fantasy.ObjectCall,
) (fantasy.ObjectStreamResponse, error) {
	return nil, errors.New("unused in cli tests")
}

func TestNewAgentRejectsInvalidTemperature(t *testing.T) {
	t.Parallel()

	_, err := cli.NewAgent(stubModel{}, "review this UI", 3.5)
	require.ErrorIs(t, err, vision.ErrInvalidTemperature)
	require.Contains(t, err.Error(), "temperature=3.50", "error must carry the temperature context")
}

func TestNewAgentRejectsNilModel(t *testing.T) {
	t.Parallel()

	_, err := cli.NewAgent(nil, "review this UI")
	require.ErrorIs(t, err, vision.ErrNoModel)
	require.Contains(t, err.Error(), "temperature=0.30", "error must carry the default temperature context")
}

func TestNewAgentBuildsAgentWithDefaultTemperature(t *testing.T) {
	t.Parallel()

	agent, err := cli.NewAgent(stubModel{}, "review this UI")
	require.NoError(t, err)
	require.NotNil(t, agent)
}

func TestRequireArgcPassesWithEnoughArgs(t *testing.T) {
	t.Parallel()

	require.GreaterOrEqual(t, len(os.Args), 1)

	cli.RequireArgc(1)
}

func TestRequireArgcExitsWithUsageWhenShort(t *testing.T) {
	t.Parallel()

	if os.Getenv("GO_TEST_REQUIRE_ARGC_CHILD") == "1" {
		cli.RequireArgc(len(os.Args) + 99)

		return
	}

	//nolint:gosec // re-exec of the test binary itself is the canonical subprocess-test pattern
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=TestRequireArgcExitsWithUsageWhenShort")

	cmd.Env = append(os.Environ(), "GO_TEST_REQUIRE_ARGC_CHILD=1")

	var exitErr *exec.ExitError

	err := cmd.Run()
	require.ErrorAs(t, err, &exitErr, "RequireArgc must exit the process when short on args")
	require.Equal(t, 1, exitErr.ExitCode())
}
