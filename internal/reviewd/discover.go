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

// screenshotDirBases are directory base names that conventionally hold UI
// screenshots. "visual" additionally requires a "testdata" parent (the
// DiscordSync golden pattern) to avoid false positives.
var screenshotDirBases = map[string]bool{
	"gallery-shots":  true,
	"screenshots":    true,
	"ui-screenshots": true,
}

// walkSkipDirs are never entered while discovering projects.
var walkSkipDirs = map[string]bool{
	".git":         true,
	".cache":       true,
	".venv":        true,
	"node_modules": true,
	"result":       true,
	"target":       true,
	"vendor":       true,
}

// screenshotExtensions are the file extensions counted as screenshots.
var screenshotExtensions = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".webp": true,
}

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

		if walkSkipDirs[entry.Name()] && path != root {
			return filepath.SkipDir
		}

		if !isScreenshotDir(root, path) {
			return nil
		}

		return accumulateDir(found, root, path)
	})
	if err != nil {
		return nil, err
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

	if screenshotDirBases[base] {
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
		return nil, err
	}

	counts := map[string]int{}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		extension := strings.ToLower(filepath.Ext(entry.Name()))
		if screenshotExtensions[extension] {
			counts[extension]++
		}
	}

	return counts, nil
}

// projectOf derives the owning project of a screenshot directory: the first
// path segment under root. When the directory sits directly in root, root
// itself is the project.
func projectOf(root, path string) (name string, projectRoot string, ok bool) {
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == "." {
		return "", "", false
	}

	segments := strings.SplitN(relative, string(filepath.Separator), 2)

	if len(segments) == 1 {
		base := filepath.Base(root)

		return strings.ToLower(base), root, true
	}

	return strings.ToLower(segments[0]), filepath.Join(root, segments[0]), true
}

// sortedSuggestions turns the walker accumulator into a name-sorted slice
// with absolute per-extension globs.
func sortedSuggestions(found map[string]*projectAccumulator) []SuggestedProject {
	suggestions := make([]SuggestedProject, 0, len(found))

	for _, project := range found {
		suggestion := SuggestedProject{
			Name:   project.name,
			Root:   project.root,
			Globs:  make([]string, 0, len(project.dirs)),
			Images: 0,
		}

		dirs := make([]string, 0, len(project.dirs))
		for dir := range project.dirs {
			dirs = append(dirs, dir)
		}

		sort.Strings(dirs)

		for _, dir := range dirs {
			extensions := make([]string, 0, len(project.dirs[dir]))
			for extension := range project.dirs[dir] {
				extensions = append(extensions, extension)
			}

			sort.Strings(extensions)

			for _, extension := range extensions {
				count := project.dirs[dir][extension]
				suggestion.Globs = append(suggestion.Globs, filepath.Join(dir, "*"+extension))
				suggestion.Images += count
			}
		}

		suggestions = append(suggestions, suggestion)
	}

	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Name < suggestions[j].Name
	})

	return suggestions
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
