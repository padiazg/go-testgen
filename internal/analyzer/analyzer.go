package analyzer

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"strings"

	"golang.org/x/tools/go/packages"
)

func Load(pkgPattern, funcSpec string) (*FuncInfo, error) {
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
		return nil, fmt.Errorf("no package found for pattern: %s", pkgPattern)
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		return nil, fmt.Errorf("package error: %v", pkg.Errors[0])
	}
	if len(pkg.GoFiles) == 0 {
		return nil, fmt.Errorf("no Go files found")
	}

	var funcName, receiverType string
	if strings.Contains(funcSpec, ".") {
		parts := strings.SplitN(funcSpec, ".", 2)
		receiverType = parts[0]
		funcName = parts[1]
	} else {
		funcName = funcSpec
	}

	info := &FuncInfo{
		Name:     funcName,
		IsMethod: receiverType != "",
		Receiver: &ReceiverInfo{TypeName: receiverType},
	}

	sourceFile, err := FindFileInSrc(pkgPattern, funcName)
	if err != nil {
		return nil, fmt.Errorf("find source file: %w", err)
	}
	info.SourceFile = sourceFile

	for _, syn := range pkg.Syntax {
		for _, decl := range syn.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if fn.Name.Name != funcName {
				continue
			}

			if receiverType != "" {
				if fn.Recv == nil || len(fn.Recv.List) == 0 {
					continue
				}
				recvType := typeExprToString(fn.Recv.List[0].Type)
				isPointer := false
				if strings.HasPrefix(recvType, "*") {
					isPointer = true
				}
				recvType = strings.TrimPrefix(recvType, "*")
				if recvType != receiverType {
					continue
				}
				info.Receiver.IsPointer = isPointer
			} else if fn.Recv != nil {
				continue
			}

			info.Package = pkg.Name
			info.ImportPath = pkg.PkgPath
			if fn.Doc != nil {
				info.Doc = fn.Doc.Text()
			}

			if fn.Type.Params != nil {
				for _, param := range fn.Type.Params.List {
					if len(param.Names) <= 1 {
						pi := resolveParamInfo(param, pkg)
						info.Params = append(info.Params, pi)
					} else {
						// Multi-name field: "id, email string" → expand to separate ParamInfo.
						for _, name := range param.Names {
							single := &ast.Field{Names: []*ast.Ident{name}, Type: param.Type}
							pi := resolveParamInfo(single, pkg)
							info.Params = append(info.Params, pi)
						}
					}
				}
			}

			if fn.Type.Results != nil {
				for _, result := range fn.Type.Results.List {
					ri := resolveResultInfo(result, pkg)
					info.Results = append(info.Results, ri)
				}
			}

			if len(info.Results) > 0 {
				lastResult := info.Results[len(info.Results)-1]
				if lastResult.IsError {
					info.HasError = true
				}
			}

			if len(info.Params) > 0 {
				firstParam := info.Params[0]
				if firstParam.IsContext {
					info.HasContext = true
				}
			}

			info.Imports = collectImports(pkg)
			info.ImportAliases = collectImportAliases(pkg, syn)

			return info, nil
		}
	}

	return nil, fmt.Errorf("function %s not found in package", funcSpec)
}

func typeExprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeExprToString(t.X)
	case *ast.SelectorExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name + "." + t.Sel.Name
		}
		return t.Sel.Name
	default:
		return ""
	}
}

func paramInfoFromField(field *ast.Field) ParamInfo {
	pi := ParamInfo{}
	if len(field.Names) > 0 {
		pi.Name = field.Names[0].Name
	}

	if field.Type != nil {
		pi.TypeName = typeToString(field.Type)

		if _, ok := field.Type.(*ast.StarExpr); ok {
			pi.IsPointer = true
		}

		if ident, ok := field.Type.(*ast.Ident); ok {
			if ident.Name == "context" {
				pi.IsContext = true
			}
		}
		if sel, ok := field.Type.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "context" {
				pi.IsContext = true
			}
		}
	}

	return pi
}

func resolveParamInfo(field *ast.Field, pkg *packages.Package) ParamInfo {
	pi := paramInfoFromField(field)

	if pkg.TypesInfo == nil || pkg.Types == nil {
		return pi
	}

	t, ok := pkg.TypesInfo.Types[field.Type]
	if !ok || t.Type == nil {
		return pi
	}

	prefix := extractTypePrefix(field.Type)

	// Unwrap slice/pointer to reach the named type.
	inner := t.Type
	if sl, ok := inner.(*types.Slice); ok {
		inner = sl.Elem()
	}
	if ptr, ok := inner.(*types.Pointer); ok {
		inner = ptr.Elem()
	}

	if named, ok := inner.(*types.Named); ok {
		if namedPkg := named.Obj().Pkg(); namedPkg != nil {
			if namedPkg.Path() != pkg.PkgPath {
				pi.ImportPath = namedPkg.Path()
				pi.Package = extractQualifier(field.Type)
			}
			pi.TypeName = prefix + named.Obj().Name()
			return pi
		}
	}

	typeStr := t.Type.String()
	if typeStr != "" && typeStr != "invalid" && typeStr != "nil" {
		cleanStr := strings.TrimLeft(typeStr, "[]*")
		impPath, typeName := splitQualifiedType(cleanStr)
		pi.ImportPath = impPath
		if typeName != "" {
			pi.TypeName = prefix + typeName
		}
		pi.Package = extractQualifier(field.Type)
	}

	return pi
}

