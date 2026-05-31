package bootstrap

import (
	"bytes"
	"context"
	"io"
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

func TestRunnerRejectsMissingAptPackagesByDefault(t *testing.T) {
	runner := Runner{Executor: &recordingExecutor{}, Stdout: io.Discard, Stderr: io.Discard}
	_, err := runner.filterAPTInstall(context.Background(), Command{
		Program: "apt",
		Args:    []string{"install", "-y", "uboot-package-that-should-not-exist"},
		Sudo:    true,
	})
	if err == nil {
		t.Fatal("filterAPTInstall() error = nil, want missing package error")
	}
	if !strings.Contains(err.Error(), "unavailable apt packages") {
		t.Fatalf("filterAPTInstall() error = %q", err)
	}
}

func TestRunnerAllowsMissingAptPackagesWhenConfigured(t *testing.T) {
	var stderr bytes.Buffer
	runner := Runner{
		Executor:             &recordingExecutor{},
		Stdout:               io.Discard,
		Stderr:               &stderr,
		AllowMissingPackages: true,
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
}
