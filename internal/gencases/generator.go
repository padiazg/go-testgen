package gencases

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"strings"
	"unicode"

	"github.com/padiazg/go-testgen/internal/spec"
)

// GenerateFixtureDecl generates a var declaration string for a fixture.
func GenerateFixtureDecl(fix *spec.Fixture, noHints bool) string {
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

// GenerateCaseEntry generates a struct literal string for a test case entry.
func GenerateCaseEntry(c *spec.Case, s *spec.Spec, structFields []*ast.Field, fset *token.FileSet, noHints bool) string {
	if c.Todo {
		return generateTodoCase(c)
	}

	var sb strings.Builder
	sb.WriteString("{\n")
	sb.WriteString(fmt.Sprintf("name: %q,\n", c.Name))

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

func generateFieldEntry(fieldName string, fieldType ast.Expr, c *spec.Case, s *spec.Spec, fset *token.FileSet, noHints bool) string {
	switch fieldName {
	case "before":
		return generateBeforeEntry(c, fieldType, fset, noHints)
	case "after":
		return generateAfterEntry(c, fieldType, fset, noHints)
	case "checks":
		return generateChecksEntry(c, s, fieldType, fset, noHints)
	default:
		return generateSimpleField(fieldName, c, s, noHints)
	}
}

func generateSimpleField(fieldName string, c *spec.Case, s *spec.Spec, noHints bool) string {
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

func generateBeforeEntry(c *spec.Case, fieldType ast.Expr, fset *token.FileSet, noHints bool) string {
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
	sb.WriteString(fmt.Sprintf("before: func(%s)%s {\n", params, returnType))
	if !noHints {
		if c.Before.Mechanism != "" {
			sb.WriteString(fmt.Sprintf("// ai-hint: %s\n", c.Before.Mechanism))
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
			sb.WriteString(fmt.Sprintf("return %s\n", zero))
		} else {
			sb.WriteString(fmt.Sprintf("return %s // ai-hint: return the value described above\n", zero))
		}
	}
	sb.WriteString("},\n")
	return sb.String()
}

func generateAfterEntry(c *spec.Case, fieldType ast.Expr, fset *token.FileSet, noHints bool) string {
	if c.After == nil {
		return ""
	}

	funcType, ok := fieldType.(*ast.FuncType)
	if !ok {
		return ""
	}

	params := formatFuncParams(funcType.Params, fset)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("after: func(%s) {\n", params))
	if !noHints {
		if c.After.Mechanism != "" {
			sb.WriteString(fmt.Sprintf("// ai-hint: %s\n", c.After.Mechanism))
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

func generateChecksEntry(c *spec.Case, s *spec.Spec, fieldType ast.Expr, fset *token.FileSet, noHints bool) string {
	composerCall, qualifier := findComposerCall(fieldType, s, fset)
	if composerCall == "" {
		return ""
	}

	var sb strings.Builder
	if len(c.Checks) > 0 {
		sb.WriteString(fmt.Sprintf("checks: %s(\n", composerCall))
		for _, chk := range c.Checks {
			sb.WriteString(prefixCheckCall(chk, s, qualifier))
			sb.WriteString(",\n")
		}
		sb.WriteString("),\n")
	} else if !noHints {
		sb.WriteString(fmt.Sprintf("checks: %s(\n", composerCall))
		sb.WriteString(fmt.Sprintf("// ai-hint: add checks for case %q\n", c.Name))
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

func generateTodoCase(c *spec.Case) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("// TODO: implement case %q\n", c.Name))
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

// findComposerCall returns the composer function call string and package qualifier.
func findComposerCall(fieldType ast.Expr, s *spec.Spec, fset *token.FileSet) (string, string) {
	typeStr := strings.TrimPrefix(typeToString(fieldType, fset), "[]")

	for _, ct := range s.CheckTypes {
		if ct.Composer == "" {
			continue
		}
		qualifier := ""
		fullType := ct.TypeName
		if ct.Package != "" {
			parts := strings.Split(ct.Package, "/")
			qualifier = parts[len(parts)-1]
			fullType = qualifier + "." + ct.TypeName
		}
		if fullType == typeStr {
			composerCall := ct.Composer
			if qualifier != "" {
				composerCall = qualifier + "." + ct.Composer
			}
			return composerCall, qualifier
		}
	}
	return "", ""
}

// prefixCheckCall adds a package qualifier to a check call if needed.
// "hasErrors(true)" + qualifier="" → "hasErrors(true)"
// "CheckResultError(...)" + qualifier="model" → "model.CheckResultError(...)"
func prefixCheckCall(callStr string, s *spec.Spec, qualifier string) string {
	if qualifier == "" {
		return callStr
	}
	// Extract function name (before first '(')
	idx := strings.Index(callStr, "(")
	if idx < 0 {
		return callStr
	}
	funcName := callStr[:idx]

	// Find if this check belongs to the cross-package check type
	for _, chk := range s.Checks {
		if chk.ID == funcName {
			ct := s.CheckTypeByID(chk.ForType)
			if ct != nil && ct.Package != "" {
				return qualifier + "." + callStr
			}
			break
		}
	}
	return callStr
}

// typeToString converts an AST type expression to its string representation.
func typeToString(expr ast.Expr, fset *token.FileSet) string {
	if expr == nil {
		return ""
	}
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, expr); err != nil {
		return fmt.Sprintf("%T", expr)
	}
	return buf.String()
}

// formatFuncParams formats the parameter list, assigning names if absent.
func formatFuncParams(params *ast.FieldList, fset *token.FileSet) string {
	if params == nil || len(params.List) == 0 {
		return ""
	}
	var parts []string
	for i, field := range params.List {
		typeStr := typeToString(field.Type, fset)
		if len(field.Names) > 0 {
			names := make([]string, len(field.Names))
			for j, n := range field.Names {
				names[j] = n.Name
			}
			parts = append(parts, strings.Join(names, ", ")+" "+typeStr)
		} else {
			parts = append(parts, paramNameFromType(typeStr, i)+" "+typeStr)
		}
	}
	return strings.Join(parts, ", ")
}

// paramNameFromType derives a parameter name from a type string.
// "*WebhookNotifier" → "w", "*Engine" → "e"
func paramNameFromType(typeStr string, idx int) string {
	s := strings.TrimPrefix(typeStr, "*")
	s = strings.TrimPrefix(s, "[]")
	for _, r := range s {
		if unicode.IsLetter(r) {
			return string(unicode.ToLower(r))
		}
	}
	return fmt.Sprintf("p%d", idx)
}

// zeroValueFor returns a zero/nil literal for the given type string.
func zeroValueFor(typeStr string) string {
	if typeStr == "" {
		return "nil"
	}
	if strings.HasPrefix(typeStr, "*") ||
		strings.HasPrefix(typeStr, "[]") ||
		strings.HasPrefix(typeStr, "map[") ||
		typeStr == "error" {
		return "nil"
	}
	switch typeStr {
	case "bool":
		return "false"
	case "string":
		return `""`
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64":
		return "0"
	}
	return "nil"
}

// wrapText wraps text to width, splitting on whitespace.
func wrapText(text string, width int) []string {
	var result []string
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.TrimSpace(text)

	for _, para := range strings.Split(text, "\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		if len(para) <= width {
			result = append(result, para)
			continue
		}
		words := strings.Fields(para)
		line := ""
		for _, word := range words {
			if line == "" {
				line = word
			} else if len(line)+1+len(word) <= width {
				line += " " + word
			} else {
				result = append(result, line)
				line = word
			}
		}
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}
