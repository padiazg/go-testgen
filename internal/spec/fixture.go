package spec

import (
	"strings"
)

// GenerateFixtureDecl generates a var declaration string for a fixture.
func (fix *Fixture) generateFixtureDecl(noHints bool) string {
	var sb strings.Builder
	if fix.Description != "" {
		for _, line := range wrapText(fix.Description, 76) {
			sb.WriteString("// ")
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}
	sb.WriteString("var ")
	sb.WriteString(fix.Name)
	sb.WriteString(" = ")
	sb.WriteString(fix.Type)
	sb.WriteString("{\n")
	if fix.Value != "" {
		sb.WriteString(strings.TrimRight(fix.Value, "\n"))
		sb.WriteString("\n")
	} else if !noHints {
		sb.WriteString("// ai-hint: fill with the value described above\n")
		if fix.Description != "" {
			for _, line := range wrapText(fix.Description, 72) {
				sb.WriteString("// ")
				sb.WriteString(line)
				sb.WriteString("\n")
			}
		}
	}
	sb.WriteString("}\n\n")
	return sb.String()
}
