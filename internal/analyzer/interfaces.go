package analyzer

import (
	"fmt"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

type InterfaceParams struct {
	PkgPattern       string
	Qualifier        string
	IfaceName        string
	ConsumingAliases map[string]string
	ConsumingPkgPath string
}

// trySamePackage checks if ifaceName is an interface exported by the consuming package.
func (ip *InterfaceParams) trySamePackage() string {
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedTypesInfo | packages.NeedName |
			packages.NeedImports | packages.NeedDeps | packages.NeedSyntax,
	}
	pkgs, err := packages.Load(cfg, ip.PkgPattern)
	if err != nil || len(pkgs) == 0 {
		return ""
	}
	obj := pkgs[0].Types.Scope().Lookup(ip.IfaceName)
	if obj == nil {
		return ""
	}
	if _, ok := obj.Type().Underlying().(*types.Interface); !ok {
		return ""
	}
	return pkgs[0].Types.Path()
}

// findImportPathByAlias re-loads a package to scan import specs for an alias.
func (ip *InterfaceParams) findImportPathByAlias(alias string) string {
	cfg := &packages.Config{
		Mode: packages.NeedSyntax | packages.NeedName,
	}
	pkgs, err := packages.Load(cfg, ip.PkgPattern)
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

func (ip *InterfaceParams) importPath() string {
	importPath := ""
	if ip.Qualifier == "" {
		// Bare name: try same package first (check if ifaceName is an interface in consuming scope)
		importPath = ip.trySamePackage()
	}

	if importPath == "" && ip.Qualifier != "" {
		// External: resolve qualifier → import path
		for path, alias := range ip.ConsumingAliases {
			if alias == ip.Qualifier {
				importPath = path
				break
			}
		}
	}

	if importPath == "" {
		importPath = ip.findImportPathByAlias(ip.Qualifier)
	}

	return importPath
}

func (ip *InterfaceParams) typesNamed(tt *types.Named, p *MethodParam) string {
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
	if pkgPath == ip.ConsumingPkgPath {
		return typeName
	}

	// External type — qualify using the alias the consuming package uses.
	qualifier := resolveQualifierByPath(pkgPath, obj.Pkg().Name(), ip.ConsumingAliases)
	if p != nil {
		p.ImportPath = pkgPath
		p.Package = qualifier
	}
	return qualifier + "." + typeName
}

// resolveTypeExpr converts a types.Type to the string as it should appear in the CONSUMING
// package's generated code, setting import/package info on param as a side effect.
// consumingPkgPath is the import path of the package where the test/mock will live.
func (ip *InterfaceParams) resolveTypeExpr(t types.Type, p *MethodParam) string {
	switch tt := t.(type) {
	case *types.Pointer:
		if p != nil {
			p.IsPointer = true
		}
		inner := ip.resolveTypeExpr(tt.Elem(), p)
		return "*" + inner

	case *types.Slice:
		inner := ip.resolveTypeExpr(tt.Elem(), p)
		return "[]" + inner

	case *types.Chan:
		elemStr := ip.resolveTypeExpr(tt.Elem(), p)
		switch tt.Dir() {
		case types.SendOnly:
			return "chan<- " + elemStr
		case types.RecvOnly:
			return "<-chan " + elemStr
		default:
			return "chan " + elemStr
		}

	case *types.Named:
		return ip.typesNamed(tt, p)

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

func (ip *InterfaceParams) resolveMethodParam(v *types.Var) MethodParam {
	p := MethodParam{Name: v.Name()}
	p.TypeName = ip.resolveTypeExpr(v.Type(), &p)
	return p
}

func (ip *InterfaceParams) extractMethodParams(tuple *types.Tuple) []MethodParam {
	if tuple == nil {
		return nil
	}
	params := make([]MethodParam, 0, tuple.Len())
	for i := 0; i < tuple.Len(); i++ {
		params = append(params, ip.resolveMethodParam(tuple.At(i)))
	}
	return params
}

// resolveQualifierByPath looks up the alias for an import path in the consuming package's aliases map.
// Falls back to pkgName if not found.
func resolveQualifierByPath(importPath, pkgName string, consumingAliases map[string]string) string {
	if alias, ok := consumingAliases[importPath]; ok && alias != "" {
		return alias
	}
	return pkgName
}

// LoadInterface resolves an interface from a consuming package's imports.
//
// pkgPattern is the consuming package pattern (e.g., "github.com/.../services/user").
// qualifier is the local alias used in the consuming package (e.g., "userDomain").
// ifaceName is the interface name (e.g., "UserRepository").
// consumingAliases is info.ImportAliases from the already-analyzed FuncInfo — maps importPath→alias.
// consumingPkgPath is info.ImportPath — the full import path of the consuming package.
func LoadInterface(ip *InterfaceParams) (*InterfaceInfo, error) {
	// Resolve qualifier → import path by reversing the alias map.
	importPath := ip.importPath()

	if importPath == "" {
		return nil, fmt.Errorf("qualifier %q not found in imports of %s", ip.Qualifier, ip.PkgPattern)
	}

	if importPath == "" {
		return nil, fmt.Errorf("%q not found as interface in package %s", ip.IfaceName, ip.ConsumingPkgPath)
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
	obj := scope.Lookup(ip.IfaceName)
	if obj == nil {
		return nil, fmt.Errorf("interface %q not found in package %s", ip.IfaceName, importPath)
	}
	iface, ok := obj.Type().Underlying().(*types.Interface)
	if !ok {
		return nil, fmt.Errorf("%q is not an interface in package %s", ip.IfaceName, importPath)
	}

	info := &InterfaceInfo{
		Name:       ip.IfaceName,
		Package:    pkg.Name,
		Qualifier:  ip.Qualifier,
		ImportPath: importPath,
	}

	for method := range iface.Methods() {
		sig := method.Type().(*types.Signature)

		m := IfaceMethod{
			Name:    method.Name(),
			Params:  ip.extractMethodParams(sig.Params()),
			Results: ip.extractMethodParams(sig.Results()),
		}
		info.Methods = append(info.Methods, m)
	}

	return info, nil
}
