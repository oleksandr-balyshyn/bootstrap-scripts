package bootstrap

import (
	"fmt"
	"os"
	"strings"
)

type aptAsset struct {
	ID       string   `yaml:"id"`
	Action   string   `yaml:"action"`
	Packages []string `yaml:"packages"`
}

func (a aptAsset) commands() ([]Command, error) {
	switch a.Action {
	case "update":
		return []Command{{Program: "apt", Args: []string{"update"}, Sudo: true}}, nil
	case "upgrade":
		return []Command{{Program: "apt", Args: []string{"upgrade", "-y"}, Sudo: true}}, nil
	case "":
		if len(a.Packages) == 0 {
			return nil, fmt.Errorf("apt asset %q has no packages or action", a.ID)
		}
		args := append([]string{"install", "-y"}, a.Packages...)
		return []Command{{Program: "apt", Args: args, Sudo: true}}, nil
	default:
		return nil, fmt.Errorf("unsupported apt action %q", a.Action)
	}
}

func (a aptAsset) getID() string { return a.ID }

type snapAsset struct {
	ID       string        `yaml:"id"`
	Packages []snapPackage `yaml:"packages"`
}

type snapPackage struct {
	Name    string `yaml:"name"`
	Classic bool   `yaml:"classic"`
	Channel string `yaml:"channel"`
}

func (a snapAsset) commands() ([]Command, error) {
	if len(a.Packages) == 0 {
		return nil, fmt.Errorf("snap asset %q has no packages", a.ID)
	}
	commands := make([]Command, 0, len(a.Packages))
	for _, pkg := range a.Packages {
		if pkg.Name == "" {
			return nil, fmt.Errorf("snap asset %q contains package without name", a.ID)
		}
		args := []string{"install", pkg.Name}
		if pkg.Classic {
			args = append(args, "--classic")
		}
		if pkg.Channel != "" {
			args = append(args, "--channel", pkg.Channel)
		}
		commands = append(commands, Command{Program: "snap", Args: args, Sudo: true})
	}
	return commands, nil
}

func (a snapAsset) getID() string { return a.ID }

type flatpakAsset struct {
	ID     string   `yaml:"id"`
	Remote string   `yaml:"remote"`
	Refs   []string `yaml:"refs"`
}

func (a flatpakAsset) commands() ([]Command, error) {
	if a.Remote == "" {
		a.Remote = "flathub"
	}
	if len(a.Refs) == 0 {
		return nil, fmt.Errorf("flatpak asset %q has no refs", a.ID)
	}
	if err := validateFlatpakRefs(a); err != nil {
		return nil, err
	}
	commands := make([]Command, 0, len(a.Refs))
	for _, ref := range a.Refs {
		commands = append(commands, Command{Program: "flatpak", Args: []string{"install", a.Remote, ref, "-y"}})
	}
	return commands, nil
}

func (a flatpakAsset) getID() string { return a.ID }

func validateFlatpakRefs(a flatpakAsset) error {
	hasOBS := false
	hasOBSPlugin := false
	for _, ref := range a.Refs {
		if ref == "com.obsproject.Studio" {
			hasOBS = true
		}
		if strings.HasPrefix(ref, "com.obsproject.Studio.Plugin.") {
			hasOBSPlugin = true
		}
	}
	if hasOBSPlugin && !hasOBS {
		return fmt.Errorf("flatpak asset %q installs OBS plugins without com.obsproject.Studio", a.ID)
	}
	return nil
}

type shellAsset struct {
	ID     string `yaml:"id"`
	Sudo   bool   `yaml:"sudo"`
	Script string `yaml:"script"`
}

func (a shellAsset) commands() ([]Command, error) {
	if strings.TrimSpace(a.Script) == "" {
		return nil, fmt.Errorf("shell asset %q has empty script", a.ID)
	}
	return shellCommand(a.Script, a.Sudo), nil
}

func (a shellAsset) getID() string { return a.ID }

type cargoAsset struct {
	ID        string   `yaml:"id"`
	Installer string   `yaml:"installer"`
	Packages  []string `yaml:"packages"`
}

func (a cargoAsset) commands() ([]Command, error) {
	if len(a.Packages) == 0 {
		return nil, fmt.Errorf("cargo asset %q has no packages", a.ID)
	}
	installer := a.Installer
	if installer == "" {
		installer = "cargo-binstall"
	}
	script := fmt.Sprintf("require_command %s\n%s -y %s", shellWord(installer), installer, strings.Join(a.Packages, " "))
	return shellCommand(script, false), nil
}

func (a cargoAsset) getID() string { return a.ID }

type sdkmanAsset struct {
	ID       string   `yaml:"id"`
	Packages []string `yaml:"packages"`
}

func (a sdkmanAsset) commands() ([]Command, error) {
	if len(a.Packages) == 0 {
		return nil, fmt.Errorf("sdkman asset %q has no packages", a.ID)
	}
	var b strings.Builder
	b.WriteString(`if [ ! -s "$HOME/.sdkman/bin/sdkman-init.sh" ]; then` + "\n")
	b.WriteString(`  curl -s "https://get.sdkman.io" | bash` + "\n")
	b.WriteString("fi\n")
	b.WriteString("source_bootstrap_env\n")
	b.WriteString("require_command sdk\n")
	for _, pkg := range a.Packages {
		fmt.Fprintf(&b, "sdk install %s || true\n", shellWord(pkg))
	}
	return shellCommand(b.String(), false), nil
}

func (a sdkmanAsset) getID() string { return a.ID }

type fontAsset struct {
	ID         string   `yaml:"id"`
	Repository string   `yaml:"repository"`
	Release    string   `yaml:"release"`
	Families   []string `yaml:"families"`
}

func (a fontAsset) commands() ([]Command, error) {
	if len(a.Families) == 0 {
		return nil, fmt.Errorf("font asset %q requires families", a.ID)
	}
	release := a.Release
	if release == "" {
		release = "latest"
	}
	executable, err := os.Executable()
	if err != nil {
		executable = os.Args[0]
	}
	args := []string{"--install-nerd-fonts", "--nerd-font-release", release}
	args = append(args, a.Families...)
	return []Command{{Program: executable, Args: args}}, nil
}

func (a fontAsset) getID() string { return a.ID }
