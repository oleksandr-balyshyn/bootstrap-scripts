#!/usr/bin/env bash
set -euo pipefail

source_file_if_exists() {
  local file="$1"
  if [ -s "$file" ]; then
    set +u
    # shellcheck disable=SC1090
    . "$file" >/dev/null 2>&1 || true
    set -u
  fi
}

run_if_available() {
  local command_name="$1"
  shift
  if command -v "$command_name" >/dev/null 2>&1; then
    "$command_name" "$@"
  else
    echo "Skipping $command_name; command is not installed."
  fi
}

echo "Initializing user environment..."
source_file_if_exists "$HOME/.profile"
source_file_if_exists "$HOME/.bashrc"
source_file_if_exists "$HOME/.zshrc"
source_file_if_exists "$HOME/.cargo/env"
source_file_if_exists "$HOME/.sdkman/bin/sdkman-init.sh"

echo
echo "## shell: ${SHELL:-unknown} ##"
echo "System info: $(uname -a)"
echo
echo "------------------------"
echo "-  Updating system...  -"
echo "------------------------"

echo
echo "Updating Ubuntu packages via apt..."
sudo apt update
sudo apt upgrade -y
sudo apt full-upgrade -y
sudo apt autoremove -y

echo
echo "Refreshing snap packages..."
if command -v snap >/dev/null 2>&1; then
  sudo snap refresh
else
  echo "Skipping snap; command is not installed."
fi

echo
echo "Updating Flatpak packages..."
if command -v flatpak >/dev/null 2>&1; then
  flatpak update -y
else
  echo "Skipping flatpak; command is not installed."
fi

echo
echo "Updating Rust toolchain..."
run_if_available rustup update

echo
echo "Updating Julia toolchain..."
run_if_available juliaup update

echo
echo "Updating SDKMAN..."
if command -v sdk >/dev/null 2>&1; then
  sdk update
  sdk upgrade
else
  echo "Skipping SDKMAN; command is not available."
fi

echo
echo "Updating NVM default Node.js..."
if command -v nvm >/dev/null 2>&1; then
  nvm install node --reinstall-packages-from=current --latest-npm
else
  echo "Skipping nvm; command is not available."
fi

echo
echo "Updating Miniforge packages..."
if command -v mamba >/dev/null 2>&1; then
  mamba update --all -y
elif command -v conda >/dev/null 2>&1; then
  conda update --all -y
else
  echo "Skipping Miniforge; mamba/conda is not available."
fi

echo
echo "Updating astral uv..."
run_if_available uv self update
