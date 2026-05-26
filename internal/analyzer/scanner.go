package analyzer

import (
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

func mergeList(pkgs []*packages.Package) (*ScanResult, error) {
	var merged ScanResult
	for _, p := range pkgs {
		if len(p.Errors) > 0 {
			return nil, fmt.Errorf("package errors: %v", p.Errors[0])
		}
		sub, err := scanSinglePackage(p)
		if err != nil {
			return nil, err
		}
		merged.Funcs = append(merged.Funcs, sub.Funcs...)
		// Copy package metadata from the last (most specific) package
		merged.ImportPath = sub.ImportPath
		merged.SourceDir = sub.SourceDir
		merged.Package = sub.Package
	}
	return &merged, nil
}

// ScanPackage analyzes all functions/methods in a package and
// returns their test status, interface dependencies, and mock file status.
// If includeUnexported is true, also includes unexported functions.
func ScanPackage(pkgPattern string) (*ScanResult, error) {
	pkgs, err := getPackages(pkgPattern)
	if err != nil {
		return nil, fmt.Errorf("load package: %w", err)
	}

	// When pattern is "." and the directory has no .go files but has go.mod,
	// retry with "./..." to scan all subpackages (module root case).
	if checkListError(pkgs) &&
		pkgPattern == "." &&
		len(pkgs) == 1 &&
		len(pkgs[0].GoFiles) == 0 &&
		len(pkgs[0].Errors) > 0 &&
		isGoModPresent(".") {
		pkgs, err = getPackages("./...")
		if err != nil {
			return nil, fmt.Errorf("load package: %w", err)
		}
	}

	// // If multiple packages were loaded (e.g. "./..."), merge results from all.
	if len(pkgs) > 1 {
		return mergeList(pkgs)
	}

	return scanSinglePackage(pkgs[0])
}

// signatureInfo holds metadata extracted from a function signature for heuristics.
type signatureInfo struct {
	NumParams        int
	NumResults       int
	HasContext       bool
	HasError         bool
	HasPointerResult bool
	HasSliceResult   bool
	HasArrayResult   bool
	HasChannelParam  bool
	HasChannelResult bool
	ReturnsInterface bool
}

// scanSinglePackage processes a single loaded package and returns its scan result.
func scanSinglePackage(pkg *packages.Package) (*ScanResult, error) {
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
			if !ok {
				continue
			}
			summary := buildFuncSummary(fn, pkg, aliases, sourceDir, filepath.Base(sourceFile))
			summary.PackageImportPath = pkg.PkgPath
			result.Funcs = append(result.Funcs, summary)
		}
	}

	return result, nil
}

func scannerInfoFromParams(info *signatureInfo, list []*ast.Field) {
	for _, field := range list {
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		info.NumParams += count

		typeStr := typeToString(field.Type)
		if typeStr == "context.Context" {
			info.HasContext = true
			continue
		}

		if strings.HasPrefix(typeStr, "chan ") ||
			strings.HasPrefix(typeStr, "<-chan ") ||
			strings.HasPrefix(typeStr, "chan<- ") {
			info.HasChannelParam = true
		}
	}
}

func scannerInfoFromResults(info *signatureInfo, pkg *packages.Package, list []*ast.Field) {
	for _, field := range list {
		typeStr := typeToString(field.Type)
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		info.NumResults += count

		switch {
		case typeStr == "error":
			info.HasError = true
		case strings.HasPrefix(typeStr, "*"):
			info.HasPointerResult = true
		case strings.HasPrefix(typeStr, "[]"):
			info.HasSliceResult = true
		case strings.HasPrefix(typeStr, "["):
			info.HasArrayResult = true
		case strings.HasPrefix(typeStr, "chan "),
			strings.HasPrefix(typeStr, "<-chan "),
			strings.HasPrefix(typeStr, "chan<- "):
			info.HasChannelResult = true
		}

		// Check if return type is an interface
		if pkg != nil && pkg.TypesInfo != nil {
			if tv, ok := pkg.TypesInfo.Types[field.Type]; ok {
				if _, isIface := tv.Type.Underlying().(*types.Interface); isIface {
					info.ReturnsInterface = true
				}
			}
		}
	}
}

