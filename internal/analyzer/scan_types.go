package analyzer

import "strings"

type InterfaceDeps []InterfaceDep

// ScanResult holds the full analysis of a package.
type ScanResult struct {
	Funcs      Funcs  `json:"funcs"`
	ImportPath string `json:"importPath"`
	Package    string `json:"package"`
	SourceDir  string `json:"sourceDir"`
}

// FuncSummary describes a single function or method.
type FuncSummary struct {
	InterfaceDeps    InterfaceDeps `json:"interfaceDeps"`
	FuncSpec         string        `json:"funcSpec"` // "ReceiverType.Name" or "Name"
	Name             string        `json:"name"`
	ReceiverType     string        `json:"receiverType,omitempty"`
	Signature        string        `json:"signature"` // fully-qualified types
	SourceFile       string        `json:"sourceFile"`
	SuggestedStyle   string        `json:"suggestedStyle,omitempty"`
	TestFuncName     string        `json:"testFuncName"`
	NumParams        int           `json:"numParams"`
	NumResults       int           `json:"numResults"`
	HasArrayResult   bool          `json:"hasArrayResult"`
	HasChannelParam  bool          `json:"hasChannelParam"`
	HasChannelResult bool          `json:"hasChannelResult"`
	HasContext       bool          `json:"hasContext"`
	HasError         bool          `json:"hasError"`
	HasPointerResult bool          `json:"hasPointerResult"`
	HasSliceResult   bool          `json:"hasSliceResult"`
	IsExported       bool          `json:"isExported"`
	IsMethod         bool          `json:"isMethod"`
	ReturnsInterface bool          `json:"returnsInterface"`
	TestExists       bool          `json:"testExists"`
}

type Funcs []FuncSummary

// InterfaceDep describes an interface dependency inferred from struct fields.
type InterfaceDep struct {
	ImportPath string `json:"importPath,omitempty"`
	MockFile   string `json:"mockFile"`            // "mock_userrepository_test.go"
	MockFrom   string `json:"mockFrom"`            // ready for --mock-from flag value
	Qualifier  string `json:"qualifier,omitempty"` // "userDomain"
	TypeName   string `json:"typeName"`            // "UserRepository"
	MockExists bool   `json:"mockExists"`
}

// Lines returns dependencies and mocks
func (i InterfaceDeps) Lines() ([]string, []string) {
	var depLines, mockLines []string
	for _, dep := range i {
		depLines = append(depLines, dep.MockFrom)
		mockStatus := "✗ "
		if dep.MockExists {
			mockStatus = "✓ "
		}
		mockLines = append(mockLines, mockStatus+dep.MockFile)
	}

	return depLines, mockLines
}

func (i InterfaceDeps) MockArgs() []string {
	var mockArgs []string
	for _, dep := range i {
		if !dep.MockExists {
			mockArgs = append(mockArgs, "--mock-from "+dep.MockFrom)
		}
	}
	return mockArgs
}

func (f Funcs) Suggestions(pkgPattern string) []string {
	var suggestions []string

	for _, fn := range f {
		if fn.TestExists {
			continue
		}

		cmd := "  go-testgen gen " + pkgPattern + " " + fn.FuncSpec
		if fn.SuggestedStyle != "" && fn.SuggestedStyle != "check" {
			cmd += " --style " + fn.SuggestedStyle
		}

		var mockArgs []string
		for _, dep := range fn.InterfaceDeps {
			if !dep.MockExists {
				mockArgs = append(mockArgs, "--mock-from "+dep.MockFrom)
			}
		}

		if len(mockArgs) > 0 {
			cmd += " " + strings.Join(mockArgs, " ")
		}

		suggestions = append(suggestions, cmd)
	}

	return suggestions
}