func resolveResultInfo(field *ast.Field, pkg *packages.Package) ResultInfo {
	ri := resultInfoFromField(field)

	if pkg.TypesInfo == nil || pkg.Types == nil {
		return ri
	}

	t, ok := pkg.TypesInfo.Types[field.Type]
	if !ok || t.Type == nil {
		return ri
	}

	prefix := extractTypePrefix(field.Type)

	// Unwrap slice/pointer to reach the named type.
	inner := t.Type
	if sl, ok := inner.(*types.Slice); ok {
		inner = sl.Elem()
	}
	if ptr, ok := inner.(*types.Pointer); ok {
		inner = ptr.Elem()
	}

	if named, ok := inner.(*types.Named); ok {
		if namedPkg := named.Obj().Pkg(); namedPkg != nil {
			if namedPkg.Path() != pkg.PkgPath {
				ri.ImportPath = namedPkg.Path()
				ri.Package = extractQualifier(field.Type)
			}
			ri.TypeName = prefix + named.Obj().Name()
			return ri
		}
	}

	typeStr := t.Type.String()
	if typeStr != "" && typeStr != "invalid" && typeStr != "nil" {
		cleanStr := strings.TrimLeft(typeStr, "[]*")
		impPath, typeName := splitQualifiedType(cleanStr)
		ri.ImportPath = impPath
		if typeName != "" {
			ri.TypeName = prefix + typeName
		}
		ri.Package = extractQualifier(field.Type)
	}

	return ri
}

// extractTypePrefix returns the leading type prefix ("[]*", "[]", "*") of an AST expression.
func extractTypePrefix(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.ArrayType:
		if _, ok := e.Elt.(*ast.StarExpr); ok {
			return "[]*"
		}
		return "[]"
	case *ast.StarExpr:
		return "*"
	default:
		return ""
	}
}

// extractQualifier returns the package qualifier used in the source AST for a type expr.
// e.g., for *userDomain.User it returns "userDomain"; for *User it returns "".
func extractQualifier(expr ast.Expr) string {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if arr, ok := expr.(*ast.ArrayType); ok {
		expr = arr.Elt
		if star, ok := expr.(*ast.StarExpr); ok {
			expr = star.X
		}
	}
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

// collectImportAliases builds a map of importPath -> local alias from a source file.
func collectImportAliases(pkg *packages.Package, file *ast.File) map[string]string {
	aliases := make(map[string]string)
	if file == nil {
		return aliases
	}
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if imp.Name != nil {
			// Explicit alias, e.g.: userDomain "github.com/.../users"
			if imp.Name.Name != "_" && imp.Name.Name != "." {
				aliases[path] = imp.Name.Name
			}
		} else {
			// No explicit alias: use the imported package's declared name
			if importedPkg, ok := pkg.Imports[path]; ok {
				aliases[path] = importedPkg.Name
			} else {
				// Fallback: last segment of path
				parts := strings.Split(path, "/")
				aliases[path] = parts[len(parts)-1]
			}
		}
	}
	return aliases
}

func splitQualifiedType(typeStr string) (importPath, typeName string) {
	parts := strings.Split(typeStr, ".")
	if len(parts) >= 2 {
		importPath = strings.Join(parts[:len(parts)-1], ".")
		typeName = parts[len(parts)-1]
	} else {
		typeName = typeStr
	}
	return
}

func typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeToString(t.X)
	case *ast.SelectorExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name + "." + t.Sel.Name
		}
		return t.Sel.Name
	case *ast.ArrayType:
		return "[]" + typeToString(t.Elt)
	case *ast.MapType:
		return "map[" + typeToString(t.Key) + "]" + typeToString(t.Value)
	case *ast.ChanType:
		return "chan " + typeToString(t.Value)
	case *ast.FuncType:
		return "func()"
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{}"
	default:
		return ""
	}
}

func resultInfoFromField(field *ast.Field) ResultInfo {
	ri := ResultInfo{
		TypeName: typeToString(field.Type),
	}

	if ident, ok := field.Type.(*ast.Ident); ok {
		if ident.Name == "error" {
			ri.IsError = true
		}
	}

	_, ri.IsPointer = field.Type.(*ast.StarExpr)

	return ri
}

func collectImports(pkg *packages.Package) []string {
	seen := make(map[string]bool)
	var imports []string

	for _, syn := range pkg.Syntax {
		for _, imp := range syn.Imports {
			path := imp.Path
			if path != nil {
				p := strings.Trim(path.Value, `"`)
				if !seen[p] && p != pkg.PkgPath {
					seen[p] = true
					imports = append(imports, p)
				}
			}
		}
	}

	return imports
}

func FormatCode(src []byte) ([]byte, error) {
	return format.Source(src)
}

func ParseFile(path string) (*ast.File, *token.FileSet, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}
	return f, fset, nil
}

func FindFileInSrc(pkgPattern, funcName string) (string, error) {
	cfg := &packages.Config{
		Mode: packages.NeedFiles,
	}
	pkgs, err := packages.Load(cfg, pkgPattern)
	if err != nil {
		return "", err
	}
	if len(pkgs) == 0 {
		return "", fmt.Errorf("no package found")
	}

	pkg := pkgs[0]
	for _, f := range pkg.GoFiles {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			continue
		}

		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				if fn.Name.Name == funcName {
					return f, nil
				}
			}
		}
	}
	return "", fmt.Errorf("file not found for %s", funcName)
}

func Getwd() (string, error) {
	return os.Getwd()
}
