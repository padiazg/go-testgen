package generator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTableGenerator_SimpleFunc(t *testing.T) {
	gen := NewTableGenerator(nil)
	result, err := gen.Generate(GenerateRequest{Info: simpleFunc()})
	require.NoError(t, err)

	src := string(result.Source)
	assert.Contains(t, src, "func TestFoo(t *testing.T)")
	assert.Contains(t, src, "want string")
	assert.Contains(t, src, "wantErr string")
	assert.Contains(t, src, "assert.Equal(t, tt.want,")
	// No check-function types.
	assert.NotContains(t, src, "type FooFn func(")
	assert.NotContains(t, src, "var checkFoo")
}

func TestTableGenerator_ErrorOnly(t *testing.T) {
	info := methodInfo() // returns only error
	gen := NewTableGenerator(nil)
	result, err := gen.Generate(GenerateRequest{Info: info})
	require.NoError(t, err)

	src := string(result.Source)
	assert.Contains(t, src, "wantErr string")
	assert.Contains(t, src, "assert.NoError(t,")
	assert.NotContains(t, src, "want error")
}

func TestTableGenerator_NoResults(t *testing.T) {
	info := &simpleFunc().Params
	_ = info
	noReturn := &simpleFunc().Name
	_ = noReturn

	// Build a no-result function info.
	from := simpleFunc()
	from.Results = nil
	from.HasError = false

	gen := NewTableGenerator(nil)
	result, err := gen.Generate(GenerateRequest{Info: from})
	require.NoError(t, err)

	src := string(result.Source)
	assert.Contains(t, src, "// no return values")
	assert.NotContains(t, src, "want ")
}

func TestTableGenerator_Method(t *testing.T) {
	gen := NewTableGenerator(nil)
	result, err := gen.Generate(GenerateRequest{Info: methodInfo()})
	require.NoError(t, err)

	src := string(result.Source)
	assert.Contains(t, src, "func TestEngine_Run(t *testing.T)")
	assert.Contains(t, src, "before func(*Engine)")
}

func TestTableGenerator_IsMerge_OmitsPackageAndImports(t *testing.T) {
	gen := NewTableGenerator(nil)
	result, err := gen.Generate(GenerateRequest{Info: simpleFunc(), IsMerge: true})
	require.NoError(t, err)

	src := string(result.Source)
	assert.False(t, strings.HasPrefix(src, "package "), "merge output must not start with package declaration")
	assert.NotContains(t, src, "import (")
}
