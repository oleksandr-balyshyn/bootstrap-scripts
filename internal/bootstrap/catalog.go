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
	// ContinueOnError marks a command whose failure is non-fatal: the runner
	// logs it, leaves it unrecorded in state (so it retries on the next run),
	// and proceeds with the remaining commands. It is used for per-package
	// installs so one bad package does not abort the whole plan.
	ContinueOnError bool
	// SkipState marks a command that bypasses uboot's resume state entirely:
	// it is always run and never recorded. This is used for delegated tools
	// (binstaller, nerd-fonts-installer, dotbot) that manage their own state
	// and are idempotent, so re-invoking them resumes their own work while
	// still picking up config changes uboot cannot see from the command line.
	SkipState bool
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
	return loadCatalogFS(os.DirFS(dir), ".", dir)
}

// LoadCatalogFS is the testable variant of LoadCatalog.
func LoadCatalogFS(fsys fs.FS, dir string) (Catalog, error) {
	return loadCatalogFS(fsys, dir, "")
}

func loadCatalogFS(fsys fs.FS, dir, localBase string) (Catalog, error) {
	compiler, err := loadCompiler(fsys, filepath.Join(dir, "installers"), localBase)
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
	apt        map[string]aptAsset
	snap       map[string]snapAsset
	flatpak    map[string]flatpakAsset
	shell      map[string]shellAsset
	cargo      map[string]cargoAsset
	sdkman     map[string]sdkmanAsset
	nerdfonts  map[string]nerdFontsAsset
	binary     map[string]binaryAsset
	binstaller map[string]binstallerAsset
	dotfile    map[string]dotfilesAsset
	localBase  string
}

func loadCompiler(fsys fs.FS, dir, localBase string) (compiler, error) {
	c := compiler{localBase: localBase}
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
		{"nerd-fonts.yaml", func() error { return loadAssetMap(fsys, filepath.Join(dir, "nerd-fonts.yaml"), &c.nerdfonts) }},
		{"binary.yaml", func() error { return loadAssetMap(fsys, filepath.Join(dir, "binary.yaml"), &c.binary) }},
		{"binstaller.yaml", func() error { return loadAssetMap(fsys, filepath.Join(dir, "binstaller.yaml"), &c.binstaller) }},
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
	case "nerd-fonts":
		asset, ok := c.nerdfonts[id]
		if !ok {
			return nil, unknownAsset(installer, id)
		}
		return asset.commands(c.localBase)
	case "binary":
		asset, ok := c.binary[id]
		if !ok {
			return nil, unknownAsset(installer, id)
		}
		return asset.commands()
	case "binstaller":
		asset, ok := c.binstaller[id]
		if !ok {
			return nil, unknownAsset(installer, id)
		}
		return asset.commands(c.localBase)
	case "dotfiles":
		asset, ok := c.dotfile[id]
		if !ok {
			return nil, unknownAsset(installer, id)
		}
		return asset.commands(c.localBase)
	default:
		return nil, fmt.Errorf("unknown installer %q", installer)
	}
}

func unknownAsset(installer, id string) error {
	return fmt.Errorf("unknown %s asset %q", installer, id)
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
