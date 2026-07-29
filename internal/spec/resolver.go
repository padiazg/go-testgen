package spec

import (
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// ResolveTestFuncName converts "Receiver.Method" or "FuncName" to "TestReceiver_Method".
func (s *Spec) resolveTestFuncName() string {
	if receiver, method, ok := strings.Cut(s.Function, "."); ok {
		return "Test" + receiver + "_" + method
	}
	return "Test" + s.Function
}

// ResolveTargetFile resolves the _test.go file path from spec + override.
func (s *Spec) resolveTargetFile(outputOverride string) (string, error) {
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
