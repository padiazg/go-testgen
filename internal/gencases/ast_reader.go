package gencases

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
)

// ParseTestFile reads and parses a _test.go file, returning the AST, FileSet, and raw bytes.
func ParseTestFile(path string) (*ast.File, *token.FileSet, []byte, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gen-cases: target file not found: %s. Run go-testgen gen first", path)
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gen-cases: parse %s: %w", path, err)
	}
	return f, fset, src, nil
}

// FindTestFunc locates a TestXxx function declaration in the file.
func FindTestFunc(f *ast.File, name string) (*ast.FuncDecl, error) {
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Name.Name == name {
			return fn, nil
		}
	}
	return nil, fmt.Errorf("gen-cases: %s not found in file. Run go-testgen gen first", name)
}

// FindTestsSlice locates the `tests := []struct{...}{...}` assignment in TestXxx.
func FindTestsSlice(fn *ast.FuncDecl) (*ast.CompositeLit, error) {
	var result *ast.CompositeLit
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if result != nil {
			return false
		}
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		if len(assign.Lhs) == 0 || len(assign.Rhs) == 0 {
			return true
		}
		ident, ok := assign.Lhs[0].(*ast.Ident)
		if !ok || ident.Name != "tests" {
			return true
		}
		lit, ok := assign.Rhs[0].(*ast.CompositeLit)
		if !ok {
			return true
		}
		result = lit
		return false
	})
	if result == nil {
		return nil, fmt.Errorf("gen-cases: could not find tests slice in %s", fn.Name.Name)
	}
	return result, nil
}

// InspectTestsStruct returns the fields of the anonymous struct in the tests slice.
func InspectTestsStruct(lit *ast.CompositeLit) ([]*ast.Field, error) {
	arr, ok := lit.Type.(*ast.ArrayType)
	if !ok {
		return nil, fmt.Errorf("gen-cases: tests slice type is not an array")
	}
	st, ok := arr.Elt.(*ast.StructType)
	if !ok {
		return nil, fmt.Errorf("gen-cases: tests slice element is not a struct")
	}
	return st.Fields.List, nil
}

// FindExistingCase reports whether a case with the given name already exists in the slice.
func FindExistingCase(lit *ast.CompositeLit, name string) bool {
	return FindExistingCaseNode(lit, name) != nil
}

// FindExistingCaseNode returns the CompositeLit for a case with the given name, or nil.
func FindExistingCaseNode(lit *ast.CompositeLit, name string) *ast.CompositeLit {
	target := `"` + name + `"`
	for _, elt := range lit.Elts {
		cl, ok := elt.(*ast.CompositeLit)
		if !ok {
			continue
		}
		for _, kv := range cl.Elts {
			pair, ok := kv.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := pair.Key.(*ast.Ident)
			if !ok || key.Name != "name" {
				continue
			}
			val, ok := pair.Value.(*ast.BasicLit)
			if !ok {
				continue
			}
			if val.Value == target {
				return cl
			}
		}
	}
	return nil
}

// FindExistingVar reports whether a var/const with the given name exists at file scope.
func FindExistingVar(f *ast.File, name string) bool {
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, sp := range gd.Specs {
			vs, ok := sp.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, n := range vs.Names {
				if n.Name == name {
					return true
				}
			}
		}
	}
	return false
}

// FindExistingLocalCheck reports whether a local check assignment exists in TestXxx.
func FindExistingLocalCheck(fn *ast.FuncDecl, name string) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if found {
			return false
		}
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		if len(assign.Lhs) == 0 {
			return true
		}
		ident, ok := assign.Lhs[0].(*ast.Ident)
		if !ok {
			return true
		}
		if ident.Name == name {
			found = true
		}
		return !found
	})
	return found
}
