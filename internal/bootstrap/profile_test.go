package bootstrap

import (
	"testing"
	"testing/fstest"
)

func TestLoadProfileMissingUsesDefaults(t *testing.T) {
	profile, found, err := LoadProfileFS(fstest.MapFS{}, ".")
	if err != nil {
		t.Fatalf("LoadProfileFS() error = %v", err)
	}
	if found {
		t.Fatal("found = true, want false for missing profile")
	}
	if !profile.Policy.RequireConfirmation {
		t.Fatal("default policy should require confirmation")
	}
	if profile.Policy.Distro != "ubuntu" {
		t.Fatalf("default distro = %q, want ubuntu", profile.Policy.Distro)
	}
}

func TestLoadProfileReadsPolicy(t *testing.T) {
	fsys := fstest.MapFS{
		"profile.yaml": {Data: []byte(`apiVersion: bootstrap.worxbend.io/v1alpha1
kind: SystemBootstrapProfile
metadata:
  name: ubuntu-workstation
spec:
  policy:
    dryRun: true
    continueOnError: true
    requireConfirmation: false
    appsDir: /custom/apps
    stateFile: custom/state.json
    distro: ubuntu
  vars:
    arch: amd64
`)},
	}

	profile, found, err := LoadProfileFS(fsys, ".")
	if err != nil {
		t.Fatalf("LoadProfileFS() error = %v", err)
	}
	if !found {
		t.Fatal("found = false, want true")
	}
	if profile.Metadata.Name != "ubuntu-workstation" {
		t.Fatalf("metadata name = %q", profile.Metadata.Name)
	}
	if !profile.Policy.DryRun || !profile.Policy.ContinueOnError {
		t.Fatalf("policy booleans not parsed: %+v", profile.Policy)
	}
	if profile.Policy.RequireConfirmation {
		t.Fatal("requireConfirmation should be false when set false")
	}
	if profile.Policy.StateFile != "custom/state.json" {
		t.Fatalf("stateFile = %q", profile.Policy.StateFile)
	}
	if profile.Vars["arch"] != "amd64" {
		t.Fatalf("vars not parsed: %+v", profile.Vars)
	}
}

func TestLoadProfileDefaultsConfirmationTrueWhenOmitted(t *testing.T) {
	fsys := fstest.MapFS{
		"profile.yaml": {Data: []byte(`kind: SystemBootstrapProfile
spec:
  policy:
    dryRun: false
`)},
	}
	profile, _, err := LoadProfileFS(fsys, ".")
	if err != nil {
		t.Fatalf("LoadProfileFS() error = %v", err)
	}
	if !profile.Policy.RequireConfirmation {
		t.Fatal("requireConfirmation should default to true when omitted")
	}
}
