package generator

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/padiazg/go-testgen/internal/analyzer"
)

// CollectImports returns the map of importPath -> alias needed for the generated test.
// Used by callers to inject imports into an existing file during merge.
func CollectImports(info *analyzer.FuncInfo) map[string]string {
	result := make(map[string]string)

	add := func(importPath, pkgAlias string) {
		if importPath == "" || importPath == info.ImportPath || importPath == "context" {
			return
		}
		alias := ""
		if pkgAlias != "" {
			parts := strings.Split(importPath, "/")
			if pkgAlias != parts[len(parts)-1] {
				alias = pkgAlias
			}
		}
		result[importPath] = alias
	}

	if info.HasContext {
		result["context"] = ""
	}
	if info.HasError {
		result["github.com/stretchr/testify/assert"] = ""
	}

	for _, p := range info.Params {
		add(p.ImportPath, p.Package)
	}
	for _, r := range info.Results {
		if !r.IsError {
			add(r.ImportPath, r.Package)
		}
	}

	return result
}

// qualifiedTypeName prepends pkgQualifier to typeName when the type is from an external package.
// Handles pointer (*) and slice ([]) prefixes correctly.
func qualifiedTypeName(typeName, pkgQualifier string) string {
	if pkgQualifier == "" {
		return typeName
	}
	if strings.HasPrefix(typeName, "*") {
		return "*" + pkgQualifier + "." + typeName[1:]
	}
	if strings.HasPrefix(typeName, "[]*") {
		return "[]*" + pkgQualifier + "." + typeName[3:]
	}
	if strings.HasPrefix(typeName, "[]") {
		return "[]" + pkgQualifier + "." + typeName[2:]
	}
	return pkgQualifier + "." + typeName
}

// buildReturnVars builds the list of variable names for capturing function return values.
// Multiple non-error results get distinct names (r, r2, r3...).
func buildReturnVars(results []analyzer.ResultInfo, resultVarName, errorVarName string) []string {
	var vars []string
	nonErrIdx := 0
	for _, r := range results {
		if r.IsError {
			vars = append(vars, errorVarName)
		} else {
			if nonErrIdx == 0 {
				vars = append(vars, resultVarName)
			} else {
				vars = append(vars, fmt.Sprintf("%s%d", resultVarName, nonErrIdx+1))
			}
			nonErrIdx++
		}
	}
	return vars
}

// deriveOutFile returns the _test.go path for a given FuncInfo.
func deriveOutFile(info *analyzer.FuncInfo) string {
	if info.SourceFile != "" {
		dir := filepath.Dir(info.SourceFile)
		base := filepath.Base(info.SourceFile)
		name := strings.TrimSuffix(base, filepath.Ext(base))
		return filepath.Join(dir, name+"_test.go")
	}
	if info.IsMethod && info.Receiver != nil {
		return strings.ToLower(info.Receiver.TypeName) + "_test.go"
	}
	return info.Name + "_test.go"
}

// generateImports builds the full import block string for a new test file.
func generateImports(info *analyzer.FuncInfo) string {
	type importEntry struct {
		Path  string
		Alias string
	}

	var imports []importEntry
	seen := make(map[string]bool)

	addImport := func(importPath, pkgAlias string) {
		if importPath == "" || seen[importPath] || importPath == info.ImportPath || importPath == "context" {
			return
		}
		seen[importPath] = true

		alias := ""
		if pkgAlias != "" {
			parts := strings.Split(importPath, "/")
			defaultName := parts[len(parts)-1]
			if pkgAlias != defaultName {
				alias = pkgAlias
			}
		}
		imports = append(imports, importEntry{Path: importPath, Alias: alias})
	}

	for _, p := range info.Params {
		addImport(p.ImportPath, p.Package)
	}
	for _, r := range info.Results {
		if !r.IsError {
			addImport(r.ImportPath, r.Package)
		}
	}

	slices.SortFunc(imports, func(a, b importEntry) int {
		return strings.Compare(a.Path, b.Path)
	})

	hasNonErrorResults := false
	for _, r := range info.Results {
		if !r.IsError {
			hasNonErrorResults = true
			break
		}
	}

	var lines []string
	lines = append(lines, "import (")
	lines = append(lines, "\t\"testing\"")

	if info.HasContext {
		lines = append(lines, "\n\t\"context\"")
	}
	if info.HasError || hasNonErrorResults {
		lines = append(lines, "\n\t\"github.com/stretchr/testify/assert\"")
	}

	for _, imp := range imports {
		if imp.Alias != "" {
			lines = append(lines, fmt.Sprintf("\n\t%s \"%s\"", imp.Alias, imp.Path))
		} else {
			lines = append(lines, "\n\t\""+imp.Path+"\"")
		}
	}

	lines = append(lines, ")\n\n")
	return strings.Join(lines, "\n")
}

