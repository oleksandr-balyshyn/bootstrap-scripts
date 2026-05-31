package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/w0rxbend/ubuntu-bootstrap/internal/bootstrap"
	"github.com/w0rxbend/ubuntu-bootstrap/internal/tui"
)

func main() {
	var (
		configDir            = flag.String("config", "configs", "directory containing modules.yaml and installer asset files")
		dryRun               = flag.Bool("dry-run", false, "print commands without executing them")
		all                  = flag.Bool("all", false, "select all modules without showing the TUI")
		noTUI                = flag.Bool("no-tui", false, "run selected modules without the interactive TUI")
		list                 = flag.Bool("list", false, "list available modules")
		assumeYes            = flag.Bool("yes", false, "skip confirmation before executing selected modules")
		allowMissingPackages = flag.Bool("allow-missing-packages", false, "skip apt packages unavailable in current repositories")
	)
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	catalog, err := bootstrap.LoadCatalog(*configDir)
	if err != nil {
		logger.Error("load catalog", "config", *configDir, "error", err)
		os.Exit(1)
	}

	if *list {
		for _, mod := range catalog.Modules {
			fmt.Printf("%-22s %s\n", mod.ID, mod.Title)
		}
		return
	}

	selected := map[string]bool{}
	for _, id := range flag.Args() {
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
			os.Exit(1)
		}
		if result.Cancelled {
			fmt.Println("Cancelled.")
			return
		}
		selected = result.Selected
		*dryRun = result.DryRun
	}

	plan, err := bootstrap.BuildPlan(catalog, selected)
	if err != nil {
		logger.Error("invalid selection", "error", err)
		os.Exit(1)
	}
	if len(plan.Modules) == 0 {
		fmt.Println("No modules selected.")
		return
	}

	if *dryRun {
		plan.Print(os.Stdout)
		return
	}
	if !*assumeYes && (*noTUI || *all || len(flag.Args()) > 0) {
		plan.Print(os.Stdout)
		fmt.Fprint(os.Stdout, "\nRun these commands? [y/N] ")
		var answer string
		_, _ = fmt.Fscan(os.Stdin, &answer)
		if answer != "y" && answer != "Y" && answer != "yes" && answer != "YES" {
			fmt.Println("Cancelled.")
			return
		}
	}

	logDir := filepath.Join(".bootstrap", "logs")
	runner := bootstrap.Runner{
		LogDir:               logDir,
		Stdout:               os.Stdout,
		Stderr:               os.Stderr,
		AllowMissingPackages: *allowMissingPackages,
	}
	if err := runner.Run(context.Background(), plan); err != nil {
		logger.Error("bootstrap failed", "error", err)
		os.Exit(1)
	}
}
