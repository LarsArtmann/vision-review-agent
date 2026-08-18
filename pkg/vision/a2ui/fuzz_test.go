package a2ui

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// The fuzz targets below pin the codec entry points against hostile or
// malformed input (plan item M13): decoding must never panic, and anything
// that decodes must re-encode. Seeds mirror the wire-format corners: valid
// envelopes, two-kind and unknown-kind envelopes, CRLF/blank/partial JSONL
// lines, and the static/dynamic ChildList shapes.

func FuzzUnmarshalMessage(f *testing.F) {
	f.Add([]byte(`{"version":"v0.9.1","createSurface":{"surfaceId":"s","catalogId":"c"}}`))
	f.Add([]byte(`{"version":"v0.9","deleteSurface":{"surfaceId":"s"}}`))
	f.Add([]byte(`{"version":"v0.9.1","createSurface":{"surfaceId":"s"},"deleteSurface":{"surfaceId":"s"}}`))
	f.Add([]byte(`{"version":"v0.9.1","spinSurface":{"surfaceId":"s"}}`))
	f.Add(
		[]byte(
			`{"version":"v0.9.1","updateComponents":{"surfaceId":"s","components":[{"id":"root","component":"Text","text":"x"}]}}`,
		),
	)
	f.Add([]byte(`{"version":"v0.9.1","updateDataModel":{"surfaceId":"s","path":"/a","value":null}}`))
	f.Add([]byte(`not json`))
	f.Add([]byte(`[]`))
	f.Add([]byte(`{"version":`))
	f.Add([]byte(`{"version":"v0.9.1","updateComponents":{"surfaceId":"s","components":"nope"}}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		message, err := UnmarshalMessage(data)
		if err != nil {
			return
		}

		require.NotNil(t, message)

		// Whatever decodes must re-encode to valid JSON.
		encoded, err := json.Marshal(message)
		require.NoError(t, err)

		// And the re-encoded form must decode again to the same message.
		decoded, err := UnmarshalMessage(encoded)
		require.NoError(t, err)
		require.Equal(t, message, decoded)
	})
}

func FuzzUnmarshalJSONL(f *testing.F) {
	f.Add([]byte("{\"version\":\"v0.9.1\",\"deleteSurface\":{\"surfaceId\":\"s\"}}"))
	f.Add(
		[]byte(
			"{\"version\":\"v0.9.1\",\"deleteSurface\":{\"surfaceId\":\"s\"}}\n{\"version\":\"v0.9.1\",\"deleteSurface\":{\"surfaceId\":\"t\"}}",
		),
	)
	f.Add([]byte("\r\n{\"version\":\"v0.9.1\",\"deleteSurface\":{\"surfaceId\":\"s\"}}\r\n"))
	f.Add([]byte("\n\n\n"))
	f.Add([]byte("{\"version\":\"v0.9.1\",\"deleteSurface\":{\"surfaceId\":\"s\"}}\n{bogus}"))
	f.Add([]byte("partial line without newline"))
	f.Add([]byte(""))

	f.Fuzz(func(t *testing.T, data []byte) {
		messages, err := UnmarshalJSONL(data)
		if err != nil {
			// Errors must locate the offending line.
			require.ErrorContains(t, err, "line ")

			return
		}

		for _, message := range messages {
			require.NotNil(t, message)
		}

		// Decoded streams must re-encode.
		_, err = MarshalJSONL(messages)
		require.NoError(t, err)
	})
}

func FuzzComponentUnmarshal(f *testing.F) {
	f.Add([]byte(`{"id":"root","component":"Column","children":["a","b"]}`))
	f.Add([]byte(`{"id":"b","component":"Button","child":"l","variant":"primary","action":{"event":{"name":"x"}}}`))
	f.Add([]byte(`{"id":"t","component":"Text","text":"hi","extra":1,"nested":{"deep":[true,null]}}`))
	f.Add([]byte(`{"id":"a","component":"Image","accessibility":{"label":"Logo"}}`))
	f.Add([]byte(`{"component":"Text"}`))
	f.Add([]byte(`{"id":42}`))
	f.Add([]byte(`"scalar"`))
	f.Add([]byte(`{"children":{"componentId":"row","path":"/items"}}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var component Component
		if err := json.Unmarshal(data, &component); err != nil {
			return
		}

		encoded, err := json.Marshal(component)
		require.NoError(t, err)

		var decoded Component
		require.NoError(t, json.Unmarshal(encoded, &decoded))

		reencoded, err := json.Marshal(decoded)
		require.NoError(t, err)
		require.JSONEq(t, string(encoded), string(reencoded))
	})
}

func FuzzChildListUnmarshal(f *testing.F) {
	f.Add([]byte(`["a","b","c"]`))
	f.Add([]byte(`[]`))
	f.Add([]byte(`{"componentId":"row","path":"/items"}`))
	f.Add([]byte(`{"componentId":"row"}`))
	f.Add([]byte(`{"path":"/only"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`7`))
	f.Add([]byte(`["a",null]`))
	f.Add([]byte(`{"componentId":12,"path":[]}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var childList ChildList
		if err := json.Unmarshal(data, &childList); err != nil {
			return
		}

		// Decoding never panics and always yields exactly one active shape.
		if childList.Dynamic != nil {
			require.Empty(t, childList.Static)
		}

		_, err := json.Marshal(&childList)
		require.NoError(t, err)
	})
}
