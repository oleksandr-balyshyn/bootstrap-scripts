package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type recordingExecutor struct {
	commands []Command
}

func (e *recordingExecutor) Run(_ context.Context, command Command, _ io.Reader, _ io.Writer, _ io.Writer) error {
	e.commands = append(e.commands, command)
	return nil
}

type failingOnceExecutor struct {
	commands []Command
}

func (e *failingOnceExecutor) Run(_ context.Context, command Command, _ io.Reader, _ io.Writer, _ io.Writer) error {
	e.commands = append(e.commands, command)
	if len(e.commands) == 1 {
		return errors.New("boom")
	}
	return nil
}

type packageCheckerFunc func(context.Context, string) bool

func (f packageCheckerFunc) Installable(ctx context.Context, pkg string) bool {
	return f(ctx, pkg)
}

func TestRunnerUsesExecutor(t *testing.T) {
	executor := &recordingExecutor{}
	plan := Plan{Modules: []Module{{
		ID:    "test",
		Title: "Test",
		Steps: []Step{{
			Name:     "Run command",
			Commands: []Command{{Program: "echo", Args: []string{"ok"}}},
		}},
	}}}

	runner := Runner{Executor: executor, Stdout: io.Discard, Stderr: io.Discard}
	if err := runner.Run(context.Background(), plan); err != nil {
		t.Fatalf("Runner.Run() error = %v", err)
	}
	if len(executor.commands) != 1 {
		t.Fatalf("executor saw %d commands, want 1", len(executor.commands))
	}
	if executor.commands[0].Program != "echo" {
		t.Fatalf("executor command = %#v", executor.commands[0])
	}
}

func TestRunnerStopsAfterCommandFailureByDefault(t *testing.T) {
	var stderr bytes.Buffer
	logDir := t.TempDir()
	executor := &failingOnceExecutor{}
	plan := Plan{Modules: []Module{{
		ID:    "test",
		Title: "Test",
		Steps: []Step{
			{Name: "Failing command", Commands: []Command{{Program: "false"}}},
			{Name: "Next command", Commands: []Command{{Program: "echo", Args: []string{"ok"}}}},
		},
	}}}

	runner := Runner{Executor: executor, LogDir: logDir, Stdout: io.Discard, Stderr: &stderr}
	if err := runner.Run(context.Background(), plan); err == nil {
		t.Fatal("Runner.Run() error = nil, want command failure")
	}
	if len(executor.commands) != 1 {
		t.Fatalf("executor saw %d commands, want 1", len(executor.commands))
	}
	if !strings.Contains(stderr.String(), "command failed") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	entries, err := os.ReadDir(filepath.Join(logDir, "warnings"))
	if err != nil {
		t.Fatal(err)
	}
	foundFailureLog := false
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "command-failed-") {
			foundFailureLog = true
		}
	}
	if !foundFailureLog {
		t.Fatalf("warning logs = %#v, want command failure log", entries)
	}
}

func TestRunnerKeepGoingContinuesAfterCommandFailure(t *testing.T) {
	var stderr bytes.Buffer
	executor := &failingOnceExecutor{}
	plan := Plan{Modules: []Module{{
		ID:    "test",
		Title: "Test",
		Steps: []Step{
			{Name: "Failing command", Commands: []Command{{Program: "false"}}},
			{Name: "Next command", Commands: []Command{{Program: "echo", Args: []string{"ok"}}}},
		},
	}}}

	runner := Runner{Executor: executor, KeepGoing: true, Stdout: io.Discard, Stderr: &stderr}
	if err := runner.Run(context.Background(), plan); err == nil {
		t.Fatal("Runner.Run() error = nil, want command failure")
	}
	if len(executor.commands) != 2 {
		t.Fatalf("executor saw %d commands, want 2", len(executor.commands))
	}
	if !strings.Contains(stderr.String(), "Bootstrap completed with 1 failed command") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunnerSkipsMissingAptPackages(t *testing.T) {
	checker := packageCheckerFunc(func(_ context.Context, pkg string) bool {
		return pkg != "uboot-package-that-should-not-exist"
	})

	var stderr bytes.Buffer
	runner := Runner{Executor: &recordingExecutor{}, PackageChecker: checker, Stdout: io.Discard, Stderr: &stderr}
	cmd, err := runner.filterAPTInstall(context.Background(), Command{
		Program: "apt",
		Args:    []string{"install", "-y", "uboot-package-that-should-not-exist"},
		Sudo:    true,
	})
	if err != nil {
		t.Fatalf("filterAPTInstall() error = %v", err)
	}
	if cmd.Program != "" {
		t.Fatalf("filterAPTInstall() command = %#v, want empty skipped command", cmd)
	}
	if !strings.Contains(stderr.String(), "skipping unavailable apt packages") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunnerKeepsAptPackagesWithInstallCandidate(t *testing.T) {
	checker := packageCheckerFunc(func(_ context.Context, pkg string) bool {
		return pkg == "qemu-system-x86"
	})

	runner := Runner{Executor: &recordingExecutor{}, PackageChecker: checker, Stdout: io.Discard, Stderr: io.Discard}
	cmd, err := runner.filterAPTInstall(context.Background(), Command{
		Program: "apt",
		Args:    []string{"install", "-y", "qemu-system-x86", "qemu-kvm"},
		Sudo:    true,
	})
	if err != nil {
		t.Fatalf("filterAPTInstall() error = %v", err)
	}
	if got := cmd.String(); got != "sudo apt install -y qemu-system-x86" {
		t.Fatalf("filterAPTInstall() command = %q", got)
	}
}

func TestRunnerLogsMissingAptPackages(t *testing.T) {
	checker := packageCheckerFunc(func(_ context.Context, pkg string) bool {
		return pkg != "uboot-package-that-should-not-exist"
	})

	var stderr bytes.Buffer
	logDir := t.TempDir()
	runner := Runner{
		Executor:       &recordingExecutor{},
		PackageChecker: checker,
		LogDir:         logDir,
		Stdout:         io.Discard,
		Stderr:         &stderr,
	}
	if err := os.MkdirAll(filepath.Join(logDir, "warnings"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd, err := runner.filterAPTInstall(context.Background(), Command{
		Program: "apt",
		Args:    []string{"install", "-y", "uboot-package-that-should-not-exist"},
		Sudo:    true,
	})
	if err != nil {
		t.Fatalf("filterAPTInstall() error = %v", err)
	}
	if cmd.Program != "" {
		t.Fatalf("filterAPTInstall() command = %#v, want empty skipped command", cmd)
	}
	if !strings.Contains(stderr.String(), "skipping unavailable apt package") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	entries, err := os.ReadDir(filepath.Join(logDir, "warnings"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("warning log count = %d, want 1", len(entries))
	}
}
