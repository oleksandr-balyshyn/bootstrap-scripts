package bootstrap

import "strings"

func shellCommand(script string, sudo bool) []Command {
	return []Command{{Program: "bash", Args: []string{"-lc", bootstrapShellPrelude() + script}, Sudo: sudo}}
}

func binaryShellCommand(script string) []Command {
	return []Command{{Program: "bash", Args: []string{"-lc", bootstrapShellPrelude() + checksumShellPrelude() + script}}}
}

func bootstrapShellPrelude() string {
	return `set -euo pipefail

source_shell_file_if_exists() {
  local file="$1"
  if [ -s "$file" ]; then
    set +u
    # shellcheck disable=SC1090
    . "$file" >/dev/null 2>&1 || true
    set -u
  fi
}

source_env_file_if_exists() {
  local file="$1"
  if [ -s "$file" ]; then
    # shellcheck disable=SC1090
    set +u
    . "$file"
    set -u
  fi
}

prepend_path() {
  local dir="$1"
  if [ -d "$dir" ]; then
    case ":$PATH:" in
      *":$dir:"*) ;;
      *) PATH="$dir:$PATH" ;;
    esac
  fi
}

merge_zsh_path() {
  if command -v zsh >/dev/null 2>&1 && [ -s "$HOME/.zshrc" ]; then
    local zsh_path
    zsh_path="$(zsh -lc 'print -r -- "$PATH"' 2>/dev/null || true)"
    if [ -n "$zsh_path" ]; then
      PATH="$zsh_path:$PATH"
    fi
  fi
}

source_bootstrap_env() {
  source_shell_file_if_exists "$HOME/.profile"
  source_shell_file_if_exists "$HOME/.bash_profile"
  source_shell_file_if_exists "$HOME/.bashrc"
  merge_zsh_path
  prepend_path "$HOME/.local/bin"
  prepend_path "$HOME/go/bin"
  prepend_path "$HOME/.go/bin"
  prepend_path "$HOME/.cargo/bin"
  prepend_path "$HOME/.local/share/uboot/toolchains/go/1.26.2/bin"
  prepend_path "$HOME/.local/share/uboot/apps/dotbot/1.24.0"
  source_env_file_if_exists "$HOME/.cargo/env"
  source_env_file_if_exists "$HOME/.sdkman/bin/sdkman-init.sh"
  export PATH
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "required command not available after sourcing bootstrap environment: $1" >&2
    return 127
  fi
}

source_bootstrap_env
`
}

func checksumShellPrelude() string {
	return `
verify_sha256() {
  local file="$1"
  local expected="$2"
  if [ -z "$expected" ]; then
    return 0
  fi
  if command -v sha256sum >/dev/null 2>&1; then
    printf '%s  %s\n' "$expected" "$file" | sha256sum -c -
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    printf '%s  %s\n' "$expected" "$file" | shasum -a 256 -c -
    return
  fi
  echo "sha256 verification requested but neither sha256sum nor shasum is available" >&2
  return 127
}

`
}

func shellWords(values []string) []string {
	words := make([]string, 0, len(values))
	for _, value := range values {
		words = append(words, shellWord(value))
	}
	return words
}

func shellWord(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func shellDouble(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

func expandHomePath(value string) string {
	if value == "~" {
		return "${HOME}"
	}
	if strings.HasPrefix(value, "~/") {
		return "${HOME}/" + strings.TrimPrefix(value, "~/")
	}
	return value
}

func escapeShellPathSegment(value string) string {
	return strings.ReplaceAll(value, `"`, `\"`)
}
