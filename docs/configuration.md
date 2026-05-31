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
- `cargo`: Rust CLI packages installed with `cargo-binstall` or another configured cargo installer.
- `sdkman`: SDKMAN candidates.
- `font`: Nerd Font family lists.
- `shell`: escape hatch for operations that are not yet worth modeling as a typed installer.

## Runtime Safety

APT package names are checked with `apt-cache show` before an install command runs. Missing packages fail the run by default. Use `--allow-missing-packages` only when you explicitly want best-effort installs.

Binary downloads install under `${HOME}/.local/share/uboot/apps/<tool>/<version>` and create symlinks under `${HOME}/.local/bin`. This avoids deleting user-managed directories from previous manual installs.

Remote script installers still exist in `shell.yaml` because the upstream bootstrap scripts use them. Treat those assets as high-trust actions. Prefer converting repeated patterns into typed installers with pinned versions and checksums over time.
