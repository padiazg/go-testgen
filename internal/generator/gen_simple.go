package generator

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/padiazg/go-testgen/internal/analyzer"
	"github.com/padiazg/go-testgen/internal/config"
)

// SimpleGenerator produces standalone test functions (no table) in AAA style.
type SimpleGenerator struct {
	cfg *config.Config
}

// Ensure SimpleGenerator satisfies the TestGenerator interface at compile time.
var _ TestGenerator = (*SimpleGenerator)(nil)

// NewSimpleGenerator creates a SimpleGenerator with the given config.
func NewSimpleGenerator(cfg *config.Config) *SimpleGenerator {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	return &SimpleGenerator{cfg: cfg}
}

func (g *SimpleGenerator) Generate(req GenerateRequest) (*GenerateResult, error) {
	info := req.Info
	if info == nil {
		return nil, fmt.Errorf("FuncInfo is nil")
	}

	var buf bytes.Buffer

	if !req.IsMerge {
		buf.WriteString("package " + info.Package + "\n\n")
		buf.WriteString(generateImports(info))
	}

	if err := g.generateSimpleTest(&buf, info, "Test"+testFuncName(info)); err != nil {
		return nil, fmt.Errorf("generate simple test: %w", err)
	}

	return &GenerateResult{
		Source:  buf.Bytes(),
		OutFile: deriveOutFile(info),
	}, nil
}

func (g *SimpleGenerator) generateSimpleTest(buf *bytes.Buffer, info *analyzer.FuncInfo, funcName string) error {
	recvVar := receiverVar(info)
	constructor := isConstructor(info)
	cfg := g.cfg

	var arrangeLine string
	var actBlock string
	var assertBlock string

	args := buildSimpleArgs(info)
	callArgs := strings.Join(args, ", ")

	if constructor {
		resultVarName := strings.ToLower(info.Results[0].TypeName[1:])
		actBlock = fmt.Sprintf("%s := New(%s)", resultVarName, callArgs)
		assertBlock = g.buildAssertBlock(info, resultVarName)
	} else if info.IsMethod {
		arrangeLine = buildReceiverInit(info, recvVar)
		actBlock = g.buildCallLine(info, recvVar+"."+info.Name, args, cfg)
		assertBlock = g.buildAssertBlock(info, "")
	} else {
		actBlock = g.buildCallLine(info, info.Name, args, cfg)
		assertBlock = g.buildAssertBlock(info, "")
	}

	t := template.Must(template.New("simpleTest").Parse(`func {{.FuncName}}(t *testing.T) {
	// Arrange
	{{- if .ArrangeLine}}
	{{.ArrangeLine}}
	{{- end}}
	// TODO: configure dependencies

	// Act
	{{.ActBlock}}

	// Assert
	{{.AssertBlock}}
}
`))

	return t.Execute(buf, map[string]any{
		"FuncName":    funcName,
		"ArrangeLine": arrangeLine,
		"ActBlock":    actBlock,
		"AssertBlock": assertBlock,
	})
}

// buildCallLine returns the act line capturing return values (or just the call).
func (g *SimpleGenerator) buildCallLine(info *analyzer.FuncInfo, callExpr string, args []string, cfg *config.Config) string {
	call := callExpr + "(" + strings.Join(args, ", ") + ")"
	returnVars := buildReturnVars(info.Results, cfg.ResultVarName, cfg.ErrorVarName)
	if len(returnVars) == 0 {
		return call
	}
	return strings.Join(returnVars, ", ") + " := " + call
}

// buildAssertBlock returns the assertion lines for the given FuncInfo.
func (g *SimpleGenerator) buildAssertBlock(info *analyzer.FuncInfo, constructorVar string) string {
	cfg := g.cfg

	if len(info.Results) == 0 {
		return "// TODO: add assertions"
	}

	var lines []string

	if info.HasError {
		lines = append(lines, fmt.Sprintf("assert.NoError(t, %s)", cfg.ErrorVarName))
	}

	if constructorVar != "" {
		lines = append(lines, fmt.Sprintf("assert.NotNil(t, %s)", constructorVar))
	} else {
		for i, r := range info.Results {
			if r.IsError {
				continue
			}
			varName := cfg.ResultVarName
			if i > 0 {
				varName = fmt.Sprintf("%s%d", cfg.ResultVarName, i+1)
			}
			lines = append(lines, fmt.Sprintf("assert.NotNil(t, %s) // TODO: refine assertion", varName))
		}
	}

	lines = append(lines, "// TODO: add more assertions")
	return strings.Join(lines, "\n\t")
}
