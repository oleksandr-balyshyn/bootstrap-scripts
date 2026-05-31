package bootstrap

type Catalog struct {
	Modules []Module
}

type Module struct {
	ID          string
	Title       string
	Description string
	Source      string
	Tags        []string
	Steps       []Step
}

type Step struct {
	Name     string
	Commands []Command
}

type Command struct {
	Program string
	Args    []string
	Sudo    bool
}

func DefaultCatalog() Catalog {
	return Catalog{Modules: []Module{
		systemUpdate(),
		shellTools(),
		terminalCLI(),
		systemUtilities(),
		buildTools(),
		debugProfiling(),
		developmentLibraries(),
		mediaCodecs(),
		gpuVideo(),
		typesettingFonts(),
		desktopUI(),
		virtualization(),
		streamingCapture(),
		documentViewers(),
		vscode(),
		containers(),
		binaryDistributions(),
		languageInstallers(),
		cargoPackages(),
		flatpakApps(),
		flatpakOBSPlugins(),
		localGoToolchain(),
		nerdFonts(),
		ohMyZshPlugins(),
		sdkmanPackages(),
		userConfigurations(),
		systemConfig(),
		currentMachine(),
	}}
}

func aptInstall(name string, packages ...string) Step {
	args := append([]string{"install", "-y"}, packages...)
	return Step{Name: name, Commands: []Command{{Program: "apt", Args: args, Sudo: true}}}
}

func snapInstall(name string, snap string, flags ...string) Step {
	args := append([]string{"install", snap}, flags...)
	return Step{Name: name, Commands: []Command{{Program: "snap", Args: args, Sudo: true}}}
}

func flatpakInstall(name string, refs ...string) Step {
	commands := make([]Command, 0, len(refs))
	for _, ref := range refs {
		commands = append(commands, Command{Program: "flatpak", Args: []string{"install", "flathub", ref, "-y"}})
	}
	return Step{Name: name, Commands: commands}
}

func shell(name string, script string) Step {
	return Step{Name: name, Commands: []Command{{Program: "bash", Args: []string{"-lc", script}}}}
}

func sudoShell(name string, script string) Step {
	return Step{Name: name, Commands: []Command{{Program: "bash", Args: []string{"-lc", script}, Sudo: true}}}
}

func systemUpdate() Module {
	return Module{
		ID:          "system-update",
		Title:       "System Update & Base",
		Description: "Refresh apt metadata, upgrade packages, and install base dependencies.",
		Source:      "scripts/fedora/00-system-update.sh",
		Tags:        []string{"system", "base"},
		Steps: []Step{
			{Name: "Update package index", Commands: []Command{{Program: "apt", Args: []string{"update"}, Sudo: true}}},
			{Name: "Upgrade installed packages", Commands: []Command{{Program: "apt", Args: []string{"upgrade", "-y"}, Sudo: true}}},
			aptInstall("Base dependencies", "ca-certificates", "curl", "wget", "gnupg2", "util-linux", "fuse3", "software-properties-common", "apt-transport-https"),
			aptInstall("PipeWire audio stack", "pipewire", "pipewire-alsa", "pipewire-pulse", "pipewire-jack", "wireplumber"),
		},
	}
}

func shellTools() Module {
	return Module{
		ID:          "shell",
		Title:       "Zsh, Git, FZF & Oh My Zsh",
		Description: "Install shell basics and switch the current user to zsh.",
		Source:      "scripts/fedora/00-system-update.sh",
		Tags:        []string{"shell"},
		Steps: []Step{
			aptInstall("Shell packages", "zsh", "fzf", "git"),
			shell("Set default shell to zsh", `if [ "$SHELL" != "$(command -v zsh)" ]; then chsh -s "$(command -v zsh)"; fi`),
			shell("Install Oh My Zsh", `if [ ! -d "$HOME/.oh-my-zsh" ]; then sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" "" --unattended; fi`),
		},
	}
}

func terminalCLI() Module {
	return Module{
		ID:          "terminal-cli",
		Title:       "Terminal & CLI Tools",
		Description: "Terminal emulators and daily command-line tools mapped from Fedora packages.",
		Source:      "scripts/fedora/01-packages.sh",
		Tags:        []string{"cli"},
		Steps: []Step{
			aptInstall("CLI packages", "alacritty", "kitty", "bat", "btop", "curl", "fd-find", "fzf", "git", "htop", "wl-clipboard", "yq", "jq", "net-tools", "hyperfine", "asciinema", "gdu", "fastfetch", "ripgrep"),
			sudoShell("Add Debian compatibility symlinks", `if command -v batcat >/dev/null 2>&1 && [ ! -e /usr/local/bin/bat ]; then ln -s "$(command -v batcat)" /usr/local/bin/bat; fi
if command -v fdfind >/dev/null 2>&1 && [ ! -e /usr/local/bin/fd ]; then ln -s "$(command -v fdfind)" /usr/local/bin/fd; fi`),
		},
	}
}

