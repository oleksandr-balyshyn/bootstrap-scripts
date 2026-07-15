package bootstrap

import (
	"fmt"
	"path/filepath"
	"strings"
)

// nerdFontsAsset delegates font installation to the external
// nerd-fonts-installer CLI (https://github.com/worxbend/nerd-fonts-installer),
// which reads its own YAML config (release, destination, refresh_font_cache,
// families) and downloads/extracts the archives.
//
// The installer binary is itself provided by the binstaller profile, so it is
// referenced by its managed path rather than assumed to be on PATH.
type nerdFontsAsset struct {
	ID      string   `yaml:"id"`
	Command string   `yaml:"command"` // installer binary path; defaults to "nerdfont-install"
	Config  string   `yaml:"config"`  // nerd-fonts-installer config path, relative to the config dir
	Args    []string `yaml:"args"`    // extra flags, e.g. ["--dry-run"]
}

func (a nerdFontsAsset) commands(localBase string) ([]Command, error) {
	if strings.TrimSpace(a.Config) == "" {
		return nil, fmt.Errorf("nerd-fonts asset %q requires a config", a.ID)
	}

	command := a.Command
	if command == "" {
		command = "nerdfont-install"
	}

	config := a.Config
	if localBase != "" && !filepath.IsAbs(config) {
		config = filepath.Join(localBase, config)
	}

	var b strings.Builder
	// shellDouble keeps ${HOME}-style references expandable so a managed
	// install path resolves at runtime.
	fmt.Fprintf(&b, "installer=%s\n", shellDouble(command))
	b.WriteString("require_command \"$installer\"\n")
	fmt.Fprintf(&b, "font_args=(--config %s)\n", shellDouble(config))
	for _, arg := range a.Args {
		fmt.Fprintf(&b, "font_args+=(%s)\n", shellWord(arg))
	}
	b.WriteString("\"$installer\" \"${font_args[@]}\"\n")

	return delegatedShellCommands(b.String()), nil
}

func (a nerdFontsAsset) getID() string { return a.ID }
