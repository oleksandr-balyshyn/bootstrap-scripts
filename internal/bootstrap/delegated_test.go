package bootstrap

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestBinstallerAssetDelegatesApply(t *testing.T) {
	fsys := minimalConfigFS(`modules:
  - id: binaries
    title: Binaries
    steps:
      - name: Install developer binaries
        installer: binstaller
        asset: developer-binaries
`)
	fsys["installers/binstaller.yaml"] = &fstest.MapFile{Data: []byte(`assets:
  - id: developer-binaries
    command: binstaller
    profile: binstaller/developer-binaries.yaml
    args: ["--verbose"]
`)}

	catalog, err := LoadCatalogFS(fsys, ".")
	if err != nil {
		t.Fatalf("LoadCatalogFS() error = %v", err)
	}
	command := catalog.Modules[0].Steps[0].Commands[0]
	if !command.SkipState {
		t.Fatal("binstaller command should bypass uboot state (SkipState)")
	}
	script := command.Args[1]
	for _, want := range []string{
		"require_command 'binstaller'",
		`apply_args=(apply --config "binstaller/developer-binaries.yaml")`,
		"apply_args+=('--verbose')",
		`binstaller "${apply_args[@]}"`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("binstaller script missing %q:\n%s", want, script)
		}
	}
}

func TestNerdFontsAssetDelegatesInstaller(t *testing.T) {
	fsys := minimalConfigFS(`modules:
  - id: fonts
    title: Fonts
    steps:
      - name: Install Nerd Fonts
        installer: nerd-fonts
        asset: nerd-fonts
`)
	fsys["installers/nerd-fonts.yaml"] = &fstest.MapFile{Data: []byte(`assets:
  - id: nerd-fonts
    command: ${HOME}/.apps/nerd-font-installer/bin/nerdfont-install
    config: nerd-fonts/fonts.yaml
`)}

	catalog, err := LoadCatalogFS(fsys, ".")
	if err != nil {
		t.Fatalf("LoadCatalogFS() error = %v", err)
	}
	command := catalog.Modules[0].Steps[0].Commands[0]
	if !command.SkipState {
		t.Fatal("nerd-fonts command should bypass uboot state (SkipState)")
	}
	script := command.Args[1]
	for _, want := range []string{
		`installer="${HOME}/.apps/nerd-font-installer/bin/nerdfont-install"`,
		`require_command "$installer"`,
		`font_args=(--config "nerd-fonts/fonts.yaml")`,
		`"$installer" "${font_args[@]}"`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("nerd-fonts script missing %q:\n%s", want, script)
		}
	}
}

func TestBinstallerAssetRequiresProfile(t *testing.T) {
	fsys := minimalConfigFS(`modules:
  - id: binaries
    title: Binaries
    steps:
      - name: Install developer binaries
        installer: binstaller
        asset: no-profile
`)
	fsys["installers/binstaller.yaml"] = &fstest.MapFile{Data: []byte(`assets:
  - id: no-profile
    command: binstaller
`)}

	_, err := LoadCatalogFS(fsys, ".")
	if err == nil || !strings.Contains(err.Error(), "requires a profile") {
		t.Fatalf("LoadCatalogFS() error = %v, want profile requirement", err)
	}
}