func systemUtilities() Module {
	return Module{
		ID:          "system-utilities",
		Title:       "System Utilities",
		Description: "Archives, sensors, network diagnostics, secrets, containers helpers, and HTTP tools.",
		Source:      "scripts/fedora/01-packages.sh",
		Tags:        []string{"system", "tools"},
		Steps: []Step{
			aptInstall("Utility packages", "unzip", "stress", "xsensors", "fontconfig", "pipx", "libsecret-1-0", "libsecret-1-dev", "podman-toolbox", "wev", "network-manager", "foot", "tig", "buildah", "mtr", "nmap", "httpie", "ripgrep", "zoxide"),
		},
	}
}

func buildTools() Module {
	return Module{
		ID:          "build-tools",
		Title:       "Compilers & Build Tools",
		Description: "C/C++, LLVM, build systems, caching, parser generators, and firmware tooling.",
		Source:      "scripts/fedora/01-packages.sh",
		Tags:        []string{"development"},
		Steps: []Step{
			aptInstall("Compiler and build packages", "build-essential", "gcc", "g++", "clang", "clangd", "clang-format", "clang-tools", "clang-tidy", "llvm", "llvm-dev", "compiler-rt", "lld", "lldb", "make", "cmake", "meson", "ninja-build", "ccache", "flex", "bison", "gperf", "libreadline-dev", "libffi-dev", "libssl-dev", "openssl", "dfu-util", "golang-go"),
		},
	}
}

func debugProfiling() Module {
	return Module{
		ID:          "debug-profiling",
		Title:       "Debugging & Profiling",
		Description: "Debuggers, tracing, performance tools, packet tools, and protobuf compiler.",
		Source:      "scripts/fedora/01-packages.sh",
		Tags:        []string{"development"},
		Steps: []Step{
			aptInstall("Debugging packages", "gdb", "valgrind", "strace", "ltrace", "linux-perf", "tshark", "protobuf-compiler"),
		},
	}
}

func developmentLibraries() Module {
	return Module{
		ID:          "dev-libraries",
		Title:       "Development Libraries",
		Description: "GTK, GObject, WebKitGTK, X11, and Mesa development packages.",
		Source:      "scripts/fedora/01-packages.sh",
		Tags:        []string{"development", "desktop"},
		Steps: []Step{
			aptInstall("Development library packages", "libgtk-3-dev", "libgtk-4-dev", "gobject-introspection", "libgirepository1.0-dev", "libwebkitgtk-6.0-dev", "libxcursor-dev", "libxrandr-dev", "libxi-dev", "libxinerama-dev", "libxxf86vm-dev", "libgl1-mesa-dev", "libgbm-dev", "libegl1-mesa-dev", "mesa-common-dev"),
		},
	}
}

func mediaCodecs() Module {
	return Module{
		ID:          "media-codecs",
		Title:       "Media Players & Codecs",
		Description: "VLC, mpv, ffmpeg, GStreamer plugins, OpenH264, and Ubuntu restricted codecs.",
		Source:      "scripts/fedora/01-packages.sh",
		Tags:        []string{"media"},
		Steps: []Step{
			aptInstall("Media packages", "vlc", "mpv", "gstreamer1.0-vaapi", "imv", "ffmpeg", "libavcodec-dev", "libavformat-dev", "libavutil-dev", "gstreamer1.0-plugins-good", "gstreamer1.0-plugins-base", "gstreamer1.0-plugins-bad", "gstreamer1.0-plugins-ugly", "gstreamer1.0-libav", "gstreamer1.0-openh264", "ubuntu-restricted-addons"),
		},
	}
}

func gpuVideo() Module {
	return Module{
		ID:          "gpu-video",
		Title:       "GPU & Video Decode",
		Description: "VA-API tools plus AMD/Intel helpers when matching hardware is detected.",
		Source:      "scripts/fedora/01-packages.sh",
		Tags:        []string{"hardware", "media"},
		Steps: []Step{
			aptInstall("VA-API packages", "vainfo", "mesa-va-drivers"),
			shell("Install AMD GPU monitor if AMD VGA is present", `if lspci | grep -i amd | grep -i vga >/dev/null; then sudo apt install -y radeontop; fi`),
			shell("Install Intel media driver if Intel VGA is present", `if lspci | grep -i intel | grep -i vga >/dev/null; then sudo apt install -y intel-media-va-driver-non-free || sudo apt install -y intel-media-va-driver; fi`),
		},
	}
}

