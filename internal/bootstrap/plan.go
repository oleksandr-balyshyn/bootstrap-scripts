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
	known := make(map[string]Module, len(catalog.Modules))
	for _, mod := range catalog.Modules {
		known[mod.ID] = mod
	}
	for id := range selected {
		if !selected[id] {
			continue
		}
		if _, ok := known[id]; !ok {
			return Plan{}, fmt.Errorf("unknown module %q", id)
		}
	}

	expanded := map[string]bool{}
	var expand func(id string) error
	expand = func(id string) error {
		if expanded[id] {
			return nil
		}
		module, ok := known[id]
		if !ok {
			return fmt.Errorf("unknown module %q", id)
		}
		expanded[id] = true
		for _, dep := range module.DependsOn {
			if err := expand(dep); err != nil {
				return err
			}
		}
		return nil
	}
	for id, ok := range selected {
		if ok {
			if err := expand(id); err != nil {
				return Plan{}, err
			}
		}
	}

	var modules []Module
	for _, mod := range catalog.Modules {
		if expanded[mod.ID] {
			modules = append(modules, mod)
		}
	}
	return Plan{Modules: modules}, nil
}

func (p Plan) Print(w io.Writer) {
	fmt.Fprintln(w, "Bootstrap plan:")
	for _, mod := range p.Modules {
		fmt.Fprintf(w, "\n[%s] %s\n", mod.ID, mod.Title)
		if len(mod.DependsOn) > 0 {
			fmt.Fprintf(w, "  depends on: %s\n", strings.Join(mod.DependsOn, ", "))
		}
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
