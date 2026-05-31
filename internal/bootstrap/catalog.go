// Package bootstrap contains the domain model used by the TUI and runner.
//
// The executable intentionally reads installable assets from YAML files under
// configs/ instead of compiling workstation choices into Go. Go code owns
// validation and command generation; configuration owns package/app/tool lists.
package bootstrap

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Catalog is the validated list of selectable bootstrap modules.
type Catalog struct {
	Modules []Module
}

// Module is a user-facing group in the TUI.
type Module struct {
	ID          string
	Title       string
	Description string
	Source      string
	Tags        []string
	Default     bool
	Steps       []Step
}

// Step is a named unit of work within a module.
type Step struct {
	Name     string
	Commands []Command
}

// Command is the low-level process invocation produced from typed config.
type Command struct {
	Program string
	Args    []string
	Sudo    bool
}

type moduleFile struct {
	Modules []moduleConfig `yaml:"modules"`
}

type moduleConfig struct {
	ID          string       `yaml:"id"`
	Title       string       `yaml:"title"`
	Description string       `yaml:"description"`
	Source      string       `yaml:"source"`
	Tags        []string     `yaml:"tags"`
	Default     bool         `yaml:"default"`
	Steps       []stepConfig `yaml:"steps"`
}

type stepConfig struct {
	Name      string `yaml:"name"`
	Installer string `yaml:"installer"`
	Asset     string `yaml:"asset"`
}

// LoadCatalog reads modules.yaml and installer-specific asset files from dir.
func LoadCatalog(dir string) (Catalog, error) {
	return LoadCatalogFS(os.DirFS(dir), ".")
}

// LoadCatalogFS is the testable variant of LoadCatalog.
func LoadCatalogFS(fsys fs.FS, dir string) (Catalog, error) {
	compiler, err := loadCompiler(fsys, filepath.Join(dir, "installers"))
	if err != nil {
		return Catalog{}, err
	}

	var file moduleFile
	if err := readYAML(fsys, filepath.Join(dir, "modules.yaml"), &file); err != nil {
		return Catalog{}, err
	}
	if len(file.Modules) == 0 {
		return Catalog{}, errors.New("catalog has no modules")
	}

	seen := map[string]bool{}
	catalog := Catalog{Modules: make([]Module, 0, len(file.Modules))}
	for _, cfg := range file.Modules {
		if cfg.ID == "" {
			return Catalog{}, errors.New("module id is required")
		}
		if seen[cfg.ID] {
			return Catalog{}, fmt.Errorf("duplicate module id %q", cfg.ID)
		}
		seen[cfg.ID] = true

		module := Module{
			ID:          cfg.ID,
			Title:       cfg.Title,
			Description: cfg.Description,
			Source:      cfg.Source,
			Tags:        cfg.Tags,
			Default:     cfg.Default,
			Steps:       make([]Step, 0, len(cfg.Steps)),
		}
		for _, step := range cfg.Steps {
			commands, err := compiler.compile(step.Installer, step.Asset)
			if err != nil {
				return Catalog{}, fmt.Errorf("module %q step %q: %w", cfg.ID, step.Name, err)
			}
			module.Steps = append(module.Steps, Step{Name: step.Name, Commands: commands})
		}
		catalog.Modules = append(catalog.Modules, module)
	}
	return catalog, nil
}

type compiler struct {
	apt     map[string]aptAsset
	snap    map[string]snapAsset
	flatpak map[string]flatpakAsset
	shell   map[string]shellAsset
	cargo   map[string]cargoAsset
	sdkman  map[string]sdkmanAsset
	fonts   map[string]fontAsset
	binary  map[string]binaryAsset
}

