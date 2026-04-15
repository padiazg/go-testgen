package analyzer

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

func DeriveTestPath(sourceFile string) string {
	dir := filepath.Dir(sourceFile)
	base := filepath.Base(sourceFile)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return filepath.Join(dir, name+"_test.go")
}

func TestExistsInFile(path, testFuncName string) (bool, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return false, fmt.Errorf("parse file: %w", err)
	}

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Name.Name == testFuncName {
			return true, nil
		}
	}
	return false, nil
}

func FindTestFuncName(info *FuncInfo) string {
	if info.IsMethod {
		return "Test" + info.Receiver.TypeName + "_" + info.Name
	}
	return "Test" + info.Name
}

// InjectImports adds missing imports (path -> alias) to an existing Go source file.
// Uses astutil.AddNamedImport which is a no-op if the import already exists.
func InjectImports(filePath string, imports map[string]string) error {
	if len(imports) == 0 {
		return nil
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse file: %w", err)
	}

	changed := false
	for path, alias := range imports {
		if astutil.AddNamedImport(fset, f, alias, path) {
			changed = true
		}
	}

	if !changed {
		return nil
	}

	ast.SortImports(fset, f)

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, f); err != nil {
		return fmt.Errorf("format: %w", err)
	}

	return os.WriteFile(filePath, buf.Bytes(), 0644)
}

func MergeTestFile(path string, newCode []byte) error {
	existing, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read existing: %w", err)
	}

	existing = append(existing, '\n')
	existing = append(existing, newCode...)

	return os.WriteFile(path, existing, 0644)
}

func PromptOverwrite(path string, testFuncName string, content []byte) error {
	fmt.Printf("Test file %s already exists and contains test %q\n", path, testFuncName)
	fmt.Println("  [S]obrescribir")
	fmt.Println("  [C]ancelar")
	fmt.Print("> ")

	var answer string
	if _, err := fmt.Scan(&answer); err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	answer = strings.ToUpper(strings.TrimSpace(answer))
	switch answer {
	case "S":
		return os.WriteFile(path, content, 0644)
	case "C":
		return fmt.Errorf("cancelled by user")
	default:
		fmt.Println("Invalid option. Cancelling.")
		return fmt.Errorf("cancelled by user")
	}
}
