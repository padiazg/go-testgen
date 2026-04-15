package analyzer

type FuncInfo struct {
	Name          string
	Package       string
	ImportPath    string
	IsMethod      bool
	Receiver      *ReceiverInfo
	Params        []ParamInfo
	Results       []ResultInfo
	HasError      bool
	HasContext     bool
	Doc           string
	Imports       []string
	ImportAliases map[string]string // importPath -> local alias
	SourceFile    string
}

type ReceiverInfo struct {
	TypeName  string
	IsPointer bool
	Fields    []FieldInfo
}

type ParamInfo struct {
	Name        string
	TypeName    string
	ImportPath  string
	IsInterface bool
	IsPointer   bool
	IsContext   bool
	Package     string
}

type ResultInfo struct {
	TypeName   string
	ImportPath string
	IsError    bool
	IsPointer  bool
	Package    string // local alias/qualifier (e.g., "userDomain")
}

type FieldInfo struct {
	Name       string
	TypeName   string
	IsExported bool
}
