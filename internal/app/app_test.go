package app

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunListsModules(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", "../../configs", "--list"}, Streams{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "system-update") {
		t.Fatalf("stdout does not list modules:\n%s", stdout.String())
	}
}

func TestRunValidatesCatalog(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", "../../configs", "--validate"}, Streams{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Catalog valid: 31 module(s)") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunDryRunSelectedModule(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", "../../configs", "--dry-run", "--no-tui", "system-update"}, Streams{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "[system-update] System Update & Base") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunRejectsUnknownModule(t *testing.T) {
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", "../../configs", "--no-tui", "missing-module"}, Streams{
		Stderr: &stderr,
	})
	if code != 1 {
		t.Fatalf("Run() code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), `unknown module \"missing-module\"`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
