package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// ScanPackage analyzes all exported functions/methods in a package and
// returns their test status, interface dependencies, and mock file status.
func ScanPackage(pkgPattern string) (*ScanResult, error) {
	fset := token.NewFileSet()
	cfg := &packages.Config{
		Mode: packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo |
			packages.NeedImports | packages.NeedDeps | packages.NeedFiles | packages.NeedName,
		Fset: fset,
	}
	pkgs, err := packages.Load(cfg, pkgPattern)
	if err != nil {
		return nil, fmt.Errorf("load package: %w", err)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no package found for %s", pkgPattern)
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		return nil, fmt.Errorf("package errors: %v", pkg.Errors[0])
	}

	// Collect import aliases from ALL source files.
	aliases := collectAllAliases(pkg)

	// Source directory.
	sourceDir := ""
	if len(pkg.GoFiles) > 0 {
		sourceDir = filepath.Dir(pkg.GoFiles[0])
	}

	result := &ScanResult{
		Package:    pkg.Name,
		ImportPath: pkg.PkgPath,
		SourceDir:  sourceDir,
	}

	// Walk source files. pkg.Syntax[i] corresponds to pkg.GoFiles[i].
	for i, syn := range pkg.Syntax {
		sourceFile := ""
		if i < len(pkg.GoFiles) {
			sourceFile = pkg.GoFiles[i]
		}
		if strings.HasSuffix(sourceFile, "_test.go") {
			continue
		}

		for _, decl := range syn.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !fn.Name.IsExported() {
				continue
			}
			summary := buildFuncSummary(fn, pkg, aliases, sourceDir)
			result.Funcs = append(result.Funcs, summary)
		}
	}

	return result, nil
}

// collectAllAliases gathers importPath→alias from every source file in the package.
func collectAllAliases(pkg *packages.Package) map[string]string {
	aliases := make(map[string]string)
	for _, syn := range pkg.Syntax {
		for _, imp := range syn.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if _, exists := aliases[path]; exists {
				continue
			}
			if imp.Name != nil && imp.Name.Name != "_" && imp.Name.Name != "." {
				aliases[path] = imp.Name.Name
			} else if importedPkg, ok := pkg.Imports[path]; ok {
				aliases[path] = importedPkg.Name
			} else {
				parts := strings.Split(path, "/")
				aliases[path] = parts[len(parts)-1]
			}
		}
	}
	return aliases
}

func buildFuncSummary(
	fn *ast.FuncDecl,
	pkg *packages.Package,
	aliases map[string]string,
	sourceDir string,
) FuncSummary {
	// Receiver type (strip pointer).
	receiverType := ""
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		receiverType = strings.TrimPrefix(typeToString(fn.Recv.List[0].Type), "*")
	}

	funcSpec := fn.Name.Name
	if receiverType != "" {
		funcSpec = receiverType + "." + fn.Name.Name
	}

	// Test function name — reuse logic from FindTestFuncName.
	testFuncName := deriveTestFuncName(fn, receiverType)

	summary := FuncSummary{
		Name:         fn.Name.Name,
		ReceiverType: receiverType,
		IsMethod:     receiverType != "",
		Signature:    buildSignatureStr(fn, receiverType),
		FuncSpec:     funcSpec,
		TestFuncName: testFuncName,
		TestExists:   testFuncExistsInDir(sourceDir, testFuncName),
	}

	summary.InterfaceDeps = extractInterfaceDeps(fn, pkg, aliases, sourceDir)

	return summary
}

// buildSignatureStr produces a fully-qualified human-readable function signature.
func buildSignatureStr(fn *ast.FuncDecl, receiverType string) string {
	prefix := fn.Name.Name
	if receiverType != "" {
		prefix = receiverType + "." + fn.Name.Name
	}

	var paramStrs []string
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			typeStr := typeToString(field.Type)
			if len(field.Names) > 0 {
				for _, name := range field.Names {
					paramStrs = append(paramStrs, name.Name+" "+typeStr)
				}
			} else {
				paramStrs = append(paramStrs, typeStr)
			}
		}
	}

	var retStrs []string
	if fn.Type.Results != nil {
		for _, field := range fn.Type.Results.List {
			typeStr := typeToString(field.Type)
			if len(field.Names) > 0 {
				for _, name := range field.Names {
					retStrs = append(retStrs, name.Name+" "+typeStr)
				}
			} else {
				retStrs = append(retStrs, typeStr)
			}
		}
	}

	sig := prefix + "(" + strings.Join(paramStrs, ", ") + ")"
	switch len(retStrs) {
	case 0:
	case 1:
		sig += " " + retStrs[0]
	default:
		sig += " (" + strings.Join(retStrs, ", ") + ")"
	}
	return sig
}

