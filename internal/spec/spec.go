package spec

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/padiazg/go-testgen/internal/astreader"
)

type Spec struct {
	Version      string         `yaml:"version"`
	Package      string         `yaml:"package"`
	Function     string         `yaml:"function"`
	TestFile     string         `yaml:"test_file"`
	Context      *Context       `yaml:"context"`
	PackageState []PackageState `yaml:"package_state"`
	Fixtures     []Fixture      `yaml:"fixtures"`
	CheckTypes   []CheckType    `yaml:"check_types"`
	Checks       []Check        `yaml:"checks"`
	TableFields  []TableField   `yaml:"table_fields"`
	Cases        []Case         `yaml:"cases"`
}

// replaceOp describes a byte-range replacement.
type replaceOp struct {
	content string
	end     int
	start   int
}

// Options controls gen-cases behavior.
type Options struct {
	Output  string
	DryRun  bool
	Force   bool
	NoHints bool
	Verbose bool
}

// insertion holds a byte-offset and the content to insert at that offset.
type insertion struct {
	content string
	offset  int
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

// Run executes the gen-cases pipeline for the given spec.
func (s *Spec) Run(opts Options) error {
	// 1. Resolve target file
	targetFile, err := s.resolveTargetFile(opts.Output)
	if err != nil {
		return err
	}

	// 2. Resolve TestXxx name
	testFuncName := s.resolveTestFuncName()

	if opts.Verbose {
		fmt.Printf("target file: %s\n", targetFile)
		fmt.Printf("test func:   %s\n", testFuncName)
	}

	// 3. Parse existing test file
	f, fset, src, err := astreader.ParseTestFile(targetFile)
	if err != nil {
		return err
	}

	// 4. Find TestXxx
	testFunc, err := astreader.FindTestFunc(f, testFuncName)
	if err != nil {
		return err
	}

	// 5. Find tests slice
	testsSlice, err := astreader.FindTestsSlice(testFunc)
	if err != nil {
		return err
	}

	// 6. Inspect struct fields
	structFields, err := astreader.InspectTestsStruct(testsSlice)
	if err != nil {
		return err
	}

	// Collect insertions (apply in reverse offset order to preserve positions).
	var insertions []insertion

	// 7. Fixtures — insert before TestFunc
	fixtureContent := s.buildFixtureContent(f, opts)
	if fixtureContent != "" {
		offset := fset.Position(testFunc.Pos()).Offset
		insertions = append(insertions, insertion{offset: offset, content: fixtureContent})
	}

	// 8. Cases — insert before Rbrace of testsSlice
	caseContent, skipped := s.buildCaseContent(testsSlice, structFields, fset, opts)
	if caseContent != "" {
		offset := fset.Position(testsSlice.Rbrace).Offset
		insertions = append(insertions, insertion{offset: offset, content: caseContent})
	}

	// Handle --force: replace existing cases
	replaceOps := s.buildReplaceOps(testsSlice, structFields, fset, opts)

	if opts.Verbose {
		fmt.Printf("cases generated: %d, skipped (duplicate): %d\n", len(s.Cases)-skipped, skipped)
	}

	// 9. Apply insertions in reverse offset order
	result := applyInsertions(src, insertions, replaceOps)

	// 10. Format
	formatted, err := formatSource(result)
	if err != nil {
		// Return unformatted with a warning — still useful for debugging
		fmt.Printf("warning: format error (output may not be valid Go): %v\n", err)
		formatted = result
	}

	// 11. Write or print
	return writeFile(targetFile, formatted, opts.DryRun)
}

// CheckTypeByID returns the CheckType with the given ID, or nil.
func (s *Spec) checkTypeByID(id string) *CheckType {
	for i := range s.CheckTypes {
		if s.CheckTypes[i].ID == id {
			return &s.CheckTypes[i]
		}
	}
	return nil
}

// CheckByID returns the Check with the given ID, or nil.
// func (s *Spec) checkByID(id string) *Check {
// 	for i := range s.Checks {
// 		if s.Checks[i].ID == id {
// 			return &s.Checks[i]
// 		}
// 	}
// 	return nil
// }

// buildFixtureContent generates the string for all new fixtures to insert.
func (s *Spec) buildFixtureContent(f *ast.File, opts Options) string {
	var sb strings.Builder
	for _, fix := range s.Fixtures {
		if astreader.FindExistingVar(f, fix.Name) {
			if opts.Verbose {
				fmt.Printf("fixture %q already exists, skipping\n", fix.Name)
			}
			continue
		}
		sb.WriteString(fix.generateFixtureDecl(opts.NoHints))
	}
	return sb.String()
}

// buildCaseContent generates the string for all new cases to insert.
// Returns (content, skippedCount).
func (s *Spec) buildCaseContent(testsSlice *ast.CompositeLit, structFields []*ast.Field, fset *token.FileSet, opts Options) (string, int) {
	var sb strings.Builder
	skipped := 0
	for i := range s.Cases {
		c := &s.Cases[i]
		exists := astreader.FindExistingCase(testsSlice, c.Name)
		if exists && !opts.Force {
			if opts.Verbose {
				fmt.Printf("case %q already exists, skipping (use --force to replace)\n", c.Name)
			}
			skipped++
			continue
		}
		if exists && opts.Force {
			// Handled by replaceOps
			continue
		}
		sb.WriteString(c.GenerateCaseEntry(s, structFields, fset, opts.NoHints))
	}
	return sb.String(), skipped
}

// buildReplaceOps builds replacement operations for --force on existing cases.
func (s *Spec) buildReplaceOps(testsSlice *ast.CompositeLit, structFields []*ast.Field, fset *token.FileSet, opts Options) []replaceOp {
	if !opts.Force {
		return nil
	}
	var ops []replaceOp
	for i := range s.Cases {
		c := &s.Cases[i]
		node := astreader.FindExistingCaseNode(testsSlice, c.Name)
		if node == nil {
			continue
		}
		start := fset.Position(node.Pos()).Offset
		end := fset.Position(node.End()).Offset
		content := c.GenerateCaseEntry(s, structFields, fset, opts.NoHints)
		// Remove trailing comma+newline from content since the original may have it
		ops = append(ops, replaceOp{start: start, end: end, content: content})
	}
	return ops
}
