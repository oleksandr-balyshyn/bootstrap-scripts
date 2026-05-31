package bootstrap

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestLoadProjectCatalog(t *testing.T) {
	catalog, err := LoadCatalog("../../configs")
	if err != nil {
		t.Fatalf("LoadCatalog() error = %v", err)
	}
	if len(catalog.Modules) != 30 {
		t.Fatalf("loaded %d modules, want 30", len(catalog.Modules))
	}

	seen := map[string]bool{}
	for _, module := range catalog.Modules {
		if module.ID == "" {
			t.Fatal("module with empty id")
		}
		if seen[module.ID] {
			t.Fatalf("duplicate module id %q", module.ID)
		}
		seen[module.ID] = true
		if module.Title == "" {
			t.Fatalf("module %q has empty title", module.ID)
		}
		if len(module.Steps) == 0 {
			t.Fatalf("module %q has no steps", module.ID)
		}
		for _, step := range module.Steps {
			if step.Name == "" {
				t.Fatalf("module %q has step with empty name", module.ID)
			}
			if len(step.Commands) == 0 {
				t.Fatalf("module %q step %q has no commands", module.ID, step.Name)
			}
		}
	}
}

func TestLoadCatalogRejectsUnknownAsset(t *testing.T) {
	fsys := minimalConfigFS(`modules:
  - id: broken
    title: Broken
    steps:
      - name: Missing package set
        installer: apt
        asset: does-not-exist
`)

	_, err := LoadCatalogFS(fsys, ".")
	if err == nil {
		t.Fatal("LoadCatalogFS() error = nil, want unknown asset error")
	}
	if !strings.Contains(err.Error(), `unknown apt asset "does-not-exist"`) {
		t.Fatalf("LoadCatalogFS() error = %q", err)
	}
}

func TestLoadCatalogBuildsTypedCommands(t *testing.T) {
	fsys := minimalConfigFS(`modules:
  - id: workstation
    title: Workstation
    default: true
    steps:
      - name: Install packages
        installer: apt
        asset: base
      - name: Install snap
        installer: snap
        asset: terminal
`)

	catalog, err := LoadCatalogFS(fsys, ".")
	if err != nil {
		t.Fatalf("LoadCatalogFS() error = %v", err)
	}
	module := catalog.Modules[0]
	if !module.Default {
		t.Fatal("module default flag was not preserved")
	}
	apt := module.Steps[0].Commands[0]
	if got := apt.String(); got != "sudo apt install -y curl git" {
		t.Fatalf("apt command = %q", got)
	}
	snap := module.Steps[1].Commands[0]
	if got := snap.String(); got != "sudo snap install ghostty --classic" {
		t.Fatalf("snap command = %q", got)
	}
}

func TestLoadCatalogRejectsUnsupportedBinaryArchive(t *testing.T) {
	fsys := minimalConfigFS(`modules:
  - id: broken-binary
    title: Broken Binary
    steps:
      - name: Install broken binary
        installer: binary
        asset: broken
`)
	fsys["installers/binary.yaml"] = &fstest.MapFile{Data: []byte(`assets:
  - id: broken
    root: ${HOME}/.local/share/uboot/apps
    tools:
      - name: bad
        version: "1"
        url: https://example.test/bad.bin
        archive: rar
        binaries: [bad]
`)}

	_, err := LoadCatalogFS(fsys, ".")
	if err == nil {
		t.Fatal("LoadCatalogFS() error = nil, want unsupported archive error")
	}
	if !strings.Contains(err.Error(), `unsupported archive type "rar"`) {
		t.Fatalf("LoadCatalogFS() error = %q", err)
	}
}

func TestLoadCatalogRejectsUnknownModuleDependency(t *testing.T) {
	fsys := minimalConfigFS(`modules:
  - id: child
    title: Child
    depends_on: [missing-parent]
    steps:
      - name: Install packages
        installer: apt
        asset: base
`)

	_, err := LoadCatalogFS(fsys, ".")
	if err == nil {
		t.Fatal("LoadCatalogFS() error = nil, want dependency error")
	}
	if !strings.Contains(err.Error(), `module "child" depends on unknown module "missing-parent"`) {
		t.Fatalf("LoadCatalogFS() error = %q", err)
	}
}

func TestLoadCatalogRejectsModuleDependencyCycle(t *testing.T) {
	fsys := minimalConfigFS(`modules:
  - id: one
    title: One
    depends_on: [two]
    steps:
      - name: Install packages
        installer: apt
        asset: base
  - id: two
    title: Two
    depends_on: [one]
    steps:
      - name: Install packages
        installer: apt
        asset: base
`)

	_, err := LoadCatalogFS(fsys, ".")
	if err == nil {
		t.Fatal("LoadCatalogFS() error = nil, want cycle error")
	}
	if !strings.Contains(err.Error(), "dependency cycle detected") {
		t.Fatalf("LoadCatalogFS() error = %q", err)
	}
}

func TestLoadCatalogRejectsOBSPluginsWithoutStudio(t *testing.T) {
	fsys := minimalConfigFS(`modules:
  - id: obs
    title: OBS
    steps:
      - name: Install plugin
        installer: flatpak
        asset: obs-plugin-only
`)
	fsys["installers/flatpak.yaml"] = &fstest.MapFile{Data: []byte(`assets:
  - id: obs-plugin-only
    refs: [com.obsproject.Studio.Plugin.DroidCam]
`)}

	_, err := LoadCatalogFS(fsys, ".")
	if err == nil {
		t.Fatal("LoadCatalogFS() error = nil, want OBS plugin dependency error")
	}
	if !strings.Contains(err.Error(), "installs OBS plugins without com.obsproject.Studio") {
		t.Fatalf("LoadCatalogFS() error = %q", err)
	}
}

func minimalConfigFS(modules string) fstest.MapFS {
	files := fstest.MapFS{
		"modules.yaml":             {Data: []byte(modules)},
		"installers/apt.yaml":      {Data: []byte("assets:\n  - id: base\n    packages: [curl, git]\n")},
		"installers/snap.yaml":     {Data: []byte("assets:\n  - id: terminal\n    packages:\n      - name: ghostty\n        classic: true\n")},
		"installers/flatpak.yaml":  {Data: []byte("assets: []\n")},
		"installers/shell.yaml":    {Data: []byte("assets: []\n")},
		"installers/cargo.yaml":    {Data: []byte("assets: []\n")},
		"installers/sdkman.yaml":   {Data: []byte("assets: []\n")},
		"installers/fonts.yaml":    {Data: []byte("assets: []\n")},
		"installers/binary.yaml":   {Data: []byte("assets: []\n")},
		"installers/dotfiles.yaml": {Data: []byte("assets: []\n")},
	}
	return files
}
