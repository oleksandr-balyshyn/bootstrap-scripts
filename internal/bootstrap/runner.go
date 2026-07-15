package bootstrap

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Runner struct {
	LogDir         string
	Stdout         io.Writer
	Stderr         io.Writer
	Executor       Executor
	PackageChecker PackageChecker
	KeepGoing      bool
	// State, when set, records completed commands and is consulted to skip
	// work that already finished on a previous run.
	State *State
	// Now returns the current time; overridable in tests. Defaults to time.Now.
	Now func() time.Time
}

type commandFailure struct {
	ModuleID string
	StepName string
	Command  Command
	Err      error
}

type RunError struct {
	Failures []commandFailure
}

func (e RunError) Error() string {
	if len(e.Failures) == 1 {
		failure := e.Failures[0]
		return fmt.Sprintf("bootstrap failed: %s: %s: %v", failure.ModuleID, failure.StepName, failure.Err)
	}
	return fmt.Sprintf("bootstrap failed: %d command(s) failed", len(e.Failures))
}

// Executor runs one compiled command. It is an interface so command execution
// can be tested without invoking package managers or sudo.
type Executor interface {
	Run(ctx context.Context, command Command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error
}

type OSExecutor struct{}

func (OSExecutor) Run(ctx context.Context, command Command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	program := command.Program
	args := command.Args
	if command.Sudo {
		args = append([]string{program}, args...)
		program = "sudo"
	}

	cmd := exec.CommandContext(ctx, program, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

type PackageChecker interface {
	Installable(ctx context.Context, pkg string) bool
}

type APTPackageChecker struct{}

func (APTPackageChecker) Installable(ctx context.Context, pkg string) bool {
	output, err := exec.CommandContext(ctx, "apt-cache", "policy", pkg).Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Candidate:") {
			candidate := strings.TrimSpace(strings.TrimPrefix(line, "Candidate:"))
			return candidate != "" && candidate != "(none)"
		}
	}
	return false
}

func (r Runner) Run(ctx context.Context, plan Plan) error {
	if r.Stdout == nil {
		r.Stdout = os.Stdout
	}
	if r.Stderr == nil {
		r.Stderr = os.Stderr
	}
	if r.Executor == nil {
		r.Executor = OSExecutor{}
	}
	if r.PackageChecker == nil {
		r.PackageChecker = APTPackageChecker{}
	}
	if r.Now == nil {
		r.Now = time.Now
	}
	if r.LogDir != "" {
		if err := os.MkdirAll(r.LogDir, 0o755); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Join(r.LogDir, "warnings"), 0o755); err != nil {
			return err
		}
	}

	failures := []commandFailure{}
	skipped := 0
	for _, mod := range plan.Modules {
		fmt.Fprintf(r.Stdout, "\n==> %s\n", mod.Title)
		for _, step := range mod.Steps {
			fmt.Fprintf(r.Stdout, " -> %s\n", step.Name)
			for _, command := range step.Commands {
				key := commandKey(mod.ID, step.Name, command)
				if !command.SkipState && r.State.Done(key) {
					fmt.Fprintf(r.Stdout, "    ✓ already applied, skipping: %s\n", command.String())
					skipped++
					continue
				}

				ran, err := r.runCommand(ctx, mod.ID, step.Name, command)
				if err != nil {
					if ctx.Err() != nil {
						return fmt.Errorf("%s: %s: %w", mod.ID, step.Name, err)
					}
					failure := commandFailure{ModuleID: mod.ID, StepName: step.Name, Command: command, Err: err}
					failures = append(failures, failure)
					r.logFailedCommand(failure)
					// Per-command best-effort failures (e.g. one apt package)
					// and --keep-going both continue; a failed command is left
					// unrecorded so it is retried on the next run.
					if command.ContinueOnError || r.KeepGoing {
						continue
					}
					return RunError{Failures: failures}
				}
				if ran && !command.SkipState {
					if err := r.State.MarkDone(key, mod.ID, step.Name, command, r.Now()); err != nil {
						return fmt.Errorf("record state for %s: %s: %w", mod.ID, step.Name, err)
					}
				}
			}
		}
	}
	if skipped > 0 {
		fmt.Fprintf(r.Stdout, "\nSkipped %d command(s) already completed in a previous run.\n", skipped)
	}
	if len(failures) > 0 {
		if r.LogDir != "" {
			fmt.Fprintf(r.Stderr, "\nBootstrap completed with %d failed command(s). See %s for details.\n", len(failures), filepath.Join(r.LogDir, "warnings"))
		} else {
			fmt.Fprintf(r.Stderr, "\nBootstrap completed with %d failed command(s).\n", len(failures))
		}
		return RunError{Failures: failures}
	}
	fmt.Fprintln(r.Stdout, "\nBootstrap complete.")
	return nil
}

