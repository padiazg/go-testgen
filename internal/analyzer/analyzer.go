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

func analyzerInfoFromParams(info *FuncInfo, pkg *packages.Package, list []*ast.Field) {
	for _, param := range list {
		if len(param.Names) <= 1 {
			info.Params = append(info.Params, resolveParamInfo(param, pkg))
		} else {
			// Multi-name field: "id, email string" → expand to separate ParamInfo.
			for _, name := range param.Names {
				single := &ast.Field{Names: []*ast.Ident{name}, Type: param.Type}
				info.Params = append(info.Params, resolveParamInfo(single, pkg))
			}
		}
	}

	if len(info.Params) > 0 {
		firstParam := info.Params[0]
		if firstParam.IsContext {
			info.HasContext = true
		}
	}
}

func analyzerInfoFromResults(info *FuncInfo, pkg *packages.Package, list []*ast.Field) {
	for _, result := range list {
		ri := resolveResultInfo(result, pkg)
		info.Results = append(info.Results, ri)
	}

	if len(info.Results) > 0 {
		lastResult := info.Results[len(info.Results)-1]
		if lastResult.IsError {
			info.HasError = true
		}
	}
}

func analyzerNewInfo(funcSpec string) *FuncInfo {
	var funcName, receiverType string
	if strings.Contains(funcSpec, ".") {
		parts := strings.SplitN(funcSpec, ".", 2)
		receiverType = parts[0]
		funcName = parts[1]
	} else {
		funcName = funcSpec
	}

	return &FuncInfo{
		Name:     funcName,
		IsMethod: receiverType != "",
		Receiver: &ReceiverInfo{TypeName: receiverType},
	}
}

func analyzerGetFn(info *FuncInfo, pkg *packages.Package) *ast.FuncDecl {
	for _, syn := range pkg.Syntax {
		for _, decl := range syn.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			if fn.Name.Name != info.Name {
				continue
			}

			switch {
			case info.Receiver.TypeName != "":
				if fn.Recv == nil || len(fn.Recv.List) == 0 {
					continue
				}
				recvType := typeExprToString(fn.Recv.List[0].Type)
				isPointer := strings.HasPrefix(recvType, "*")
				recvType = strings.TrimPrefix(recvType, "*")
				if recvType != info.Receiver.TypeName {
					continue
				}

				info.Receiver.IsPointer = isPointer

				// Find factory function for this receiver type
				info.FactoryFunc, info.FactoryParams = findFactoryFunc(pkg, recvType)
			case fn.Recv != nil:
				continue
			}

			info.ImportAliases = collectImportAliases(pkg, syn)

			return fn
		}
	}

	return nil
}

func Load(pkgPattern, funcSpec string) (*FuncInfo, error) {
	pkgs, err := getPackages(pkgPattern)
	if err != nil {
		return nil, fmt.Errorf("load package: %w", err)
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		return nil, fmt.Errorf("package error: %v", pkg.Errors[0])
	}

	if len(pkg.GoFiles) == 0 {
		return nil, fmt.Errorf("no Go files found")
	}

	info := analyzerNewInfo(funcSpec)

	sourceFile, err := FindFileInSrc(pkgPattern, info.Name)
	if err != nil {
		return nil, fmt.Errorf("find source file: %w", err)
	}
	info.SourceFile = sourceFile

	fn := analyzerGetFn(info, pkg)
	if fn == nil {
		return nil, fmt.Errorf("function %s not found in package", funcSpec)
	}

	info.Package = pkg.Name
	info.ImportPath = pkg.PkgPath
	if fn.Doc != nil {
		info.Doc = fn.Doc.Text()
	}

	if fn.Type.Params != nil {
		analyzerInfoFromParams(info, pkg, fn.Type.Params.List)
	}

	if fn.Type.Results != nil {
		analyzerInfoFromResults(info, pkg, fn.Type.Results.List)
	}

	info.Imports = collectImports(pkg)

	return info, nil
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

		if ch, ok := field.Type.(*ast.ChanType); ok {
			pi.IsChannel = true
			pi.ChanDir = int(ch.Dir)
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

	// Unwrap channel/slice/pointer to reach the named type.
	inner := t.Type
	if ch, ok := inner.(*types.Chan); ok {
		pi.IsChannel = true
		pi.ChanDir = int(ch.Dir())
		inner = ch.Elem()
	}
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
		impPath, _ := splitQualifiedType(cleanStr)
		pi.ImportPath = impPath
		pi.TypeName = typeToString(field.Type)
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

	// Unwrap channel/slice/pointer to reach the named type.
	inner := t.Type
	if ch, ok := inner.(*types.Chan); ok {
		ri.IsChannel = true
		ri.ChanDir = int(ch.Dir())
		inner = ch.Elem()
	}
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
		impPath, _ := splitQualifiedType(cleanStr)
		ri.ImportPath = impPath
		ri.TypeName = typeToString(field.Type)
		ri.Package = extractQualifier(field.Type)
	}

	return ri
}

