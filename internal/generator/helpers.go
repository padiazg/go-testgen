package generator

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/padiazg/go-testgen/internal/analyzer"
)

// CollectImports returns the map of importPath -> alias needed for the generated test.
// Used by callers to inject imports into an existing file during merge.
func CollectImports(info *analyzer.FuncInfo) map[string]string {
	result := make(map[string]string)

	add := func(importPath, pkgAlias string) {
		if importPath == "" || importPath == "context" {
			return
		}
		// When merging into X_test, also import the source package.
		if importPath == info.ImportPath && info.TargetPkg != "" && info.TargetPkg != info.Package {
			parts := strings.Split(importPath, "/")
			result[importPath] = parts[len(parts)-1]
			return
		}
		if importPath == info.ImportPath {
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
// Handles pointer (*), slice ([]), and array [N] prefixes correctly.
func qualifiedTypeName(typeName, pkgQualifier string) string {
	if pkgQualifier == "" {
		return typeName
	}

	// Channel prefixes (order matters — longer prefixes first)
	for _, prefix := range []string{
		"<-chan []*", "<-chan []", "<-chan *", "<-chan ",
		"chan<- []*", "chan<- []", "chan<- *", "chan<- ",
		"chan []*", "chan []", "chan *", "chan ",
	} {
		if strings.HasPrefix(typeName, prefix) {
			return prefix + pkgQualifier + "." + typeName[len(prefix):]
		}
	}

	switch {
	case strings.HasPrefix(typeName, "*"):
		return "*" + pkgQualifier + "." + typeName[1:]
	case strings.HasPrefix(typeName, "[]*"):
		return "[]*" + pkgQualifier + "." + typeName[3:]
	case strings.HasPrefix(typeName, "[]"):
		return "[]" + pkgQualifier + "." + typeName[2:]
	case strings.HasPrefix(typeName, "["):
		bracketIdx := strings.Index(typeName, "]")
		if bracketIdx > 0 {
			elemType := typeName[bracketIdx+1:]
			if pkgQualifier == "" {
				return typeName
			}
			dotIdx := strings.Index(elemType, ".")
			if dotIdx > 0 {
				return typeName[:bracketIdx+1] + pkgQualifier + "." + elemType[dotIdx+1:]
			}
			return typeName[:bracketIdx+1] + pkgQualifier + "." + elemType
		}
		return typeName
	default:
		return pkgQualifier + "." + typeName
	}
}

// buildReturnVars builds the list of variable names for capturing function return values.
// Multiple non-error results get distinct names (r, r2, r3...).
// Additional var names can be passed to skip collision (e.g. "err" already used by factory).
func buildReturnVars(results []analyzer.ResultInfo, resultVarName, errorVarName string, skipNames ...string) []string {
	var vars []string
	nonErrIdx := 0
	errVarName := errorVarName
	for _, n := range skipNames {
		if n == "err" {
			errVarName = "err2"
		}
	}
	for _, r := range results {
		if r.IsError {
			vars = append(vars, errVarName)
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

// qualifyForExternalTest qualifies a same-package type reference for use in an X_test package.
// E.g., "ProductRepository" -> "database.ProductRepository" when TargetPkg == "database_test"
// and infoPkg == "database". Returns the original typeName if no qualification needed.
func qualifyForExternalTest(typeName, pkgQualifier, infoPkg, targetPkg string) string {
	if targetPkg == "" || targetPkg == infoPkg {
		return typeName
	}
	if pkgQualifier != "" {
		return typeName // already qualified via import alias
	}
	return qualifiedTypeName(typeName, infoPkg)
}

// generateImports builds the full import block string for a new test file.
func generateImports(info *analyzer.FuncInfo) string {
	imports := info.GetImports()
	hasNonErrorResults := info.HasNonErrorResults()

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
		name := "tt." + p.Name
		if name == "tt._" {
			name = placeholderValue(p.TypeName)
		}
		args = append(args, name)
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
	case strings.HasPrefix(typeName, "["):
		bracketIdx := strings.Index(typeName, "]")
		if bracketIdx > 0 {
			return typeName + "{}"
		}
		return "nil"
	case strings.HasPrefix(typeName, "[]"):
		return "nil"
	case strings.HasPrefix(typeName, "chan "),
		strings.HasPrefix(typeName, "<-chan "),
		strings.HasPrefix(typeName, "chan<- "):
		return "nil"
	case typeName == "error":
		return "nil"
	case strings.HasPrefix(typeName, "float"):
		return "0"
	case strings.HasPrefix(typeName, "uint"):
		return "0"
	case strings.HasPrefix(typeName, "byte") || typeName == "rune":
		return "0"
	case strings.HasPrefix(typeName, "uintptr"):
		return "0"
	default:
		return "nil"
	}
}

// buildTableFields builds the struct field list for a table-driven test.
// Skips context params; adds before func for methods; adds factory params for methods with factories.
func buildTableFields(info *analyzer.FuncInfo, extraFields ...string) []string {
	fields := []string{"name string"}

	qualify := func(typeName, pkgQualifier string) string {
		return qualifyForExternalTest(typeName, pkgQualifier, info.Package, info.TargetPkg)
	}

	if info.IsMethod {
		for _, p := range info.Params {
			if p.IsContext {
				continue
			}
			fields = append(fields, fmt.Sprintf("%s %s", p.Name, qualify(p.TypeName, p.Package)))
		}
	} else {
		for _, p := range info.Params {
			if p.IsContext {
				continue
			}
			fields = append(fields, fmt.Sprintf("%s %s", p.Name, qualify(p.TypeName, p.Package)))
		}
	}

	fields = append(fields, extraFields...)

	if info.IsMethod && info.FactoryFunc != "" {
		for _, p := range info.FactoryParams {
			if p.IsContext {
				continue
			}
			fields = append(fields, fmt.Sprintf("%s %s", p.Name, qualify(p.TypeName, p.Package)))
		}
	}

	if info.IsMethod {
		recvQual := qualify(info.Receiver.TypeName, "")
		fields = append(fields, fmt.Sprintf("before func(*%s)", recvQual))
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
		return "s"
	}
	return "e"
}

// receiverParamType returns the check-fn parameter type for the receiver.
func receiverParamType(info *analyzer.FuncInfo) string {
	recv := qualifyForExternalTest(info.Receiver.TypeName, "", info.Package, info.TargetPkg)
	if info.Receiver.IsPointer {
		return "*" + recv
	}
	return recv
}

// buildReceiverInit returns the code to instantiate the receiver for a method test.
// skipNames are variable names that are already used and should be skipped to avoid collisions
// (e.g. "err" already declared by a subsequent method call).
func buildReceiverInit(info *analyzer.FuncInfo, varName string, skipNames ...string) string {
	recvType := info.Receiver.TypeName
	// Check if "err" should be skipped.
	errInUse := false
	for _, n := range skipNames {
		if n == "err" {
			errInUse = true
			break
		}
	}

	// Qualify receiver/factory for X_test packages.
	recvQual := qualifyForExternalTest(recvType, "", info.Package, info.TargetPkg)
	factoryQual := info.FactoryFunc
	if info.TargetPkg != "" && info.TargetPkg != info.Package && info.FactoryFunc != "" {
		factoryQual = qualifiedTypeName(info.FactoryFunc, info.Package)
	}

	if info.FactoryFunc != "" {
		// Use table fields for factory params when they're exposed (non-context params).
		var args []string
		for _, p := range info.FactoryParams {
			if p.IsContext {
				continue
			}
			args = append(args, "tt."+p.Name)
		}
		argList := strings.Join(args, ", ")
		if info.FactoryReturnsError && !errInUse {
			return fmt.Sprintf("%s, err := %s(%s)", varName, factoryQual, argList)
		}
		return fmt.Sprintf("%s := %s(%s)", varName, factoryQual, argList)
	}
	if info.Receiver.IsPointer {
		return fmt.Sprintf("%s := &%s{}", varName, recvQual)
	}
	if info.Receiver.Kind == "basic" {
		return fmt.Sprintf("%s := %s(0)", varName, recvQual)
	}
	return fmt.Sprintf("%s := %s{}", varName, recvQual)
}