func typesettingFonts() Module {
	return Module{
		ID:          "typesetting-fonts",
		Title:       "Typesetting & Fonts",
		Description: "TeX Live basic/XeTeX packages and Fira Code fonts.",
		Source:      "scripts/fedora/01-packages.sh",
		Tags:        []string{"documents"},
		Steps: []Step{
			aptInstall("Typesetting packages", "texlive-base", "texlive-xetex", "fonts-firacode"),
		},
	}
}

func desktopUI() Module {
	return Module{
		ID:          "desktop-ui",
		Title:       "GNOME Desktop Tools",
		Description: "GNOME Tweaks and extension management tools.",
		Source:      "scripts/fedora/01-packages.sh",
		Tags:        []string{"desktop"},
		Steps: []Step{
			aptInstall("GNOME packages", "gnome-tweaks", "gnome-shell-extensions", "gnome-shell-extension-manager"),
		},
	}
}

func virtualization() Module {
	return Module{
		ID:          "virtualization",
		Title:       "Virtualization",
		Description: "KVM/QEMU/libvirt stack and user group setup.",
		Source:      "scripts/fedora/01-packages.sh",
		Tags:        []string{"virtualization"},
		Steps: []Step{
			aptInstall("Virtualization packages", "qemu-kvm", "libvirt-daemon-system", "libvirt-clients", "virtinst", "bridge-utils", "virt-manager"),
			shell("Add user to libvirt group", `sudo usermod -aG libvirt "$USER"`),
		},
	}
}

func streamingCapture() Module {
	return Module{
		ID:          "streaming-capture",
		Title:       "Streaming & Capture",
		Description: "Virtual camera loopback support.",
		Source:      "scripts/fedora/01-packages.sh",
		Tags:        []string{"media"},
		Steps: []Step{
			aptInstall("Capture packages", "v4l2loopback-dkms", "v4l2loopback-utils"),
		},
	}
}

func documentViewers() Module {
	return Module{
		ID:          "document-viewers",
		Title:       "Document Viewers",
		Description: "Zathura and MuPDF document viewers.",
		Source:      "scripts/fedora/01-packages.sh",
		Tags:        []string{"documents"},
		Steps: []Step{
			aptInstall("Viewer packages", "zathura", "mupdf"),
		},
	}
}

func vscode() Module {
	return Module{
		ID:          "vscode",
		Title:       "Visual Studio Code",
		Description: "Configure Microsoft apt repository and install VS Code.",
		Source:      "scripts/fedora/02-extras.sh",
		Tags:        []string{"development", "editor"},
		Steps: []Step{
			aptInstall("Repository prerequisites", "wget", "gpg", "apt-transport-https"),
			shell("Install Microsoft apt key", `wget -qO- https://packages.microsoft.com/keys/microsoft.asc | gpg --dearmor | sudo tee /etc/apt/keyrings/packages.microsoft.gpg >/dev/null`),
			sudoShell("Configure VS Code repository", `install -d -m 0755 /etc/apt/keyrings
echo "deb [arch=amd64,arm64,armhf signed-by=/etc/apt/keyrings/packages.microsoft.gpg] https://packages.microsoft.com/repos/code stable main" > /etc/apt/sources.list.d/vscode.list`),
			{Name: "Update package index", Commands: []Command{{Program: "apt", Args: []string{"update"}, Sudo: true}}},
			aptInstall("Install VS Code", "code"),
		},
	}
}

func containers() Module {
	return Module{
		ID:          "containers",
		Title:       "Containers",
		Description: "Podman, Docker-compatible Podman shim, Buildah, and Toolbox.",
		Source:      "scripts/fedora/02-extras.sh",
		Tags:        []string{"containers"},
		Steps: []Step{
			aptInstall("Container packages", "podman", "podman-docker", "buildah", "podman-toolbox"),
		},
	}
}

func systemConfig() Module {
	return Module{
		ID:          "system-config",
		Title:       "System Configuration",
		Description: "Enable network time and keep RTC in UTC.",
		Source:      "scripts/fedora/02-extras.sh",
		Tags:        []string{"system"},
		Steps: []Step{
			sudoShell("Configure time", `timedatectl set-local-rtc 0
timedatectl set-ntp true`),
		},
	}
}

