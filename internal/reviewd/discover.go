package reviewed

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Screenshot file extensions recognized by discovery and blob storage.
const (
	ExtensionPNG  = ".png"
	ExtensionJPG  = ".jpg"
	ExtensionJPEG = ".jpeg"
	ExtensionWebP = ".webp"
)

// projectPathSegments bounds the path segments considered when deriving the
// owning project: the first segment under the discovery root.
const projectPathSegments = 2

// SuggestedProject is one discovered project with the absolute screenshot
// globs that cover everything the walker found for it.
type SuggestedProject struct {
	Name   string   `json:"name"`
	Root   string   `json:"root"`
	Globs  []string `json:"globs"`
	Images int      `json:"images"`
}

// projectAccumulator collects screenshot directories for one project while
// walking.
type projectAccumulator struct {
	name string
	root string
	dirs map[string]map[string]int
}

// isScreenshotDirBase reports whether a directory base name conventionally
// holds UI screenshots. "visual" additionally requires a "testdata" parent
// (the DiscordSync golden pattern) to avoid false positives.
func isScreenshotDirBase(base string) bool {
	switch base {
	case "gallery-shots", "screenshots", "ui-screenshots":
		return true
	default:
		return false
	}
}

// isWalkSkipDir reports whether a directory is never entered while walking.
func isWalkSkipDir(name string) bool {
	switch name {
	case ".git", ".cache", ".venv", "node_modules", "result", "target", "vendor":
		return true
	default:
		return false
	}
}

// isScreenshotExtension reports whether a file extension is a screenshot.
func isScreenshotExtension(extension string) bool {
	switch extension {
	case ExtensionPNG, ExtensionJPG, ExtensionJPEG, ExtensionWebP:
		return true
	default:
		return false
	}
}

// DiscoverProjects walks root looking for known screenshot directory
// patterns (testdata/visual, gallery-shots, screenshots, ui-screenshots) and
// returns one suggestion per project, named after the directory directly
// under root (lowercased). Suggestions are sorted by name.
func DiscoverProjects(root string) ([]SuggestedProject, error) {
	found := map[string]*projectAccumulator{}

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if !entry.IsDir() {
			return nil
		}

		if isWalkSkipDir(entry.Name()) && path != root {
			return filepath.SkipDir
		}

		if !isScreenshotDir(root, path) {
			return nil
		}

		if err := accumulateDir(found, root, path); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("discover projects under %s: %w", root, err)
	}

	return sortedSuggestions(found), nil
}

// isScreenshotDir reports whether path matches a known screenshot
// convention.
func isScreenshotDir(root, path string) bool {
	if path == root {
		return false
	}

	base := filepath.Base(path)

	if isScreenshotDirBase(base) {
		return true
	}

	return base == "visual" && filepath.Base(filepath.Dir(path)) == "testdata"
}

// accumulateDir counts the screenshots in dir and records them under the
// project that owns dir.
func accumulateDir(found map[string]*projectAccumulator, root, dir string) error {
	images, err := countImagesByExtension(dir)
	if err != nil {
		return err
	}

	name, projectRoot, ok := projectOf(root, dir)
	if !ok {
		return nil
	}

	project, ok := found[name]
	if !ok {
		project = &projectAccumulator{name: name, root: projectRoot, dirs: map[string]map[string]int{}}
		found[name] = project
	}

	if project.dirs[dir] == nil {
		project.dirs[dir] = map[string]int{}
	}

	for extension, count := range images {
		project.dirs[dir][extension] += count
	}

	return nil
}

// countImagesByExtension counts screenshot files in dir per extension.
func countImagesByExtension(dir string) (map[string]int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", dir, err)
	}

	counts := map[string]int{}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		extension := strings.ToLower(filepath.Ext(entry.Name()))
		if isScreenshotExtension(extension) {
			counts[extension]++
		}
	}

	return counts, nil
}

// projectOf derives the owning project of a screenshot directory: the first
// path segment under root. When the directory sits directly in root, root
// itself is the project.
func projectOf(root, path string) (string, string, bool) {
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == "." {
		return "", "", false
	}

	segments := strings.SplitN(relative, string(filepath.Separator), projectPathSegments)

	if len(segments) == 1 {
		return strings.ToLower(filepath.Base(root)), root, true
	}

	return strings.ToLower(segments[0]), filepath.Join(root, segments[0]), true
}

// sortedSuggestions turns the walker accumulator into a name-sorted slice
// with absolute per-extension globs.
func sortedSuggestions(found map[string]*projectAccumulator) []SuggestedProject {
	suggestions := make([]SuggestedProject, 0, len(found))

	for _, project := range found {
		suggestions = append(suggestions, suggestionOf(project))
	}

	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Name < suggestions[j].Name
	})

	return suggestions
}

func suggestionOf(project *projectAccumulator) SuggestedProject {
	suggestion := SuggestedProject{
		Name:   project.name,
		Root:   project.root,
		Globs:  make([]string, 0, len(project.dirs)),
		Images: 0,
	}

	for _, dir := range sortedNestedKeys(project.dirs) {
		for _, extension := range sortedIntMapKeys(project.dirs[dir]) {
			count := project.dirs[dir][extension]
			suggestion.Globs = append(suggestion.Globs, filepath.Join(dir, "*"+extension))
			suggestion.Images += count
		}
	}

	return suggestion
}

// sortedIntMapKeys returns the sorted keys of a map[string]int.
func sortedIntMapKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}

// SuggestedConfigJSON renders discovered projects as the pretty-printed JSON
// "projects" object ready to paste into the daemon config.
func SuggestedConfigJSON(suggestions []SuggestedProject) (string, error) {
	projects := make(map[string][]string, len(suggestions))

	for _, suggestion := range suggestions {
		projects[suggestion.Name] = suggestion.Globs
	}

	encoded, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode suggested projects: %w", err)
	}

	return string(encoded), nil
}

// sortedNestedKeys returns the sorted keys of a map[string]map[string]int.
func sortedNestedKeys(m map[string]map[string]int) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}
