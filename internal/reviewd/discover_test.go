package reviewed

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createShots(t *testing.T, dir string, names ...string) {
	t.Helper()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}

	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("png"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}

func TestDiscoverProjectsDiscordSyncPattern(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	visual := filepath.Join(root, "DiscordSync", "internal", "web", "testdata", "visual")
	createShots(t, visual,
		"Settings--dark--desktop.png",
		"Settings--light--desktop.png",
		"Login--dark--mobile.png",
		"notes.txt",
	)

	suggestions, err := DiscoverProjects(root)
	if err != nil {
		t.Fatalf("DiscoverProjects: %v", err)
	}

	if len(suggestions) != 1 {
		t.Fatalf("suggestions = %+v, want exactly 1", suggestions)
	}

	got := suggestions[0]
	if got.Name != "discordsync" {
		t.Fatalf("name = %q, want discordsync", got.Name)
	}

	if got.Images != 3 {
		t.Fatalf("images = %d, want 3 (txt ignored)", got.Images)
	}

	wantGlob := filepath.Join(visual, "*.png")
	if len(got.Globs) != 1 || got.Globs[0] != wantGlob {
		t.Fatalf("globs = %v, want [%s]", got.Globs, wantGlob)
	}
}

func TestDiscoverProjectsMultipleDirsAndExtensions(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	createShots(t, filepath.Join(root, "Alpha", "gallery-shots"), "a.png", "b.png")
	createShots(t, filepath.Join(root, "Alpha", "docs", "gallery-shots"), "c.jpg")
	createShots(t, filepath.Join(root, "beta", "ui-screenshots"), "d.webp")

	suggestions, err := DiscoverProjects(root)
	if err != nil {
		t.Fatalf("DiscoverProjects: %v", err)
	}

	if len(suggestions) != 2 {
		t.Fatalf("suggestions = %+v, want 2 projects", suggestions)
	}

	if suggestions[0].Name != "alpha" || suggestions[1].Name != "beta" {
		t.Fatalf("order = [%s, %s], want [alpha, beta]", suggestions[0].Name, suggestions[1].Name)
	}

	alpha := suggestions[0]
	if alpha.Images != 3 {
		t.Fatalf("alpha images = %d, want 3", alpha.Images)
	}

	joined := strings.Join(alpha.Globs, " ")
	if !strings.Contains(joined, "*.png") || !strings.Contains(joined, "*.jpg") {
		t.Fatalf("alpha globs should cover png and jpg: %v", alpha.Globs)
	}
}

func TestDiscoverProjectsSkipsNoiseDirs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	createShots(t, filepath.Join(root, "Proj", "node_modules", "gallery-shots"), "a.png")
	createShots(t, filepath.Join(root, "Proj", ".git", "screenshots"), "b.png")

	suggestions, err := DiscoverProjects(root)
	if err != nil {
		t.Fatalf("DiscoverProjects: %v", err)
	}

	if len(suggestions) != 0 {
		t.Fatalf("suggestions = %+v, want none (noise dirs skipped)", suggestions)
	}
}

func TestDiscoverProjectsVisualRequiresTestdataParent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	createShots(t, filepath.Join(root, "Proj", "src", "visual"), "a.png")

	suggestions, err := DiscoverProjects(root)
	if err != nil {
		t.Fatalf("DiscoverProjects: %v", err)
	}

	if len(suggestions) != 0 {
		t.Fatalf("suggestions = %+v, want none (visual without testdata parent)", suggestions)
	}
}

func TestDiscoverProjectsEmptyRoot(t *testing.T) {
	t.Parallel()

	suggestions, err := DiscoverProjects(t.TempDir())
	if err != nil {
		t.Fatalf("DiscoverProjects: %v", err)
	}

	if len(suggestions) != 0 {
		t.Fatalf("suggestions = %+v, want none", suggestions)
	}
}

func TestSuggestedConfigJSON(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	visual := filepath.Join(root, "DiscordSync", "testdata", "visual")
	createShots(t, visual, "a.png", "b.png")

	suggestions, err := DiscoverProjects(root)
	if err != nil {
		t.Fatalf("DiscoverProjects: %v", err)
	}

	rendered, err := SuggestedConfigJSON(suggestions)
	if err != nil {
		t.Fatalf("SuggestedConfigJSON: %v", err)
	}

	wantGlob := filepath.Join(visual, "*.png")
	if !strings.Contains(rendered, `"discordsync"`) || !strings.Contains(rendered, wantGlob) {
		t.Fatalf("rendered config should contain project and absolute glob:\n%s", rendered)
	}
}