func binaryDistributions() Module {
	return Module{
		ID:          "binary-dists",
		Title:       "Binary Distributions",
		Description: "Install upstream binary releases under ~/.apps: Yazi, Zig, Minikube, xplr, kind, Zellij, Helm, kubectl, Kustomize, Neovide, Neovim, Lazygit, Jujutsu, and Dotbot.",
		Source:      "scripts/binary-dist.sh",
		Tags:        []string{"development", "cli", "kubernetes"},
		Steps: []Step{
			aptInstall("Binary install prerequisites", "curl", "unzip", "tar", "xz-utils", "ca-certificates"),
			shell("Install binary distributions", `set -euo pipefail
APPS_DIR="${HOME}/.apps"
LAZYGIT_VERSION=0.61.0
KIND_VERSION=0.31.0
ZELLIJ_VERSION=0.44.1
JJ_VERSION=0.40.0
DOTBOT_VERSION=1.24.0
ZIG_VERSION=0.15.2
YAZI_VERSION=v26.5.6
mkdir -p "$APPS_DIR"

rm -rf "$APPS_DIR/yazi"
mkdir -p "$APPS_DIR/yazi/bin"
curl -L -o "$APPS_DIR/yazi/yazi.zip" "https://github.com/sxyazi/yazi/releases/download/${YAZI_VERSION}/yazi-x86_64-unknown-linux-gnu.zip"
unzip -q "$APPS_DIR/yazi/yazi.zip" -d "$APPS_DIR/yazi"
mv -f "$APPS_DIR"/yazi/yazi-x86_64-unknown-linux-gnu/* "$APPS_DIR/yazi/bin/"
chmod +x "$APPS_DIR/yazi/bin/yazi" "$APPS_DIR/yazi/bin/ya"
rm -f "$APPS_DIR/yazi/yazi.zip"

rm -rf "$APPS_DIR/zig"
mkdir -p "$APPS_DIR/zig/bin"
curl -L -o "$APPS_DIR/zig/zig.tar.xz" "https://ziglang.org/download/${ZIG_VERSION}/zig-x86_64-linux-${ZIG_VERSION}.tar.xz"
tar -xf "$APPS_DIR/zig/zig.tar.xz" -C "$APPS_DIR/zig/bin/"
ln -sf "$APPS_DIR"/zig/bin/zig-x86_64-*/zig "$APPS_DIR/zig/bin/zig"
rm -f "$APPS_DIR/zig/zig.tar.xz"

rm -rf "$APPS_DIR/minikube"
mkdir -p "$APPS_DIR/minikube/bin"
curl -L -o "$APPS_DIR/minikube/bin/minikube" https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
chmod +x "$APPS_DIR/minikube/bin/minikube"

rm -rf "$APPS_DIR/xplr"
mkdir -p "$APPS_DIR/xplr/bin"
curl -L -o "$APPS_DIR/xplr/xplr-linux.tar.gz" https://github.com/sayanarijit/xplr/releases/latest/download/xplr-linux.tar.gz
tar -zxf "$APPS_DIR/xplr/xplr-linux.tar.gz" -C "$APPS_DIR/xplr/bin/"
chmod +x "$APPS_DIR/xplr/bin/xplr"
rm -f "$APPS_DIR/xplr/xplr-linux.tar.gz"

rm -rf "$APPS_DIR/kind"
mkdir -p "$APPS_DIR/kind/bin"
curl -Lo "$APPS_DIR/kind/bin/kind" "https://kind.sigs.k8s.io/dl/v${KIND_VERSION}/kind-linux-amd64"
chmod +x "$APPS_DIR/kind/bin/kind"

rm -rf "$APPS_DIR/zellij"
mkdir -p "$APPS_DIR/zellij/bin"
curl -Lo "$APPS_DIR/zellij/zellij.tar.gz" "https://github.com/zellij-org/zellij/releases/download/v${ZELLIJ_VERSION}/zellij-x86_64-unknown-linux-musl.tar.gz"
tar -zxf "$APPS_DIR/zellij/zellij.tar.gz" -C "$APPS_DIR/zellij/bin/"
chmod +x "$APPS_DIR/zellij/bin/zellij"
rm -f "$APPS_DIR/zellij/zellij.tar.gz"

rm -rf "$APPS_DIR/helm"
mkdir -p "$APPS_DIR/helm/bin"
curl -fsSL -o "$APPS_DIR/helm/get_helm.sh" https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
chmod +x "$APPS_DIR/helm/get_helm.sh"
USE_SUDO=false HELM_INSTALL_DIR="$APPS_DIR/helm/bin" "$APPS_DIR/helm/get_helm.sh"
rm -f "$APPS_DIR/helm/get_helm.sh"

rm -rf "$APPS_DIR/kubectl"
mkdir -p "$APPS_DIR/kubectl/bin"
curl -L -o "$APPS_DIR/kubectl/bin/kubectl" "https://dl.k8s.io/release/$(curl -Ls https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x "$APPS_DIR/kubectl/bin/kubectl"

rm -rf "$APPS_DIR/kustomize"
mkdir -p "$APPS_DIR/kustomize/bin"
curl -L -o "$APPS_DIR/kustomize/install_kustomize.sh" https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh
chmod +x "$APPS_DIR/kustomize/install_kustomize.sh"
"$APPS_DIR/kustomize/install_kustomize.sh" "$APPS_DIR/kustomize/bin"
rm -f "$APPS_DIR/kustomize/install_kustomize.sh"

rm -rf "$APPS_DIR/neovide"
mkdir -p "$APPS_DIR/neovide/bin"
curl -L -o "$APPS_DIR/neovide/bin/neovide" https://github.com/neovide/neovide/releases/latest/download/neovide.AppImage
chmod +x "$APPS_DIR/neovide/bin/neovide"

rm -rf "$APPS_DIR/neovim"
mkdir -p "$APPS_DIR/neovim/bin"
curl -L -o "$APPS_DIR/neovim/neovim.tar.gz" https://github.com/neovim/neovim/releases/latest/download/nvim-linux-x86_64.tar.gz
tar -zxf "$APPS_DIR/neovim/neovim.tar.gz" -C "$APPS_DIR/neovim/"
mv -f "$APPS_DIR"/neovim/nvim-linux-x86_64/* "$APPS_DIR/neovim/"
chmod +x "$APPS_DIR/neovim/bin/nvim"
ln -sf "$APPS_DIR/neovim/bin/nvim" "$APPS_DIR/neovim/bin/neovim"
ln -sf "$APPS_DIR/neovim/bin/nvim" "$APPS_DIR/neovim/bin/vim"
sudo ln -sf "$APPS_DIR/neovim/bin/nvim" /usr/local/bin/neovim
sudo ln -sf "$APPS_DIR/neovim/bin/nvim" /usr/local/bin/vim
sudo ln -sf "$APPS_DIR/neovim/bin/nvim" /usr/local/bin/nvim
rm -rf "$APPS_DIR/neovim/neovim.tar.gz" "$APPS_DIR/neovim/nvim-linux-x86_64"

mkdir -p "$APPS_DIR/lazygit/bin"
curl -L -o "$APPS_DIR/lazygit/lazygit.tar.gz" "https://github.com/jesseduffield/lazygit/releases/download/v${LAZYGIT_VERSION}/lazygit_${LAZYGIT_VERSION}_Linux_x86_64.tar.gz"
tar -zxf "$APPS_DIR/lazygit/lazygit.tar.gz" -C "$APPS_DIR/lazygit/bin/"
chmod +x "$APPS_DIR/lazygit/bin/lazygit"
ln -sf "$APPS_DIR/lazygit/bin/lazygit" "$APPS_DIR/lazygit/bin/lzg"
rm -f "$APPS_DIR/lazygit/lazygit.tar.gz"

mkdir -p "$APPS_DIR/jujutsu/bin"
curl -L -o "$APPS_DIR/jujutsu/jj.tar.gz" "https://github.com/jj-vcs/jj/releases/download/v${JJ_VERSION}/jj-v${JJ_VERSION}-x86_64-unknown-linux-musl.tar.gz"
tar -zxf "$APPS_DIR/jujutsu/jj.tar.gz" -C "$APPS_DIR/jujutsu/bin/"
chmod +x "$APPS_DIR/jujutsu/bin/jj"
ln -sf "$APPS_DIR/jujutsu/bin/jj" "$APPS_DIR/jujutsu/bin/jj-scm"
ln -sf "$APPS_DIR/jujutsu/bin/jj" "$APPS_DIR/jujutsu/bin/jujutsu"
rm -f "$APPS_DIR/jujutsu/jj.tar.gz"

rm -rf "$APPS_DIR/dotbot"
mkdir -p "$APPS_DIR/dotbot/bin"
curl -L -o "$APPS_DIR/dotbot/dotbot.tar.gz" "https://github.com/anishathalye/dotbot/releases/download/v${DOTBOT_VERSION}/dotbot-linux-x64.tar.gz"
tar -zxf "$APPS_DIR/dotbot/dotbot.tar.gz" -C "$APPS_DIR/dotbot/bin/"
chmod +x "$APPS_DIR/dotbot/bin/dotbot"
rm -f "$APPS_DIR/dotbot/dotbot.tar.gz"`),
		},
	}
}

