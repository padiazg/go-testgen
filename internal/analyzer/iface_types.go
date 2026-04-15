package analyzer

// InterfaceInfo describes a resolved Go interface and all its methods.
type InterfaceInfo struct {
	Name       string          // e.g., "UserRepository"
	Package    string          // declared package name, e.g., "users"
	Qualifier  string          // alias used in consuming package, e.g., "userDomain"
	ImportPath string          // full import path, e.g., "github.com/.../domain/users"
	Methods    []IfaceMethod
}

// IfaceMethod describes a single method on an interface.
type IfaceMethod struct {
	Name    string
	Params  []MethodParam
	Results []MethodParam
}

// MethodParam describes a single parameter or return value of an interface method.
type MethodParam struct {
	Name       string // param name ("ctx", "id"), may be empty for return values
	TypeName   string // as it should appear in generated code, e.g., "*userDomain.User"
	ImportPath string // full import path for the type's package (empty for builtins/same pkg)
	Package    string // local qualifier to use in generated code, e.g., "userDomain"
	IsError    bool   // true if the type is the built-in error interface
	IsPointer  bool   // true if the type is a pointer
}
