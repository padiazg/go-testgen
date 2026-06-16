package analyzer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadInterface_DirectImportPath_IOWriter(t *testing.T) {
	info, err := LoadInterface(&InterfaceParams{
		IfaceName:        "Writer",
		DirectImportPath: "io",
	})
	require.NoError(t, err)
	assert.Equal(t, "Writer", info.Name)
	assert.Equal(t, "io", info.Package)
	assert.Equal(t, "io", info.Qualifier)
	assert.Equal(t, "io", info.ImportPath)
	require.Len(t, info.Methods, 1)

	m := info.Methods[0]
	assert.Equal(t, "Write", m.Name)
	require.Len(t, m.Params, 1)
	assert.Equal(t, "[]byte", m.Params[0].TypeName)
	require.Len(t, m.Results, 2)
	assert.Equal(t, "int", m.Results[0].TypeName)
	assert.True(t, m.Results[1].IsError)
}

func TestLoadInterface_DirectImportPath_IOReader(t *testing.T) {
	info, err := LoadInterface(&InterfaceParams{
		IfaceName:        "Reader",
		DirectImportPath: "io",
	})
	require.NoError(t, err)
	assert.Equal(t, "Reader", info.Name)
	assert.Equal(t, "io", info.Package)
	require.Len(t, info.Methods, 1)
	assert.Equal(t, "Read", info.Methods[0].Name)
}

func TestLoadInterface_DirectImportPath_NetHTTPHandler(t *testing.T) {
	info, err := LoadInterface(&InterfaceParams{
		IfaceName:        "Handler",
		DirectImportPath: "net/http",
	})
	require.NoError(t, err)
	assert.Equal(t, "Handler", info.Name)
	assert.Equal(t, "http", info.Package)
	assert.Equal(t, "http", info.Qualifier)
	assert.Equal(t, "net/http", info.ImportPath)
	require.Len(t, info.Methods, 1)

	m := info.Methods[0]
	assert.Equal(t, "ServeHTTP", m.Name)
	require.Len(t, m.Params, 2)
	// ResponseWriter is an interface from net/http — should be qualified
	assert.Contains(t, m.Params[0].TypeName, "ResponseWriter")
	// *Request is a pointer type from net/http
	assert.Contains(t, m.Params[1].TypeName, "Request")
	assert.True(t, m.Params[1].IsPointer)
}

func TestLoadInterface_DirectImportPath_IOFS(t *testing.T) {
	info, err := LoadInterface(&InterfaceParams{
		IfaceName:        "FS",
		DirectImportPath: "io/fs",
	})
	require.NoError(t, err)
	assert.Equal(t, "FS", info.Name)
	assert.Equal(t, "fs", info.Package)
	assert.Equal(t, "fs", info.Qualifier)
	assert.Equal(t, "io/fs", info.ImportPath)
	require.Len(t, info.Methods, 1)

	m := info.Methods[0]
	assert.Equal(t, "Open", m.Name)
	require.Len(t, m.Params, 1)
	assert.Equal(t, "string", m.Params[0].TypeName)
	require.Len(t, m.Results, 2)
	assert.True(t, m.Results[1].IsError)
}

func TestLoadInterface_DirectImportPath_NotFound(t *testing.T) {
	_, err := LoadInterface(&InterfaceParams{
		IfaceName:        "DoesNotExist",
		DirectImportPath: "io",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestLoadInterface_DirectImportPath_NotInterface(t *testing.T) {
	_, err := LoadInterface(&InterfaceParams{
		IfaceName:        "SeekStart", // io.SeekStart is a const int, not an interface
		DirectImportPath: "io",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not an interface")
}

func TestLoadInterface_DirectImportPath_WithExplicitQualifier(t *testing.T) {
	// When both DirectImportPath and Qualifier are set, Qualifier should be used.
	info, err := LoadInterface(&InterfaceParams{
		IfaceName:        "Writer",
		DirectImportPath: "io",
		Qualifier:        "myio",
	})
	require.NoError(t, err)
	assert.Equal(t, "myio", info.Qualifier)
	assert.Equal(t, "io", info.Package)
}
