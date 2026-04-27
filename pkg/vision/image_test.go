package vision

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadImageFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.png")
	if err := os.WriteFile(tmpFile, []byte("fake png data"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid file",
			path:    tmpFile,
			wantErr: false,
		},
		{
			name:    "missing file",
			path:    filepath.Join(tmpDir, "missing.png"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			img, err := LoadImageFromFile(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if img == nil {
				t.Error("expected image, got nil")
				return
			}
			if img.MediaType != "image/png" {
				t.Errorf("expected media type image/png, got %s", img.MediaType)
			}
			if string(img.Data) != "fake png data" {
				t.Error("data mismatch")
			}
		})
	}
}

func TestLoadImageFromFile_MediaTypeDetection(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		ext      string
		wantType string
	}{
		{".png", "image/png"},
		{".jpg", "image/jpeg"},
		{".jpeg", "image/jpeg"},
		{".gif", "image/gif"},
		{".webp", "image/webp"},
		{".unknown", "image/png"}, // fallback
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(tmpDir, "test"+tt.ext)
			if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
				t.Fatal(err)
			}
			img, err := LoadImageFromFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if img.MediaType != tt.wantType {
				t.Errorf("expected %s, got %s", tt.wantType, img.MediaType)
			}
		})
	}
}

func TestLoadImageFromReader(t *testing.T) {
	tests := []struct {
		name      string
		reader    *strings.Reader
		mediaType string
		filename  string
		wantData  string
		wantType  string
		wantName  string
		wantErr   bool
	}{
		{
			name:      "jpeg image",
			reader:    strings.NewReader("test image data"),
			mediaType: "image/jpeg",
			filename:  "photo.jpg",
			wantData:  "test image data",
			wantType:  "image/jpeg",
			wantName:  "photo.jpg",
			wantErr:   false,
		},
		{
			name:      "png image",
			reader:    strings.NewReader("png data"),
			mediaType: "image/png",
			filename:  "screenshot.png",
			wantData:  "png data",
			wantType:  "image/png",
			wantName:  "screenshot.png",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			img, err := LoadImageFromReader(tt.reader, tt.mediaType, tt.filename)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if img == nil {
				t.Fatal("expected image, got nil")
			}
			if string(img.Data) != tt.wantData {
				t.Errorf("expected data %q, got %q", tt.wantData, string(img.Data))
			}
			if img.MediaType != tt.wantType {
				t.Errorf("expected media type %q, got %q", tt.wantType, img.MediaType)
			}
			if img.Filename != tt.wantName {
				t.Errorf("expected filename %q, got %q", tt.wantName, img.Filename)
			}
		})
	}
}
