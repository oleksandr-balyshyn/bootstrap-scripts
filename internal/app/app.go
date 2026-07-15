package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/w0rxbend/ubuntu-bootstrap/internal/bootstrap"
	"github.com/w0rxbend/ubuntu-bootstrap/internal/tui"
)

type Streams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func Run(ctx context.Context, args []string, streams Streams) int {
	if streams.Stdin == nil {
		streams.Stdin = strings.NewReader("")
	}
	if streams.Stdout == nil {
		streams.Stdout = io.Discard
	}
	if streams.Stderr == nil {
		streams.Stderr = io.Discard
	}

	flags := flag.NewFlagSet("uboot", flag.ContinueOnError)
	flags.SetOutput(streams.Stderr)
	configDir := flags.String("config", "configs", "directory containing profile.yaml, modules.yaml, and installer asset files")
	dryRun := flags.Bool("dry-run", false, "print commands without executing them")
	all := flags.Bool("all", false, "select all modules without showing the TUI")
	noTUI := flags.Bool("no-tui", false, "run selected modules without the interactive TUI")
	list := flags.Bool("list", false, "list available modules")
	validate := flags.Bool("validate", false, "validate the catalog and all module plans without running commands")
	assumeYes := flags.Bool("yes", false, "skip confirmation before executing selected modules")
	keepGoing := flags.Bool("keep-going", false, "continue running remaining commands after a command fails")
	statePath := flags.String("state", "", "resume state file path (default from profile policy)")
	resetState := flags.Bool("reset-state", false, "discard saved resume state before running")
	skipOSCheck := flags.Bool("skip-os-check", false, "skip the Ubuntu-only host check")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	setFlags := map[string]bool{}
	flags.Visit(func(f *flag.Flag) { setFlags[f.Name] = true })

	logger := slog.New(slog.NewTextHandler(streams.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	catalog, err := bootstrap.LoadCatalog(*configDir)
	if err != nil {
		logger.Error("load catalog", "config", *configDir, "error", err)
		return 1
	}

	profile, _, err := bootstrap.LoadProfile(*configDir)
	if err != nil {
		logger.Error("load profile", "config", *configDir, "error", err)
		return 1
	}
	policy := profile.Policy
	// Policy provides defaults; explicit CLI flags win.
	if !setFlags["dry-run"] && policy.DryRun {
		*dryRun = true
	}
	if !setFlags["keep-going"] && policy.ContinueOnError {
		*keepGoing = true
	}
	if !setFlags["yes"] && !policy.RequireConfirmation {
		*assumeYes = true
	}
	if !setFlags["state"] {
		*statePath = policy.StateFile
	}

	if *list {
		for _, mod := range catalog.Modules {
			fmt.Fprintf(streams.Stdout, "%-22s %s\n", mod.ID, mod.Title)
		}
		return 0
	}

	if *validate {
		if err := bootstrap.ValidateCatalog(catalog); err != nil {
			logger.Error("validate catalog", "error", err)
			return 1
		}
		fmt.Fprintf(streams.Stdout, "Catalog valid: %d module(s)\n", len(catalog.Modules))
		return 0
	}

	selected := map[string]bool{}
	for _, id := range flags.Args() {
		selected[id] = true
	}
	if *all {
		for _, mod := range catalog.Modules {
			selected[mod.ID] = true
		}
	}

	if !*noTUI && !*all && len(selected) == 0 {
		result, err := tui.Run(catalog)
		if err != nil {
			logger.Error("tui failed", "error", err)
			return 1
		}
		if result.Cancelled {
			fmt.Fprintln(streams.Stdout, "Cancelled.")
			return 0
		}
		selected = result.Selected
		*dryRun = result.DryRun
	}

	plan, err := bootstrap.BuildPlan(catalog, selected)
	if err != nil {
		logger.Error("invalid selection", "error", err)
		return 1
	}
	if len(plan.Modules) == 0 {
		fmt.Fprintln(streams.Stdout, "No modules selected.")
		return 0
	}

	if *dryRun {
		plan.Print(streams.Stdout)
		return 0
	}

	if !*skipOSCheck {
		if err := bootstrap.CheckUbuntu(""); err != nil {
			logger.Error("unsupported host", "error", err)
			return 1
		}
	}

	if !*assumeYes && (*noTUI || *all || len(flags.Args()) > 0) {
		plan.Print(streams.Stdout)
		fmt.Fprint(streams.Stdout, "\nRun these commands? [y/N] ")
		var answer string
		_, _ = fmt.Fscan(streams.Stdin, &answer)
		if answer != "y" && answer != "Y" && answer != "yes" && answer != "YES" {
			fmt.Fprintln(streams.Stdout, "Cancelled.")
			return 0
		}
	}

	if *resetState && *statePath != "" {
		if err := os.Remove(*statePath); err != nil && !os.IsNotExist(err) {
			logger.Error("reset state", "path", *statePath, "error", err)
			return 1
		}
	}
	state, err := bootstrap.LoadState(*statePath)
	if err != nil {
		logger.Error("load state", "path", *statePath, "error", err)
		return 1
	}
	if state.Profile == "" {
		state.Profile = profile.Metadata.Name
	}
	if count := state.Count(); count > 0 {
		fmt.Fprintf(streams.Stdout, "Resuming: %d command(s) already completed will be skipped.\n", count)
	}

	runner := bootstrap.Runner{
		LogDir:    filepath.Join(".bootstrap", "logs"),
		Stdout:    streams.Stdout,
		Stderr:    streams.Stderr,
		KeepGoing: *keepGoing,
		State:     state,
	}
	if err := runner.Run(ctx, plan); err != nil {
		logger.Error("bootstrap failed", "error", err)
		return 1
	}
	return 0
}