// buildArgs builds the call argument list for a function call in generated test code.
// Uses "tt." prefix for table-driven tests.
func buildArgs(info *analyzer.FuncInfo) []string {
	var args []string
	if info.HasContext {
		args = append(args, "context.Background()")
	}
	for _, p := range info.Params {
		if p.IsContext {
			continue
		}
		args = append(args, "tt."+p.Name)
	}
	return args
}

// buildSimpleArgs builds placeholder arguments for simple style tests.
func buildSimpleArgs(info *analyzer.FuncInfo) []string {
	var args []string
	if info.HasContext {
		args = append(args, "context.Background()")
	}
	for _, p := range info.Params {
		if p.IsContext {
			continue
		}
		// Use placeholder value based on type
		arg := placeholderValue(p.TypeName)
		args = append(args, arg)
	}
	return args
}

// placeholderValue returns a placeholder value for a given type.
func placeholderValue(typeName string) string {
	switch {
	case strings.HasPrefix(typeName, "string"):
		return `"value"`
	case strings.HasPrefix(typeName, "int"):
		return "0"
	case strings.HasPrefix(typeName, "bool"):
		return "false"
	case strings.HasPrefix(typeName, "[]"):
		return "nil"
	case typeName == "error":
		return "nil"
	default:
		return "nil"
	}
}

// buildTableFields builds the struct field list for a table-driven test.
// Skips context params; adds before func for methods.
func buildTableFields(info *analyzer.FuncInfo, extraFields ...string) []string {
	fields := []string{"name string"}

	if info.IsMethod {
		for _, p := range info.Params {
			if p.IsContext {
				continue
			}
			fields = append(fields, fmt.Sprintf("%s %s", p.Name, qualifiedTypeName(p.TypeName, p.Package)))
		}
	} else {
		for _, p := range info.Params {
			if p.IsContext {
				continue
			}
			fields = append(fields, fmt.Sprintf("%s %s", p.Name, qualifiedTypeName(p.TypeName, p.Package)))
		}
	}

	fields = append(fields, extraFields...)

	if info.IsMethod {
		fields = append(fields, fmt.Sprintf("before func(*%s)", info.Receiver.TypeName))
	}

	return fields
}

// isConstructor returns true when info represents a New() constructor.
func isConstructor(info *analyzer.FuncInfo) bool {
	return !info.IsMethod && info.Name == "New" && len(info.Results) > 0 && info.Results[0].IsPointer
}

// testFuncName derives the test function name from a FuncInfo.
// Adds underscore prefix if the name starts with lowercase for Go test to recognize it.
func testFuncName(info *analyzer.FuncInfo) string {
	var base string
	if info.IsMethod {
		base = info.Receiver.TypeName + "_" + info.Name
	} else if isConstructor(info) {
		base = strings.TrimPrefix(info.Results[0].TypeName, "*") + "_" + info.Name
	} else {
		base = info.Name
	}
	// If first letter is lowercase, prefix with underscore for Go test recognition
	if len(base) > 0 && base[0] >= 'a' && base[0] <= 'z' {
		return "_" + base
	}
	return base
}

// receiverVar returns a one-letter variable name for the receiver.
func receiverVar(info *analyzer.FuncInfo) string {
	if info.IsMethod && info.Receiver != nil && len(info.Receiver.TypeName) > 0 {
		// return strings.ToLower(info.Receiver.TypeName[:1])
		return "s"
	}
	return "e"
}

// buildReceiverInit returns the code to instantiate the receiver for a method test.
func buildReceiverInit(info *analyzer.FuncInfo, varName string) string {
	recvType := info.Receiver.TypeName
	if info.FactoryFunc != "" {
		// Build arguments using placeholder values based on param types
		var args []string
		for _, p := range info.FactoryParams {
			if p.IsContext {
				continue
			}
			args = append(args, placeholderValue(p.TypeName))
		}
		argList := strings.Join(args, ", ")
		return fmt.Sprintf("%s := %s(%s)", varName, info.FactoryFunc, argList)
	}
	if info.Receiver.IsPointer {
		return fmt.Sprintf("%s := &%s{}", varName, recvType)
	}
	return fmt.Sprintf("%s := %s{}", varName, recvType)
}