func languageInstallers() Module {
	return Module{
		ID:          "language-installers",
		Title:       "Language Toolchain Installers",
		Description: "Run upstream installers for Rust, SDKMAN, NVM, pyenv, Poetry, Starship, pnpm, Juliaup, dotenvx, Miniforge, uv, and cargo-binstall.",
		Source:      "scripts/cli-tools.sh",
		Tags:        []string{"development", "languages"},
		Steps: []Step{
			aptInstall("Installer prerequisites", "curl", "python3", "build-essential", "git", "ca-certificates"),
			shell("Run language installers", `set -euo pipefail
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
curl -s "https://get.sdkman.io" | bash
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.4/install.sh | bash
curl https://pyenv.run | bash
curl -sSL https://install.python-poetry.org | python3 -
curl -sS https://starship.rs/install.sh | sh -s -- -y
curl -fsSL https://get.pnpm.io/install.sh | sh -
curl -fsSL https://install.julialang.org | sh -s -- -y
curl -sfS 'https://dotenvx.sh?directory=/home/worxbend/.local/bin' | sh
curl -L -O "https://github.com/conda-forge/miniforge/releases/latest/download/Miniforge3-$(uname)-$(uname -m).sh"
bash "Miniforge3-$(uname)-$(uname -m).sh" -b -p "$HOME/miniforge3"
rm -f "Miniforge3-$(uname)-$(uname -m).sh"
curl -LsSf https://astral.sh/uv/install.sh | sh
curl -L --proto '=https' --tlsv1.2 -sSf https://raw.githubusercontent.com/cargo-bins/cargo-binstall/main/install-from-binstall-release.sh | bash`),
		},
	}
}

