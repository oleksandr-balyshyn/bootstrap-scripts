set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

go_bin := env_var_or_default("GO", "go")

default:
    @just --list

fmt:
    {{ go_bin }} fmt ./...
    just --fmt --unstable
    yamlfmt .
    prettier --write "**/*.{md,json,jsonc}" --ignore-path .prettierignore

check:
    test -z "$$({{ go_bin }}fmt -l .)"
    just --fmt --check --unstable
    prettier --check "**/*.{md,json,jsonc}" --ignore-path .prettierignore
    yamllint .
    {{ go_bin }} vet ./...
    {{ go_bin }} test ./...
    {{ go_bin }} run ./cmd/uboot --config ./configs --validate
    actionlint

build:
    {{ go_bin }} build -o bin/uboot ./cmd/uboot

ci: fmt check build

install-tooling:
    sudo apt update
    sudo apt install -y just npm yamllint shellcheck shfmt golangci-lint
    {{ go_bin }} install github.com/google/yamlfmt/cmd/yamlfmt@latest
    {{ go_bin }} install github.com/rhysd/actionlint/cmd/actionlint@latest
    {{ go_bin }} install golang.org/x/tools/gopls@latest
    npm config set prefix "$$HOME/.local"
    npm install -g prettier prettier-plugin-sh yaml-language-server

dry-run module:
    ./bin/uboot --dry-run --no-tui {{ module }}
