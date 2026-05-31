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
	DependsOn   []string
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
	DependsOn   []string     `yaml:"depends_on"`
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
			DependsOn:   cfg.DependsOn,
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
	if err := validateModuleDependencies(catalog); err != nil {
		return Catalog{}, err
	}
	return catalog, nil
}

func validateModuleDependencies(catalog Catalog) error {
	known := make(map[string]bool, len(catalog.Modules))
	for _, module := range catalog.Modules {
		known[module.ID] = true
	}
	for _, module := range catalog.Modules {
		for _, dep := range module.DependsOn {
			if !known[dep] {
				return fmt.Errorf("module %q depends on unknown module %q", module.ID, dep)
			}
		}
	}
	return detectDependencyCycles(catalog)
}

func detectDependencyCycles(catalog Catalog) error {
	modules := make(map[string]Module, len(catalog.Modules))
	for _, module := range catalog.Modules {
		modules[module.ID] = module
	}

	const (
		unvisited = 0
		visiting  = 1
		visited   = 2
	)

	state := map[string]int{}
	var visit func(id string, stack []string) error
	visit = func(id string, stack []string) error {
		switch state[id] {
		case visiting:
			return fmt.Errorf("dependency cycle detected: %s -> %s", strings.Join(stack, " -> "), id)
		case visited:
			return nil
		}

		state[id] = visiting
		for _, dep := range modules[id].DependsOn {
			if err := visit(dep, append(stack, id)); err != nil {
				return err
			}
		}
		state[id] = visited
		return nil
	}

	for _, module := range catalog.Modules {
		if state[module.ID] == unvisited {
			if err := visit(module.ID, nil); err != nil {
				return err
			}
		}
	}
	return nil
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
	dotfile map[string]dotfilesAsset
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
		{"dotfiles.yaml", func() error { return loadAssetMap(fsys, filepath.Join(dir, "dotfiles.yaml"), &c.dotfile) }},
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
	case "dotfiles":
		asset, ok := c.dotfile[id]
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
	if err := validateFlatpakRefs(a); err != nil {
		return nil, err
	}
	commands := make([]Command, 0, len(a.Refs))
	for _, ref := range a.Refs {
		commands = append(commands, Command{Program: "flatpak", Args: []string{"install", a.Remote, ref, "-y"}})
	}
	return commands, nil
}

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
	return shellCommand(script, false), nil
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

type dotfilesAsset struct {
	ID         string        `yaml:"id"`
	Repository string        `yaml:"repository"`
	Ref        string        `yaml:"ref"`
	Checkout   string        `yaml:"checkout"`
	Dotbot     string        `yaml:"dotbot"`
	Excludes   []string      `yaml:"excludes"`
	Create     []string      `yaml:"create"`
	Shell      []string      `yaml:"shell"`
	Links      []dotfileLink `yaml:"links"`
}

type dotfileLink struct {
	Target string `yaml:"target"`
	Source string `yaml:"source"`
}

