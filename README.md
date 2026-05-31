# Ubuntu Bootstrap

A Go/Charm TUI bootstrapper for this Ubuntu workstation. It converts the package intent from `w0rxbend/system-bootstrap` into selectable Ubuntu modules backed by external YAML config files.

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
./bin/uboot --dry-run --no-tui system-update shell terminal-cli
./bin/uboot --all --dry-run
./bin/uboot --all --yes
./bin/uboot --config ./configs --list
```

## Build

```bash
go build -o bin/uboot ./cmd/uboot
```

If system Go is not installed yet, `bootstrap.sh` also knows how to use the managed local Go symlink at `~/.local/bin/go`. It also supports the temporary local compiler path used while this VM was first bootstrapped: `~/.local/opt/go1.26.2/bin/go`.

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

The install catalog is configured in `configs/`; see [docs/configuration.md](docs/configuration.md). Go code in `internal/bootstrap` loads, validates, and compiles those installer assets into executable plans. The runner logs command output under `.bootstrap/logs/` when executing.

Module dependencies are declared with `depends_on` in `configs/modules.yaml` and are automatically included in plans. For example, selecting `cargo-packages` adds `language-installers` first, and selecting `zsh-plugins` adds `shell`.

Some Fedora packages do not have exact Ubuntu names, so the catalog uses Ubuntu equivalents where appropriate. Examples: `fd` becomes `fd-find`, `perf` becomes `linux-perf`, `wireshark-cli` becomes `tshark`, and GTK/WebKit development packages use Debian `-dev` names.

Before executing an `apt install` command, the runner checks each package with `apt-cache show`. Unavailable package names fail the run by default; pass `--allow-missing-packages` for a best-effort run.
