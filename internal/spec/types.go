package spec

type Context struct {
	SubjectInit string `yaml:"subject_init"`
}

type PackageState struct {
	Description       string `yaml:"description"`
	Name              string `yaml:"name"`
	Type              string `yaml:"type"`
	ClearBetweenCases bool   `yaml:"clear_between_cases"`
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

type Before struct {
	Returns     *Returns `yaml:"returns"`
	Description string   `yaml:"description"`
	Mechanism   string   `yaml:"mechanism"`
}

type After struct {
	Description string `yaml:"description"`
	Mechanism   string `yaml:"mechanism"`
}

type Returns struct {
	Type   string `yaml:"type"`
	UsedAs string `yaml:"used_as"`
}
