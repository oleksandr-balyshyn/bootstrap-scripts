package bootstrap

import (
	"encoding/hex"
	"fmt"
	"strings"
)

type binaryAsset struct {
	ID    string       `yaml:"id"`
	Root  string       `yaml:"root"`
	Links string       `yaml:"links"`
	Tools []binaryTool `yaml:"tools"`
}

type binaryTool struct {
	Name     string   `yaml:"name"`
	Version  string   `yaml:"version"`
	URL      string   `yaml:"url"`
	Archive  string   `yaml:"archive"`
	SHA256   string   `yaml:"sha256"`
	Binaries []string `yaml:"binaries"`
	Aliases  []string `yaml:"aliases"`
}

func (a binaryAsset) commands() ([]Command, error) {
	if a.Root == "" || len(a.Tools) == 0 {
		return nil, fmt.Errorf("binary asset %q requires root and tools", a.ID)
	}
	if a.Links == "" {
		a.Links = "~/.local/bin"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "root=%s\nlinks=%s\nmkdir -p \"$root\" \"$links\"\n", shellDouble(a.Root), shellDouble(a.Links))
	for _, tool := range a.Tools {
		if tool.Name == "" || tool.URL == "" || len(tool.Binaries) == 0 {
			return nil, fmt.Errorf("binary asset %q has incomplete tool entry", a.ID)
		}
		if err := writeBinaryInstall(&b, tool); err != nil {
			return nil, fmt.Errorf("binary asset %q tool %q: %w", a.ID, tool.Name, err)
		}
	}
	return binaryShellCommand(b.String()), nil
}

func (a binaryAsset) getID() string { return a.ID }

func writeBinaryInstall(b *strings.Builder, tool binaryTool) error {
	if tool.SHA256 != "" {
		if len(tool.SHA256) != 64 || tool.SHA256 != strings.ToLower(tool.SHA256) {
			return fmt.Errorf("sha256 must be 64 lowercase hex characters")
		}
		if _, err := hex.DecodeString(tool.SHA256); err != nil {
			return fmt.Errorf("sha256 must be 64 lowercase hex characters")
		}
	}

	url := shellWord(tool.URL)
	sha256 := shellWord(tool.SHA256)
	fmt.Fprintf(b, "\n# %s %s\n", tool.Name, tool.Version)
	fmt.Fprintf(b, "tool_dir=\"$root/%s/%s\"\n", tool.Name, tool.Version)
	fmt.Fprintln(b, "stage=\"$(mktemp -d)\"")
	fmt.Fprintln(b, "mkdir -p \"$tool_dir\"")
	switch tool.Archive {
	case "none":
		fmt.Fprintf(b, "curl -L -o \"$stage/%s\" %s\nverify_sha256 \"$stage/%s\" %s\ncp \"$stage/%s\" \"$tool_dir/%s\"\nchmod +x \"$tool_dir/%s\"\n", tool.Binaries[0], url, tool.Binaries[0], sha256, tool.Binaries[0], tool.Binaries[0], tool.Binaries[0])
	case "zip":
		fmt.Fprintf(b, "curl -L -o \"$stage/archive.zip\" %s\nverify_sha256 \"$stage/archive.zip\" %s\nunzip -q \"$stage/archive.zip\" -d \"$stage/out\"\nfind \"$stage/out\" -type f -perm /111 -exec cp {} \"$tool_dir/\" \\;\n", url, sha256)
	case "tar.gz":
		fmt.Fprintf(b, "curl -L -o \"$stage/archive.tar.gz\" %s\nverify_sha256 \"$stage/archive.tar.gz\" %s\ntar -zxf \"$stage/archive.tar.gz\" -C \"$stage\"\nfind \"$stage\" -type f -perm /111 -exec cp {} \"$tool_dir/\" \\;\n", url, sha256)
	case "tar.xz":
		fmt.Fprintf(b, "curl -L -o \"$stage/archive.tar.xz\" %s\nverify_sha256 \"$stage/archive.tar.xz\" %s\ntar -xf \"$stage/archive.tar.xz\" -C \"$stage\"\nfind \"$stage\" -type f -perm /111 -exec cp {} \"$tool_dir/\" \\;\n", url, sha256)
	case "kubectl-stable":
		if tool.SHA256 != "" {
			return fmt.Errorf("sha256 is not supported with kubectl-stable archive")
		}
		fmt.Fprintln(b, `kubectl_version="$(curl -Ls https://dl.k8s.io/release/stable.txt)"`)
		fmt.Fprintln(b, `curl -L -o "$tool_dir/kubectl" "https://dl.k8s.io/release/${kubectl_version}/bin/linux/amd64/kubectl"`)
		fmt.Fprintln(b, `chmod +x "$tool_dir/kubectl"`)
	case "helm-script":
		fmt.Fprintf(b, "curl -fsSL -o \"$stage/get_helm.sh\" %s\n", url)
		fmt.Fprintf(b, "verify_sha256 \"$stage/get_helm.sh\" %s\n", sha256)
		fmt.Fprintln(b, `chmod +x "$stage/get_helm.sh"`)
		fmt.Fprintln(b, `PATH="$tool_dir:$PATH" USE_SUDO=false HELM_INSTALL_DIR="$tool_dir" "$stage/get_helm.sh"`)
	case "kustomize-script":
		fmt.Fprintf(b, "curl -fsSL -o \"$stage/install_kustomize.sh\" %s\n", url)
		fmt.Fprintf(b, "verify_sha256 \"$stage/install_kustomize.sh\" %s\n", sha256)
		fmt.Fprintln(b, `chmod +x "$stage/install_kustomize.sh"`)
		fmt.Fprintln(b, `"$stage/install_kustomize.sh" "$tool_dir"`)
	default:
		return fmt.Errorf("unsupported archive type %q", tool.Archive)
	}
	for _, bin := range tool.Binaries {
		fmt.Fprintf(b, "if [ -x \"$tool_dir/%s\" ]; then ln -sfn \"$tool_dir/%s\" \"$links/%s\"; fi\n", bin, bin, bin)
	}
	for _, alias := range tool.Aliases {
		if len(tool.Binaries) > 0 {
			fmt.Fprintf(b, "if [ -x \"$tool_dir/%s\" ]; then ln -sfn \"$tool_dir/%s\" \"$links/%s\"; fi\n", tool.Binaries[0], tool.Binaries[0], alias)
		}
	}
	fmt.Fprintln(b, "rm -rf \"$stage\"")
	return nil
}
