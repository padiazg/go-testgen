package generator

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/padiazg/go-testgen/internal/analyzer"
	"github.com/padiazg/go-testgen/internal/config"
)

// GenerateRequest is the input to any TestGenerator implementation.
type GenerateRequest struct {
	Info    *analyzer.FuncInfo
	OutPkg  string
	IsMerge bool
}

// GenerateResult is the output of any TestGenerator implementation.
type GenerateResult struct {
	OutFile string
	Source  []byte
}

// CheckGenerator produces table-driven tests with check-function closures.
type CheckGenerator struct {
	cfg *config.Config
}

// Ensure CheckGenerator satisfies the TestGenerator interface at compile time.
var _ TestGenerator = (*CheckGenerator)(nil)

// NewCheckGenerator creates a CheckGenerator with the given config.
func NewCheckGenerator(cfg *config.Config) *CheckGenerator {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	return &CheckGenerator{cfg: cfg}
}

// New is a convenience alias that returns a TestGenerator using the check style.
// Kept for backward compatibility with existing callers.
func New(cfg *config.Config) TestGenerator {
	return NewCheckGenerator(cfg)
}

func (g *CheckGenerator) Generate(req GenerateRequest) (*GenerateResult, error) {
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

	if err := generateCheckType(&buf, checkTypeName, checkVarName, info); err != nil {
		return nil, fmt.Errorf("generate check type: %w", err)
	}

	if info.HasError {
		if err := generateCheckError(&buf, checkTypeName, info); err != nil {
			return nil, fmt.Errorf("generate check error: %w", err)
		}
	}

	if err := generateTestTable(&buf, checkTypeName, checkVarName, info, g.cfg); err != nil {
		return nil, fmt.Errorf("generate table: %w", err)
	}

	return &GenerateResult{
		Source:  buf.Bytes(),
		OutFile: deriveOutFile(info),
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

	return t.Execute(buf, map[string]any{
		"CheckTypeName": checkTypeName,
		"CheckVarName":  checkVarName,
		"Params":        strings.Join(paramList, ", "),
	})
}

func generateCheckError(buf *bytes.Buffer, checkTypeName string, info *analyzer.FuncInfo) error {
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

	return t.Execute(buf, map[string]any{
		"FuncName":      info.Name,
		"CheckTypeName": checkTypeName,
		"Params":        strings.Join(params, ", "),
		"ErrVar":        errVarName,
	})
}

func generateTestTable(buf *bytes.Buffer, checkTypeName, checkVarName string, info *analyzer.FuncInfo, cfg *config.Config) error {
	recvVar := receiverVar(info)
	constructor := isConstructor(info)

	hasResultVars := constructor || (info.IsMethod && info.Receiver != nil) || info.HasError || (len(info.Results) > 0 && !info.Results[0].IsError)

	tableFields := buildTableFields(info, "checks []"+checkTypeName)
	fieldList := strings.Join(tableFields, "\n\t")

	tfName := testFuncName(info)

	args := buildArgs(info)

	var (
		setupLines []string
		resultVars string
	)

	switch {
	case constructor:
		resultVarName := strings.ToLower(info.Results[0].TypeName[1:])
		setupLines = append(setupLines, fmt.Sprintf("%s := New(%s)", resultVarName, "tt."+info.Params[0].Name))
		resultVars = resultVarName

	case info.IsMethod:
		setupLines = append(setupLines, buildReceiverInit(info, recvVar))

		if len(info.Params) > 0 {
			setupLines = append(setupLines, fmt.Sprintf("if tt.before != nil {\n\t\t\ttt.before(%s)\n\t\t}", recvVar))
		}

		callExpr := recvVar + "." + info.Name + "(" + strings.Join(args, ", ") + ")"

		returnVars := buildReturnVars(info.Results, cfg.ResultVarName, cfg.ErrorVarName)
		if len(returnVars) > 0 {
			setupLines = append(setupLines, fmt.Sprintf("%s := %s", strings.Join(returnVars, ", "), callExpr))
			resultVars = strings.Join(returnVars, ", ")
		} else {
			setupLines = append(setupLines, callExpr)
		}

	default:
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

	return t.Execute(buf, map[string]any{
		"TestFuncName":  tfName,
		"FieldList":     fieldList,
		"CheckVarName":  checkVarName,
		"SetupBlock":    setupBlock,
		"ResultVars":    resultVars,
		"HasResultVars": hasResultVars,
	})
}
