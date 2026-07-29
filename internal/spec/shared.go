package spec

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"os"
	"sort"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// ParseFile reads and parses a .testspec.yaml file.
func ParseFile(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("gen-cases: spec file not found: %s", path)
	}
	var s Spec
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("gen-cases: invalid spec: %w", err)
	}
	if err := s.validate(); err != nil {
		return nil, fmt.Errorf("gen-cases: invalid spec: %w", err)
	}
	return &s, nil
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

// prefixCheckCall adds a package qualifier to a check call if needed.
// "hasErrors(true)" + qualifier="" → "hasErrors(true)"
// "CheckResultError(...)" + qualifier="model" → "model.CheckResultError(...)"
func prefixCheckCall(callStr string, s *Spec, qualifier string) string {
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
			ct := s.checkTypeByID(chk.ForType)
			if ct != nil && ct.Package != "" {
				return qualifier + "." + callStr
			}
			break
		}
	}
	return callStr
}

// findComposerCall returns the composer function call string and package qualifier.
func findComposerCall(fieldType ast.Expr, s *Spec, fset *token.FileSet) (string, string) {
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

// applyInsertions applies insertions and replacements to src.
// Insertions are applied at given offsets; replacements substitute byte ranges.
// All operations are applied in reverse offset order to preserve positions.
func applyInsertions(src []byte, insertions []insertion, replaces []replaceOp) []byte {
	type op struct {
		content string
		end     int // for insertions: start == end
		start   int
	}

	var ops []op
	for _, ins := range insertions {
		ops = append(ops, op{start: ins.offset, end: ins.offset, content: ins.content})
	}
	for _, r := range replaces {
		ops = append(ops, op(r))
	}

	// Sort descending by start offset so we apply from end to start
	sort.Slice(ops, func(i, j int) bool {
		return ops[i].start > ops[j].start
	})

	result := make([]byte, len(src))
	copy(result, src)

	for _, op := range ops {
		ins := []byte(op.content)
		result = append(result[:op.start], append(ins, result[op.end:]...)...)
	}

	return result
}

// deriveTestFileName converts a FuncSpec to the base test filename (without _test.go).
// "WebhookNotifier.Deliver" → "webhook_notifier"
// "Engine.Start"            → "engine"
// "NewEngine"               → "engine"
func deriveTestFileName(funcSpec string) string {
	if receiver, _, ok := strings.Cut(funcSpec, "."); ok {
		return camelToSnake(receiver)
	}
	name := funcSpec
	if strings.HasPrefix(name, "New") && len(name) > 3 {
		name = name[3:]
	}
	return camelToSnake(name)
}

// camelToSnake converts CamelCase to snake_case.
// "WebhookNotifier" → "webhook_notifier", "ZH07i" → "zh07i"
func camelToSnake(s string) string {
	var result []rune
	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsUpper(r) && i > 0 {
			prev := runes[i-1]
			if unicode.IsLower(prev) || unicode.IsDigit(prev) {
				result = append(result, '_')
			} else if i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
				result = append(result, '_')
			}
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}
