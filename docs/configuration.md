# Configuration

`uboot` is intentionally data-driven. Go code validates and compiles installer assets into commands; YAML files define what can be installed.

## Layout

```text
configs/
  modules.yaml
  installers/
    apt.yaml
    binary.yaml
    cargo.yaml
    flatpak.yaml
    fonts.yaml
    sdkman.yaml
    shell.yaml
    snap.yaml
```

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

The dotfiles module installs Dotbot first, clones `w0rxbend/system-bootstrap`, removes excluded `.files` folders, then runs Dotbot with a generated config from `configs/installers/dotfiles.yaml`.

The `repo-tooling` module installs the local developer workflow: `just`, `yamlfmt`, `yamllint`, `actionlint`, `gopls`, Prettier, `yaml-language-server`, ShellCheck/shfmt, VS Code extensions, and Neovim-compatible LSP tools.

The installer file owns the entity list:

```yaml
assets:
  - id: terminal-cli
    packages: [alacritty, kitty, ripgrep]
```

## Installer Types

- `apt`: package groups plus `update` and `upgrade` actions.
- `snap`: snap package entries with `classic` and `channel` options.
- `flatpak`: Flatpak refs grouped by remote.
- `binary`: upstream binary downloads into a managed root, with symlinks into `~/.local/bin`.
- `dotfiles`: Git-backed dotfile checkout plus generated Dotbot link config.
- `cargo`: Rust CLI packages installed with `cargo-binstall` or another configured cargo installer.
- `sdkman`: SDKMAN candidates.
- `font`: Nerd Font family lists.
- `shell`: escape hatch for operations that are not yet worth modeling as a typed installer.

## Runtime Safety

APT package names are checked with `apt-cache show` before an install command runs. Missing packages are reported, written to `.bootstrap/logs/warnings/`, skipped, and the rest of the install continues.

Binary downloads install under `${HOME}/.local/share/uboot/apps/<tool>/<version>` and create symlinks under `${HOME}/.local/bin`. This avoids deleting user-managed directories from previous manual installs.

The catalog loader validates module dependencies, rejects dependency cycles, and catches known asset-level dependency problems such as OBS plugin Flatpaks without `com.obsproject.Studio`.

Remote script installers still exist in `shell.yaml` because the upstream bootstrap scripts use them. Treat those assets as high-trust actions. Prefer converting repeated patterns into typed installers with pinned versions and checksums over time.
