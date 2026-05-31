package bootstrap

import (
	"fmt"
	"io"
	"strings"
)

type Plan struct {
	Modules []Module
}

func BuildPlan(catalog Catalog, selected map[string]bool) (Plan, error) {
	known := map[string]bool{}
	var modules []Module
	for _, mod := range catalog.Modules {
		known[mod.ID] = true
		if selected[mod.ID] {
			modules = append(modules, mod)
		}
	}
	for id := range selected {
		if !known[id] {
			return Plan{}, fmt.Errorf("unknown module %q", id)
		}
	}
	return Plan{Modules: modules}, nil
}

func (p Plan) Print(w io.Writer) {
	fmt.Fprintln(w, "Bootstrap plan:")
	for _, mod := range p.Modules {
		fmt.Fprintf(w, "\n[%s] %s\n", mod.ID, mod.Title)
		for _, step := range mod.Steps {
			fmt.Fprintf(w, "  - %s\n", step.Name)
			for _, cmd := range step.Commands {
				fmt.Fprintf(w, "    %s\n", cmd.String())
			}
		}
	}
}

func (c Command) String() string {
	parts := append([]string{c.Program}, c.Args...)
	if c.Sudo {
		parts = append([]string{"sudo"}, parts...)
	}
	for i, part := range parts {
		if strings.ContainsAny(part, " \n\t\"'") {
			parts[i] = shellQuote(part)
		}
	}
	return strings.Join(parts, " ")
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
