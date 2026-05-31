package bootstrap

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func InstallNerdFonts(ctx context.Context, release string, families []string, stdout, stderr io.Writer) error {
	if len(families) == 0 {
		return fmt.Errorf("at least one Nerd Font family is required")
	}
	if release == "" {
		release = "latest"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	root := filepath.Join(home, ".local", "share", "fonts", "NerdFonts")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}

	client := &http.Client{Timeout: 10 * time.Minute}
	for _, family := range families {
		if strings.TrimSpace(family) == "" {
			return fmt.Errorf("empty Nerd Font family")
		}
		if err := installNerdFontFamily(ctx, client, release, family, root, stdout); err != nil {
			return err
		}
	}

	if _, err := exec.LookPath("fc-cache"); err == nil {
		fmt.Fprintln(stdout, "Refreshing font cache...")
		cmd := exec.CommandContext(ctx, "fc-cache", "-f", root)
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

func installNerdFontFamily(ctx context.Context, client *http.Client, release, family, root string, stdout io.Writer) error {
	url := nerdFontReleaseURL(release, family)
	fmt.Fprintf(stdout, "Installing Nerd Font %s from %s\n", family, url)

	temp, err := os.CreateTemp("", "uboot-nerd-font-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	defer temp.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("download %s: %s", url, resp.Status)
	}
	if _, err := io.Copy(temp, resp.Body); err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}

	destination := filepath.Join(root, family)
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	return extractFontZip(temp.Name(), destination)
}

func nerdFontReleaseURL(release, family string) string {
	if release == "latest" {
		return fmt.Sprintf("https://github.com/ryanoasis/nerd-fonts/releases/latest/download/%s.zip", family)
	}
	return fmt.Sprintf("https://github.com/ryanoasis/nerd-fonts/releases/download/%s/%s.zip", release, family)
}

func extractFontZip(path, destination string) error {
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	archive, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer archive.Close()

	for _, file := range archive.File {
		if file.FileInfo().IsDir() || !isFontFile(file.Name) {
			continue
		}
		if err := extractZipFile(file, filepath.Join(destination, filepath.Base(file.Name))); err != nil {
			return err
		}
	}
	return nil
}

func isFontFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".otf", ".ttc", ".ttf":
		return true
	default:
		return false
	}
}

func extractZipFile(file *zip.File, destination string) error {
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()

	out, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, reader)
	return err
}
