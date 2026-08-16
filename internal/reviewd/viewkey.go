package reviewed

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// Default theme and viewport used when a screenshot filename does not follow
// the {Page}--{theme}--{viewport} convention. Deterministic fallback keeps
// dedupe stable across passes.
const (
	FallbackTheme    = "default"
	FallbackViewport = "desktop"
)

// ErrEmptyViewKey is returned when a filename carries no usable stem.
var ErrEmptyViewKey = errors.New("view key: empty filename")

// ViewKey identifies one visual variant of a page: the page name plus theme
// and viewport it was rendered under.
type ViewKey struct {
	Page     string
	Theme    string
	Viewport string
}

// ParseViewKey extracts a ViewKey from a screenshot filename following the
// {Page}--{theme}--{viewport} convention. When the name does not conform, the
// whole stem becomes the page and deterministic fallbacks fill theme and
// viewport. The last two "--" separated segments are always theme and
// viewport, so pages may themselves contain "--".
func ParseViewKey(name string) (ViewKey, error) {
	base := filepath.Base(name)

	stem := strings.TrimSuffix(base, filepath.Ext(base))
	if stem == "" || stem == "." || stem == string(filepath.Separator) {
		return ViewKey{}, ErrEmptyViewKey
	}

	parts := strings.Split(stem, "--")

	var page, theme, viewport string

	switch len(parts) {
	case 1:
		page, theme, viewport = parts[0], FallbackTheme, FallbackViewport
	case 2:
		page, theme, viewport = parts[0], parts[1], FallbackViewport
	default:
		viewport = parts[len(parts)-1]
		theme = parts[len(parts)-2]
		page = strings.Join(parts[:len(parts)-2], "--")
	}

	return ViewKey{Page: page, Theme: theme, Viewport: viewport}, nil
}

// String renders the canonical {Page}--{theme}--{viewport} form used for
// stream IDs and markdown filenames.
func (v ViewKey) String() string {
	return v.Page + "--" + v.Theme + "--" + v.Viewport
}

// ErrEmptyProject is returned when a view stream ID is requested for an
// unnamed project; the stream ID would otherwise silently start with ":".
var ErrEmptyProject = errors.New("view stream: empty project")

// ViewStreamID returns the event stream ID for a project's view. Stream IDs
// are <project>:<viewKey>.
func ViewStreamID(project string, viewKey ViewKey) (id.StreamID, error) {
	if project == "" {
		return id.StreamID{}, ErrEmptyProject
	}

	return id.ParseStreamID(project + ":" + viewKey.String())
}
