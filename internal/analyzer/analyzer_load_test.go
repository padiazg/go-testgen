package analyzer

import (
	"go/ast"
	"go/token"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestLoadDebug(t *testing.T) {
	cfg := &packages.Config{
		Mode: packages.NeedSyntax | packages.NeedTypes | packages.NeedFiles,
		Fset: token.NewFileSet(),
	}

	pkgs, _ := packages.Load(cfg, ".")
	pkg := pkgs[0]

	t.Logf("=== Package: %s, Files: %d ===", pkg.Name, len(pkg.GoFiles))

	for _, syn := range pkg.Syntax {
		for _, decl := range syn.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			hasRecv := fn.Recv != nil && len(fn.Recv.List) > 0
			t.Logf("Func: %s, HasRecv: %v", fn.Name.Name, hasRecv)
		}
	}
}