func cargoPackages() Module {
	return Module{
		ID:          "cargo-packages",
		Title:       "Cargo CLI Packages",
		Description: "Install Rust CLI packages listed in cargo-packages.sh with cargo-binstall.",
		Source:      "scripts/cargo-packages.sh",
		Tags:        []string{"rust", "cli"},
		Steps: []Step{
			shell("Install cargo packages", `set -euo pipefail
. "$HOME/.cargo/env"
cargo-binstall -y eza lsd fd-find ripgrep bingrep hx just sd procs du-dust gping tree-sitter-cli macchina multibg-wayland bottom broot`),
		},
	}
}

func flatpakApps() Module {
	return Module{
		ID:          "flatpak-apps",
		Title:       "Flatpak Desktop Apps",
		Description: "Install Flatpak, enable Flathub, and install the desktop applications from flatpak.sh.",
		Source:      "scripts/flatpak.sh",
		Tags:        []string{"desktop", "flatpak"},
		Steps: []Step{
			aptInstall("Flatpak prerequisites", "flatpak", "gnome-software-plugin-flatpak"),
			{Name: "Enable Flathub", Commands: []Command{{Program: "flatpak", Args: []string{"remote-add", "--if-not-exists", "flathub", "https://flathub.org/repo/flathub.flatpakrepo"}, Sudo: true}}},
			flatpakInstall("Browsers and editors", "io.gitlab.librewolf-community", "com.google.Chrome", "com.brave.Browser", "app.zen_browser.zen", "dev.zed.Zed"),
			flatpakInstall("Communication", "org.telegram.desktop", "com.discordapp.Discord", "org.zulip.Zulip", "dev.vencord.Vesktop"),
			flatpakInstall("Media and audio", "com.spotify.Client", "org.audacityteam.Audacity", "org.kde.audiotube", "io.github.hrkfdn.ncspot", "org.gnome.Decibels", "io.bassi.Amberol", "com.github.neithern.g4music"),
			flatpakInstall("Video and graphics creation", "org.kde.kdenlive", "org.inkscape.Inkscape", "org.kde.krita", "org.blender.Blender", "org.freecad.FreeCAD", "org.godotengine.Godot", "org.librecad.librecad", "com.bambulab.BambuStudio"),
			flatpakInstall("Writing and text tools", "io.gitlab.liferooter.TextPieces", "org.gnome.gitlab.somas.Apostrophe", "org.gnome.gitlab.ilhooq.Bookup", "io.github.diegopvlk.Dosage", "page.codeberg.censor.Censor", "com.logseq.Logseq"),
			flatpakInstall("Terminal and shell", "app.devsuite.Ptyxis", "org.wezfurlong.wezterm"),
			flatpakInstall("System utilities", "com.mattjakeman.ExtensionManager", "com.github.tchx84.Flatseal", "io.github.giantpinkrobots.flatsweep", "io.github.flattool.Warehouse", "net.nokyan.Resources", "page.tesk.Refine", "io.missioncenter.MissionCenter", "be.alexandervanhee.gradia", "io.github.mrvladus.List", "com.belmoussaoui.Authenticator", "org.gnome.Polari", "org.gnome.dspy", "io.github.swordpuffin.rewaita", "org.gnome.design.Emblem", "org.mozilla.vpn", "io.github.zingytomato.netpeek", "com.oguzhaninan.Stacer"),
			flatpakInstall("Productivity", "com.pojtinger.felicitas.Sessions", "com.rafaelmardojai.Blanket", "io.github.nozwock.Packet", "org.localsend.localsend_app"),
			flatpakInstall("Dev tools", "rest.insomnia.Insomnia", "com.vscodium.codium"),
			flatpakInstall("News, health, and miscellaneous", "io.gitlab.news_flash.NewsFlash", "dev.Cogitri.Health", "io.github.nokse22.Exhibit"),
		},
	}
}

