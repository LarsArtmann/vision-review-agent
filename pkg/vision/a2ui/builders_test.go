package a2ui

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildersProduceWireShapes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		component Component
		kind      string
		props     map[string]any
	}{
		{
			name:      "list with direction",
			component: NewList("lst", DirectionHorizontal, "a", "b"),
			kind:      KindList,
			props:     map[string]any{propDirection: "horizontal"},
		},
		{
			name:      "list without direction",
			component: NewList("lst", "", "a"),
			kind:      KindList,
			props:     nil,
		},
		{
			name:      "modal",
			component: NewModal("m", "open-btn", "sheet"),
			kind:      KindModal,
			props:     map[string]any{propTrigger: "open-btn", propContent: "sheet"},
		},
		{
			name: "tabs",
			component: NewTabs("tabs",
				Tab{Title: "One", ChildID: "pane1"},
				Tab{Title: "Two", ChildID: "pane2"},
			),
			kind: KindTabs,
			props: map[string]any{propTabs: []map[string]any{
				{propTitle: "One", propChild: "pane1"},
				{propTitle: "Two", propChild: "pane2"},
			}},
		},
		{
			name:      "checkbox",
			component: NewCheckBox("cb", "Subscribe", true),
			kind:      KindCheckBox,
			props:     map[string]any{propLabel: "Subscribe", propValue: true},
		},
		{
			name: "choicepicker with literal selection",
			component: NewChoicePicker("cp", "Pick", []ChoicePickerOption{
				{Label: "Red", Value: "red"}, {Label: "Blue", Value: "blue"},
			}, []string{"red"}, ChoiceMutuallyExclusive),
			kind: KindChoicePicker,
			props: map[string]any{
				propLabel: "Pick",
				propOptions: []map[string]any{
					{propLabel: "Red", propValue: "red"},
					{propLabel: "Blue", propValue: "blue"},
				},
				propValue:   []string{"red"},
				propVariant: ChoiceMutuallyExclusive,
			},
		},
		{
			name: "choicepicker with bound selection",
			component: NewChoicePicker("cp", "Pick", []ChoicePickerOption{
				{Label: "Red", Value: "red"},
			}, Bind("/colors"), ""),
			kind: KindChoicePicker,
			props: map[string]any{
				propLabel:   "Pick",
				propOptions: []map[string]any{{propLabel: "Red", propValue: "red"}},
				propValue:   map[string]any{"path": "/colors"},
			},
		},
		{
			name:      "textfield full",
			component: NewTextField("tf", "Email", "a@b.c", FieldObscured),
			kind:      KindTextField,
			props:     map[string]any{propLabel: "Email", propValue: "a@b.c", propVariant: FieldObscured},
		},
		{
			name:      "textfield minimal",
			component: NewTextField("tf", "Name", "", ""),
			kind:      KindTextField,
			props:     map[string]any{propLabel: "Name"},
		},
		{
			name:      "datetimeinput",
			component: NewDateTimeInput("dt", "Starts", "2026-08-19T10:00", true, false),
			kind:      KindDateTimeInput,
			props:     map[string]any{propLabel: "Starts", propValue: "2026-08-19T10:00", propEnableDate: true},
		},
		{
			name:      "slider",
			component: NewSlider("sl", "Volume", 7, 0, 11),
			kind:      KindSlider,
			props: map[string]any{
				propLabel: "Volume",
				propValue: float64(7),
				propMin:   float64(0),
				propMax:   float64(11),
			},
		},
		{
			name:      "audioplayer",
			component: NewAudioPlayer("au", "https://example.com/a.mp3", "Episode 1"),
			kind:      KindAudioPlayer,
			props:     map[string]any{propURL: "https://example.com/a.mp3", propDescription: "Episode 1"},
		},
		{
			name:      "video",
			component: NewVideo("vid", "https://example.com/v.mp4"),
			kind:      KindVideo,
			props:     map[string]any{propURL: "https://example.com/v.mp4"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.kind, tc.component.Kind)
			require.Equal(t, tc.props, tc.component.Props)

			encoded, err := json.Marshal(tc.component)
			require.NoError(t, err)

			// Round trips preserve the wire shape exactly; Go slice types
			// inside Props may widen ([]map[string]any -> []any), so the
			// comparison is on the re-encoded JSON, not the struct.
			var decoded Component
			require.NoError(t, json.Unmarshal(encoded, &decoded))

			reencoded, err := json.Marshal(decoded)
			require.NoError(t, err)
			require.JSONEq(t, string(encoded), string(reencoded))
		})
	}
}
