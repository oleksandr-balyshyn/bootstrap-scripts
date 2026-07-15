package bootstrap

import (
	"fmt"
	"path/filepath"
	"strings"
)

type dotfilesAsset struct {
	ID         string        `yaml:"id"`
	Repository string        `yaml:"repository"`
	Ref        string        `yaml:"ref"`
	Checkout   string        `yaml:"checkout"`
	Local      string        `yaml:"local"`
	Dotbot     string        `yaml:"dotbot"`
	Excludes   []string      `yaml:"excludes"`
	Create     []string      `yaml:"create"`
	Shell      []string      `yaml:"shell"`
	Links      []dotfileLink `yaml:"links"`
}

type dotfileLink struct {
	Target string `yaml:"target"`
	Source string `yaml:"source"`
}

func (a dotfilesAsset) commands(localBase string) ([]Command, error) {
	if a.Local == "" && (a.Repository == "" || a.Checkout == "") {
		return nil, fmt.Errorf("dotfiles asset %q requires either local or repository and checkout", a.ID)
	}
	if len(a.Links) == 0 {
		return nil, fmt.Errorf("dotfiles asset %q requires links", a.ID)
	}
	if a.Ref == "" {
		a.Ref = "main"
	}
	if a.Dotbot == "" {
		a.Dotbot = "${HOME}/.apps/dotbot/bin/dotbot"
	}
	for _, link := range a.Links {
		if link.Target == "" || link.Source == "" {
			return nil, fmt.Errorf("dotfiles asset %q has incomplete link", a.ID)
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "dotbot=%s\n", shellDouble(a.Dotbot))
	b.WriteString(`if [ ! -x "$dotbot" ]; then
  dotbot="${HOME}/.local/bin/dotbot"
fi
`)
	if a.Local != "" {
		sourceDir := a.Local
		if localBase != "" && !filepath.IsAbs(sourceDir) {
			sourceDir = filepath.Join(localBase, sourceDir)
		}
		fmt.Fprintf(&b, "source_dir=%s\n", shellDouble(sourceDir))
		b.WriteString(`if [ ! -d "$source_dir" ]; then
  echo "dotfiles source directory does not exist: $source_dir" >&2
  exit 1
fi
`)
	} else {
		fmt.Fprintf(&b, "repo_dir=%s\n", shellDouble(a.Checkout))
		fmt.Fprintf(&b, "repo_url=%s\n", shellWord(a.Repository))
		fmt.Fprintf(&b, "repo_ref=%s\n", shellWord(a.Ref))
		b.WriteString(`source_dir="$repo_dir/.files"
mkdir -p "$(dirname "$repo_dir")"
if [ -d "$repo_dir/.git" ]; then
  git -C "$repo_dir" fetch --depth 1 origin "$repo_ref"
  git -C "$repo_dir" reset --hard "origin/$repo_ref"
else
  git clone --depth 1 --branch "$repo_ref" "$repo_url" "$repo_dir"
fi
`)
		for _, exclude := range a.Excludes {
			fmt.Fprintf(&b, "rm -rf \"$source_dir/%s\"\n", escapeShellPathSegment(exclude))
		}
	}

	b.WriteString(`prepare_dotfile_target() {
  local target="$1"
  if [ -e "$target" ] && [ ! -L "$target" ]; then
    printf "Dotfile target %s already exists. Remove it and replace with a symlink? [y/N] " "$target" >&2
    local answer
    if ! read -r answer; then
      echo "No input available; keeping existing target: $target" >&2
      return 1
    fi
    case "$answer" in
      y|Y|yes|YES)
        rm -rf -- "$target"
        ;;
      *)
        echo "Keeping existing target: $target" >&2
        return 1
        ;;
    esac
  fi
}`)
	b.WriteString("\n\n")
	for _, link := range a.Links {
		fmt.Fprintf(&b, "prepare_dotfile_target %s\n", shellDouble(expandHomePath(link.Target)))
	}
	b.WriteString("\n")

	b.WriteString(`config="$(mktemp)"
trap 'rm -f "$config"' EXIT
cat > "$config" <<'DOTBOT_CONFIG'
- defaults:
    link:
      relink: true
      create: true
`)
	if len(a.Create) > 0 {
		b.WriteString("- create:\n")
		for _, dir := range a.Create {
			fmt.Fprintf(&b, "    - %s\n", dir)
		}
	}
	if len(a.Shell) > 0 {
		b.WriteString("- shell:\n")
		for _, command := range a.Shell {
			fmt.Fprintf(&b, "    - %s\n", command)
		}
	}
	b.WriteString("- link:\n")
	for _, link := range a.Links {
		fmt.Fprintf(&b, "    %s: %s\n", link.Target, link.Source)
	}
	b.WriteString("DOTBOT_CONFIG\n")
	b.WriteString(`"$dotbot" -d "$source_dir" -c "$config"` + "\n")
	return shellCommand(b.String(), false), nil
}

func (a dotfilesAsset) getID() string {
	return a.ID
}