func loadCompiler(fsys fs.FS, dir string) (compiler, error) {
	c := compiler{}
	loaders := []struct {
		name string
		load func() error
	}{
		{"apt.yaml", func() error { return loadAssetMap(fsys, filepath.Join(dir, "apt.yaml"), &c.apt) }},
		{"snap.yaml", func() error { return loadAssetMap(fsys, filepath.Join(dir, "snap.yaml"), &c.snap) }},
		{"flatpak.yaml", func() error { return loadAssetMap(fsys, filepath.Join(dir, "flatpak.yaml"), &c.flatpak) }},
		{"shell.yaml", func() error { return loadAssetMap(fsys, filepath.Join(dir, "shell.yaml"), &c.shell) }},
		{"cargo.yaml", func() error { return loadAssetMap(fsys, filepath.Join(dir, "cargo.yaml"), &c.cargo) }},
		{"sdkman.yaml", func() error { return loadAssetMap(fsys, filepath.Join(dir, "sdkman.yaml"), &c.sdkman) }},
		{"fonts.yaml", func() error { return loadAssetMap(fsys, filepath.Join(dir, "fonts.yaml"), &c.fonts) }},
		{"binary.yaml", func() error { return loadAssetMap(fsys, filepath.Join(dir, "binary.yaml"), &c.binary) }},
	}
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			return compiler{}, fmt.Errorf("%s: %w", loader.name, err)
		}
	}
	return c, nil
}

func (c compiler) compile(installer, id string) ([]Command, error) {
	switch installer {
	case "apt":
		asset, ok := c.apt[id]
		if !ok {
			return nil, unknownAsset(installer, id)
		}
		return asset.commands()
	case "snap":
		asset, ok := c.snap[id]
		if !ok {
			return nil, unknownAsset(installer, id)
		}
		return asset.commands()
	case "flatpak":
		asset, ok := c.flatpak[id]
		if !ok {
			return nil, unknownAsset(installer, id)
		}
		return asset.commands()
	case "shell":
		asset, ok := c.shell[id]
		if !ok {
			return nil, unknownAsset(installer, id)
		}
		return asset.commands()
	case "cargo":
		asset, ok := c.cargo[id]
		if !ok {
			return nil, unknownAsset(installer, id)
		}
		return asset.commands()
	case "sdkman":
		asset, ok := c.sdkman[id]
		if !ok {
			return nil, unknownAsset(installer, id)
		}
		return asset.commands()
	case "font":
		asset, ok := c.fonts[id]
		if !ok {
			return nil, unknownAsset(installer, id)
		}
		return asset.commands()
	case "binary":
		asset, ok := c.binary[id]
		if !ok {
			return nil, unknownAsset(installer, id)
		}
		return asset.commands()
	default:
		return nil, fmt.Errorf("unknown installer %q", installer)
	}
}

func unknownAsset(installer, id string) error {
	return fmt.Errorf("unknown %s asset %q", installer, id)
}

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
	commands := make([]Command, 0, len(a.Refs))
	for _, ref := range a.Refs {
		commands = append(commands, Command{Program: "flatpak", Args: []string{"install", a.Remote, ref, "-y"}})
	}
	return commands, nil
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
	script := "set -euo pipefail\n" + a.Script
	return []Command{{Program: "bash", Args: []string{"-lc", script}, Sudo: a.Sudo}}, nil
}

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
	script := fmt.Sprintf(". \"$HOME/.cargo/env\"\n%s -y %s", installer, strings.Join(a.Packages, " "))
	return []Command{{Program: "bash", Args: []string{"-lc", "set -euo pipefail\n" + script}}}, nil
}

type sdkmanAsset struct {
	ID       string   `yaml:"id"`
	Packages []string `yaml:"packages"`
}

func (a sdkmanAsset) commands() ([]Command, error) {
	if len(a.Packages) == 0 {
		return nil, fmt.Errorf("sdkman asset %q has no packages", a.ID)
	}
	var b strings.Builder
	b.WriteString(`curl -s "https://get.sdkman.io" | bash` + "\n")
	b.WriteString(`. "$HOME/.sdkman/bin/sdkman-init.sh"` + "\n")
	for _, pkg := range a.Packages {
		fmt.Fprintf(&b, "sdk install %s || true\n", shellWord(pkg))
	}
	return []Command{{Program: "bash", Args: []string{"-lc", "set -euo pipefail\n" + b.String()}}}, nil
}

