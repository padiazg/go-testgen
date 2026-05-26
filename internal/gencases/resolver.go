package gencases

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/padiazg/go-testgen/internal/spec"
	"golang.org/x/tools/go/packages"
)

// ResolveTestFuncName converts "Receiver.Method" or "FuncName" to "TestReceiver_Method".
func ResolveTestFuncName(s *spec.Spec) string {
	if receiver, method, ok := strings.Cut(s.Function, "."); ok {
		return "Test" + receiver + "_" + method
	}
	return "Test" + s.Function
}

// ResolveTargetFile resolves the _test.go file path from spec + override.
func ResolveTargetFile(s *spec.Spec, outputOverride string) (string, error) {
	if outputOverride != "" {
		return outputOverride, nil
	}
	if s.TestFile != "" {
		return s.TestFile, nil
	}

	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedFiles,
	}, s.Package)
	if err != nil {
		return "", fmt.Errorf("load package %s: %w", s.Package, err)
	}
	if len(pkgs) == 0 || len(pkgs[0].GoFiles) == 0 {
		return "", fmt.Errorf("package %s not found or has no Go files", s.Package)
	}

	pkgDir := filepath.Dir(pkgs[0].GoFiles[0])
	filename := deriveTestFileName(s.Function) + "_test.go"
	return filepath.Join(pkgDir, filename), nil
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
