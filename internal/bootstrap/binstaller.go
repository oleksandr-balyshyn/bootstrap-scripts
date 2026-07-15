package bootstrap

import (
	"fmt"
	"path/filepath"
	"strings"
)

// binstallerAsset delegates binary distribution installs to the external
// `binstaller` CLI (https://github.com/worxbend/binstaller). uboot only
// bootstraps the binstaller binary; binstaller then downloads, verifies, and
// installs every tool in the referenced BinaryDistributionProfile and keeps its
// own per-tool state, so re-running resumes binstaller's work natively.
type binstallerAsset struct {
	ID      string   `yaml:"id"`
	Command string   `yaml:"command"` // binstaller binary name/path; defaults to "binstaller"
	Profile string   `yaml:"profile"` // BinaryDistributionProfile path, relative to the config dir
	State   string   `yaml:"state"`   // optional --state path override
	Args    []string `yaml:"args"`    // extra flags, e.g. ["--only", "kubectl"]
}

func (a binstallerAsset) commands(localBase string) ([]Command, error) {
	if strings.TrimSpace(a.Profile) == "" {
		return nil, fmt.Errorf("binstaller asset %q requires a profile", a.ID)
	}

	command := a.Command
	if command == "" {
		command = "binstaller"
	}

	profile := a.Profile
	if localBase != "" && !filepath.IsAbs(profile) {
		profile = filepath.Join(localBase, profile)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "require_command %s\n", shellWord(command))
	fmt.Fprintf(&b, "apply_args=(apply --config %s)\n", shellDouble(profile))
	if strings.TrimSpace(a.State) != "" {
		fmt.Fprintf(&b, "apply_args+=(--state %s)\n", shellDouble(expandHomePath(a.State)))
	}
	for _, arg := range a.Args {
		fmt.Fprintf(&b, "apply_args+=(%s)\n", shellWord(arg))
	}
	fmt.Fprintf(&b, "%s \"${apply_args[@]}\"\n", command)

	return delegatedShellCommands(b.String()), nil
}

func (a binstallerAsset) getID() string { return a.ID }
