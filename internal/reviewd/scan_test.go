package reviewed

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func scanTestStamp() time.Time {
	return time.Date(2026, time.August, 16, 20, 0, 0, 0, time.UTC)
}

func TestScanProjectFindsAndHashes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	home := writeScanPNG(t, dir, "Home")
	settings := writeScanPNG(t, dir, "Settings--dark--desktop")

	captures, err := ScanProject([]string{filepath.Join(dir, "*.png")})
	if err != nil {
		t.Fatalf("ScanProject: %v", err)
	}

	if len(captures) != 2 {
		t.Fatalf("captures = %d, want 2", len(captures))
	}

	if captures[0].ViewKey.String() != "Home--default--desktop" {
		t.Fatalf("first key = %q, want Home--default--desktop", captures[0].ViewKey.String())
	}

	if captures[1].ViewKey.String() != "Settings--dark--desktop" {
		t.Fatalf("second key = %q, want Settings--dark--desktop", captures[1].ViewKey.String())
	}

	for _, capture := range captures {
		sha, err := SHA256File(capture.Path)
		if err != nil {
			t.Fatalf("hash %s: %v", capture.Path, err)
		}

		if capture.SHA256 != sha {
			t.Fatalf("capture sha %q != file sha %q", capture.SHA256, sha)
		}

		if capture.Path != home && capture.Path != settings {
			t.Fatalf("unexpected path %q", capture.Path)
		}
	}
}

func TestScanProjectDedupesToNewestFile(t *testing.T) {
	t.Parallel()

	old := t.TempDir()
	newDir := t.TempDir()

	oldPath := writeScanPNG(t, old, "Login--dark--desktop")
	newPath := writeScanPNG(t, newDir, "Login--dark--desktop")

	stampOld := scanTestStamp()
	stampNew := scanTestStamp().Add(time.Hour)

	if err := os.Chtimes(oldPath, stampOld, stampOld); err != nil {
		t.Fatalf("chtimes old: %v", err)
	}

	if err := os.Chtimes(newPath, stampNew, stampNew); err != nil {
		t.Fatalf("chtimes new: %v", err)
	}

	captures, err := ScanProject([]string{
		filepath.Join(old, "*.png"),
		filepath.Join(newDir, "*.png"),
	})
	if err != nil {
		t.Fatalf("ScanProject: %v", err)
	}

	if len(captures) != 1 {
		t.Fatalf("captures = %d, want 1", len(captures))
	}

	if captures[0].Path != newPath {
		t.Fatalf("kept %q, want newest %q", captures[0].Path, newPath)
	}

	if !captures[0].ModifiedAt.Equal(stampNew) {
		t.Fatalf("modifiedAt = %v, want %v", captures[0].ModifiedAt, stampNew)
	}
}

func TestScanProjectIgnoresDirectoriesAndOtherFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeScanPNG(t, dir, "Home")

	if err := os.Mkdir(filepath.Join(dir, "nested--x--desktop"), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("not a shot"), 0o600); err != nil {
		t.Fatalf("write txt: %v", err)
	}

	captures, err := ScanProject([]string{filepath.Join(dir, "*")})
	if err != nil {
		t.Fatalf("ScanProject: %v", err)
	}

	if len(captures) != 1 {
		t.Fatalf("captures = %d, want 1 (Home only)", len(captures))
	}

	if captures[0].ViewKey.String() != "Home--default--desktop" {
		t.Fatalf("key = %q, want Home--default--desktop", captures[0].ViewKey.String())
	}
}

func TestScanProjectBadGlob(t *testing.T) {
	t.Parallel()

	_, err := ScanProject([]string{"["})
	if err == nil {
		t.Fatal("want error for bad glob pattern")
	}
}

func writeScanPNG(t *testing.T, dir string, name string) string {
	t.Helper()

	path := filepath.Join(dir, name+".png")
	if err := os.WriteFile(path, scanTestPNG, 0o644); err != nil {
		t.Fatalf("write png %s: %v", path, err)
	}

	return path
}

var scanTestPNG = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4, 0x89, 0x00, 0x00, 0x00,
	0x0D, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9C, 0x62, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00, 0x00, 0x00, 0x00, 0x49,
	0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
}
