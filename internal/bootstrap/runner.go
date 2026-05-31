package bootstrap

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type Runner struct {
	LogDir string
	Stdout io.Writer
	Stderr io.Writer
}

func (r Runner) Run(ctx context.Context, plan Plan) error {
	if r.Stdout == nil {
		r.Stdout = os.Stdout
	}
	if r.Stderr == nil {
		r.Stderr = os.Stderr
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
	command = r.filterAPTInstall(ctx, command)
	if command.Program == "" {
		return nil
	}

	program := command.Program
	args := command.Args
	if command.Sudo {
		args = append([]string{program}, args...)
		program = "sudo"
	}

	fmt.Fprintf(r.Stdout, "    $ %s\n", command.String())
	cmd := exec.CommandContext(ctx, program, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = r.Stdout
	cmd.Stderr = r.Stderr

	var logFile *os.File
	if r.LogDir != "" {
		name := fmt.Sprintf("%s-%d.log", safeName(moduleID+"-"+stepName), time.Now().UnixNano())
		file, err := os.Create(filepath.Join(r.LogDir, name))
		if err != nil {
			return err
		}
		defer file.Close()
		logFile = file
		cmd.Stdout = io.MultiWriter(r.Stdout, logFile)
		cmd.Stderr = io.MultiWriter(r.Stderr, logFile)
	}

	err := cmd.Run()
	if logFile != nil {
		_, _ = fmt.Fprintf(logFile, "\nexit_error=%v\n", err)
	}
	return err
}

func (r Runner) filterAPTInstall(ctx context.Context, command Command) Command {
	if command.Program != "apt" || len(command.Args) < 3 || command.Args[0] != "install" {
		return command
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
	for _, pkg := range packages {
		check := exec.CommandContext(ctx, "apt-cache", "show", pkg)
		if err := check.Run(); err != nil {
			fmt.Fprintf(r.Stderr, "    ! skipping unavailable apt package: %s\n", pkg)
			continue
		}
		available = append(available, pkg)
	}
	if len(available) == 0 {
		fmt.Fprintf(r.Stderr, "    ! no available apt packages left for: %s\n", command.String())
		return Command{}
	}

	command.Args = append([]string{"install"}, append(prefix, available...)...)
	return command
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