type fontAsset struct {
	ID         string   `yaml:"id"`
	Repository string   `yaml:"repository"`
	Families   []string `yaml:"families"`
}

func (a fontAsset) commands() ([]Command, error) {
	if a.Repository == "" || len(a.Families) == 0 {
		return nil, fmt.Errorf("font asset %q requires repository and families", a.ID)
	}
	script := fmt.Sprintf(`FONT_DIR="${HOME}/.fonts"
mkdir -p "$FONT_DIR"
workdir="$(mktemp -d)"
git clone --depth 1 --filter=blob:none %s "$workdir/nerd-fonts"
cd "$workdir/nerd-fonts"
for font in %s; do
  ./install.sh -U "$font"
done
fc-cache -vf
rm -rf "$workdir"`, shellWord(a.Repository), strings.Join(shellWords(a.Families), " "))
	return []Command{{Program: "bash", Args: []string{"-lc", "set -euo pipefail\n" + script}}}, nil
}

type binaryAsset struct {
	ID    string       `yaml:"id"`
	Root  string       `yaml:"root"`
	Links string       `yaml:"links"`
	Tools []binaryTool `yaml:"tools"`
}

type binaryTool struct {
	Name     string   `yaml:"name"`
	Version  string   `yaml:"version"`
	URL      string   `yaml:"url"`
	Archive  string   `yaml:"archive"`
	Binaries []string `yaml:"binaries"`
	Aliases  []string `yaml:"aliases"`
}

func (a binaryAsset) commands() ([]Command, error) {
	if a.Root == "" || len(a.Tools) == 0 {
		return nil, fmt.Errorf("binary asset %q requires root and tools", a.ID)
	}
	if a.Links == "" {
		a.Links = "~/.local/bin"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "root=%s\nlinks=%s\nmkdir -p \"$root\" \"$links\"\n", shellDouble(a.Root), shellDouble(a.Links))
	for _, tool := range a.Tools {
		if tool.Name == "" || tool.URL == "" || len(tool.Binaries) == 0 {
			return nil, fmt.Errorf("binary asset %q has incomplete tool entry", a.ID)
		}
		if err := writeBinaryInstall(&b, tool); err != nil {
			return nil, fmt.Errorf("binary asset %q tool %q: %w", a.ID, tool.Name, err)
		}
	}
	return []Command{{Program: "bash", Args: []string{"-lc", "set -euo pipefail\n" + b.String()}}}, nil
}

