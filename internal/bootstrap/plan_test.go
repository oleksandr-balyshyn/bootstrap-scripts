package bootstrap

import "testing"

func TestBuildPlanRejectsUnknownSelection(t *testing.T) {
	catalog := Catalog{Modules: []Module{{ID: "known", Title: "Known"}}}
	_, err := BuildPlan(catalog, map[string]bool{"missing": true})
	if err == nil {
		t.Fatal("BuildPlan() error = nil, want error")
	}
}

func TestBuildPlanExpandsDependenciesInCatalogOrder(t *testing.T) {
	catalog := Catalog{Modules: []Module{
		{ID: "language-installers", Title: "Language Installers"},
		{ID: "cargo-packages", Title: "Cargo Packages", DependsOn: []string{"language-installers"}},
	}}

	plan, err := BuildPlan(catalog, map[string]bool{"cargo-packages": true})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if len(plan.Modules) != 2 {
		t.Fatalf("BuildPlan() modules = %d, want 2", len(plan.Modules))
	}
	if plan.Modules[0].ID != "language-installers" || plan.Modules[1].ID != "cargo-packages" {
		t.Fatalf("BuildPlan() order = [%s, %s]", plan.Modules[0].ID, plan.Modules[1].ID)
	}
}

func TestBuildPlanIgnoresFalseSelections(t *testing.T) {
	catalog := Catalog{Modules: []Module{{ID: "known", Title: "Known"}}}
	plan, err := BuildPlan(catalog, map[string]bool{"known": false})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if len(plan.Modules) != 0 {
		t.Fatalf("BuildPlan() modules = %d, want 0", len(plan.Modules))
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
