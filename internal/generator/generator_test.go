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

func TestQualifyForExternalTest(t *testing.T) {
	tests := []struct {
		name         string
		typeName     string
		pkgQualifier string
		infoPkg      string
		targetPkg    string
		want         string
	}{
		{name: "external bare, fresh file", typeName: "*Category", pkgQualifier: "categoriesDomain", infoPkg: "categories", targetPkg: "", want: "*categoriesDomain.Category"},
		{name: "external bare, merged internal test", typeName: "*Category", pkgQualifier: "categoriesDomain", infoPkg: "categories", targetPkg: "categories", want: "*categoriesDomain.Category"},
		{name: "external bare, external test", typeName: "*Category", pkgQualifier: "categoriesDomain", infoPkg: "categories", targetPkg: "categories_test", want: "*categoriesDomain.Category"},
		{name: "external slice", typeName: "[]*Category", pkgQualifier: "categoriesDomain", infoPkg: "categories", targetPkg: "", want: "[]*categoriesDomain.Category"},
		{name: "external default-name pkg", typeName: "*DB", pkgQualifier: "sql", infoPkg: "categories", targetPkg: "", want: "*sql.DB"},
		{name: "external already qualified", typeName: "*sql.DB", pkgQualifier: "sql", infoPkg: "categories", targetPkg: "", want: "*sql.DB"},
		{name: "local, internal test", typeName: "Engine", pkgQualifier: "", infoPkg: "mypkg", targetPkg: "mypkg", want: "Engine"},
		{name: "local, external test", typeName: "Engine", pkgQualifier: "", infoPkg: "mypkg", targetPkg: "mypkg_test", want: "mypkg.Engine"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, qualifyForExternalTest(tt.typeName, tt.pkgQualifier, tt.infoPkg, tt.targetPkg))
		})
	}
}

// categoryStoreMethod mirrors a database adapter method:
// (s *CategoryStore) Create(ctx context.Context, e *categoriesDomain.Category) error
// with factory NewCategoryStore(db *sql.DB).
func categoryStoreMethod() *analyzer.FuncInfo {
	return &analyzer.FuncInfo{
		Name:       "Create",
		Package:    "categories",
		ImportPath: "github.com/example/app/internal/adapters/secondary/database/categories",
		IsMethod:   true,
		Receiver:   &analyzer.ReceiverInfo{TypeName: "CategoryStore", IsPointer: true},
		Params: []analyzer.ParamInfo{
			{Name: "ctx", TypeName: "context.Context", IsContext: true},
			{Name: "e", TypeName: "*Category", Package: "categoriesDomain", ImportPath: "github.com/example/app/internal/core/domain/categories"},
		},
		Results:     []analyzer.ResultInfo{{TypeName: "error", IsError: true}},
		HasError:    true,
		HasContext:  true,
		FactoryFunc: "NewCategoryStore",
		FactoryParams: []analyzer.ParamInfo{
			{Name: "db", TypeName: "*DB", Package: "sql", ImportPath: "database/sql"},
		},
	}
}

func TestCheckGenerator_ExternalPkgTypes(t *testing.T) {
	gen := NewCheckGenerator(nil)
	result, err := gen.Generate(GenerateRequest{Info: categoryStoreMethod()})
	require.NoError(t, err)

	src := string(result.Source)
	// Table fields must carry the import alias, not bare type names.
	assert.Contains(t, src, "e *categoriesDomain.Category")
	assert.Contains(t, src, "db *sql.DB")
	// Both packages must be imported (factory param import included).
	assert.Contains(t, src, "categoriesDomain \"github.com/example/app/internal/core/domain/categories\"")
	assert.Contains(t, src, "\"database/sql\"")
}

func TestQualifiedTypeName_Array(t *testing.T) {
	tests := []struct {
		name         string
		typeName     string
		pkgQualifier string
		want         string
	}{
		{name: "no qualifier", typeName: "[100]types.PriceBar", pkgQualifier: "", want: "[100]types.PriceBar"},
		{name: "with qualifier", typeName: "[100]types.PriceBar", pkgQualifier: "domain", want: "[100]domain.PriceBar"},
		{name: "pointer element", typeName: "[5]*user.User", pkgQualifier: "domain", want: "[5]domain.User"},
		{name: "element no dot", typeName: "[5]int", pkgQualifier: "pkg", want: "[5]pkg.int"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := qualifiedTypeName(tt.typeName, tt.pkgQualifier)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPlaceholderValue_Array(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		want     string
	}{
		{name: "fixed array", typeName: "[100]types.PriceBar", want: "[100]types.PriceBar{}"},
		{name: "small array", typeName: "[5]int", want: "[5]int{}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := placeholderValue(tt.typeName)
			assert.Equal(t, tt.want, got)
		})
	}
}
