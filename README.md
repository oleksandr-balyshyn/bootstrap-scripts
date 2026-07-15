# Ubuntu Bootstrap

A Go/Charm TUI bootstrapper for this Ubuntu workstation. It converts the package intent from `w0rxbend/system-bootstrap` into selectable Ubuntu modules backed by external YAML config files.

It is Ubuntu-only by design: apt/snap-oriented plans run only after an `/etc/os-release` check confirms Ubuntu (or a close Debian/Ubuntu derivative). Override with `--skip-os-check`.

Binary distributions are delegated to [`worxbend/binstaller`](https://github.com/worxbend/binstaller) and Nerd Fonts to [`worxbend/nerd-fonts-installer`](https://github.com/worxbend/nerd-fonts-installer); `uboot` only bootstraps the `binstaller` CLI and orchestrates the rest.

## Run

```bash
./bootstrap.sh
```

Keys:

```text
up/down or k/j  move
space           toggle module
a               select or clear all
d               dry-run mode
enter           run selected modules
q or esc        quit
```

Useful non-interactive commands:

```bash
./bin/uboot --list
./bin/uboot --validate
./bin/uboot --dry-run --no-tui system-update shell terminal-cli
./bin/uboot --all --dry-run
./bin/uboot --all --yes
./bin/uboot --all --yes --keep-going
./bin/uboot --config ./configs --list
./bin/uboot --all --yes --reset-state   # ignore saved progress and start over
./bin/uboot --all --yes --skip-os-check # run on a non-Ubuntu host
```

## Resume

Runs are resumable. Each command that completes successfully is recorded in a state file (default `.bootstrap/state.json`, configurable via profile policy or `--state`). If a run is interrupted — a reboot, `Ctrl-C`, or a fatal failure — re-running the same selection skips everything already completed and continues from the first unfinished command. The state file is written atomically after every successful command, so progress survives a crash mid-run.

- `--reset-state` discards saved progress and runs the full plan again.
- Package installs (apt/snap/flatpak) run **one entity per command** and are best-effort: a single failed or unavailable package is logged, skipped, and left unrecorded so it is retried next run, while the rest of the set still installs.
- Delegated tools (`binstaller`, `nerd-fonts-installer`, `dotbot`) are always invoked and manage their own resume state internally.

## Build

```bash
go build -o bin/uboot ./cmd/uboot
```

If system Go is not installed yet, `bootstrap.sh` also knows how to use the managed local Go symlink at `~/.local/bin/go`. It also supports the temporary local compiler path used while this VM was first bootstrapped: `~/.local/opt/go1.26.2/bin/go`.

## Tooling

Use `just` as the repo entrypoint:

```bash
just fmt
just check
just build
```

The `repo-tooling` bootstrap module installs `just`, YAML tooling, shell tooling, Go LSP/tools, VS Code extensions, and Neovim LSP/formatter dependencies.

## Module Sources

Fedora-derived Ubuntu package groups:

- `scripts/fedora/00-system-update.sh`
- `scripts/fedora/01-packages.sh`
- `scripts/fedora/02-extras.sh`

Root scripts also replicated, excluding only `arch`, `fedora`, and `opensuse` directories:

- `scripts/binary-dist.sh`
- `scripts/cargo-packages.sh`
- `scripts/cli-tools.sh`
- `scripts/configurations.sh`
- `scripts/flatpak-obs-plugins.sh`
- `scripts/flatpak.sh`
- `scripts/install_golang.sh`
- `scripts/nerd-fonts.p0.sh`
- `scripts/oh-my-zsh-plugins.sh`
- `scripts/sdkman-packages.sh`

Local VM history is preserved as `current-machine`:

```bash
sudo apt install -y curl 1password zsh git
sudo apt install -y snapd
sudo snap install ghostty --classic
```

## Notes

The install catalog is configured in `configs/`; see [docs/configuration.md](docs/configuration.md). Run policy lives in `configs/profile.yaml` (a binstaller-style `SystemBootstrapProfile`); the ordered plan lives in `configs/modules.yaml`. Go code in `internal/bootstrap` loads, validates, and compiles those installer assets into executable plans. The runner logs command output under `.bootstrap/logs/` when executing.

`profile.yaml` supplies defaults (`dryRun`, `continueOnError`, `requireConfirmation`, `appsDir`, `stateFile`, `distro`); any CLI flag overrides the matching policy value.

Module dependencies are declared with `depends_on` in `configs/modules.yaml` and are automatically included in plans. For example, selecting `cargo-packages` adds `language-installers` first, `binary-dists` adds `binstaller`, and `nerd-fonts` and `dotfiles` add `binary-dists` (which provides `dotbot` and the fonts installer binary).

Binary distributions are installed by `binstaller apply` against `configs/binstaller/developer-binaries.yaml`. helm and kustomize stay in `uboot`'s own `binary` installer because their upstream install scripts cannot be expressed as binstaller archive downloads.

Dotfile management is a selectable `dotfiles` module. It applies the vendored `dotfiles/` tree copied from `w0rxbend/system-bootstrap/.files` (excluding `arch+hypr` and `opensuse`) using the `dotbot` binary that the binstaller profile installs under `${HOME}/.apps`.

Some Fedora packages do not have exact Ubuntu names, so the catalog uses Ubuntu equivalents where appropriate. Examples: `fd` becomes `fd-find`, `perf` becomes `linux-perf`, `wireshark-cli` becomes `tshark`, and GTK/WebKit development packages use Debian `-dev` names.

apt/snap/flatpak installs run one entity per command. Before executing an `apt install`, the runner checks the package with `apt-cache policy`; unavailable names are reported, logged under `.bootstrap/logs/warnings/`, and skipped. A package that fails to install is also logged and skipped so the rest of the set still installs.

Command failures stop execution by default and return a non-zero exit status. Use `--keep-going` (or `continueOnError` in policy) to continue through remaining commands while still returning a failure at the end if any command failed. Per-package install failures always continue regardless.
