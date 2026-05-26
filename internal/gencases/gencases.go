// Package gencases implements the gen-cases subcommand: reads a .testspec.yaml
// and materializes test case entries into an existing _test.go file.
package gencases

import (
	"fmt"
	"go/ast"
	"go/token"
	"sort"
	"strings"

	"github.com/padiazg/go-testgen/internal/spec"
)

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

// Run executes the gen-cases pipeline for the given spec.
func Run(s *spec.Spec, opts Options) error {
	// 1. Resolve target file
	targetFile, err := ResolveTargetFile(s, opts.Output)
	if err != nil {
		return err
	}

	// 2. Resolve TestXxx name
	testFuncName := ResolveTestFuncName(s)

	if opts.Verbose {
		fmt.Printf("target file: %s\n", targetFile)
		fmt.Printf("test func:   %s\n", testFuncName)
	}

	// 3. Parse existing test file
	f, fset, src, err := ParseTestFile(targetFile)
	if err != nil {
		return err
	}

	// 4. Find TestXxx
	testFunc, err := FindTestFunc(f, testFuncName)
	if err != nil {
		return err
	}

	// 5. Find tests slice
	testsSlice, err := FindTestsSlice(testFunc)
	if err != nil {
		return err
	}

	// 6. Inspect struct fields
	structFields, err := InspectTestsStruct(testsSlice)
	if err != nil {
		return err
	}

	// Collect insertions (apply in reverse offset order to preserve positions).
	var insertions []insertion

	// 7. Fixtures — insert before TestFunc
	fixtureContent := buildFixtureContent(s, f, opts)
	if fixtureContent != "" {
		offset := fset.Position(testFunc.Pos()).Offset
		insertions = append(insertions, insertion{offset: offset, content: fixtureContent})
	}

	// 8. Cases — insert before Rbrace of testsSlice
	caseContent, skipped := buildCaseContent(s, testsSlice, structFields, fset, opts)
	if caseContent != "" {
		offset := fset.Position(testsSlice.Rbrace).Offset
		insertions = append(insertions, insertion{offset: offset, content: caseContent})
	}

	// Handle --force: replace existing cases
	replaceOps := buildReplaceOps(s, testsSlice, structFields, fset, opts)

	if opts.Verbose {
		fmt.Printf("cases generated: %d, skipped (duplicate): %d\n", len(s.Cases)-skipped, skipped)
	}

	// 9. Apply insertions in reverse offset order
	result := applyInsertions(src, insertions, replaceOps)

	// 10. Format
	formatted, err := Format(result)
	if err != nil {
		// Return unformatted with a warning — still useful for debugging
		fmt.Printf("warning: format error (output may not be valid Go): %v\n", err)
		formatted = result
	}

	// 11. Write or print
	return WriteFile(targetFile, formatted, opts.DryRun)
}

// buildFixtureContent generates the string for all new fixtures to insert.
func buildFixtureContent(s *spec.Spec, f *ast.File, opts Options) string {
	var sb strings.Builder
	for _, fix := range s.Fixtures {
		if FindExistingVar(f, fix.Name) {
			if opts.Verbose {
				fmt.Printf("fixture %q already exists, skipping\n", fix.Name)
			}
			continue
		}
		sb.WriteString(GenerateFixtureDecl(&fix, opts.NoHints))
	}
	return sb.String()
}

// buildCaseContent generates the string for all new cases to insert.
// Returns (content, skippedCount).
func buildCaseContent(s *spec.Spec, testsSlice *ast.CompositeLit, structFields []*ast.Field, fset *token.FileSet, opts Options) (string, int) {
	var sb strings.Builder
	skipped := 0
	for i := range s.Cases {
		c := &s.Cases[i]
		exists := FindExistingCase(testsSlice, c.Name)
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
		sb.WriteString(GenerateCaseEntry(c, s, structFields, fset, opts.NoHints))
	}
	return sb.String(), skipped
}

// replaceOp describes a byte-range replacement.
type replaceOp struct {
	content string
	end     int
	start   int
}

// buildReplaceOps builds replacement operations for --force on existing cases.
func buildReplaceOps(s *spec.Spec, testsSlice *ast.CompositeLit, structFields []*ast.Field, fset *token.FileSet, opts Options) []replaceOp {
	if !opts.Force {
		return nil
	}
	var ops []replaceOp
	for i := range s.Cases {
		c := &s.Cases[i]
		node := FindExistingCaseNode(testsSlice, c.Name)
		if node == nil {
			continue
		}
		start := fset.Position(node.Pos()).Offset
		end := fset.Position(node.End()).Offset
		content := GenerateCaseEntry(c, s, structFields, fset, opts.NoHints)
		// Remove trailing comma+newline from content since the original may have it
		ops = append(ops, replaceOp{start: start, end: end, content: content})
	}
	return ops
}

// applyInsertions applies insertions and replacements to src.
// Insertions are applied at given offsets; replacements substitute byte ranges.
// All operations are applied in reverse offset order to preserve positions.
func applyInsertions(src []byte, insertions []insertion, replaces []replaceOp) []byte {
	type op struct {
		content string
		end     int // for insertions: start == end
		start   int
	}

	var ops []op
	for _, ins := range insertions {
		ops = append(ops, op{start: ins.offset, end: ins.offset, content: ins.content})
	}
	for _, r := range replaces {
		ops = append(ops, op{start: r.start, end: r.end, content: r.content})
	}

	// Sort descending by start offset so we apply from end to start
	sort.Slice(ops, func(i, j int) bool {
		return ops[i].start > ops[j].start
	})

	result := make([]byte, len(src))
	copy(result, src)

	for _, op := range ops {
		ins := []byte(op.content)
		result = append(result[:op.start], append(ins, result[op.end:]...)...)
	}
	return result
}
