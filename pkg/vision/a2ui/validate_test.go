package a2ui

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// validStream builds the canonical valid sequence: create, components, data
// model, delete.
func validStream() []Message {
	return []Message{
		NewCreateSurface("main", DefaultCatalogID),
		NewUpdateComponents("main",
			NewColumn(RootID, "title", "button"),
			NewText("title", "Hello", TextH1),
			NewButton("button", "button-label", "clicked"),
			NewText("button-label", "Click me", ""),
		),
		NewUpdateDataModel("main", "", map[string]any{"greeting": "Hello"}),
		NewDeleteSurface("main"),
	}
}

// mutate builds a fresh valid stream and hands it to fn for mutation.
func mutate(t *testing.T, fn func(messages []Message)) []Message {
	t.Helper()

	messages := validStream()
	fn(messages)

	return messages
}

// updateComponentsAt returns the UpdateComponents message of a validStream
// (always index 1), asserting the type.
func updateComponentsAt(t *testing.T, messages []Message) *UpdateComponents {
	t.Helper()

	msg, ok := messages[1].(*UpdateComponents)
	require.True(t, ok, "message 1 must be updateComponents")

	return msg
}

func TestValidateAcceptsValidStreams(t *testing.T) {
	t.Parallel()

	require.NoError(t, Validate(validStream()))
	require.Empty(t, Issues(validStream()))

	t.Run("surface may be recreated after deletion", func(t *testing.T) {
		t.Parallel()

		messages := append(validStream(),
			NewCreateSurface("main", DefaultCatalogID),
			NewUpdateComponents("main", NewText(RootID, "Hello again", "")),
		)
		require.NoError(t, Validate(messages))
	})

	t.Run("multiple updateComponents for one surface", func(t *testing.T) {
		t.Parallel()

		messages := []Message{
			NewCreateSurface("main", DefaultCatalogID),
			NewUpdateComponents("main", NewText(RootID, "First paint", "")),
			NewUpdateComponents("main", NewText(RootID, "Refreshed", "")),
		}
		require.NoError(t, Validate(messages))
	})
}

func TestValidateRejectsStructuralProblems(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		messages func(t *testing.T) []Message
		contains string
	}{
		{
			name: "missing root",
			messages: func(t *testing.T) []Message {
				t.Helper()

				return mutate(t, func(messages []Message) {
					update := updateComponentsAt(t, messages)
					for i := range update.Components {
						if update.Components[i].ID == RootID {
							update.Components[i].ID = "was-root"
						}
					}
				})
			},
			contains: `no component with id "root"`,
		},
		{
			name: "duplicate ids",
			messages: func(t *testing.T) []Message {
				t.Helper()

				return mutate(t, func(messages []Message) {
					update := updateComponentsAt(t, messages)
					update.Components = append(update.Components, NewText("title", "Dupe", ""))
				})
			},
			contains: "duplicate id",
		},
		{
			name: "dangling child reference",
			messages: func(t *testing.T) []Message {
				t.Helper()

				return mutate(t, func(messages []Message) {
					update := updateComponentsAt(t, messages)
					update.Components[0] = NewColumn(RootID, "title", "ghost")
				})
			},
			contains: `child "ghost" is not defined`,
		},
		{
			name: "child cycle",
			messages: func(t *testing.T) []Message {
				t.Helper()

				return mutate(t, func(messages []Message) {
					update := updateComponentsAt(t, messages)
					update.Components[0] = NewColumn(RootID, "title")
					update.Components[1] = NewColumn("title", RootID)
				})
			},
			contains: "reference cycle",
		},
		{
			name: "both child and children",
			messages: func(t *testing.T) []Message {
				t.Helper()

				return mutate(t, func(messages []Message) {
					update := updateComponentsAt(t, messages)
					update.Components[0] = Component{
						ID:       RootID,
						Kind:     "Card",
						Child:    new("title"),
						Children: StaticChildren("title"),
					}
				})
			},
			contains: "both child and children set",
		},
		{
			name: "empty components list",
			messages: func(t *testing.T) []Message {
				t.Helper()

				return mutate(t, func(messages []Message) {
					update := updateComponentsAt(t, messages)
					update.Components = nil
				})
			},
			contains: "components list is empty",
		},
		{
			name: "update before create",
			messages: func(t *testing.T) []Message {
				t.Helper()

				messages := validStream()

				return messages[1:]
			},
			contains: `surface "main" was never created`,
		},
		{
			name: "duplicate create",
			messages: func(t *testing.T) []Message {
				t.Helper()

				messages := validStream()[:3] // create, components, data model; surface still open

				return append(messages, NewCreateSurface("main", DefaultCatalogID))
			},
			contains: "already exists",
		},
		{
			name: "delete unknown surface",
			messages: func(t *testing.T) []Message {
				t.Helper()

				return []Message{NewDeleteSurface("ghost")}
			},
			contains: `surface "ghost" was never created`,
		},
		{
			name: "bad data model path",
			messages: func(t *testing.T) []Message {
				t.Helper()

				return mutate(t, func(messages []Message) {
					messages[2] = NewUpdateDataModel("main", "user/name", "x")
				})
			},
			contains: "must be a JSON Pointer",
		},
		{
			name: "empty component id",
			messages: func(t *testing.T) []Message {
				t.Helper()

				return mutate(t, func(messages []Message) {
					update := updateComponentsAt(t, messages)
					update.Components = append(
						update.Components,
						Component{Kind: "Text", Props: map[string]any{"text": "x"}},
					)
				})
			},
			contains: "empty id",
		},
		{
			name: "empty component type",
			messages: func(t *testing.T) []Message {
				t.Helper()

				return mutate(t, func(messages []Message) {
					update := updateComponentsAt(t, messages)
					update.Components = append(update.Components, Component{ID: "loose"})
				})
			},
			contains: "empty component type",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := Validate(tc.messages(t))
			require.ErrorIs(t, err, ErrValidation)
			require.ErrorContains(t, err, tc.contains)
		})
	}
}

func TestValidateDetectsDeepCycle(t *testing.T) {
	t.Parallel()

	messages := []Message{
		NewCreateSurface("main", DefaultCatalogID),
		NewUpdateComponents("main",
			NewColumn(RootID, "a"),
			NewColumn("a", "b"),
			NewColumn("b", "a"),
		),
	}

	err := Validate(messages)
	require.ErrorIs(t, err, ErrValidation)
	require.ErrorIs(t, err, ErrComponentCycle)
}

func TestValidatePermitsOrphans(t *testing.T) {
	t.Parallel()

	messages := []Message{
		NewCreateSurface("main", DefaultCatalogID),
		NewUpdateComponents("main",
			NewText(RootID, "Root", ""),
			NewText("orphan", "Unreferenced but legal", ""),
		),
	}

	require.NoError(t, Validate(messages))
}

func TestValidateRejectsUnknownVersion(t *testing.T) {
	t.Parallel()

	messages := []Message{NewCreateSurface("main", DefaultCatalogID)}
	create, ok := messages[0].(*CreateSurface)
	require.True(t, ok)

	create.version = "v1.0"

	err := Validate(messages)
	require.ErrorContains(t, err, "unknown envelope version")
}

func TestValidateNilStream(t *testing.T) {
	t.Parallel()

	require.NoError(t, Validate(nil))
}

func TestNilStreamAndIssueData(t *testing.T) {
	t.Parallel()

	require.NoError(t, Validate(nil))

	issues := Issues([]Message{nil})
	require.Len(t, issues, 1)
	require.Equal(t, 0, issues[0].MessageIndex)
	require.Equal(t, "message is nil", issues[0].Detail)
}
