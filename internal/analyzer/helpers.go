package analyzer

import (
	"errors"
	"fmt"
	"go/token"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/packages"
)

func getPackages(pkgPattern string) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedFiles |
			packages.NeedName,
		Fset: token.NewFileSet(),
	}

	pkgs, err := packages.Load(cfg, pkgPattern)
	if err != nil {
		return nil, fmt.Errorf("load package: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no package found for: %s", pkgPattern)
	}

	return pkgs, nil
}

func checkListError(pkgs []*packages.Package) bool {
	var pkgErr packages.Error
	for _, pkg := range pkgs {
		for _, err := range pkg.Errors {
			if errors.As(err, &pkgErr) && err.Kind == packages.ListError {
				return true
			}
		}
	}

	return false
}

// isGoModPresent checks whether a go.mod file exists in the given directory.
func isGoModPresent(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "go.mod"))
	return err == nil
}