// inspectSignature extracts basic signature metadata used for style suggestion heuristics.
func inspectSignature(fn *ast.FuncDecl, pkg *packages.Package) signatureInfo {
	var info signatureInfo
	if fn.Type.Params != nil {
		scannerInfoFromParams(&info, fn.Type.Params.List)
	}
	if fn.Type.Results != nil {
		scannerInfoFromResults(&info, pkg, fn.Type.Results.List)
	}
	return info
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
	sourceDir, sourceFile string,
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

	sig := inspectSignature(fn, pkg)

	summary := FuncSummary{
		Name:             fn.Name.Name,
		ReceiverType:     receiverType,
		IsMethod:         receiverType != "",
		IsExported:       fn.Name.IsExported(),
		Signature:        buildSignatureStr(fn, receiverType),
		FuncSpec:         funcSpec,
		TestFuncName:     testFuncName,
		SourceFile:       sourceFile,
		TestExists:       testFuncExistsInDir(sourceDir, testFuncName),
		HasContext:       sig.HasContext,
		NumParams:        sig.NumParams,
		NumResults:       sig.NumResults,
		HasError:         sig.HasError,
		HasPointerResult: sig.HasPointerResult,
		HasSliceResult:   sig.HasSliceResult,
		HasArrayResult:   sig.HasArrayResult,
		HasChannelParam:  sig.HasChannelParam,
		HasChannelResult: sig.HasChannelResult,
		ReturnsInterface: sig.ReturnsInterface,
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
// Adds underscore prefix if the name starts with lowercase for Go test recognition.
func deriveTestFuncName(fn *ast.FuncDecl, receiverType string) string {
	var base string
	if receiverType != "" {
		base = receiverType + "_" + fn.Name.Name
	} else if strings.HasPrefix(fn.Name.Name, "New") && fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		firstRet := typeToString(fn.Type.Results.List[0].Type)
		typeName := strings.TrimPrefix(firstRet, "*")
		if typeName != "" && typeName != firstRet {
			base = typeName + "_" + fn.Name.Name
		} else {
			base = fn.Name.Name
		}
	} else {
		base = fn.Name.Name
	}
	// If first letter is lowercase, prefix with underscore for Go test recognition
	if len(base) > 0 && base[0] >= 'a' && base[0] <= 'z' {
		return "Test_" + base
	}
	return "Test" + base
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

type interfaceDepsData struct {
	aliases   map[string]string
	pkg       *packages.Package
	seen      map[string]bool
	sourceDir string
	deps      []InterfaceDep
}

func newInterfaceDepsData(pkg *packages.Package, aliases map[string]string, sourceDir string) *interfaceDepsData {
	return &interfaceDepsData{
		pkg:       pkg,
		aliases:   aliases,
		seen:      make(map[string]bool),
		sourceDir: sourceDir,
	}
}

func (id *interfaceDepsData) add(strct *types.Struct) {
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
		if id.seen[typeName] {
			continue
		}
		id.seen[typeName] = true

		importPath := obj.Pkg().Path()
		qualifier := resolveQualifierByPath(importPath, obj.Pkg().Name(), id.aliases)

		mockFrom := qualifier + "." + typeName
		if qualifier == "" || qualifier == id.pkg.Name {
			mockFrom = typeName
		}

		mockFile := "mock_" + strings.ToLower(typeName) + "_test.go"
		mockPath := filepath.Join(id.sourceDir, mockFile)
		_, statErr := os.Stat(mockPath)

		id.deps = append(id.deps, InterfaceDep{
			TypeName:   typeName,
			Qualifier:  qualifier,
			ImportPath: importPath,
			MockFile:   mockFile,
			MockExists: statErr == nil,
			MockFrom:   mockFrom,
		})
	}
}

func getStruct(t types.Type) (*types.Struct, bool) {
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	if named, ok := t.(*types.Named); ok {
		if strct, ok := named.Underlying().(*types.Struct); ok {
			return strct, true
		}
	}

	return nil, false
}

// extractInterfaceDeps detects interface-typed struct fields from the receiver or
// constructor config param, returning deduplicated InterfaceDep entries.
// extractInterfaceDeps detects interface-typed struct fields from the receiver or
// constructor config param, returning deduplicated InterfaceDep entries.
func extractInterfaceDeps(fn *ast.FuncDecl, pkg *packages.Package, aliases map[string]string, sourceDir string) []InterfaceDep {
	data := newInterfaceDepsData(pkg, aliases, sourceDir)

	// Methods: inspect receiver struct fields.
	if fn.Recv != nil && len(fn.Recv.List) > 0 && pkg.TypesInfo != nil {
		recvTypeExpr := fn.Recv.List[0].Type
		if tv, ok := pkg.TypesInfo.Types[recvTypeExpr]; ok {
			if strct, ok := getStruct(tv.Type); ok {
				data.add(strct)
			}
		}
	}

	// Functions (constructors): inspect first non-context param if it's *Struct.
	if fn.Recv == nil && fn.Type.Params != nil && pkg.TypesInfo != nil {
		for _, param := range fn.Type.Params.List {
			if tv, ok := pkg.TypesInfo.Types[param.Type]; ok {
				if strct, ok := getStruct(tv.Type); ok {
					data.add(strct)
					break
				}
			}
		}
	}

	return data.deps
}
