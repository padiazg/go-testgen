package generator

import (
	"bytes"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/padiazg/testgen/internal/analyzer"
	"github.com/padiazg/testgen/internal/config"
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

type Generator struct {
	cfg *config.Config
}

type GenerateRequest struct {
	Info    *analyzer.FuncInfo
	OutPkg  string
	IsMerge bool
}

type GenerateResult struct {
	Source  []byte
	OutFile string
}

func New(cfg *config.Config) *Generator {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	return &Generator{cfg: cfg}
}

func (g *Generator) Generate(req GenerateRequest) (*GenerateResult, error) {
	info := req.Info
	if info == nil {
		return nil, fmt.Errorf("FuncInfo is nil")
	}

	var buf bytes.Buffer

	if !req.IsMerge {
		buf.WriteString("package " + info.Package + "\n\n")
		buf.WriteString(generateImports(info))
	}

	checkTypeName := g.cfg.CheckTypePrefix + info.Name + g.cfg.CheckTypeSuffix
	checkVarName := "check" + info.Name

	if info.IsMethod && info.Receiver != nil {
		checkTypeName = "check" + info.Receiver.TypeName + info.Name + "Fn"
		checkVarName = "check" + info.Receiver.TypeName + info.Name
	} else if len(info.Results) > 0 && !info.Results[0].IsError {
		checkTypeName = info.Name + "Fn"
		checkVarName = "check" + info.Name
	}

	err := generateCheckType(&buf, checkTypeName, checkVarName, info)
	if err != nil {
		return nil, fmt.Errorf("generate check type: %w", err)
	}

	if info.HasError {
		err = generateCheckError(&buf, checkTypeName, info)
		if err != nil {
			return nil, fmt.Errorf("generate check error: %w", err)
		}
	}

	err = generateTestTable(&buf, checkTypeName, checkVarName, info, g.cfg)
	if err != nil {
		return nil, fmt.Errorf("generate table: %w", err)
	}

	var outFile string
	if info.SourceFile != "" {
		dir := filepath.Dir(info.SourceFile)
		base := filepath.Base(info.SourceFile)
		name := strings.TrimSuffix(base, filepath.Ext(base))
		outFile = filepath.Join(dir, name+"_test.go")
	} else {
		outFile = info.Name + "_test.go"
		if info.IsMethod && info.Receiver != nil {
			outFile = strings.ToLower(info.Receiver.TypeName) + "_test.go"
		}
	}

	return &GenerateResult{
		Source:  buf.Bytes(),
		OutFile: outFile,
	}, nil
}

func generateCheckType(buf *bytes.Buffer, checkTypeName, checkVarName string, info *analyzer.FuncInfo) error {
	var paramList []string
	paramList = append(paramList, "*testing.T")

	for _, res := range info.Results {
		paramList = append(paramList, qualifiedTypeName(res.TypeName, res.Package))
	}

	t := template.Must(template.New("checkType").Parse(`type {{.CheckTypeName}} func({{.Params}})

var {{.CheckVarName}} = func(fns ...{{.CheckTypeName}}) []{{.CheckTypeName}} { return fns }
`))

	return t.Execute(buf, map[string]interface{}{
		"CheckTypeName": checkTypeName,
		"CheckVarName":  checkVarName,
		"Params":        strings.Join(paramList, ", "),
	})
}

func generateCheckError(buf *bytes.Buffer, checkTypeName string, info *analyzer.FuncInfo) error {
	// Build params matching the checkFn signature: *testing.T + all results
	// Non-error results use _ to avoid unused-variable errors.
	var params []string
	params = append(params, "t *testing.T")
	errVarName := "err"
	for _, r := range info.Results {
		typeName := qualifiedTypeName(r.TypeName, r.Package)
		if r.IsError {
			params = append(params, errVarName+" "+typeName)
		} else {
			params = append(params, "_ "+typeName)
		}
	}

	t := template.Must(template.New("checkError").Parse(`
func check{{.FuncName}}Error(want string) {{.CheckTypeName}} {
	return func({{.Params}}) {
		t.Helper()
		if want == "" {
			assert.NoErrorf(t, {{.ErrVar}}, "check{{.FuncName}}Error: expected no error, got %v", {{.ErrVar}})
			return
		}
		if assert.Errorf(t, {{.ErrVar}}, "check{{.FuncName}}Error: expected error %q", want) {
			assert.Containsf(t, {{.ErrVar}}.Error(), want, "check{{.FuncName}}Error mismatch")
		}
	}
}
`))

	return t.Execute(buf, map[string]interface{}{
		"FuncName":      info.Name,
		"CheckTypeName": checkTypeName,
		"Params":        strings.Join(params, ", "),
		"ErrVar":        errVarName,
	})
}

func generateTestTable(buf *bytes.Buffer, checkTypeName, checkVarName string, info *analyzer.FuncInfo, cfg *config.Config) error {
	receiverVar := "e"
	if info.IsMethod && info.Receiver != nil {
		receiverVar = strings.ToLower(info.Receiver.TypeName[:1])
	}

	isConstructor := !info.IsMethod && info.Name == "New" && len(info.Results) > 0 && info.Results[0].IsPointer

	hasResultVars := isConstructor || (info.IsMethod && info.Receiver != nil) || info.HasError || (len(info.Results) > 0 && !info.Results[0].IsError)

	tableFields := []string{"name string"}

	if info.IsMethod {
		if len(info.Params) == 0 {
			tableFields = append(tableFields, "config *Config")
		} else {
			for _, p := range info.Params {
				if p.IsContext {
					continue
				}
				tableFields = append(tableFields, fmt.Sprintf("%s %s", p.Name, qualifiedTypeName(p.TypeName, p.Package)))
			}
		}
	} else {
		for _, p := range info.Params {
			if p.IsContext {
				continue
			}
			tableFields = append(tableFields, fmt.Sprintf("%s %s", p.Name, qualifiedTypeName(p.TypeName, p.Package)))
		}
	}

	// if info.HasError {
	// 	tableFields = append(tableFields, "wantErr bool")
	// }

	if info.IsMethod {
		tableFields = append(tableFields, fmt.Sprintf("before func(*%s)", info.Receiver.TypeName))
	}

	tableFields = append(tableFields, "checks []"+checkTypeName)
	fieldList := strings.Join(tableFields, "\n\t")

	testFuncName := info.Name

	if info.IsMethod {
		testFuncName = info.Receiver.TypeName + "_" + info.Name
	} else if isConstructor {
		testFuncName = strings.TrimPrefix(info.Results[0].TypeName, "*") + "_" + info.Name
	}

	var setupLines []string
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

	var resultVars string

	if isConstructor {
		resultVarName := strings.ToLower(info.Results[0].TypeName[1:])
		setupLines = append(setupLines, fmt.Sprintf("%s := New(%s)", resultVarName, "tt."+info.Params[0].Name))
		resultVars = resultVarName
	} else if info.IsMethod {
		setupLines = append(setupLines, fmt.Sprintf("%s := New(nil)", receiverVar))

		if len(info.Params) > 0 {
			setupLines = append(setupLines, fmt.Sprintf("if tt.before != nil {\n\t\t\ttt.before(%s)\n\t\t}", receiverVar))
		}

		callExpr := receiverVar + "." + info.Name + "(" + strings.Join(args, ", ") + ")"

		returnVars := buildReturnVars(info.Results, cfg.ResultVarName, cfg.ErrorVarName)
		if len(returnVars) > 0 {
			setupLines = append(setupLines, fmt.Sprintf("%s := %s", strings.Join(returnVars, ", "), callExpr))
			resultVars = strings.Join(returnVars, ", ")
		} else {
			setupLines = append(setupLines, callExpr)
		}
	} else {
		callExpr := info.Name + "(" + strings.Join(args, ", ") + ")"

		returnVars := buildReturnVars(info.Results, cfg.ResultVarName, cfg.ErrorVarName)
		if len(returnVars) > 0 {
			setupLines = append(setupLines, fmt.Sprintf("%s := %s", strings.Join(returnVars, ", "), callExpr))
			resultVars = strings.Join(returnVars, ", ")
		} else {
			setupLines = append(setupLines, callExpr)
		}
	}

	setupBlock := strings.Join(setupLines, "\n\t\t\t")

	t := template.Must(template.New("testTable").Parse(`func Test{{.TestFuncName}}(t *testing.T) {
	tests := []struct {
		{{.FieldList}}
	}{
		{
			name: "TODO: success case",
			checks: {{.CheckVarName}}(
			),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			{{.SetupBlock}}
			{{if .HasResultVars}}for _, c := range tt.checks {
				c(t, {{.ResultVars}})
			}{{else}}for _, c := range tt.checks {
				c(t)
			}{{end}}
		})
	}
}
`))

	return t.Execute(buf, map[string]interface{}{
		"TestFuncName":  testFuncName,
		"FieldList":     fieldList,
		"CheckVarName":  checkVarName,
		"SetupBlock":    setupBlock,
		"ResultVars":    resultVars,
		"HasResultVars": hasResultVars,
	})
}

func generateImports(info *analyzer.FuncInfo) string {
	type importEntry struct {
		Path  string
		Alias string // empty = no explicit alias
	}

	var imports []importEntry
	seen := make(map[string]bool)

	addImport := func(importPath, pkgAlias string) {
		if importPath == "" || seen[importPath] || importPath == info.ImportPath || importPath == "context" {
			return
		}
		seen[importPath] = true

		// Determine if an explicit alias is needed:
		// only emit alias when it differs from the package's own name.
		alias := ""
		if pkgAlias != "" {
			// Resolve the package's actual name via ImportAliases (set by analyzer).
			// If the source already used an alias, pkgAlias IS that alias.
			// Check if it differs from the last path segment (Go default import name).
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

	var lines []string
	lines = append(lines, "import (")
	lines = append(lines, "\t\"testing\"")

	if info.HasContext {
		lines = append(lines, "\n\t\"context\"")
	}

	if info.HasError {
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
