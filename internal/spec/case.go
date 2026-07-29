package spec

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

type Case struct {
	After       *After            `yaml:"after"`
	Before      *Before           `yaml:"before"`
	Fields      map[string]string `yaml:"fields"`
	Gates       map[string]string `yaml:"gates"`
	Description string            `yaml:"description"`
	Name        string            `yaml:"name"`
	Checks      []string          `yaml:"checks"`
	Todo        bool              `yaml:"todo"`
}

// GenerateCaseEntry generates a struct literal string for a test case entry.
func (c *Case) GenerateCaseEntry(s *Spec, structFields []*ast.Field, fset *token.FileSet, noHints bool) string {
	if c.Todo {
		return c.GenerateTodoCase()
	}

	var sb strings.Builder
	sb.WriteString("{\n")
	fmt.Fprintf(&sb, "name: %q,\n", c.Name)

	for _, field := range structFields {
		for _, nameIdent := range field.Names {
			fieldName := nameIdent.Name
			if fieldName == "name" {
				continue
			}
			entry := generateFieldEntry(fieldName, field.Type, c, s, fset, noHints)
			if entry != "" {
				sb.WriteString(entry)
			}
		}
	}

	sb.WriteString("},\n")
	return sb.String()
}

func (c *Case) GenerateTodoCase() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "// TODO: implement case %q\n", c.Name)
	if c.Description != "" {
		for _, line := range wrapText(c.Description, 76) {
			sb.WriteString("// ")
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}
	if len(c.Checks) > 0 {
		sb.WriteString("// Suggested checks: ")
		sb.WriteString(strings.Join(c.Checks, ", "))
		sb.WriteString("\n")
	}
	return sb.String()
}

func generateFieldEntry(fieldName string, fieldType ast.Expr, c *Case, s *Spec, fset *token.FileSet, noHints bool) string {
	switch fieldName {
	case "before":
		return c.generateBeforeEntry(fieldType, fset, noHints)
	case "after":
		return c.generateAfterEntry(fieldType, fset, noHints)
	case "checks":
		return c.generateChecksEntry(s, fieldType, fset, noHints)
	default:
		return c.generateSimpleField(fieldName, s, noHints)
	}
}

func (c *Case) generateBeforeEntry(fieldType ast.Expr, fset *token.FileSet, noHints bool) string {
	if c.Before == nil {
		return ""
	}

	funcType, ok := fieldType.(*ast.FuncType)
	if !ok {
		if noHints {
			return "before: nil,\n"
		}
		return "before: nil, // ai-hint: set before for this case\n"
	}

	params := formatFuncParams(funcType.Params, fset)
	returnType := ""
	if funcType.Results != nil && len(funcType.Results.List) > 0 {
		returnType = " " + typeToString(funcType.Results.List[0].Type, fset)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "before: func(%s)%s {\n", params, returnType)
	if !noHints {
		if c.Before.Mechanism != "" {
			fmt.Fprintf(&sb, "// ai-hint: %s\n", c.Before.Mechanism)
		}
		if c.Before.Description != "" {
			for _, line := range wrapText(c.Before.Description, 72) {
				sb.WriteString("// ")
				sb.WriteString(line)
				sb.WriteString("\n")
			}
		}
	}
	if c.Before.Returns != nil {
		zero := zeroValueFor(c.Before.Returns.Type)
		if noHints {
			fmt.Fprintf(&sb, "return %s\n", zero)
		} else {
			fmt.Fprintf(&sb, "return %s // ai-hint: return the value described above\n", zero)
		}
	}
	sb.WriteString("},\n")
	return sb.String()
}

func (c *Case) generateAfterEntry(fieldType ast.Expr, fset *token.FileSet, noHints bool) string {
	if c.After == nil {
		return ""
	}

	funcType, ok := fieldType.(*ast.FuncType)
	if !ok {
		return ""
	}

	params := formatFuncParams(funcType.Params, fset)

	var sb strings.Builder
	fmt.Fprintf(&sb, "after: func(%s) {\n", params)
	if !noHints {
		if c.After.Mechanism != "" {
			fmt.Fprintf(&sb, "// ai-hint: %s\n", c.After.Mechanism)
		}
		if c.After.Description != "" {
			for _, line := range wrapText(c.After.Description, 72) {
				sb.WriteString("// ")
				sb.WriteString(line)
				sb.WriteString("\n")
			}
		}
	}
	sb.WriteString("},\n")
	return sb.String()
}

func (c *Case) generateChecksEntry(s *Spec, fieldType ast.Expr, fset *token.FileSet, noHints bool) string {
	composerCall, qualifier := findComposerCall(fieldType, s, fset)
	if composerCall == "" {
		return ""
	}

	var sb strings.Builder
	if len(c.Checks) > 0 {
		fmt.Fprintf(&sb, "checks: %s(\n", composerCall)
		for _, chk := range c.Checks {
			sb.WriteString(prefixCheckCall(chk, s, qualifier))
			sb.WriteString(",\n")
		}
		sb.WriteString("),\n")
	} else if !noHints {
		fmt.Fprintf(&sb, "checks: %s(\n", composerCall)
		fmt.Fprintf(&sb, "// ai-hint: add checks for case %q\n", c.Name)
		if c.Description != "" {
			for _, line := range wrapText(c.Description, 72) {
				sb.WriteString("// ")
				sb.WriteString(line)
				sb.WriteString("\n")
			}
		}
		sb.WriteString("),\n")
	}
	return sb.String()
}

func (c *Case) generateSimpleField(fieldName string, s *Spec, noHints bool) string {
	// Check fields map first (covers input, state, and gate values stored in fields)
	if v, ok := c.Fields[fieldName]; ok {
		return fmt.Sprintf("%s: %s,\n", fieldName, strings.TrimRight(v, "\n\r\t "))
	}
	// Check gates map
	if v, ok := c.Gates[fieldName]; ok {
		return fmt.Sprintf("%s: %s,\n", fieldName, strings.TrimRight(v, "\n\r\t "))
	}
	// No value in spec — emit ai-hint or skip
	if noHints {
		return ""
	}
	// Find the type from table_fields for the hint
	typeStr := ""
	for _, tf := range s.TableFields {
		if tf.Name == fieldName {
			typeStr = tf.Type
			break
		}
	}
	zero := zeroValueFor(typeStr)
	return fmt.Sprintf("%s: %s, // ai-hint: set %s for this case\n", fieldName, zero, fieldName)
}
