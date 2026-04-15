package generator

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"text/template"

	"github.com/padiazg/testgen/internal/analyzer"
)

const mockTemplate = `package {{.Package}}

{{.Imports}}
// mock{{.Name}} implements {{.Qualifier}}.{{.Name}} for testing.
type mock{{.Name}} struct {
	mock.Mock
}
{{range .Methods}}
func (m *mock{{$.Name}}) {{.Name}}({{.ParamList}}) {{.ReturnList}} {
	args := m.Called({{.CallArgs}})
	{{.ReturnStmts}}
}
{{end}}`

// MockGenRequest describes what mock to generate.
type MockGenRequest struct {
	Info    *analyzer.InterfaceInfo
	PkgName string // package name for the generated file (consuming package's name)
}

// GenerateMock produces a complete testify mock for the given interface.
func GenerateMock(req MockGenRequest) ([]byte, error) {
	info := req.Info
	pkgName := req.PkgName

	type methodData struct {
		Name        string
		ParamList   string
		ReturnList  string
		CallArgs    string
		ReturnStmts string
	}

	methods := make([]methodData, 0, len(info.Methods))
	for _, m := range info.Methods {
		methods = append(methods, buildMethodData(m))
	}

	imports := buildMockImports(info, pkgName)

	t := template.Must(template.New("mock").Parse(mockTemplate))
	var buf bytes.Buffer
	err := t.Execute(&buf, map[string]interface{}{
		"Package":   pkgName,
		"Imports":   imports,
		"Name":      info.Name,
		"Qualifier": info.Qualifier,
		"Methods":   methods,
	})
	if err != nil {
		return nil, fmt.Errorf("render mock template: %w", err)
	}
	return buf.Bytes(), nil
}

// MockFileName returns the _test.go file name for a mock (e.g., "mock_userrepository_test.go").
func MockFileName(ifaceName string) string {
	return "mock_" + strings.ToLower(ifaceName) + "_test.go"
}

func buildMethodData(m analyzer.IfaceMethod) struct {
	Name        string
	ParamList   string
	ReturnList  string
	CallArgs    string
	ReturnStmts string
} {
	// ParamList: "ctx context.Context, req *userDomain.UserCreateRequest"
	var paramParts []string
	var callArgParts []string
	for i, p := range m.Params {
		name := p.Name
		if name == "" {
			name = fmt.Sprintf("p%d", i)
		}
		paramParts = append(paramParts, name+" "+p.TypeName)
		callArgParts = append(callArgParts, name)
	}

	// ReturnList: "(*userDomain.User, error)" or single "error"
	var returnTypes []string
	for _, r := range m.Results {
		returnTypes = append(returnTypes, r.TypeName)
	}
	returnList := ""
	if len(returnTypes) == 1 {
		returnList = returnTypes[0]
	} else if len(returnTypes) > 1 {
		returnList = "(" + strings.Join(returnTypes, ", ") + ")"
	}

	// ReturnStmts: assign each result from args, then return
	returnStmts := buildReturnStatements(m.Results)

	return struct {
		Name        string
		ParamList   string
		ReturnList  string
		CallArgs    string
		ReturnStmts string
	}{
		Name:        m.Name,
		ParamList:   strings.Join(paramParts, ", "),
		ReturnList:  returnList,
		CallArgs:    strings.Join(callArgParts, ", "),
		ReturnStmts: returnStmts,
	}
}

// buildReturnStatements generates the return body for a mock method.
// Uses comma-ok type assertions for pointer/slice types (nil-safe).
// Uses args.Error(i) for error results.
func buildReturnStatements(results []analyzer.MethodParam) string {
	if len(results) == 0 {
		return ""
	}

	// Simple single-error case: return args.Error(0)
	if len(results) == 1 && results[0].IsError {
		return "return args.Error(0)"
	}

	// Build individual variable assignments then a single return.
	var lines []string
	var retVars []string

	for i, r := range results {
		if r.IsError {
			varName := fmt.Sprintf("r%d", i)
			lines = append(lines, fmt.Sprintf("%s := args.Error(%d)", varName, i))
			retVars = append(retVars, varName)
			continue
		}

		varName := fmt.Sprintf("r%d", i)
		// Use comma-ok for pointer and slice types (nil-safe).
		if r.IsPointer || strings.HasPrefix(r.TypeName, "*") || strings.HasPrefix(r.TypeName, "[]") {
			lines = append(lines, fmt.Sprintf("%s, _ := args.Get(%d).(%s)", varName, i, r.TypeName))
		} else {
			lines = append(lines, fmt.Sprintf("%s := args.Get(%d).(%s)", varName, i, r.TypeName))
		}
		retVars = append(retVars, varName)
	}

	lines = append(lines, "return "+strings.Join(retVars, ", "))
	return strings.Join(lines, "\n\t")
}

// buildMockImports builds the import block for a mock file.
func buildMockImports(info *analyzer.InterfaceInfo, pkgName string) string {
	type entry struct {
		Path  string
		Alias string
	}

	seen := make(map[string]bool)
	var imports []entry

	add := func(path, alias string) {
		if path == "" || seen[path] {
			return
		}
		seen[path] = true
		// Only emit alias if it differs from the last path segment.
		parts := strings.Split(path, "/")
		defName := parts[len(parts)-1]
		if alias == defName {
			alias = ""
		}
		imports = append(imports, entry{Path: path, Alias: alias})
	}

	// Always need testify/mock.
	add("github.com/stretchr/testify/mock", "")

	for _, m := range info.Methods {
		for _, p := range append(m.Params, m.Results...) {
			if p.ImportPath != "" {
				add(p.ImportPath, p.Package)
			}
		}
	}

	slices.SortFunc(imports, func(a, b entry) int {
		return strings.Compare(a.Path, b.Path)
	})

	var lines []string
	lines = append(lines, "import (")
	for _, imp := range imports {
		if imp.Alias != "" {
			lines = append(lines, fmt.Sprintf("\t%s %q", imp.Alias, imp.Path))
		} else {
			lines = append(lines, fmt.Sprintf("\t%q", imp.Path))
		}
	}
	lines = append(lines, ")")
	return strings.Join(lines, "\n") + "\n"
}
