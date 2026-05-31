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
	if len(catalog.Modules) != 31 {
		t.Fatalf("loaded %d modules, want 31", len(catalog.Modules))
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

func TestShellCommandsRefreshBootstrapEnvironment(t *testing.T) {
	fsys := minimalConfigFS(`modules:
  - id: shell-module
    title: Shell Module
    steps:
      - name: Run shell
        installer: shell
        asset: custom
      - name: Run cargo
        installer: cargo
        asset: rust-tools
      - name: Run sdkman
        installer: sdkman
        asset: jvm-tools
`)
	fsys["installers/shell.yaml"] = &fstest.MapFile{Data: []byte(`assets:
  - id: custom
    script: echo ready
`)}
	fsys["installers/cargo.yaml"] = &fstest.MapFile{Data: []byte(`assets:
  - id: rust-tools
    packages: [cargo-nextest]
`)}
	fsys["installers/sdkman.yaml"] = &fstest.MapFile{Data: []byte(`assets:
  - id: jvm-tools
    packages: [java]
`)}

	catalog, err := LoadCatalogFS(fsys, ".")
	if err != nil {
		t.Fatalf("LoadCatalogFS() error = %v", err)
	}

	for _, step := range catalog.Modules[0].Steps {
		command := step.Commands[0]
		if command.Program != "bash" {
			t.Fatalf("%s command program = %q, want bash", step.Name, command.Program)
		}
		script := command.Args[1]
		for _, want := range []string{
			`source_shell_file_if_exists "$HOME/.bashrc"`,
			"merge_zsh_path",
			`source_env_file_if_exists "$HOME/.cargo/env"`,
			`source_env_file_if_exists "$HOME/.sdkman/bin/sdkman-init.sh"`,
			"source_bootstrap_env",
		} {
			if !strings.Contains(script, want) {
				t.Fatalf("%s script does not contain %q:\n%s", step.Name, want, script)
			}
		}
		if strings.Contains(script, "verify_sha256") {
			t.Fatalf("%s script unexpectedly contains binary checksum helper:\n%s", step.Name, script)
		}
	}

	cargoScript := catalog.Modules[0].Steps[1].Commands[0].Args[1]
	if !strings.Contains(cargoScript, "require_command 'cargo-binstall'") {
		t.Fatalf("cargo script does not require cargo-binstall:\n%s", cargoScript)
	}

	sdkmanScript := catalog.Modules[0].Steps[2].Commands[0].Args[1]
	if !strings.Contains(sdkmanScript, `curl -s "https://get.sdkman.io" | bash`) {
		t.Fatalf("sdkman script does not install sdkman when missing:\n%s", sdkmanScript)
	}
	if !strings.Contains(sdkmanScript, "require_command sdk") {
		t.Fatalf("sdkman script does not require sdk after sourcing environment:\n%s", sdkmanScript)
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

func TestLoadCatalogBuildsBinaryChecksumCommand(t *testing.T) {
	fsys := minimalConfigFS(`modules:
  - id: binary
    title: Binary
    steps:
      - name: Install binary
        installer: binary
        asset: tools
`)
	fsys["installers/binary.yaml"] = &fstest.MapFile{Data: []byte(`assets:
  - id: tools
    root: ${HOME}/.local/share/uboot/apps
    tools:
      - name: sample
        version: "1"
        url: https://example.test/sample.tar.gz
        archive: tar.gz
        sha256: 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
        binaries: [sample]
`)}

	catalog, err := LoadCatalogFS(fsys, ".")
	if err != nil {
		t.Fatalf("LoadCatalogFS() error = %v", err)
	}
	script := catalog.Modules[0].Steps[0].Commands[0].Args[1]
	for _, want := range []string{
		"verify_sha256()",
		`verify_sha256 "$stage/archive.tar.gz" '0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef'`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("binary script does not contain %q:\n%s", want, script)
		}
	}
}

func TestLoadCatalogRejectsInvalidBinaryChecksum(t *testing.T) {
	fsys := minimalConfigFS(`modules:
  - id: binary
    title: Binary
    steps:
      - name: Install binary
        installer: binary
        asset: tools
`)
	fsys["installers/binary.yaml"] = &fstest.MapFile{Data: []byte(`assets:
  - id: tools
    root: ${HOME}/.local/share/uboot/apps
    tools:
      - name: sample
        version: "1"
        url: https://example.test/sample.tar.gz
        archive: tar.gz
        sha256: not-a-checksum
        binaries: [sample]
`)}

	_, err := LoadCatalogFS(fsys, ".")
	if err == nil {
		t.Fatal("LoadCatalogFS() error = nil, want invalid checksum error")
	}
	if !strings.Contains(err.Error(), "sha256 must be 64 lowercase hex characters") {
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

func TestLoadCatalogBuildsLocalDotfilesCommand(t *testing.T) {
	fsys := minimalConfigFS(`modules:
  - id: dotfiles
    title: Dotfiles
    steps:
      - name: Symlink dotfiles
        installer: dotfiles
        asset: local-dotfiles
`)
	fsys["installers/dotfiles.yaml"] = &fstest.MapFile{Data: []byte(`assets:
  - id: local-dotfiles
    local: ../dotfiles
    links:
      - target: ~/.zshrc
        source: .zshrc
`)}

	catalog, err := LoadCatalogFS(fsys, ".")
	if err != nil {
		t.Fatalf("LoadCatalogFS() error = %v", err)
	}

	script := catalog.Modules[0].Steps[0].Commands[0].Args[1]
	for _, want := range []string{
		`source_dir="../dotfiles"`,
		`prepare_dotfile_target "${HOME}/.zshrc"`,
		`Remove it and replace with a symlink? [y/N]`,
		`rm -rf -- "$target"`,
		`"$dotbot" -d "$source_dir" -c "$config"`,
		"~/.zshrc: .zshrc",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("local dotfiles script does not contain %q:\n%s", want, script)
		}
	}
	if strings.Contains(script, "git clone") {
		t.Fatalf("local dotfiles script unexpectedly clones repository:\n%s", script)
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
