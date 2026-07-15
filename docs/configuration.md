# Configuration

`uboot` is intentionally data-driven. Go code validates and compiles installer assets into commands; YAML files define what can be installed.

## Layout

```text
configs/
  profile.yaml            # SystemBootstrapProfile: run policy + metadata
  modules.yaml            # ordered, selectable plan
  binstaller/
    developer-binaries.yaml   # BinaryDistributionProfile for binstaller
  nerd-fonts/
    fonts.yaml            # nerd-fonts-installer config
  installers/
    apt.yaml
    binary.yaml           # bootstraps binstaller + helm/kustomize only
    binstaller.yaml       # delegates a profile to the binstaller CLI
    cargo.yaml
    dotfiles.yaml
    flatpak.yaml
    nerd-fonts.yaml       # delegates a config to nerd-fonts-installer
    sdkman.yaml
    shell.yaml
    snap.yaml
```

## Profile and Policy

`profile.yaml` is a binstaller-style `SystemBootstrapProfile` envelope that carries run policy and metadata for the whole bootstrap. It is optional; without it `uboot` uses built-in defaults.

```yaml
apiVersion: bootstrap.worxbend.io/v1alpha1
kind: SystemBootstrapProfile
metadata:
  name: ubuntu-workstation
spec:
  policy:
    dryRun: false
    continueOnError: false
    requireConfirmation: true
    appsDir: "${HOME}/.apps"
    stateFile: .bootstrap/state.json
    distro: ubuntu
  vars:
    arch: amd64
```

Policy values are defaults only; a CLI flag always wins (`--dry-run`, `--keep-going`, `--yes`, `--state`). `requireConfirmation: false` behaves like `--yes`; `continueOnError: true` behaves like `--keep-going`.

`uboot` targets Ubuntu only. Before executing, it confirms the host is Ubuntu (or a close Debian/Ubuntu derivative) from `/etc/os-release`. Use `--skip-os-check` to override, or set `UBOOT_OS_RELEASE` to point the check at a different file.

## Modules

`modules.yaml` controls the TUI. Each module references assets by installer type and asset id:

```yaml
modules:
  - id: terminal-cli
    title: Terminal & CLI Tools
    default: true
    steps:
      - name: CLI packages
        installer: apt
        asset: terminal-cli
```

Modules can depend on other modules. Dependencies are automatically added to the execution plan and emitted before dependent modules:

```yaml
modules:
  - id: cargo-packages
    depends_on: [language-installers]
```

This is used for cases where a selected module needs a provider created elsewhere, such as Cargo packages needing Rust/Cargo bootstrap, SDKMAN packages needing SDKMAN bootstrap, Oh My Zsh plugins needing Oh My Zsh, and dotfiles needing Dotbot.

The dotfiles module installs Dotbot first, then applies the vendored `dotfiles/` tree with a generated config from `configs/installers/dotfiles.yaml`. The vendored tree comes from `w0rxbend/system-bootstrap/.files` with `arch+hypr` and `opensuse` omitted, so applying dotfiles no longer depends on cloning `system-bootstrap` at runtime.

The `repo-tooling` module installs the local developer workflow: `just`, `yamlfmt`, `yamllint`, `actionlint`, `gopls`, Prettier, `yaml-language-server`, ShellCheck/shfmt, VS Code extensions, and Neovim-compatible LSP tools.

The installer file owns the entity list:

```yaml
assets:
  - id: terminal-cli
    packages: [alacritty, kitty, ripgrep]
```

Validate catalog structure and command generation before running installs:

```bash
./bin/uboot --config ./configs --validate
```

## Installer Types

- `apt`: package groups plus `update` and `upgrade` actions. Each package installs in its own `apt install -y <pkg>` command.
- `snap`: snap package entries with `classic` and `channel` options, one command per package.
- `flatpak`: Flatpak refs grouped by remote, one command per ref.
- `binary`: upstream binary downloads into a managed root, with symlinks into `~/.local/bin`. Used only to bootstrap the `binstaller` CLI and to install helm/kustomize (whose upstream install scripts binstaller cannot express).
- `binstaller`: delegates a `BinaryDistributionProfile` to the external `binstaller` CLI (`binstaller apply --config <profile>`). binstaller owns download, checksum verification, extraction, symlinking, and per-tool resume state.
- `nerd-fonts`: delegates a font config to the external `nerd-fonts-installer` CLI. The installer binary is provided by the binstaller profile under `${appsDir}`.
- `dotfiles`: vendored local directory or Git-backed dotfile checkout plus generated Dotbot link config, applied with the `dotbot` binary from the binstaller profile.
- `cargo`: Rust CLI packages installed with `cargo-binstall` or another configured cargo installer.
- `sdkman`: SDKMAN candidates.
- `shell`: escape hatch for operations that are not yet worth modeling as a typed installer.

### Delegated installer assets

```yaml
# installers/binstaller.yaml
assets:
  - id: developer-binaries
    command: binstaller # binary name or path; defaults to "binstaller"
    profile: binstaller/developer-binaries.yaml
    # state: developer-binaries.state.json   # optional --state override
    # args: ["--only", "kubectl"]            # optional extra flags

# installers/nerd-fonts.yaml
assets:
  - id: nerd-fonts
    command: ${HOME}/.apps/nerd-font-installer/bin/nerdfont-install
    config: nerd-fonts/fonts.yaml
```

## Resume State

Every command that finishes successfully is recorded in a JSON state file (default `.bootstrap/state.json`, from `policy.stateFile` or `--state`). Entries are keyed by a content hash of the command, so editing a step in config re-runs it while untouched steps are skipped. The file is rewritten atomically after each success, so an interrupted run (reboot, `Ctrl-C`, fatal error) resumes from the first unfinished command on the next invocation. `--reset-state` discards it.

Delegated commands (`binstaller`, `nerd-fonts`, `dotfiles`) bypass this state: they always run and let the delegated tool decide what work remains, which also means config changes those tools see are always applied.

## Runtime Safety

apt/snap/flatpak installs run one entity per command. Each apt package is checked with `apt-cache policy` before its install command runs; missing packages are reported, written to `.bootstrap/logs/warnings/`, and skipped. Because packages install individually and best-effort, one failed or unavailable package does not abort the rest of the set — it is logged and left unrecorded so it is retried on the next run.

The `binary` installer downloads under `${HOME}/.local/share/uboot/apps/<tool>/<version>` and creates symlinks under `${HOME}/.local/bin`, avoiding deletion of user-managed directories from previous manual installs. Its tools can include an optional lowercase hex `sha256` field verified before extract/execute:

```yaml
tools:
  - name: example
    version: "1.0.0"
    url: https://example.test/example-linux-amd64.tar.gz
    archive: tar.gz
    sha256: 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
    binaries: [example]
```

The `binstaller`-delegated dev binaries live under `policy.appsDir` (`${HOME}/.apps`) and are checksum-verified by binstaller itself; the `BinaryDistributionProfile` pins versions and per-tool `checksum.sha256` values.

Runtime command failures stop the bootstrap run by default. `--keep-going` (or `policy.continueOnError`) continues through the rest of the selected plan but still returns a non-zero exit status if any command failed. Per-package install failures always continue.

The catalog loader validates module dependencies, rejects dependency cycles, and catches known asset-level dependency problems such as OBS plugin Flatpaks without `com.obsproject.Studio`.

Remote script installers still exist in `shell.yaml` because the upstream bootstrap scripts use them. Treat those assets as high-trust actions. Prefer converting repeated patterns into typed installers with pinned versions and checksums over time.
