package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/w0rxbend/ubuntu-bootstrap/internal/bootstrap"
)

type Result struct {
	Selected  map[string]bool
	DryRun    bool
	Cancelled bool
}

type model struct {
	catalog  bootstrap.Catalog
	cursor   int
	selected map[string]bool
	dryRun   bool
	cancel   bool
	width    int
	height   int
}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	panelStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")).Padding(1, 2)
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	tagStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
)

func Run(catalog bootstrap.Catalog) (Result, error) {
	m := model{catalog: catalog, selected: defaultSelection(catalog)}
	program := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return Result{}, err
	}
	m, ok := finalModel.(model)
	if !ok {
		return Result{}, fmt.Errorf("unexpected TUI model %T", finalModel)
	}
	return Result{Selected: m.selected, DryRun: m.dryRun, Cancelled: m.cancel}, nil
}

func defaultSelection(catalog bootstrap.Catalog) map[string]bool {
	selected := map[string]bool{}
	for _, mod := range catalog.Modules {
		switch mod.ID {
		case "system-update", "shell", "terminal-cli", "current-machine":
			selected[mod.ID] = true
		}
	}
	return selected
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			m.cancel = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.catalog.Modules)-1 {
				m.cursor++
			}
		case " ", "x":
			id := m.catalog.Modules[m.cursor].ID
			m.selected[id] = !m.selected[id]
		case "a":
			all := len(m.selected) != len(m.catalog.Modules)
			m.selected = map[string]bool{}
			if all {
				for _, mod := range m.catalog.Modules {
					m.selected[mod.ID] = true
				}
			}
		case "d":
			m.dryRun = !m.dryRun
		case "enter":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	leftWidth := 46
	if m.width > 0 && m.width < 100 {
		leftWidth = max(34, m.width-8)
	}

	var left strings.Builder
	left.WriteString(titleStyle.Render("Ubuntu Bootstrap") + "\n")
	left.WriteString(helpStyle.Render("space toggle  a all  d dry-run  enter run  q quit") + "\n\n")
	for i, mod := range m.catalog.Modules {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render(">")
		}
		box := "[ ]"
		if m.selected[mod.ID] {
			box = selectedStyle.Render("[x]")
		}
		line := fmt.Sprintf("%s %s %s", cursor, box, mod.Title)
		if i == m.cursor {
			line = cursorStyle.Render(line)
		}
		left.WriteString(line + "\n")
	}

	detail := m.detailView()
	body := lipgloss.JoinHorizontal(lipgloss.Top,
		panelStyle.Width(leftWidth).Render(left.String()),
		panelStyle.Width(max(42, m.width-leftWidth-10)).Render(detail),
	)
	return body
}

func (m model) detailView() string {
	mod := m.catalog.Modules[m.cursor]
	var b strings.Builder
	b.WriteString(titleStyle.Render(mod.Title) + "\n\n")
	b.WriteString(mod.Description + "\n\n")
	b.WriteString(tagStyle.Render("source: "+mod.Source) + "\n")
	b.WriteString(tagStyle.Render("tags: "+strings.Join(mod.Tags, ", ")) + "\n")
	if m.dryRun {
		b.WriteString(selectedStyle.Render("mode: dry-run") + "\n\n")
	} else {
		b.WriteString("mode: execute\n\n")
	}
	b.WriteString("Steps:\n")
	for _, step := range mod.Steps {
		b.WriteString("  - " + step.Name + "\n")
	}
	count := 0
	for _, ok := range m.selected {
		if ok {
			count++
		}
	}
	b.WriteString(fmt.Sprintf("\nSelected modules: %d/%d\n", count, len(m.catalog.Modules)))
	return b.String()
}