// extractTypePrefix returns the leading type prefix ("[]*", "[]", "[N]", "*") of an AST expression.
func extractTypePrefix(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.ArrayType:
		if e.Len != nil {
			return "[" + typeToString(e.Len) + "]"
		}
		if _, ok := e.Elt.(*ast.StarExpr); ok {
			return "[]*"
		}
		return "[]"
	case *ast.StarExpr:
		return "*"
	case *ast.ChanType:
		inner := extractTypePrefix(e.Value)
		switch e.Dir {
		case ast.SEND:
			return "chan<- " + inner
		case ast.RECV:
			return "<-chan " + inner
		default:
			return "chan " + inner
		}
	default:
		return ""
	}
}

// extractQualifier returns the package qualifier used in the source AST for a type expr.
// e.g., for *userDomain.User it returns "userDomain"; for *User it returns "".
func extractQualifier(expr ast.Expr) string {
	if ch, ok := expr.(*ast.ChanType); ok {
		expr = ch.Value
	}
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if arr, ok := expr.(*ast.ArrayType); ok {
		expr = arr.Elt
	}
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
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
			continue
		}

		// No explicit alias: use the imported package's declared name
		if importedPkg, ok := pkg.Imports[path]; ok {
			aliases[path] = importedPkg.Name
		} else {
			// Fallback: last segment of path
			parts := strings.Split(path, "/")
			aliases[path] = parts[len(parts)-1]
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
		if t.Len != nil {
			return "[" + typeToString(t.Len) + "]" + typeToString(t.Elt)
		}
		return "[]" + typeToString(t.Elt)
	case *ast.MapType:
		return "map[" + typeToString(t.Key) + "]" + typeToString(t.Value)
	case *ast.ChanType:
		switch t.Dir {
		case ast.SEND:
			return "chan<- " + typeToString(t.Value)
		case ast.RECV:
			return "<-chan " + typeToString(t.Value)
		default:
			return "chan " + typeToString(t.Value)
		}
	case *ast.FuncType:
		return "func()"
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{}"
	case *ast.Ellipsis:
		return "..." + typeToString(t.Elt)
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

	if ch, ok := field.Type.(*ast.ChanType); ok {
		ri.IsChannel = true
		ri.ChanDir = int(ch.Dir)
	}

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

// findFactoryFunc searches for a factory function that returns *receiverType.
// It looks for functions starting with "New" that return a pointer to the receiver type.
// Returns the function name and its parameters.
func findFactoryFunc(pkg *packages.Package, receiverType string) (string, []ParamInfo) {
	if pkg.Syntax == nil {
		return "", nil
	}

	for _, syn := range pkg.Syntax {
		for _, decl := range syn.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			// Skip methods (functions with receivers)
			if fn.Recv != nil {
				continue
			}
			// Look for functions starting with "New"
			if !strings.HasPrefix(fn.Name.Name, "New") {
				continue
			}
			// Check if it returns *receiverType
			if fn.Type.Results == nil || len(fn.Type.Results.List) == 0 {
				continue
			}
			retType := typeExprToString(fn.Type.Results.List[0].Type)
			if retType == "*"+receiverType {
				// Capture the factory function's parameters
				var factoryParams []ParamInfo
				if fn.Type.Params != nil {
					for _, param := range fn.Type.Params.List {
						if len(param.Names) <= 1 {
							pi := resolveParamInfo(param, pkg)
							factoryParams = append(factoryParams, pi)
						} else {
							for _, name := range param.Names {
								single := &ast.Field{Names: []*ast.Ident{name}, Type: param.Type}
								pi := resolveParamInfo(single, pkg)
								factoryParams = append(factoryParams, pi)
							}
						}
					}
				}
				return fn.Name.Name, factoryParams
			}
		}
	}
	return "", nil
}
