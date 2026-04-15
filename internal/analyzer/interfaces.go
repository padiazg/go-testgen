package analyzer

import (
	"fmt"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

// LoadInterface resolves an interface from a consuming package's imports.
//
// pkgPattern is the consuming package pattern (e.g., "github.com/.../services/user").
// qualifier is the local alias used in the consuming package (e.g., "userDomain").
// ifaceName is the interface name (e.g., "UserRepository").
// consumingAliases is info.ImportAliases from the already-analyzed FuncInfo — maps importPath→alias.
// consumingPkgPath is info.ImportPath — the full import path of the consuming package.
func LoadInterface(pkgPattern, qualifier, ifaceName string, consumingAliases map[string]string, consumingPkgPath string) (*InterfaceInfo, error) {
	// Resolve qualifier → import path by reversing the alias map.
	importPath := ""
	for path, alias := range consumingAliases {
		if alias == qualifier {
			importPath = path
			break
		}
	}
	if importPath == "" {
		// Fallback: re-load the consuming package and scan import specs.
		importPath = findImportPathByAlias(pkgPattern, qualifier)
	}
	if importPath == "" {
		return nil, fmt.Errorf("qualifier %q not found in imports of %s", qualifier, pkgPattern)
	}

	// Load the target package that declares the interface.
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedTypesInfo | packages.NeedName |
			packages.NeedImports | packages.NeedDeps | packages.NeedSyntax,
	}
	pkgs, err := packages.Load(cfg, importPath)
	if err != nil {
		return nil, fmt.Errorf("load package %s: %w", importPath, err)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no package found for %s", importPath)
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		return nil, fmt.Errorf("package errors: %v", pkg.Errors[0])
	}

	// Find the interface by name in the type scope.
	scope := pkg.Types.Scope()
	obj := scope.Lookup(ifaceName)
	if obj == nil {
		return nil, fmt.Errorf("interface %q not found in package %s", ifaceName, importPath)
	}
	iface, ok := obj.Type().Underlying().(*types.Interface)
	if !ok {
		return nil, fmt.Errorf("%q is not an interface in package %s", ifaceName, importPath)
	}

	info := &InterfaceInfo{
		Name:       ifaceName,
		Package:    pkg.Name,
		Qualifier:  qualifier,
		ImportPath: importPath,
	}

	for i := 0; i < iface.NumMethods(); i++ {
		method := iface.Method(i)
		sig := method.Type().(*types.Signature)

		m := IfaceMethod{
			Name:    method.Name(),
			Params:  extractMethodParams(sig.Params(), consumingAliases, consumingPkgPath),
			Results: extractMethodParams(sig.Results(), consumingAliases, consumingPkgPath),
		}
		info.Methods = append(info.Methods, m)
	}

	return info, nil
}

// findImportPathByAlias re-loads a package to scan import specs for an alias.
func findImportPathByAlias(pkgPattern, alias string) string {
	cfg := &packages.Config{
		Mode: packages.NeedSyntax | packages.NeedName,
	}
	pkgs, err := packages.Load(cfg, pkgPattern)
	if err != nil || len(pkgs) == 0 {
		return ""
	}
	for _, syn := range pkgs[0].Syntax {
		for _, imp := range syn.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if imp.Name != nil && imp.Name.Name == alias {
				return path
			}
		}
	}
	return ""
}

func extractMethodParams(tuple *types.Tuple, consumingAliases map[string]string, consumingPkgPath string) []MethodParam {
	if tuple == nil {
		return nil
	}
	params := make([]MethodParam, 0, tuple.Len())
	for i := 0; i < tuple.Len(); i++ {
		params = append(params, resolveMethodParam(tuple.At(i), consumingAliases, consumingPkgPath))
	}
	return params
}

func resolveMethodParam(v *types.Var, consumingAliases map[string]string, consumingPkgPath string) MethodParam {
	p := MethodParam{Name: v.Name()}
	p.TypeName = resolveTypeExpr(v.Type(), consumingAliases, consumingPkgPath, &p)
	return p
}

// resolveTypeExpr converts a types.Type to the string as it should appear in the CONSUMING
// package's generated code, setting import/package info on param as a side effect.
// consumingPkgPath is the import path of the package where the test/mock will live.
func resolveTypeExpr(t types.Type, consumingAliases map[string]string, consumingPkgPath string, p *MethodParam) string {
	switch tt := t.(type) {
	case *types.Pointer:
		if p != nil {
			p.IsPointer = true
		}
		inner := resolveTypeExpr(tt.Elem(), consumingAliases, consumingPkgPath, p)
		return "*" + inner

	case *types.Slice:
		inner := resolveTypeExpr(tt.Elem(), consumingAliases, consumingPkgPath, p)
		return "[]" + inner

	case *types.Named:
		obj := tt.Obj()
		typeName := obj.Name()

		if obj.Pkg() == nil {
			// Built-in (error interface)
			if typeName == "error" || tt.String() == "error" {
				if p != nil {
					p.IsError = true
				}
			}
			return typeName
		}

		pkgPath := obj.Pkg().Path()

		// context.Context special case
		if pkgPath == "context" && typeName == "Context" {
			if p != nil {
				p.ImportPath = "context"
				p.Package = "context"
			}
			return "context.Context"
		}

		// Types from the consuming package itself need no qualifier.
		if pkgPath == consumingPkgPath {
			return typeName
		}

		// External type — qualify using the alias the consuming package uses.
		qualifier := resolveQualifierByPath(pkgPath, obj.Pkg().Name(), consumingAliases)
		if p != nil {
			p.ImportPath = pkgPath
			p.Package = qualifier
		}
		return qualifier + "." + typeName

	case *types.Basic:
		return tt.Name()

	case *types.Interface:
		// Built-in error
		if tt.String() == "error" {
			if p != nil {
				p.IsError = true
			}
			return "error"
		}
		return tt.String()

	default:
		// Check for error via String() fallback
		if t.String() == "error" {
			if p != nil {
				p.IsError = true
			}
			return "error"
		}
		return t.String()
	}
}

// resolveQualifierByPath looks up the alias for an import path in the consuming package's aliases map.
// Falls back to pkgName if not found.
func resolveQualifierByPath(importPath, pkgName string, consumingAliases map[string]string) string {
	if alias, ok := consumingAliases[importPath]; ok && alias != "" {
		return alias
	}
	return pkgName
}
