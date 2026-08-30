package analyzer

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

// TestAdd_SQLDB_HasNoMockableDeps is the regression test for #18: sql.DB has
// unexported interface fields (e.g. connector driver.Connector) that must not
// be reported as mockable interface deps.
func TestAdd_SQLDB_HasNoMockableDeps(t *testing.T) {
	cfg := &packages.Config{Mode: packages.NeedTypes | packages.NeedDeps}
	pkgs, err := packages.Load(cfg, "database/sql")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	dbObj := pkgs[0].Types.Scope().Lookup("DB")
	require.NotNil(t, dbObj)
	named, ok := dbObj.Type().(*types.Named)
	require.True(t, ok)
	strct, ok := named.Underlying().(*types.Struct)
	require.True(t, ok)

	data := newInterfaceDepsData(pkgs[0], nil, t.TempDir())
	data.add(strct)
	assert.Empty(t, data.deps)
}

func TestAdd_SkipsUnexportedKeepsExportedInterfaceFields(t *testing.T) {
	pkg := types.NewPackage("example.com/pkg", "pkg")
	driverPkg := types.NewPackage("database/sql/driver", "driver")

	exportedIface := types.NewNamed(
		types.NewTypeName(0, pkg, "Repo", nil),
		types.NewInterfaceType(nil, nil),
		nil,
	)
	unexportedIface := types.NewNamed(
		types.NewTypeName(0, driverPkg, "Connector", nil),
		types.NewInterfaceType(nil, nil),
		nil,
	)

	strct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Repo", exportedIface, false),
		types.NewField(token.NoPos, pkg, "connector", unexportedIface, false),
	}, nil)

	data := newInterfaceDepsData(&packages.Package{Name: "pkg"}, nil, t.TempDir())
	data.add(strct)
	require.Len(t, data.deps, 1)
	assert.Equal(t, "Repo", data.deps[0].TypeName)
}
