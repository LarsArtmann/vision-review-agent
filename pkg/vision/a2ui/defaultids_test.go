package a2ui

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDefaultIDs pins the surface/catalog ID fallback rule at its unit:
// each ID falls back independently, so a model may omit one or both.
func TestDefaultIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		surfaceID    string
		catalogID    string
		wantSurface  string
		wantCatalog  string
	}{
		{
			name:        "both empty",
			surfaceID:   "",
			catalogID:   "",
			wantSurface: defaultSurfaceID,
			wantCatalog: DefaultCatalogID,
		},
		{
			name:        "surface empty only",
			surfaceID:   "",
			catalogID:   "custom-catalog",
			wantSurface: defaultSurfaceID,
			wantCatalog: "custom-catalog",
		},
		{
			name:        "catalog empty only",
			surfaceID:   "editor",
			catalogID:   "",
			wantSurface: "editor",
			wantCatalog: DefaultCatalogID,
		},
		{
			name:        "neither empty",
			surfaceID:   "editor",
			catalogID:   "custom-catalog",
			wantSurface: "editor",
			wantCatalog: "custom-catalog",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			surfaceID, catalogID := defaultIDs(tt.surfaceID, tt.catalogID)
			require.Equal(t, tt.wantSurface, surfaceID)
			require.Equal(t, tt.wantCatalog, catalogID)
		})
	}
}
