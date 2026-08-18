package a2ui

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildPrompt(t *testing.T) {
	t.Parallel()

	prompt := BuildPrompt("")

	require.Contains(t, prompt, "A2UI surface")
	require.Contains(t, prompt, "Column: children (required")
	require.Contains(t, prompt, "Button: child (required")
	require.Contains(t, prompt, `(required) marks a mandatory property`)
	require.Contains(t, prompt, `"properties": {"text": "Hello", "variant": "h1"}`)
	require.Contains(t, prompt, `"root"`)
	require.Contains(t, prompt, DefaultTask)

	custom := BuildPrompt("Design a login form")
	require.Contains(t, custom, "Design a login form")
	require.NotContains(t, custom, DefaultTask, "custom task must replace the default")
}