// runCommand executes a single command. It reports whether the command was
// actually run: a command filtered down to nothing (e.g. an apt install whose
// only package is unavailable) reports ran=false so it is not recorded as
// completed and is reconsidered on the next run.
func (r Runner) runCommand(ctx context.Context, moduleID, stepName string, command Command) (bool, error) {
	command, err := r.filterAPTInstall(ctx, command)
	if err != nil {
		return false, err
	}
	if command.Program == "" {
		return false, nil
	}

	fmt.Fprintf(r.Stdout, "    $ %s\n", command.String())
	stdout := r.Stdout
	stderr := r.Stderr

	var logFile *os.File
	if r.LogDir != "" {
		name := fmt.Sprintf("%s-%d.log", safeName(moduleID+"-"+stepName), time.Now().UnixNano())
		file, err := os.Create(filepath.Join(r.LogDir, name))
		if err != nil {
			return false, err
		}
		defer file.Close()
		logFile = file
		stdout = io.MultiWriter(r.Stdout, logFile)
		stderr = io.MultiWriter(r.Stderr, logFile)
	}

	err = r.Executor.Run(ctx, command, os.Stdin, stdout, stderr)
	if logFile != nil {
		_, _ = fmt.Fprintf(logFile, "\nexit_error=%v\n", err)
	}
	return true, err
}

func (r Runner) logFailedCommand(failure commandFailure) {
	message := fmt.Sprintf("%s: %s: command failed: %v", failure.ModuleID, failure.StepName, failure.Err)
	fmt.Fprintf(r.Stderr, "    ! %s\n", message)
	if r.LogDir == "" {
		return
	}

	name := fmt.Sprintf("command-failed-%s-%d.log", safeName(failure.ModuleID+"-"+failure.StepName), time.Now().UnixNano())
	path := filepath.Join(r.LogDir, "warnings", name)
	body := fmt.Sprintf(
		"warning=%s\nmodule=%s\nstep=%s\ncommand=%s\nerror=%v\n",
		message,
		failure.ModuleID,
		failure.StepName,
		failure.Command.String(),
		failure.Err,
	)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		fmt.Fprintf(r.Stderr, "    ! failed to write warning log: %v\n", err)
	}
}

func (r Runner) filterAPTInstall(ctx context.Context, command Command) (Command, error) {
	if command.Program != "apt" || len(command.Args) < 3 || command.Args[0] != "install" {
		return command, nil
	}

	prefix := []string{}
	packages := []string{}
	for _, arg := range command.Args[1:] {
		if len(arg) > 0 && arg[0] == '-' {
			prefix = append(prefix, arg)
			continue
		}
		packages = append(packages, arg)
	}

	available := make([]string, 0, len(packages))
	missing := make([]string, 0)
	for _, pkg := range packages {
		if !r.PackageChecker.Installable(ctx, pkg) {
			missing = append(missing, pkg)
			continue
		}
		available = append(available, pkg)
	}
	if len(missing) > 0 {
		r.logSkippedPackages(command, missing)
	}
	if len(available) == 0 {
		return Command{}, nil
	}

	command.Args = append([]string{"install"}, append(prefix, available...)...)
	return command, nil
}

func (r Runner) logSkippedPackages(command Command, packages []string) {
	message := fmt.Sprintf("skipping unavailable apt packages: %s", strings.Join(packages, ", "))
	fmt.Fprintf(r.Stderr, "    ! %s\n", message)
	if r.LogDir == "" {
		return
	}

	name := fmt.Sprintf("apt-missing-%d.log", time.Now().UnixNano())
	path := filepath.Join(r.LogDir, "warnings", name)
	body := fmt.Sprintf("warning=%s\ncommand=%s\npackages=%s\n", message, command.String(), strings.Join(packages, "\n"))
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		fmt.Fprintf(r.Stderr, "    ! failed to write warning log: %v\n", err)
	}
}

func safeName(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			out = append(out, r)
			continue
		}
		out = append(out, '-')
	}
	return string(out)
}