func writeBinaryInstall(b *strings.Builder, tool binaryTool) error {
	url := shellWord(tool.URL)
	fmt.Fprintf(b, "\n# %s %s\n", tool.Name, tool.Version)
	fmt.Fprintf(b, "tool_dir=\"$root/%s/%s\"\n", tool.Name, tool.Version)
	fmt.Fprintln(b, "stage=\"$(mktemp -d)\"")
	fmt.Fprintln(b, "mkdir -p \"$tool_dir\"")
	switch tool.Archive {
	case "none":
		fmt.Fprintf(b, "curl -L -o \"$tool_dir/%s\" %s\nchmod +x \"$tool_dir/%s\"\n", tool.Binaries[0], url, tool.Binaries[0])
	case "zip":
		fmt.Fprintf(b, "curl -L -o \"$stage/archive.zip\" %s\nunzip -q \"$stage/archive.zip\" -d \"$stage/out\"\nfind \"$stage/out\" -type f -perm /111 -exec cp {} \"$tool_dir/\" \\;\n", url)
	case "tar.gz":
		fmt.Fprintf(b, "curl -L -o \"$stage/archive.tar.gz\" %s\ntar -zxf \"$stage/archive.tar.gz\" -C \"$stage\"\nfind \"$stage\" -type f -perm /111 -exec cp {} \"$tool_dir/\" \\;\n", url)
	case "tar.xz":
		fmt.Fprintf(b, "curl -L -o \"$stage/archive.tar.xz\" %s\ntar -xf \"$stage/archive.tar.xz\" -C \"$stage\"\nfind \"$stage\" -type f -perm /111 -exec cp {} \"$tool_dir/\" \\;\n", url)
	case "kubectl-stable":
		fmt.Fprintln(b, `kubectl_version="$(curl -Ls https://dl.k8s.io/release/stable.txt)"`)
		fmt.Fprintln(b, `curl -L -o "$tool_dir/kubectl" "https://dl.k8s.io/release/${kubectl_version}/bin/linux/amd64/kubectl"`)
		fmt.Fprintln(b, `chmod +x "$tool_dir/kubectl"`)
	case "helm-script":
		fmt.Fprintf(b, "curl -fsSL -o \"$stage/get_helm.sh\" %s\n", url)
		fmt.Fprintln(b, `chmod +x "$stage/get_helm.sh"`)
		fmt.Fprintln(b, `USE_SUDO=false HELM_INSTALL_DIR="$tool_dir" "$stage/get_helm.sh"`)
	case "kustomize-script":
		fmt.Fprintf(b, "curl -fsSL -o \"$stage/install_kustomize.sh\" %s\n", url)
		fmt.Fprintln(b, `chmod +x "$stage/install_kustomize.sh"`)
		fmt.Fprintln(b, `"$stage/install_kustomize.sh" "$tool_dir"`)
	default:
		return fmt.Errorf("unsupported archive type %q", tool.Archive)
	}
	for _, bin := range tool.Binaries {
		fmt.Fprintf(b, "if [ -x \"$tool_dir/%s\" ]; then ln -sfn \"$tool_dir/%s\" \"$links/%s\"; fi\n", bin, bin, bin)
	}
	for _, alias := range tool.Aliases {
		if len(tool.Binaries) > 0 {
			fmt.Fprintf(b, "if [ -x \"$tool_dir/%s\" ]; then ln -sfn \"$tool_dir/%s\" \"$links/%s\"; fi\n", tool.Binaries[0], tool.Binaries[0], alias)
		}
	}
	fmt.Fprintln(b, "rm -rf \"$stage\"")
	return nil
}

type assetFile[T interface{ getID() string }] struct {
	Assets []T `yaml:"assets"`
}

func loadAssetMap[T interface{ getID() string }](fsys fs.FS, path string, dest *map[string]T) error {
	var file assetFile[T]
	if err := readYAML(fsys, path, &file); err != nil {
		return err
	}
	assets := make(map[string]T, len(file.Assets))
	for _, asset := range file.Assets {
		id := asset.getID()
		if id == "" {
			return fmt.Errorf("%s contains asset without id", path)
		}
		if _, exists := assets[id]; exists {
			return fmt.Errorf("%s contains duplicate asset id %q", path, id)
		}
		assets[id] = asset
	}
	*dest = assets
	return nil
}

func (a aptAsset) getID() string     { return a.ID }
func (a snapAsset) getID() string    { return a.ID }
func (a flatpakAsset) getID() string { return a.ID }
func (a shellAsset) getID() string   { return a.ID }
func (a cargoAsset) getID() string   { return a.ID }
func (a sdkmanAsset) getID() string  { return a.ID }
func (a fontAsset) getID() string    { return a.ID }
func (a binaryAsset) getID() string  { return a.ID }

func readYAML(fsys fs.FS, path string, out any) error {
	data, err := fs.ReadFile(fsys, filepath.ToSlash(path))
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, out); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func shellWords(values []string) []string {
	words := make([]string, 0, len(values))
	for _, value := range values {
		words = append(words, shellWord(value))
	}
	return words
}

func shellWord(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func shellDouble(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}
