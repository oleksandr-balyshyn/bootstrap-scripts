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
	LogDir               string
	Stdout               io.Writer
	Stderr               io.Writer
	Executor             Executor
	AllowMissingPackages bool
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
	if r.LogDir != "" {
		if err := os.MkdirAll(r.LogDir, 0o755); err != nil {
			return err
		}
	}

	for _, mod := range plan.Modules {
		fmt.Fprintf(r.Stdout, "\n==> %s\n", mod.Title)
		for _, step := range mod.Steps {
			fmt.Fprintf(r.Stdout, " -> %s\n", step.Name)
			for _, command := range step.Commands {
				if err := r.runCommand(ctx, mod.ID, step.Name, command); err != nil {
					return fmt.Errorf("%s: %s: %w", mod.ID, step.Name, err)
				}
			}
		}
	}
	fmt.Fprintln(r.Stdout, "\nBootstrap complete.")
	return nil
}

func (r Runner) runCommand(ctx context.Context, moduleID, stepName string, command Command) error {
	command, err := r.filterAPTInstall(ctx, command)
	if err != nil {
		return err
	}
	if command.Program == "" {
		return nil
	}

	fmt.Fprintf(r.Stdout, "    $ %s\n", command.String())
	stdout := r.Stdout
	stderr := r.Stderr

	var logFile *os.File
	if r.LogDir != "" {
		name := fmt.Sprintf("%s-%d.log", safeName(moduleID+"-"+stepName), time.Now().UnixNano())
		file, err := os.Create(filepath.Join(r.LogDir, name))
		if err != nil {
			return err
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
	return err
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
		check := exec.CommandContext(ctx, "apt-cache", "show", pkg)
		if err := check.Run(); err != nil {
			missing = append(missing, pkg)
			continue
		}
		available = append(available, pkg)
	}
	if len(missing) > 0 && !r.AllowMissingPackages {
		return Command{}, fmt.Errorf("unavailable apt packages: %s", strings.Join(missing, ", "))
	}
	for _, pkg := range missing {
		fmt.Fprintf(r.Stderr, "    ! skipping unavailable apt package: %s\n", pkg)
	}
	if len(available) == 0 {
		return Command{}, nil
	}

	command.Args = append([]string{"install"}, append(prefix, available...)...)
	return command, nil
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
