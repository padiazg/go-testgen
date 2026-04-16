package generator

import (
	"strings"
	"testing"

	"github.com/padiazg/go-testgen/internal/analyzer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// simpleFunc returns a FuncInfo for a plain function: Foo(x int) (string, error)
func simpleFunc() *analyzer.FuncInfo {
	return &analyzer.FuncInfo{
		Name:    "Foo",
		Package: "mypkg",
		Params: []analyzer.ParamInfo{
			{Name: "x", TypeName: "int"},
		},
		Results: []analyzer.ResultInfo{
			{TypeName: "string"},
			{TypeName: "error", IsError: true},
		},
		HasError: true,
	}
}

// methodInfo returns a FuncInfo for a method: (e *Engine) Run(ctx context.Context) error
func methodInfo() *analyzer.FuncInfo {
	return &analyzer.FuncInfo{
		Name:     "Run",
		Package:  "mypkg",
		IsMethod: true,
		Receiver: &analyzer.ReceiverInfo{TypeName: "Engine", IsPointer: true},
		Params: []analyzer.ParamInfo{
			{Name: "ctx", TypeName: "context.Context", IsContext: true},
		},
		Results:    []analyzer.ResultInfo{{TypeName: "error", IsError: true}},
		HasError:   true,
		HasContext: true,
	}
}

// constructorInfo returns a FuncInfo for New(cfg *Config) *Engine
func constructorInfo() *analyzer.FuncInfo {
	return &analyzer.FuncInfo{
		Name:    "New",
		Package: "mypkg",
		Params:  []analyzer.ParamInfo{{Name: "cfg", TypeName: "*Config"}},
		Results: []analyzer.ResultInfo{{TypeName: "*Engine", IsPointer: true}},
	}
}

func TestCheckGenerator_SimpleFunc(t *testing.T) {
	gen := NewCheckGenerator(nil)
	result, err := gen.Generate(GenerateRequest{Info: simpleFunc()})
	require.NoError(t, err)

	src := string(result.Source)
	assert.Contains(t, src, "type FooFn func(")
	assert.Contains(t, src, "var checkFoo = func(")
	assert.Contains(t, src, "func checkFooError(")
	assert.Contains(t, src, "func TestFoo(t *testing.T)")
	assert.Contains(t, src, "tt.x")
}

func TestCheckGenerator_Method(t *testing.T) {
	gen := NewCheckGenerator(nil)
	result, err := gen.Generate(GenerateRequest{Info: methodInfo()})
	require.NoError(t, err)

	src := string(result.Source)
	assert.Contains(t, src, "type checkEngineRunFn func(")
	assert.Contains(t, src, "var checkEngineRun = func(")
	assert.Contains(t, src, "func TestEngine_Run(t *testing.T)")
	assert.Contains(t, src, "before func(*Engine)")
	assert.Contains(t, src, "context.Background()")
}

func TestCheckGenerator_Constructor(t *testing.T) {
	gen := NewCheckGenerator(nil)
	result, err := gen.Generate(GenerateRequest{Info: constructorInfo()})
	require.NoError(t, err)

	src := string(result.Source)
	assert.Contains(t, src, "func TestEngine_New(t *testing.T)")
}

func TestCheckGenerator_IsMerge_OmitsPackageAndImports(t *testing.T) {
	gen := NewCheckGenerator(nil)
	result, err := gen.Generate(GenerateRequest{Info: simpleFunc(), IsMerge: true})
	require.NoError(t, err)

	src := string(result.Source)
	assert.False(t, strings.HasPrefix(src, "package "), "merge output must not start with package declaration")
	assert.NotContains(t, src, "import (")
}

func TestCheckGenerator_NilFuncInfo_ReturnsError(t *testing.T) {
	gen := NewCheckGenerator(nil)
	_, err := gen.Generate(GenerateRequest{Info: nil})
	require.Error(t, err)
}
