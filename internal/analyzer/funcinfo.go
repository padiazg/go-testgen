package analyzer

import (
	"slices"
	"strings"
)

type FuncInfo struct {
	ImportAliases map[string]string // importPath -> local alias
	Receiver      *ReceiverInfo
	Doc           string
	FactoryFunc   string      // factory function name for methods (e.g., "NewClient")
	FactoryParams []ParamInfo // factory function parameters (captured for proper instantiation)
	ImportPath    string
	Name          string
	Package       string
	SourceFile    string
	Imports       []string
	Params        []ParamInfo
	Results       []ResultInfo
	HasContext    bool
	HasError      bool
	IsMethod      bool
}

type ReceiverInfo struct {
	TypeName  string
	Fields    []FieldInfo
	IsPointer bool
}

type ParamInfo struct {
	ImportPath  string
	Name        string
	Package     string
	TypeName    string
	ChanDir     int // 0=bidi, 1=send-only (ast.SEND), 2=recv-only (ast.RECV)
	IsChannel   bool
	IsContext   bool
	IsInterface bool
	IsPointer   bool
}

type ResultInfo struct {
	ImportPath string
	Package    string // local alias/qualifier (e.g., "userDomain")
	TypeName   string
	ChanDir    int // 0=bidi, 1=send-only (ast.SEND), 2=recv-only (ast.RECV)
	IsChannel  bool
	IsError    bool
	IsPointer  bool
}

type FieldInfo struct {
	Name       string
	TypeName   string
	IsExported bool
}

type ImportEntry struct {
	Path  string
	Alias string
}

func (i *FuncInfo) GetImports() []ImportEntry {
	var imports []ImportEntry
	seen := make(map[string]bool)

	addImport := func(importPath, pkgAlias string) {
		if importPath == "" || seen[importPath] || importPath == i.ImportPath || importPath == "context" {
			return
		}
		seen[importPath] = true

		alias := ""
		if pkgAlias != "" {
			parts := strings.Split(importPath, "/")
			defaultName := parts[len(parts)-1]
			if pkgAlias != defaultName {
				alias = pkgAlias
			}
		}

		imports = append(imports, ImportEntry{Path: importPath, Alias: alias})
	}

	for _, p := range i.Params {
		addImport(p.ImportPath, p.Package)
	}
	for _, r := range i.Results {
		if !r.IsError {
			addImport(r.ImportPath, r.Package)
		}
	}

	slices.SortFunc(imports, func(a, b ImportEntry) int {
		return strings.Compare(a.Path, b.Path)
	})

	return imports
}

func (i *FuncInfo) HasNonErrorResults() bool {
	for _, r := range i.Results {
		if !r.IsError {
			return true
		}
	}

	return false
}
