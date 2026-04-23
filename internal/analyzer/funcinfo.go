package analyzer

type FuncInfo struct {
	ImportAliases map[string]string // importPath -> local alias
	Receiver      *ReceiverInfo
	Doc           string
	FactoryFunc   string       // factory function name for methods (e.g., "NewClient")
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
	IsContext   bool
	IsInterface bool
	IsPointer   bool
}

type ResultInfo struct {
	ImportPath string
	Package    string // local alias/qualifier (e.g., "userDomain")
	TypeName   string
	IsError    bool
	IsPointer  bool
}

type FieldInfo struct {
	Name       string
	TypeName   string
	IsExported bool
}
