package generator

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/padiazg/go-testgen/internal/analyzer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerator_ChannelReturn(t *testing.T) {
	info := &analyzer.FuncInfo{
		Name:    "ChannelRecvReturn",
		Package: "sample",
		Results: []analyzer.ResultInfo{
			{TypeName: "<-chan *sample.SomeType", IsChannel: true, ChanDir: 2},
		},
	}

	gen := NewTableGenerator(nil)
	result, err := gen.Generate(GenerateRequest{Info: info})
	require.NoError(t, err)

	src := string(result.Source)
	assert.Contains(t, src, "func TestChannelRecvReturn(t *testing.T)")
	assert.Contains(t, src, "<-chan *sample.SomeType")
	// assert is imported when HasError or has non-error results with channels
	// The table generator includes assert for comparing results
	assert.NotContains(t, src, `"context"`)
}

func TestGenerator_ChannelSendParam(t *testing.T) {
	info := &analyzer.FuncInfo{
		Name:    "ChannelSendParam",
		Package: "sample",
		Params: []analyzer.ParamInfo{
			{Name: "ch", TypeName: "chan<- string", IsChannel: true, ChanDir: 1},
		},
	}

	gen := NewCheckGenerator(nil)
	result, err := gen.Generate(GenerateRequest{Info: info})
	require.NoError(t, err)

	src := string(result.Source)
	assert.Contains(t, src, "func TestChannelSendParam(t *testing.T)")
	assert.Contains(t, src, "chan<- string")
}

func TestGenerator_ChannelBidi(t *testing.T) {
	info := &analyzer.FuncInfo{
		Name:    "ChannelBidi",
		Package: "sample",
		Params: []analyzer.ParamInfo{
			{Name: "ch", TypeName: "chan int", IsChannel: true, ChanDir: 0},
		},
		Results: []analyzer.ResultInfo{
			{TypeName: "chan int", IsChannel: true, ChanDir: 0},
		},
	}

	gen := NewSimpleGenerator(nil)
	result, err := gen.Generate(GenerateRequest{Info: info})
	require.NoError(t, err)

	src := string(result.Source)
	assert.Contains(t, src, "func TestChannelBidi(t *testing.T)")
	assert.Contains(t, src, "nil")
}

func TestGenerator_ChannelSlice(t *testing.T) {
	info := &analyzer.FuncInfo{
		Name:    "ChannelSlice",
		Package: "sample",
		Results: []analyzer.ResultInfo{
			{TypeName: "<-chan []string", IsChannel: true, ChanDir: 2},
		},
	}

	gen := NewTableGenerator(nil)
	result, err := gen.Generate(GenerateRequest{Info: info})
	require.NoError(t, err)

	src := string(result.Source)
	assert.Contains(t, src, "func TestChannelSlice(t *testing.T)")
	assert.Contains(t, src, "<-chan []string")
}

func TestGenerator_ChannelMultiParam(t *testing.T) {
	info := &analyzer.FuncInfo{
		Name:    "MultiChannelParam",
		Package: "sample",
		Params: []analyzer.ParamInfo{
			{Name: "input", TypeName: "chan<- int", IsChannel: true, ChanDir: 1},
			{Name: "output", TypeName: "chan int", IsChannel: true, ChanDir: 0},
			{Name: "recv", TypeName: "<-chan string", IsChannel: true, ChanDir: 2},
		},
	}

	gen := NewCheckGenerator(nil)
	result, err := gen.Generate(GenerateRequest{Info: info})
	require.NoError(t, err)

	src := string(result.Source)
	assert.Contains(t, src, "func TestMultiChannelParam(t *testing.T)")
	assert.Contains(t, src, "chan<- int")
	assert.Contains(t, src, "<-chan string")
}

func TestGenerator_ChannelOutput_ValidGo(t *testing.T) {
	info := &analyzer.FuncInfo{
		Name:       "ChannelRecvReturn",
		Package:    "sample",
		ImportPath: "github.com/padiazg/go-testgen/internal/testdata/sample",
		Results: []analyzer.ResultInfo{
			{TypeName: "<-chan *sample.SomeType", IsChannel: true, ChanDir: 2},
		},
	}

	for _, style := range []TestStyle{StyleTable, StyleCheck, StyleSimple} {
		t.Run(string(style), func(t *testing.T) {
			gen, err := NewForStyle(style, nil)
			require.NoError(t, err)

			result, err := gen.Generate(GenerateRequest{Info: info})
			require.NoError(t, err)

			fset := token.NewFileSet()
			_, err = parser.ParseFile(fset, "generated_test.go", string(result.Source), parser.ParseComments)
			assert.NoError(t, err, "generated code must be valid Go")
		})
	}
}
