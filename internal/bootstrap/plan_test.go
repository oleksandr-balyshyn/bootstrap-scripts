package bootstrap

import "testing"

func TestBuildPlanRejectsUnknownSelection(t *testing.T) {
	catalog := Catalog{Modules: []Module{{ID: "known", Title: "Known"}}}
	_, err := BuildPlan(catalog, map[string]bool{"missing": true})
	if err == nil {
		t.Fatal("BuildPlan() error = nil, want error")
	}
}

func TestCommandStringQuotesArguments(t *testing.T) {
	cmd := Command{Program: "bash", Args: []string{"-lc", `echo "hello world"`}, Sudo: true}
	got := cmd.String()
	want := `sudo bash -lc 'echo "hello world"'`
	if got != want {
		t.Fatalf("Command.String() = %q, want %q", got, want)
	}
}
