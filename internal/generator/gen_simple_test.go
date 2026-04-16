package generator

import (
	"strings"
	"testing"

	"github.com/padiazg/go-testgen/internal/analyzer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleGenerator_SimpleFunc(t *testing.T) {
	gen := NewSimpleGenerator(nil)
	result, err := gen.Generate(GenerateRequest{Info: simpleFunc()})
	require.NoError(t, err)

	src := string(result.Source)
	assert.Contains(t, src, "func TestFoo(t *testing.T)")
	// No table.
	assert.NotContains(t, src, "tests := []struct")
	// AAA sections present.
	assert.Contains(t, src, "// Arrange")
	assert.Contains(t, src, "// Act")
	assert.Contains(t, src, "// Assert")
	assert.Contains(t, src, "assert.NoError(t,")
}

func TestSimpleGenerator_Method(t *testing.T) {
	gen := NewSimpleGenerator(nil)
	result, err := gen.Generate(GenerateRequest{Info: methodInfo()})
	require.NoError(t, err)

	src := string(result.Source)
	assert.Contains(t, src, "func TestEngine_Run(t *testing.T)")
	// No factory function in mock data, so should use direct instantiation
	assert.Contains(t, src, "&Engine{}")
	assert.NotContains(t, src, "tests := []struct")
}

func TestSimpleGenerator_Constructor(t *testing.T) {
	gen := NewSimpleGenerator(nil)
	result, err := gen.Generate(GenerateRequest{Info: constructorInfo()})
	require.NoError(t, err)

	src := string(result.Source)
	assert.Contains(t, src, "func TestEngine_New(t *testing.T)")
	assert.Contains(t, src, "assert.NotNil(t,")
}

func TestSimpleGenerator_NoResults(t *testing.T) {
	info := simpleFunc()
	info.Results = nil
	info.HasError = false

	gen := NewSimpleGenerator(nil)
	result, err := gen.Generate(GenerateRequest{Info: info})
	require.NoError(t, err)

	src := string(result.Source)
	assert.Contains(t, src, "// TODO: add assertions")
}

func TestSimpleGenerator_IsMerge_OmitsPackageAndImports(t *testing.T) {
	gen := NewSimpleGenerator(nil)
	result, err := gen.Generate(GenerateRequest{Info: simpleFunc(), IsMerge: true})
	require.NoError(t, err)

	src := string(result.Source)
	assert.False(t, strings.HasPrefix(src, "package "), "merge must not emit package declaration")
	assert.NotContains(t, src, "import (")
}

func TestSimpleGenerator_NoError_NoErrAssert(t *testing.T) {
	info := &analyzer.FuncInfo{
		Name:    "Process",
		Package: "mypkg",
		Results: []analyzer.ResultInfo{{TypeName: "string"}},
	}

	gen := NewSimpleGenerator(nil)
	result, err := gen.Generate(GenerateRequest{Info: info})
	require.NoError(t, err)

	src := string(result.Source)
	assert.NotContains(t, src, "assert.NoError")
	assert.Contains(t, src, "assert.NotNil(t, r)")
}
