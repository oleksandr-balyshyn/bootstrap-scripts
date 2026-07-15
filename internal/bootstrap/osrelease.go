package bootstrap

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// defaultOSReleasePath is the standard location of the OS identification file
// on systemd-based distributions.
const defaultOSReleasePath = "/etc/os-release"

// CheckUbuntu verifies that the host is Ubuntu (or a close Debian/Ubuntu
// derivative). uboot targets Ubuntu only, so this guards against running the
// apt/snap-oriented plan on an unsupported distribution. An empty path uses the
// UBOOT_OS_RELEASE override if set, otherwise the system default.
func CheckUbuntu(path string) error {
	if path == "" {
		if override := os.Getenv("UBOOT_OS_RELEASE"); override != "" {
			path = override
		} else {
			path = defaultOSReleasePath
		}
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("cannot read %s to confirm Ubuntu: %w", path, err)
	}
	defer file.Close()
	return checkUbuntuFrom(file)
}

func checkUbuntuFrom(r io.Reader) error {
	fields := parseOSRelease(r)
	id := strings.ToLower(fields["ID"])
	if id == "ubuntu" {
		return nil
	}
	idLike := strings.ToLower(fields["ID_LIKE"])
	for _, like := range strings.Fields(idLike) {
		if like == "ubuntu" || like == "debian" {
			return nil
		}
	}

	name := fields["PRETTY_NAME"]
	if name == "" {
		name = fields["NAME"]
	}
	if name == "" {
		name = "this system"
	}
	return fmt.Errorf("uboot supports Ubuntu only, but detected %s (ID=%q); re-run with --skip-os-check to override", name, fields["ID"])
}

// parseOSRelease parses the key=value, optionally quoted, os-release format.
func parseOSRelease(r io.Reader) map[string]string {
	fields := map[string]string{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		fields[strings.TrimSpace(key)] = value
	}
	return fields
}
