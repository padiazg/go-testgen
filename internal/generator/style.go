package generator

import "fmt"

// TestStyle identifies the test generation style.
type TestStyle string

const (
	StyleCheck  TestStyle = "check"  // table-driven + check-function closures (default)
	StyleTable  TestStyle = "table"  // table-driven + want value comparison
	StyleSimple TestStyle = "simple" // standalone test functions, no table
)

// ParseTestStyle validates and normalises a style string.
// Empty string is accepted and treated as StyleCheck by callers.
func ParseTestStyle(s string) (TestStyle, error) {
	switch TestStyle(s) {
	case StyleCheck, StyleTable, StyleSimple:
		return TestStyle(s), nil
	case "":
		return StyleCheck, nil
	default:
		return "", fmt.Errorf("unknown test style %q: must be one of check, table, simple", s)
	}
}

func (s TestStyle) String() string {
	return string(s)
}
