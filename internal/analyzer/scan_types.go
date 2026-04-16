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
	Name          string         `json:"name"`
	ReceiverType  string         `json:"receiverType,omitempty"`
	IsMethod      bool           `json:"isMethod"`
	IsExported    bool           `json:"isExported"`
	Signature     string         `json:"signature"` // fully-qualified types
	FuncSpec      string         `json:"funcSpec"`  // "ReceiverType.Name" or "Name"
	TestFuncName  string         `json:"testFuncName"`
	TestExists    bool           `json:"testExists"`
	InterfaceDeps []InterfaceDep `json:"interfaceDeps"`

	// Fields used by style suggestion heuristics.
	HasContext       bool `json:"hasContext"`
	NumParams        int  `json:"numParams"`
	NumResults       int  `json:"numResults"`
	HasError         bool `json:"hasError"`
	HasPointerResult bool `json:"hasPointerResult"`
	HasSliceResult   bool `json:"hasSliceResult"`
	ReturnsInterface bool `json:"returnsInterface"`

	// SuggestedStyle is populated by the report command (not the analyzer).
	SuggestedStyle string `json:"suggestedStyle,omitempty"`
}

// InterfaceDep describes an interface dependency inferred from struct fields.
type InterfaceDep struct {
	TypeName   string `json:"typeName"`            // "UserRepository"
	Qualifier  string `json:"qualifier,omitempty"` // "userDomain"
	ImportPath string `json:"importPath,omitempty"`
	MockFile   string `json:"mockFile"` // "mock_userrepository_test.go"
	MockExists bool   `json:"mockExists"`
	MockFrom   string `json:"mockFrom"` // ready for --mock-from flag value
}