func flatpakOBSPlugins() Module {
	return Module{
		ID:          "flatpak-obs",
		Title:       "OBS Studio Flatpak Plugins",
		Description: "Install OBS Studio and the OBS Flatpak plugins from flatpak-obs-plugins.sh.",
		Source:      "scripts/flatpak-obs-plugins.sh",
		Tags:        []string{"streaming", "flatpak"},
		Steps: []Step{
			aptInstall("Flatpak prerequisites", "flatpak"),
			{Name: "Enable Flathub", Commands: []Command{{Program: "flatpak", Args: []string{"remote-add", "--if-not-exists", "flathub", "https://flathub.org/repo/flathub.flatpakrepo"}, Sudo: true}}},
			flatpakInstall("OBS Studio and plugins", "com.obsproject.Studio", "com.obsproject.Studio.Plugin.DroidCam", "com.obsproject.Studio.Plugin.BackgroundRemoval", "com.obsproject.Studio.Plugin.Gstreamer", "com.obsproject.Studio.Plugin.GStreamerVaapi", "com.obsproject.Studio.Plugin.OBSPWVideo", "com.obsproject.Studio.Plugin.AitumMultistream", "com.obsproject.Studio.Plugin.SceneSwitcher", "com.obsproject.Studio.Plugin.WaylandHotkeys", "com.obsproject.Studio.Plugin.CompositeBlur", "com.obsproject.Studio.Plugin.AdvancedMasks", "com.obsproject.Studio.Plugin._3DEffect", "com.obsproject.Studio.Plugin.Shaderfilter", "com.obsproject.Studio.Plugin.TransitionTable"),
		},
	}
}

func localGoToolchain() Module {
	return Module{
		ID:          "local-go",
		Title:       "Local Go Toolchain",
		Description: "Install Go 1.26.2 under ~/.go and create ~/.go-workspace, matching install_golang.sh.",
		Source:      "scripts/install_golang.sh",
		Tags:        []string{"go", "development"},
		Steps: []Step{
			aptInstall("Go install prerequisites", "curl", "tar"),
			shell("Install local Go", `set -euo pipefail
GOPATH="${HOME}/.go"
GO_VERSION=1.26.2
rm -rf "$GOPATH"
mkdir -p "$GOPATH"
curl -L -o "$GOPATH/go.tar.gz" "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
tar -zxf "$GOPATH/go.tar.gz" -C "$GOPATH"
mv -f "$GOPATH"/go/* "$GOPATH/"
rm -rf "$GOPATH/go" "$GOPATH/go.tar.gz"
mkdir -p "${HOME}/.go-workspace"`),
		},
	}
}

