package spec

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Spec struct {
	Version      string        `yaml:"version"`
	Package      string        `yaml:"package"`
	Function     string        `yaml:"function"`
	TestFile     string        `yaml:"test_file"`
	Context      *Context      `yaml:"context"`
	PackageState []PackageState `yaml:"package_state"`
	Fixtures     []Fixture     `yaml:"fixtures"`
	CheckTypes   []CheckType   `yaml:"check_types"`
	Checks       []Check       `yaml:"checks"`
	TableFields  []TableField  `yaml:"table_fields"`
	Cases        []Case        `yaml:"cases"`
}

type Context struct {
	SubjectInit string `yaml:"subject_init"`
	SharedSetup string `yaml:"shared_setup"`
}

type PackageState struct {
	Name              string `yaml:"name"`
	Type              string `yaml:"type"`
	ClearBetweenCases bool   `yaml:"clear_between_cases"`
	Description       string `yaml:"description"`
}

type Fixture struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
	Value       string `yaml:"value"`
}

type CheckType struct {
	ID          string `yaml:"id"`
	TypeName    string `yaml:"type_name"`
	Signature   string `yaml:"signature"`
	Composer    string `yaml:"composer"`
	Package     string `yaml:"package"`
	Description string `yaml:"description"`
}

type Check struct {
	ID        string       `yaml:"id"`
	ForType   string       `yaml:"for_type"`
	Scope     string       `yaml:"scope"`
	Signature string       `yaml:"signature"`
	When      string       `yaml:"when"`
	Params    []CheckParam `yaml:"params"`
	Captures  []string     `yaml:"captures"`
}

type CheckParam struct {
	Name          string `yaml:"name"`
	Type          string `yaml:"type"`
	Doc           string `yaml:"doc"`
	SentinelEmpty string `yaml:"sentinel_empty"`
}

type TableField struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
	Role string `yaml:"role"`
	Doc  string `yaml:"doc"`
}

type Case struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Fields      map[string]string `yaml:"fields"`
	Before      *Before           `yaml:"before"`
	After       *After            `yaml:"after"`
	Checks      []string          `yaml:"checks"`
	Gates       map[string]string `yaml:"gates"`
	Todo        bool              `yaml:"todo"`
}

type Before struct {
	Description string   `yaml:"description"`
	Mechanism   string   `yaml:"mechanism"`
	Returns     *Returns `yaml:"returns"`
}

type After struct {
	Description string `yaml:"description"`
	Mechanism   string `yaml:"mechanism"`
}

type Returns struct {
	Type   string `yaml:"type"`
	UsedAs string `yaml:"used_as"`
}

// ParseFile reads and parses a .testspec.yaml file.
func ParseFile(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("gen-cases: spec file not found: %s", path)
	}
	var s Spec
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("gen-cases: invalid spec: %w", err)
	}
	if err := s.validate(); err != nil {
		return nil, fmt.Errorf("gen-cases: invalid spec: %w", err)
	}
	return &s, nil
}

func (s *Spec) validate() error {
	if s.Package == "" {
		return fmt.Errorf("package is required")
	}
	if s.Function == "" {
		return fmt.Errorf("function is required")
	}
	return nil
}

// CheckTypeByID returns the CheckType with the given ID, or nil.
func (s *Spec) CheckTypeByID(id string) *CheckType {
	for i := range s.CheckTypes {
		if s.CheckTypes[i].ID == id {
			return &s.CheckTypes[i]
		}
	}
	return nil
}

// CheckByID returns the Check with the given ID, or nil.
func (s *Spec) CheckByID(id string) *Check {
	for i := range s.Checks {
		if s.Checks[i].ID == id {
			return &s.Checks[i]
		}
	}
	return nil
}
