package bootstrap

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func fixedNow() func() time.Time {
	t := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	return func() time.Time { return t }
}

func samplePlan() Plan {
	return Plan{Modules: []Module{{
		ID:    "test",
		Title: "Test",
		Steps: []Step{{
			Name: "Install packages",
			Commands: []Command{
				{Program: "apt", Args: []string{"install", "-y", "curl"}, Sudo: true},
				{Program: "apt", Args: []string{"install", "-y", "git"}, Sudo: true},
			},
		}},
	}}}
}

func TestRunnerRecordsAndResumesState(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")

	state, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	first := &recordingExecutor{}
	runner := Runner{Executor: first, PackageChecker: allowAll(), State: state, Now: fixedNow(), Stdout: io.Discard, Stderr: io.Discard}
	if err := runner.Run(context.Background(), samplePlan()); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}
	if len(first.commands) != 2 {
		t.Fatalf("first run executed %d commands, want 2", len(first.commands))
	}
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("state file was not written: %v", err)
	}

	// A fresh process reloads state and must skip both completed commands.
	resumed, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("reload LoadState() error = %v", err)
	}
	if resumed.Count() != 2 {
		t.Fatalf("reloaded state has %d entries, want 2", resumed.Count())
	}
	second := &recordingExecutor{}
	rerun := Runner{Executor: second, PackageChecker: allowAll(), State: resumed, Now: fixedNow(), Stdout: io.Discard, Stderr: io.Discard}
	if err := rerun.Run(context.Background(), samplePlan()); err != nil {
		t.Fatalf("second Run() error = %v", err)
	}
	if len(second.commands) != 0 {
		t.Fatalf("second run executed %d commands, want 0 (all resumed)", len(second.commands))
	}
}

func TestRunnerResumesFromInterruption(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	state, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}

	// Fail on the second command to simulate an interruption after the first
	// command has already been recorded.
	interrupted := &failingOnceExecutor{}
	runner := Runner{Executor: interrupted, PackageChecker: allowAll(), State: state, Now: fixedNow(), Stdout: io.Discard, Stderr: io.Discard}
	_ = runner.Run(context.Background(), samplePlan())

	reloaded, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("reload error = %v", err)
	}
	// failingOnceExecutor fails its first call, so nothing is recorded and the
	// run aborts before the second command.
	if reloaded.Count() != 0 {
		t.Fatalf("state count = %d, want 0 after immediate failure", reloaded.Count())
	}

	// On rerun the executor no longer fails; both commands should run.
	retry := &failingOnceExecutor{}
	// Prime it so the first call succeeds this time.
	retry.commands = append(retry.commands, Command{})
	rerun := Runner{Executor: retry, PackageChecker: allowAll(), State: reloaded, Now: fixedNow(), Stdout: io.Discard, Stderr: io.Discard}
	if err := rerun.Run(context.Background(), samplePlan()); err != nil {
		t.Fatalf("rerun error = %v", err)
	}
	if reloaded.Count() != 2 {
		t.Fatalf("state count after rerun = %d, want 2", reloaded.Count())
	}
}

func TestRunnerContinueOnErrorSkipsAndRetries(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	state, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}

	plan := Plan{Modules: []Module{{
		ID:    "test",
		Title: "Test",
		Steps: []Step{{
			Name: "Install packages",
			Commands: []Command{
				{Program: "apt", Args: []string{"install", "-y", "broken"}, Sudo: true, ContinueOnError: true},
				{Program: "apt", Args: []string{"install", "-y", "good"}, Sudo: true, ContinueOnError: true},
			},
		}},
	}}}

	executor := &failingOnceExecutor{} // first command fails, rest succeed
	runner := Runner{Executor: executor, PackageChecker: allowAll(), State: state, Now: fixedNow(), Stdout: io.Discard, Stderr: io.Discard}
	err = runner.Run(context.Background(), plan)
	if err == nil {
		t.Fatal("Run() error = nil, want aggregated failure")
	}
	if len(executor.commands) != 2 {
		t.Fatalf("executed %d commands, want 2 (continue past the failure)", len(executor.commands))
	}
	// Only the successful command is recorded; the failed one retries next run.
	if state.Count() != 1 {
		t.Fatalf("state count = %d, want 1 (only the succeeded package)", state.Count())
	}
}

func TestLoadStateDiscardsIncompatibleVersion(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	if err := os.WriteFile(statePath, []byte(`{"version":99,"completed":{"x":{}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	state, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if state.Count() != 0 {
		t.Fatalf("expected incompatible state to be discarded, got %d entries", state.Count())
	}
}

func TestStateErrorsOnCorruptFile(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	if err := os.WriteFile(statePath, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadState(statePath); err == nil {
		t.Fatal("LoadState() error = nil, want parse error")
	}
}
