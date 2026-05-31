#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [[ -x "$repo_dir/bin/uboot" ]]; then
  exec "$repo_dir/bin/uboot" --config "$repo_dir/configs" "$@"
fi

if command -v go >/dev/null 2>&1; then
  cd "$repo_dir"
  exec go run ./cmd/uboot --config "$repo_dir/configs" "$@"
fi

if [[ -x "$HOME/.local/bin/go" ]]; then
  cd "$repo_dir"
  exec "$HOME/.local/bin/go" run ./cmd/uboot --config "$repo_dir/configs" "$@"
fi

if [[ -x "$HOME/.local/opt/go1.26.2/bin/go" ]]; then
  cd "$repo_dir"
  exec "$HOME/.local/opt/go1.26.2/bin/go" run ./cmd/uboot --config "$repo_dir/configs" "$@"
fi

cat >&2 <<'EOF'
Go is required to run this bootstrap project.

Install one of:
  sudo apt install -y golang-go
  sudo snap install go --classic
EOF
exit 1