func nerdFonts() Module {
	return Module{
		ID:          "nerd-fonts",
		Title:       "Nerd Fonts",
		Description: "Install the Nerd Font families listed in nerd-fonts.p0.sh.",
		Source:      "scripts/nerd-fonts.p0.sh",
		Tags:        []string{"fonts", "desktop"},
		Steps: []Step{
			aptInstall("Font install prerequisites", "git", "fontconfig"),
			shell("Install Nerd Fonts", `set -euo pipefail
FONT_DIR="${HOME}/.fonts"
mkdir -p "$FONT_DIR"
workdir="$(mktemp -d)"
git clone --depth 1 --filter=blob:none https://github.com/ryanoasis/nerd-fonts "$workdir/nerd-fonts"
cd "$workdir/nerd-fonts"
for font in JetBrainsMono MPlus Terminus FantasqueSansMono Noto Hack HeavyData 3270 FiraCode LiberationMono RobotoMono Mononoki Ubuntu DroidSansMono Monoid SpaceMono SourceCodePro ComicShannsMono NerdFontsSymbolsOnly DaddyTimeMono UbuntuMono Meslo FiraMono CodeNewRoman CascadiaCode Hasklig DejaVuSansMono ZedMono Inconsolata Hermit CommitMono Agave GeistMono Monaspace ShareTechMono Recursive D2Coding EnvyCodeR IosevkaTerm Lekton Lilex VictorMono; do
  ./install.sh -U "$font"
done
fc-cache -vf
rm -rf "$workdir"`),
		},
	}
}

func ohMyZshPlugins() Module {
	return Module{
		ID:          "zsh-plugins",
		Title:       "Oh My Zsh Plugins",
		Description: "Install zsh-syntax-highlighting, zsh-autosuggestions, and zsh-history-substring-search.",
		Source:      "scripts/oh-my-zsh-plugins.sh",
		Tags:        []string{"shell"},
		Steps: []Step{
			aptInstall("Plugin prerequisites", "git"),
			shell("Clone plugins", `set -euo pipefail
custom="${ZSH_CUSTOM:-$HOME/.oh-my-zsh/custom}"
mkdir -p "$custom/plugins"
[ -d "$custom/plugins/zsh-syntax-highlighting" ] || git clone https://github.com/zsh-users/zsh-syntax-highlighting.git "$custom/plugins/zsh-syntax-highlighting"
[ -d "$custom/plugins/zsh-autosuggestions" ] || git clone https://github.com/zsh-users/zsh-autosuggestions "$custom/plugins/zsh-autosuggestions"
[ -d "$custom/plugins/zsh-history-substring-search" ] || git clone https://github.com/zsh-users/zsh-history-substring-search "$custom/plugins/zsh-history-substring-search"`),
		},
	}
}

func sdkmanPackages() Module {
	return Module{
		ID:          "sdkman-packages",
		Title:       "SDKMAN Packages",
		Description: "Install SDKMAN and Java, Gradle, Maven, sbt, Scala, Micronaut, Vert.x, and VisualVM.",
		Source:      "scripts/sdkman-packages.sh",
		Tags:        []string{"jvm", "development"},
		Steps: []Step{
			aptInstall("SDKMAN prerequisites", "curl", "zip", "unzip"),
			shell("Install SDKMAN packages", `set -euo pipefail
curl -s "https://get.sdkman.io" | bash
. "$HOME/.sdkman/bin/sdkman-init.sh"
sdk version
sdk install java || true
sdk install gradle || true
sdk install maven || true
sdk install sbt || true
sdk install scala || true
sdk install micronaut || true
sdk install vertx || true
sdk install visualvm || true`),
		},
	}
}

func userConfigurations() Module {
	return Module{
		ID:          "user-config",
		Title:       "User Configuration",
		Description: "Clone tmux plugin manager, set global git identity/options, and configure time.",
		Source:      "scripts/configurations.sh",
		Tags:        []string{"configuration"},
		Steps: []Step{
			aptInstall("Configuration prerequisites", "git", "tmux"),
			shell("Clone tmux plugin manager", `if [ ! -d "$HOME/.tmux/plugins/tpm" ]; then git clone https://github.com/tmux-plugins/tpm "$HOME/.tmux/plugins/tpm"; fi`),
			shell("Configure git", `git config --global user.email "balyszyn@gmail.com"
git config --global user.name "w0rxbend"
git config --global pull.rebase true
git config --global init.defaultBranch main
git config --global core.autocrlf input`),
			sudoShell("Configure time", `timedatectl set-local-rtc 0
timedatectl set-ntp true`),
		},
	}
}

func currentMachine() Module {
	return Module{
		ID:          "current-machine",
		Title:       "Current VM Installs",
		Description: "Packages reconstructed from this VM's apt/snap history before the TUI project was created.",
		Source:      "local apt/snap history",
		Tags:        []string{"local"},
		Steps: []Step{
			aptInstall("Current apt packages", "curl", "1password", "zsh", "git"),
			snapInstall("Current snap packages", "ghostty", "--classic"),
		},
	}
}
