package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeResumeConfig writes a minimal, side-effect-free config (shell echoes)
// plus a profile that skips confirmation, and returns the config dir.
func writeResumeConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "installers"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"profile.yaml": `kind: SystemBootstrapProfile
metadata:
  name: resume-test
spec:
  policy:
    requireConfirmation: false
    distro: ubuntu
`,
		"modules.yaml": `modules:
  - id: hello
    title: Hello
    default: true
    steps:
      - name: Echo one
        installer: shell
        asset: one
      - name: Echo two
        installer: shell
        asset: two
`,
		"installers/shell.yaml": `assets:
  - id: one
    script: echo one
  - id: two
    script: echo two
`,
	}
	for _, name := range []string{"apt", "snap", "flatpak", "cargo", "sdkman", "nerd-fonts", "binary", "binstaller", "dotfiles"} {
		files["installers/"+name+".yaml"] = "assets: []\n"
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestRunResumesCompletedCommands(t *testing.T) {
	configDir := writeResumeConfig(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	args := []string{
		"--config", configDir,
		"--no-tui", "--yes", "--skip-os-check",
		"--state", statePath,
		"hello",
	}

	var out1 bytes.Buffer
	if code := Run(context.Background(), args, Streams{Stdout: &out1, Stderr: &out1}); code != 0 {
		t.Fatalf("first Run() code = %d, output:\n%s", code, out1.String())
	}
	if !strings.Contains(out1.String(), "Bootstrap complete.") {
		t.Fatalf("first run did not complete:\n%s", out1.String())
	}
	if strings.Contains(out1.String(), "already applied") {
		t.Fatalf("first run should not skip anything:\n%s", out1.String())
	}
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("state file not written: %v", err)
	}

	var out2 bytes.Buffer
	if code := Run(context.Background(), args, Streams{Stdout: &out2, Stderr: &out2}); code != 0 {
		t.Fatalf("second Run() code = %d, output:\n%s", code, out2.String())
	}
	if !strings.Contains(out2.String(), "Resuming: 2 command(s)") {
		t.Fatalf("second run did not report resume:\n%s", out2.String())
	}
	if strings.Count(out2.String(), "already applied, skipping") != 2 {
		t.Fatalf("second run should skip both commands:\n%s", out2.String())
	}
}

func TestRunRejectsNonUbuntuHost(t *testing.T) {
	configDir := writeResumeConfig(t)
	osRelease := filepath.Join(t.TempDir(), "os-release")
	if err := os.WriteFile(osRelease, []byte("ID=fedora\nPRETTY_NAME=\"Fedora\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("UBOOT_OS_RELEASE", osRelease)

	var out bytes.Buffer
	code := Run(context.Background(), []string{
		"--config", configDir, "--no-tui", "--yes",
		"--state", filepath.Join(t.TempDir(), "state.json"),
		"hello",
	}, Streams{Stdout: &out, Stderr: &out})
	if code != 1 {
		t.Fatalf("Run() code = %d, want 1 on non-Ubuntu host:\n%s", code, out.String())
	}
}
