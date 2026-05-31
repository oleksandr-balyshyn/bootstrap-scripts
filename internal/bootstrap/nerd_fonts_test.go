package bootstrap

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestNerdFontReleaseURL(t *testing.T) {
	tests := []struct {
		name    string
		release string
		family  string
		want    string
	}{
		{
			name:    "latest",
			release: "latest",
			family:  "JetBrainsMono",
			want:    "https://github.com/ryanoasis/nerd-fonts/releases/latest/download/JetBrainsMono.zip",
		},
		{
			name:    "tag",
			release: "v3.4.0",
			family:  "Hack",
			want:    "https://github.com/ryanoasis/nerd-fonts/releases/download/v3.4.0/Hack.zip",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nerdFontReleaseURL(tt.release, tt.family); got != tt.want {
				t.Fatalf("nerdFontReleaseURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractFontZipOnlyExtractsFonts(t *testing.T) {
	temp := t.TempDir()
	archivePath := filepath.Join(temp, "font.zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for name, body := range map[string]string{
		"Font.ttf":        "font",
		"nested/Font.otf": "font",
		"README.md":       "docs",
	} {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	destination := filepath.Join(temp, "out")
	if err := extractFontZip(archivePath, destination); err != nil {
		t.Fatalf("extractFontZip() error = %v", err)
	}
	for _, name := range []string{"Font.ttf", "Font.otf"} {
		if _, err := os.Stat(filepath.Join(destination, name)); err != nil {
			t.Fatalf("expected extracted font %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(destination, "README.md")); !os.IsNotExist(err) {
		t.Fatalf("README.md should not be extracted, stat err = %v", err)
	}
}
