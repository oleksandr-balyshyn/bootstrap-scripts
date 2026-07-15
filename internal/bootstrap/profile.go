package bootstrap

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Profile is the binstaller-style envelope for a whole-system bootstrap run. It
// carries run policy and metadata; the ordered plan of work lives in
// modules.yaml. A run without a profile file uses DefaultPolicy.
type Profile struct {
	APIVersion string
	Kind       string
	Metadata   ProfileMetadata
	Policy     Policy
	Vars       map[string]string
}

// ProfileMetadata mirrors the metadata block of the binstaller profile format.
type ProfileMetadata struct {
	Name        string
	Labels      map[string]string
	Annotations map[string]string
}

// Policy holds runtime defaults. CLI flags override any value set here.
type Policy struct {
	DryRun              bool
	ContinueOnError     bool
	RequireConfirmation bool
	AppsDir             string
	StateFile           string
	Distro              string
}

// DefaultPolicy is used when no profile.yaml is present. It matches uboot's
// historical CLI defaults: confirm before running, stop on the first failure,
// and track resume state under .bootstrap/state.json.
func DefaultPolicy() Policy {
	return Policy{
		RequireConfirmation: true,
		AppsDir:             "${HOME}/.apps",
		StateFile:           filepath.Join(".bootstrap", "state.json"),
		Distro:              "ubuntu",
	}
}

type profileFile struct {
	APIVersion string          `yaml:"apiVersion"`
	Kind       string          `yaml:"kind"`
	Metadata   metadataConfig  `yaml:"metadata"`
	Spec       profileSpecFile `yaml:"spec"`
}

type metadataConfig struct {
	Name        string            `yaml:"name"`
	Labels      map[string]string `yaml:"labels"`
	Annotations map[string]string `yaml:"annotations"`
}

type profileSpecFile struct {
	Policy policyConfig      `yaml:"policy"`
	Vars   map[string]string `yaml:"vars"`
}

type policyConfig struct {
	DryRun              bool   `yaml:"dryRun"`
	ContinueOnError     bool   `yaml:"continueOnError"`
	RequireConfirmation bool   `yaml:"requireConfirmation"`
	AppsDir             string `yaml:"appsDir"`
	StateFile           string `yaml:"stateFile"`
	Distro              string `yaml:"distro"`
}

// LoadProfile reads profile.yaml from dir. A missing file is not an error: it
// yields a profile carrying DefaultPolicy so uboot works without a profile.
func LoadProfile(dir string) (Profile, bool, error) {
	return loadProfileFS(os.DirFS(dir), ".")
}

// LoadProfileFS is the testable variant of LoadProfile.
func LoadProfileFS(fsys fs.FS, dir string) (Profile, bool, error) {
	return loadProfileFS(fsys, dir)
}

func loadProfileFS(fsys fs.FS, dir string) (Profile, bool, error) {
	path := filepath.ToSlash(filepath.Join(dir, "profile.yaml"))
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		if os.IsNotExist(err) {
			return Profile{Policy: DefaultPolicy()}, false, nil
		}
		return Profile{}, false, err
	}

	// Pre-seed defaults so fields absent from the YAML keep their default
	// (notably RequireConfirmation, which defaults to true).
	policy := DefaultPolicy()
	file := profileFile{Spec: profileSpecFile{Policy: policyConfig{
		DryRun:              policy.DryRun,
		ContinueOnError:     policy.ContinueOnError,
		RequireConfirmation: policy.RequireConfirmation,
		AppsDir:             policy.AppsDir,
		StateFile:           policy.StateFile,
		Distro:              policy.Distro,
	}}}
	if err := yaml.Unmarshal(data, &file); err != nil {
		return Profile{}, false, fmt.Errorf("parse %s: %w", path, err)
	}

	profile := Profile{
		APIVersion: file.APIVersion,
		Kind:       file.Kind,
		Metadata: ProfileMetadata{
			Name:        file.Metadata.Name,
			Labels:      file.Metadata.Labels,
			Annotations: file.Metadata.Annotations,
		},
		Policy: Policy{
			DryRun:              file.Spec.Policy.DryRun,
			ContinueOnError:     file.Spec.Policy.ContinueOnError,
			RequireConfirmation: file.Spec.Policy.RequireConfirmation,
			AppsDir:             file.Spec.Policy.AppsDir,
			StateFile:           file.Spec.Policy.StateFile,
			Distro:              file.Spec.Policy.Distro,
		},
		Vars: file.Spec.Vars,
	}
	return profile, true, nil
}
