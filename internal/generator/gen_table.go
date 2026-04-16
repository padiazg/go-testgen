package generator

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/padiazg/go-testgen/internal/analyzer"
	"github.com/padiazg/go-testgen/internal/config"
)

// TableGenerator produces classic table-driven tests with a `want` field.
type TableGenerator struct {
	cfg *config.Config
}

// Ensure TableGenerator satisfies the TestGenerator interface at compile time.
var _ TestGenerator = (*TableGenerator)(nil)

// NewTableGenerator creates a TableGenerator with the given config.
func NewTableGenerator(cfg *config.Config) *TableGenerator {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	return &TableGenerator{cfg: cfg}
}

func (g *TableGenerator) Generate(req GenerateRequest) (*GenerateResult, error) {
	info := req.Info
	if info == nil {
		return nil, fmt.Errorf("FuncInfo is nil")
	}

	var buf bytes.Buffer

	if !req.IsMerge {
		buf.WriteString("package " + info.Package + "\n\n")
		buf.WriteString(generateImports(info))
	}

	if err := g.generateTableTest(&buf, info); err != nil {
		return nil, fmt.Errorf("generate table test: %w", err)
	}

	return &GenerateResult{
		Source:  buf.Bytes(),
		OutFile: deriveOutFile(info),
	}, nil
}

func (g *TableGenerator) generateTableTest(buf *bytes.Buffer, info *analyzer.FuncInfo) error {
	recvVar := receiverVar(info)
	constructor := isConstructor(info)
	tfName := testFuncName(info)

	// Determine want fields and assertion logic based on return values.
	wantFields, assertBlock := g.buildWantFieldsAndAssert(info)

	tableFields := buildTableFields(info, wantFields...)
	fieldList := strings.Join(tableFields, "\n\t\t")

	args := buildArgs(info)

	var setupLines []string
	var callLine string

	if constructor {
		resultVarName := strings.ToLower(info.Results[0].TypeName[1:])
		setupLines = append(setupLines, fmt.Sprintf("%s := New(%s)", resultVarName, "tt."+info.Params[0].Name))
	} else if info.IsMethod {
		setupLines = append(setupLines, buildReceiverInit(info, recvVar))
		if len(info.Params) > 0 {
			setupLines = append(setupLines, fmt.Sprintf("if tt.before != nil {\n\t\t\ttt.before(%s)\n\t\t}", recvVar))
		}
		callLine = g.buildCallLine(info, recvVar+"."+info.Name, args)
	} else {
		callLine = g.buildCallLine(info, info.Name, args)
	}

	if callLine != "" {
		setupLines = append(setupLines, callLine)
	}

	setupBlock := strings.Join(setupLines, "\n\t\t\t")

	t := template.Must(template.New("tableTest").Parse(`func Test{{.TestFuncName}}(t *testing.T) {
	tests := []struct {
		{{.FieldList}}
	}{
		{name: "TODO: success case"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			{{.SetupBlock}}
			{{.AssertBlock}}
		})
	}
}
`))

	return t.Execute(buf, map[string]any{
		"TestFuncName": tfName,
		"FieldList":    fieldList,
		"SetupBlock":   setupBlock,
		"AssertBlock":  assertBlock,
	})
}

// buildWantFieldsAndAssert returns the extra struct fields for want values and the assertion code.
func (g *TableGenerator) buildWantFieldsAndAssert(info *analyzer.FuncInfo) (fields []string, assertBlock string) {
	cfg := g.cfg
	resultVarName := cfg.ResultVarName
	errVarName := cfg.ErrorVarName

	var nonErrResults []analyzer.ResultInfo
	for _, r := range info.Results {
		if !r.IsError {
			nonErrResults = append(nonErrResults, r)
		}
	}

	switch {
	case len(info.Results) == 0:
		// No return values — nothing to assert.
		assertBlock = "// no return values"

	case len(info.Results) == 1 && info.Results[0].IsError:
		// Only an error return.
		fields = append(fields, "wantErr string")
		assertBlock = fmt.Sprintf(
			"if tt.wantErr != \"\" {\n\t\t\t\tif assert.Error(t, %s) { assert.Contains(t, %s.Error(), tt.wantErr) }\n\t\t\t} else {\n\t\t\t\tassert.NoError(t, %s)\n\t\t\t}",
			errVarName, errVarName, errVarName,
		)

	case len(nonErrResults) == 1 && !info.HasError:
		// Single non-error result.
		r := nonErrResults[0]
		wantType := qualifiedTypeName(r.TypeName, r.Package)
		fields = append(fields, "want "+wantType)
		assertBlock = fmt.Sprintf("assert.Equal(t, tt.want, %s)", resultVarName)

	default:
		// Mixed: one or more non-error results + optional error.
		for i, r := range nonErrResults {
			wantType := qualifiedTypeName(r.TypeName, r.Package)
			if i == 0 {
				fields = append(fields, "want "+wantType)
			} else {
				fields = append(fields, fmt.Sprintf("want%d %s", i+1, wantType))
			}
		}
		if info.HasError {
			fields = append(fields, "wantErr string")
		}

		var asserts []string
		for i := range nonErrResults {
			varName := resultVarName
			wantField := "tt.want"
			if i > 0 {
				varName = fmt.Sprintf("%s%d", resultVarName, i+1)
				wantField = fmt.Sprintf("tt.want%d", i+1)
			}
			asserts = append(asserts, fmt.Sprintf("assert.Equal(t, %s, %s)", wantField, varName))
		}
		if info.HasError {
			asserts = append(asserts, fmt.Sprintf(
				"if tt.wantErr != \"\" {\n\t\t\t\tif assert.Error(t, %s) { assert.Contains(t, %s.Error(), tt.wantErr) }\n\t\t\t} else {\n\t\t\t\tassert.NoError(t, %s)\n\t\t\t}",
				errVarName, errVarName, errVarName,
			))
		}
		assertBlock = strings.Join(asserts, "\n\t\t\t")
	}

	return fields, assertBlock
}

// buildCallLine generates the assignment line for capturing return values.
func (g *TableGenerator) buildCallLine(info *analyzer.FuncInfo, callExpr string, args []string) string {
	call := callExpr + "(" + strings.Join(args, ", ") + ")"
	returnVars := buildReturnVars(info.Results, g.cfg.ResultVarName, g.cfg.ErrorVarName)
	if len(returnVars) == 0 {
		return call
	}
	return strings.Join(returnVars, ", ") + " := " + call
}