func (a dotfilesAsset) commands() ([]Command, error) {
	if a.Repository == "" || a.Checkout == "" || len(a.Links) == 0 {
		return nil, fmt.Errorf("dotfiles asset %q requires repository, checkout, and links", a.ID)
	}
	if a.Ref == "" {
		a.Ref = "main"
	}
	if a.Dotbot == "" {
		a.Dotbot = "${HOME}/.local/bin/dotbot"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "repo_dir=%s\n", shellDouble(a.Checkout))
	fmt.Fprintf(&b, "repo_url=%s\n", shellWord(a.Repository))
	fmt.Fprintf(&b, "repo_ref=%s\n", shellWord(a.Ref))
	fmt.Fprintf(&b, "dotbot=%s\n", shellDouble(a.Dotbot))
	b.WriteString(`if [ ! -x "$dotbot" ]; then
  dotbot="${HOME}/.local/share/uboot/apps/dotbot/1.24.0/dotbot"
fi
mkdir -p "$(dirname "$repo_dir")"
if [ -d "$repo_dir/.git" ]; then
  git -C "$repo_dir" fetch --depth 1 origin "$repo_ref"
  git -C "$repo_dir" reset --hard "origin/$repo_ref"
else
  git clone --depth 1 --branch "$repo_ref" "$repo_url" "$repo_dir"
fi
`)
	for _, exclude := range a.Excludes {
		fmt.Fprintf(&b, "rm -rf \"$repo_dir/.files/%s\"\n", escapeShellPathSegment(exclude))
	}

	b.WriteString(`config="$repo_dir/.files/uboot.install.conf.yaml"
cat > "$config" <<'DOTBOT_CONFIG'
- defaults:
    link:
      relink: true
      create: true
`)
	if len(a.Create) > 0 {
		b.WriteString("- create:\n")
		for _, dir := range a.Create {
			fmt.Fprintf(&b, "    - %s\n", dir)
		}
	}
	if len(a.Shell) > 0 {
		b.WriteString("- shell:\n")
		for _, command := range a.Shell {
			fmt.Fprintf(&b, "    - %s\n", command)
		}
	}
	b.WriteString("- link:\n")
	for _, link := range a.Links {
		if link.Target == "" || link.Source == "" {
			return nil, fmt.Errorf("dotfiles asset %q has incomplete link", a.ID)
		}
		fmt.Fprintf(&b, "    %s: %s\n", link.Target, link.Source)
	}
	b.WriteString("DOTBOT_CONFIG\n")
	b.WriteString(`"$dotbot" -d "$repo_dir/.files" -c "$config"` + "\n")
	return shellCommand(b.String(), false), nil
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
	return shellCommand(b.String(), false), nil
}

func shellCommand(script string, sudo bool) []Command {
	return []Command{{Program: "bash", Args: []string{"-lc", bootstrapShellPrelude() + script}, Sudo: sudo}}
}

func bootstrapShellPrelude() string {
	return `set -euo pipefail

source_shell_file_if_exists() {
  local file="$1"
  if [ -s "$file" ]; then
    set +u
    # shellcheck disable=SC1090
    . "$file" >/dev/null 2>&1 || true
    set -u
  fi
}

source_env_file_if_exists() {
  local file="$1"
  if [ -s "$file" ]; then
    # shellcheck disable=SC1090
    . "$file"
  fi
}

prepend_path() {
  local dir="$1"
  if [ -d "$dir" ]; then
    case ":$PATH:" in
      *":$dir:"*) ;;
      *) PATH="$dir:$PATH" ;;
    esac
  fi
}

merge_zsh_path() {
  if command -v zsh >/dev/null 2>&1 && [ -s "$HOME/.zshrc" ]; then
    local zsh_path
    zsh_path="$(zsh -lc 'print -r -- "$PATH"' 2>/dev/null || true)"
    if [ -n "$zsh_path" ]; then
      PATH="$zsh_path:$PATH"
    fi
  fi
}

source_bootstrap_env() {
  source_shell_file_if_exists "$HOME/.profile"
  source_shell_file_if_exists "$HOME/.bash_profile"
  source_shell_file_if_exists "$HOME/.bashrc"
  merge_zsh_path
  prepend_path "$HOME/.local/bin"
  prepend_path "$HOME/go/bin"
  prepend_path "$HOME/.go/bin"
  prepend_path "$HOME/.cargo/bin"
  prepend_path "$HOME/.local/share/uboot/toolchains/go/1.26.2/bin"
  prepend_path "$HOME/.local/share/uboot/apps/dotbot/1.24.0"
  source_env_file_if_exists "$HOME/.cargo/env"
  source_env_file_if_exists "$HOME/.sdkman/bin/sdkman-init.sh"
  export PATH
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "required command not available after sourcing bootstrap environment: $1" >&2
    return 127
  fi
}

source_bootstrap_env
`
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
func (a dotfilesAsset) getID() string {
	return a.ID
}

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

func escapeShellPathSegment(value string) string {
	return strings.ReplaceAll(value, `"`, `\"`)
}
