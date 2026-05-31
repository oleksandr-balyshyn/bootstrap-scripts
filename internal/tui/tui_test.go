package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/w0rxbend/ubuntu-bootstrap/internal/bootstrap"
)

func TestDefaultSelectionUsesCatalogDefaults(t *testing.T) {
	selected := defaultSelection(bootstrap.Catalog{Modules: []bootstrap.Module{
		{ID: "one", Default: true},
		{ID: "two"},
	}})
	if !selected["one"] {
		t.Fatal("default module was not selected")
	}
	if selected["two"] {
		t.Fatal("non-default module was selected")
	}
}

func TestToggleDeletesDeselectedEntries(t *testing.T) {
	m := model{
		catalog:  bootstrap.Catalog{Modules: []bootstrap.Module{{ID: "one"}}},
		selected: map[string]bool{"one": true},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	got := updated.(model)
	if _, exists := got.selected["one"]; exists {
		t.Fatal("deselected module key was retained")
	}
}
