package a2ui

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMessageWireShape(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		msg  Message
		want string
	}{
		{
			name: "createSurface",
			msg:  NewCreateSurface("main", DefaultCatalogID),
			want: `{"version":"v0.9.1","createSurface":{"surfaceId":"main","catalogId":"` + DefaultCatalogID + `"}}`,
		},
		{
			name: "createSurface with theme",
			msg: func() Message {
				msg := NewCreateSurface("main", "cat")
				msg.Theme = map[string]any{"primaryColor": "#FF0000"}

				return msg
			}(),
			want: `{"version":"v0.9.1","createSurface":{"surfaceId":"main","catalogId":"cat","theme":{"primaryColor":"#FF0000"}}}`,
		},
		{
			name: "updateComponents",
			msg:  NewUpdateComponents("main", NewText(RootID, "Hello", "")),
			want: `{"version":"v0.9.1","updateComponents":{"surfaceId":"main","components":[{"id":"root","component":"Text","text":"Hello"}]}}`,
		},
		{
			name: "updateDataModel whole model",
			msg:  NewUpdateDataModel("main", "", map[string]any{"user": "ada"}),
			want: `{"version":"v0.9.1","updateDataModel":{"surfaceId":"main","value":{"user":"ada"}}}`,
		},
		{
			name: "updateDataModel at path",
			msg:  NewUpdateDataModel("main", "/user/name", "ada"),
			want: `{"version":"v0.9.1","updateDataModel":{"surfaceId":"main","path":"/user/name","value":"ada"}}`,
		},
		{
			name: "updateDataModel remove",
			msg:  NewRemoveDataModelEntry("main", "/user/name"),
			want: `{"version":"v0.9.1","updateDataModel":{"surfaceId":"main","path":"/user/name"}}`,
		},
		{
			name: "deleteSurface",
			msg:  NewDeleteSurface("main"),
			want: `{"version":"v0.9.1","deleteSurface":{"surfaceId":"main"}}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			encoded, err := json.Marshal(tc.msg)
			require.NoError(t, err)
			require.JSONEq(t, tc.want, string(encoded))
		})
	}
}

func TestUnmarshalMessageDispatch(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		wire string
		want Message
	}{
		{
			name: "createSurface v0.9",
			wire: `{"version":"v0.9","createSurface":{"surfaceId":"s","catalogId":"c"}}`,
			want: &CreateSurface{SurfaceID: "s", CatalogID: "c", version: "v0.9"},
		},
		{
			name: "updateComponents",
			wire: `{"version":"v0.9.1","updateComponents":{"surfaceId":"s","components":[{"id":"root","component":"Text","text":"hi"}]}}`,
			want: &UpdateComponents{
				SurfaceID: "s",
				Components: []Component{{
					ID:    RootID,
					Kind:  "Text",
					Props: map[string]any{"text": "hi"},
				}},
				version: VersionV091,
			},
		},
		{
			name: "updateDataModel remove detected",
			wire: `{"version":"v0.9.1","updateDataModel":{"surfaceId":"s","path":"/a"}}`,
			want: &UpdateDataModel{SurfaceID: "s", Path: "/a", Remove: true, version: VersionV091},
		},
		{
			name: "deleteSurface",
			wire: `{"version":"v0.9.1","deleteSurface":{"surfaceId":"s"}}`,
			want: &DeleteSurface{SurfaceID: "s", version: VersionV091},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			msg, err := UnmarshalMessage([]byte(tc.wire))
			require.NoError(t, err)
			require.Equal(t, tc.want, msg)
		})
	}
}

func TestUnmarshalMessageRejectsBadEnvelopes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		wire string
	}{
		{"no kind key", `{"version":"v0.9.1"}`},
		{
			"two kind keys",
			`{"version":"v0.9.1","createSurface":{"surfaceId":"s","catalogId":"c"},"deleteSurface":{"surfaceId":"s"}}`,
		},
		{"unknown kind key", `{"version":"v0.9.1","explodeSurface":{"surfaceId":"s"}}`},
		{"not json", `nope`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := UnmarshalMessage([]byte(tc.wire))
			require.Error(t, err)
		})
	}
}

func TestJSONLRoundtrip(t *testing.T) {
	t.Parallel()

	messages := []Message{
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

	encoded, err := MarshalJSONL(messages)
	require.NoError(t, err)

	decoded, err := UnmarshalJSONL(encoded)
	require.NoError(t, err)
	require.Len(t, decoded, len(messages))

	for i := range messages {
		require.Equal(t, messages[i], decoded[i], "message %d must survive the roundtrip", i)
	}
}

func TestJSONLSkipsBlankLinesAndReportsLineNumbers(t *testing.T) {
	t.Parallel()

	decoded, err := UnmarshalJSONL([]byte("\n" + `{"version":"v0.9.1","deleteSurface":{"surfaceId":"s"}}` + "\n\n"))
	require.NoError(t, err)
	require.Len(t, decoded, 1)

	_, err = UnmarshalJSONL([]byte(`{"version":"v0.9.1"}\n{bogus}`))
	require.ErrorContains(t, err, "line ")
}

func TestUpdateDataModelExplicitNullIsAWrite(t *testing.T) {
	t.Parallel()

	// The spec allows "value": null as a legal write; omission means remove.
	// The two must stay distinguishable across a decode/encode round trip.
	writeNull := []byte(`{"version":"v0.9.1","updateDataModel":{"surfaceId":"s","path":"/a","value":null}}`)
	decoded, err := UnmarshalMessage(writeNull)
	require.NoError(t, err)

	update, ok := decoded.(*UpdateDataModel)
	require.True(t, ok)
	require.False(t, update.Remove, "explicit null must decode as a write, not a remove")
	require.Nil(t, update.Value)

	reencoded, err := json.Marshal(decoded)
	require.NoError(t, err)
	require.JSONEq(t, string(writeNull), string(reencoded))

	remove := []byte(`{"version":"v0.9.1","updateDataModel":{"surfaceId":"s","path":"/a"}}`)
	decodedRemove, err := UnmarshalMessage(remove)
	require.NoError(t, err)

	updateRemove, ok := decodedRemove.(*UpdateDataModel)
	require.True(t, ok)
	require.True(t, updateRemove.Remove, "omitted value with a path must decode as a remove")

	reencodedRemove, err := json.Marshal(decodedRemove)
	require.NoError(t, err)
	require.JSONEq(t, string(remove), string(reencodedRemove))
}

func TestUnmarshalMessageRejectsUnknownEnvelopeKeys(t *testing.T) {
	t.Parallel()

	// The official schema sets additionalProperties: false on the envelope;
	// unknown top-level keys are rejected instead of silently ignored.
	_, err := UnmarshalMessage(
		[]byte(`{"version":"v0.9.1","surfaceProperties":{"a":1},"deleteSurface":{"surfaceId":"s"}}`),
	)
	require.ErrorIs(t, err, ErrMalformedMessage)
	require.ErrorContains(t, err, "unknown envelope key")

	// Payload-level leniency is deliberate: additive payload fields from
	// newer minor revisions must not break decoding.
	decoded, err := UnmarshalMessage(
		[]byte(`{"version":"v0.9.1","deleteSurface":{"surfaceId":"s","futureField":true}}`),
	)
	require.NoError(t, err)
	require.Equal(t, "s", decoded.Surface())
}
