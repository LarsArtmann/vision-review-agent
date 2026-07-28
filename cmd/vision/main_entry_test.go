package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMainEntryVersionFlag builds the real binary and runs it with -version,
// verifying the end-to-end binary execution path (flag parsing, version output,
// exit code). This catches issues that unit tests miss (link errors, missing
// imports, ldflags injection).
func TestMainEntryVersionFlag(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "vision")

	cmd := exec.Command("go", "build", "-o", binary, "./")
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run(), "go build must succeed")

	out, err := exec.Command(binary, "-version").Output()
	require.NoError(t, err, "binary must exit 0 with -version")
	require.True(t, strings.HasPrefix(string(out), "vision "), "output must start with 'vision '")
}

// TestMainEntryNoArgsExitsNonZero verifies that running the binary without
// positional arguments exits non-zero (usage error).
func TestMainEntryNoArgsExitsNonZero(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "vision")

	build := exec.Command("go", "build", "-o", binary, "./")
	build.Stderr = os.Stderr
	require.NoError(t, build.Run())

	err := exec.Command(binary).Run()
	require.Error(t, err, "binary must exit non-zero when no images provided")
}
