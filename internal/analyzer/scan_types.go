package analyzer

// ScanResult holds the full analysis of a package.
type ScanResult struct {
	Package    string        `json:"package"`
	ImportPath string        `json:"importPath"`
	SourceDir  string        `json:"sourceDir"`
	Funcs      []FuncSummary `json:"funcs"`
}

// FuncSummary describes a single function or method.
type FuncSummary struct {
	FuncSpec         string         `json:"funcSpec"` // "ReceiverType.Name" or "Name"
	Name             string         `json:"name"`
	ReceiverType     string         `json:"receiverType,omitempty"`
	Signature        string         `json:"signature"` // fully-qualified types
	SuggestedStyle   string         `json:"suggestedStyle,omitempty"`
	TestFuncName     string         `json:"testFuncName"`
	InterfaceDeps    []InterfaceDep `json:"interfaceDeps"`
	NumParams        int            `json:"numParams"`
	NumResults       int            `json:"numResults"`
	HasContext       bool           `json:"hasContext"`
	HasError         bool           `json:"hasError"`
	HasPointerResult bool           `json:"hasPointerResult"`
	HasSliceResult   bool           `json:"hasSliceResult"`
	IsExported       bool           `json:"isExported"`
	IsMethod         bool           `json:"isMethod"`
	ReturnsInterface bool           `json:"returnsInterface"`
	TestExists       bool           `json:"testExists"`
}

// InterfaceDep describes an interface dependency inferred from struct fields.
type InterfaceDep struct {
	ImportPath string `json:"importPath,omitempty"`
	MockFile   string `json:"mockFile"`            // "mock_userrepository_test.go"
	MockFrom   string `json:"mockFrom"`            // ready for --mock-from flag value
	Qualifier  string `json:"qualifier,omitempty"` // "userDomain"
	TypeName   string `json:"typeName"`            // "UserRepository"
	MockExists bool   `json:"mockExists"`
}