// deriveTestFuncName mirrors FindTestFuncName without needing a full FuncInfo.
func deriveTestFuncName(fn *ast.FuncDecl, receiverType string) string {
	if receiverType != "" {
		return "Test" + receiverType + "_" + fn.Name.Name
	}
	// Constructor heuristic: New or NewXxx returning a pointer.
	if strings.HasPrefix(fn.Name.Name, "New") && fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		firstRet := typeToString(fn.Type.Results.List[0].Type)
		typeName := strings.TrimPrefix(firstRet, "*")
		if typeName != "" && typeName != firstRet {
			return "Test" + typeName + "_" + fn.Name.Name
		}
	}
	return "Test" + fn.Name.Name
}

// testFuncExistsInDir checks if testFuncName exists in any _test.go file in sourceDir.
func testFuncExistsInDir(sourceDir, testFuncName string) bool {
	if sourceDir == "" || testFuncName == "" {
		return false
	}
	entries, err := filepath.Glob(filepath.Join(sourceDir, "*_test.go"))
	if err != nil {
		return false
	}
	for _, f := range entries {
		if ok, _ := TestExistsInFile(f, testFuncName); ok {
			return true
		}
	}
	return false
}

// extractInterfaceDeps detects interface-typed struct fields from the receiver or
// constructor config param, returning deduplicated InterfaceDep entries.
func extractInterfaceDeps(
	fn *ast.FuncDecl,
	pkg *packages.Package,
	aliases map[string]string,
	sourceDir string,
) []InterfaceDep {
	seen := make(map[string]bool)
	var deps []InterfaceDep

	addDepsFromStruct := func(strct *types.Struct) {
		for i := 0; i < strct.NumFields(); i++ {
			field := strct.Field(i)
			if _, isIface := field.Type().Underlying().(*types.Interface); !isIface {
				continue
			}
			named, ok := field.Type().(*types.Named)
			if !ok {
				continue
			}
			obj := named.Obj()
			if obj.Pkg() == nil {
				continue // built-in (error, etc.)
			}
			typeName := obj.Name()
			if seen[typeName] {
				continue
			}
			seen[typeName] = true

			importPath := obj.Pkg().Path()
			qualifier := resolveQualifierByPath(importPath, obj.Pkg().Name(), aliases)

			mockFrom := qualifier + "." + typeName
			if qualifier == "" || qualifier == pkg.Name {
				mockFrom = typeName
			}

			mockFile := "mock_" + strings.ToLower(typeName) + "_test.go"
			mockPath := filepath.Join(sourceDir, mockFile)
			_, statErr := os.Stat(mockPath)

			deps = append(deps, InterfaceDep{
				TypeName:   typeName,
				Qualifier:  qualifier,
				ImportPath: importPath,
				MockFile:   mockFile,
				MockExists: statErr == nil,
				MockFrom:   mockFrom,
			})
		}
	}

	// Methods: inspect receiver struct fields.
	if fn.Recv != nil && len(fn.Recv.List) > 0 && pkg.TypesInfo != nil {
		recvTypeExpr := fn.Recv.List[0].Type
		if tv, ok := pkg.TypesInfo.Types[recvTypeExpr]; ok {
			t := tv.Type
			if ptr, ok := t.(*types.Pointer); ok {
				t = ptr.Elem()
			}
			if named, ok := t.(*types.Named); ok {
				if strct, ok := named.Underlying().(*types.Struct); ok {
					addDepsFromStruct(strct)
				}
			}
		}
	}

	// Functions (constructors): inspect first non-context param if it's *Struct.
	if fn.Recv == nil && fn.Type.Params != nil && pkg.TypesInfo != nil {
		for _, param := range fn.Type.Params.List {
			if tv, ok := pkg.TypesInfo.Types[param.Type]; ok {
				t := tv.Type
				if ptr, ok := t.(*types.Pointer); ok {
					t = ptr.Elem()
				}
				if named, ok := t.(*types.Named); ok {
					if strct, ok := named.Underlying().(*types.Struct); ok {
						addDepsFromStruct(strct)
						break // only inspect first eligible param
					}
				}
			}
		}
	}

	return deps
}
