/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/padiazg/go-testgen/internal/analyzer"
	"github.com/padiazg/go-testgen/internal/config"
	"github.com/padiazg/go-testgen/internal/generator"
	"github.com/spf13/cobra"
)

type mockSource struct {
	ifaceName string
	qualifier string
	source    []byte
}

// mockSpec holds the parsed components of a --mock-from value.
type mockSpec struct {
	qualifier  string // local alias (e.g., "userDomain") — empty for direct path mode
	ifaceName  string // e.g., "Writer", "Handler"
	importPath string // full import path when specified (e.g., "io/fs", "net/http") — empty for qualifier mode
}

// genCmd represents the gen command
var (
	configFlag    string
	outputFlag    string
	verboseFlag   bool
	styleFlag     string
	mockFromFlags []string
	pkgFlag       string

	genCmd = &cobra.Command{
		Use:   "gen [<./pkg/path> <FuncSpec>]",
		Short: "Generates test scaffolding for a function or method",
		Long: `Generates test scaffolding for a function or method.

Normal mode (2 args): gen <./pkg/path> <FuncSpec>
Standalone mock mode (0 args): gen --mock-from <spec> --pkg <name> --output <file|->

In standalone mode, --pkg and --output are required.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 2 {
				return nil
			}
			if len(args) == 0 && len(mockFromFlags) > 0 {
				return nil
			}
			if len(mockFromFlags) > 0 {
				return fmt.Errorf("standalone mock mode requires 0 positional args, got %d", len(args))
			}
			return fmt.Errorf("requires exactly 2 args (<pkg/path> <FuncSpec>) or 0 args with --mock-from")
		},
		RunE: runGen,
	}
)

func init() {
	rootCmd.AddCommand(genCmd)
	genCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "output file (default: auto-detect, use '-' for stdout)")
	genCmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "show parsed FuncInfo")
	genCmd.Flags().StringVar(&configFlag, "config", "", "path to .go-testgen.yaml config file")
	genCmd.Flags().StringVar(&styleFlag, "style", "", "test generation style: check, table, simple (default: from config or check)")
	genCmd.Flags().StringSliceVar(&mockFromFlags, "mock-from", nil,
		`generate mock for interface (repeatable). Formats:
  qualifier.Interface   — resolve via consuming package imports
  .Interface            — same package
  io/fs.FS              — full import path (stdlib or external)
  github.com/x/y.Iface  — full import path (module)
Can be used without positional args (standalone mode) with --pkg and --output.`)
	genCmd.Flags().StringVar(&pkgFlag, "pkg", "", "package name for generated mock files (required in standalone mode)")

}

func runGen(cmd *cobra.Command, args []string) error {
	// Standalone mock mode: no positional args, just --mock-from.
	if len(args) == 0 {
		return runStandaloneMocks()
	}

	pkgPath := args[0]
	funcSpec := args[1]

	cfg, err := config.Load(configFlag)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	info, err := analyzer.Load(pkgPath, funcSpec)
	if err != nil {
		return fmt.Errorf("analyze: %w", err)
	}

	if verboseFlag {
		b, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return fmt.Errorf("json marshal: %w", err)
		}
		fmt.Fprintf(os.Stderr, "FuncInfo:\n%s\n\n", b)
	}

	testFuncName, isMerge := funcName(info)

	gen, err := getGenerator(cfg)
	if err != nil {
		return fmt.Errorf("create generator: %w", err)
	}

	result, err := gen.Generate(generator.GenerateRequest{Info: info, IsMerge: isMerge})
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	formatted, err := analyzer.FormatCode(result.Source)
	if err != nil {
		formatted = result.Source
	}

	if isMerge && outputFlag == "" {
		targetPath := analyzer.DeriveTestPath(info.SourceFile)
		if _, statErr := os.Stat(targetPath); statErr == nil {
			if err := analyzer.InjectImports(targetPath, generator.CollectImports(info)); err != nil {
				return fmt.Errorf("inject imports: %w", err)
			}
		}
	}

	if err := generateMocks(pkgPath, info, outputFlag); err != nil {
		return fmt.Errorf("generate mocks: %w", err)
	}

	return writeOutput(formatted, outputFlag, info.SourceFile, testFuncName, isMerge, info)
}

func funcName(info *analyzer.FuncInfo) (string, bool) {
	testFuncName := analyzer.FindTestFuncName(info)

	var isMerge bool
	if outputFlag == "" {
		targetPath := analyzer.DeriveTestPath(info.SourceFile)
		if _, err := os.Stat(targetPath); err == nil {
			exists, _ := analyzer.TestExistsInFile(targetPath, testFuncName)
			isMerge = !exists
		}
	}

	return testFuncName, isMerge
}

func getGenerator(cfg *config.Config) (generator.TestGenerator, error) {
	// Resolve test style: flag > config > default "check".
	styleName := styleFlag
	if styleName == "" {
		styleName = cfg.TestStyle
	}

	testStyle, err := generator.ParseTestStyle(styleName)
	if err != nil {
		return nil, fmt.Errorf("invalid --style: %w", err)
	}

	gen, err := generator.NewForStyle(testStyle, cfg)
	if err != nil {
		return nil, fmt.Errorf("create generator: %w", err)
	}

	return gen, nil
}

func generateMocks(pkgPath string, info *analyzer.FuncInfo, outputFlag string) error {
	if len(mockFromFlags) == 0 {
		return nil
	}

	// Determine output directory for mock files.
	outDir := ""
	switch {
	case outputFlag != "" && outputFlag != "-":
		outDir = filepath.Dir(outputFlag)
	case info.SourceFile != "":
		outDir = filepath.Dir(info.SourceFile)
	default:
		return fmt.Errorf("cannot determine output directory for mock files")
	}

	for _, mockSpec := range mockFromFlags {
		mockSource, err := source(pkgPath, info, mockSpec)
		if err != nil {
			return fmt.Errorf("generate mock: %w", err)
		}

		formatted, err := analyzer.FormatCode(mockSource.source)
		if err != nil {
			// Don't fail on format errors — write unformatted so user can inspect.
			formatted = mockSource.source
		}

		mockFile := filepath.Join(outDir, generator.MockFileName(mockSource.ifaceName))

		if outputFlag == "-" {
			fmt.Printf("\n// --- mock: %s ---\n", mockFile)
			os.Stdout.Write(formatted)
			continue
		}

		// Don't overwrite existing mock files (user may have customised them).
		if _, err := os.Stat(mockFile); err == nil {
			fmt.Fprintf(os.Stderr, "mock file %s already exists, skipping\n", mockFile)
			continue
		}

		if err := os.WriteFile(mockFile, formatted, 0644); err != nil {
			return fmt.Errorf("write mock file %s: %w", mockFile, err)
		}
	}

	return nil
}

func writeOutput(content []byte, outputFlag, sourceFile, testFuncName string, isMerge bool, info *analyzer.FuncInfo) error {
	var imports map[string]string
	if isMerge && info != nil {
		imports = generator.CollectImports(info)
	}

	switch outputFlag {
	case "-":
		os.Stdout.Write(content)
		return nil
	case "":
		targetPath := analyzer.DeriveTestPath(sourceFile)
		return writeToFile(targetPath, content, testFuncName, isMerge, imports)
	default:
		return writeToFile(outputFlag, content, testFuncName, false, imports)
	}
}

func writeToFile(path string, content []byte, testFuncName string, isMerge bool, imports map[string]string) error {
	if _, err := os.Stat(path); err == nil {
		exists, err := analyzer.TestExistsInFile(path, testFuncName)
		if err != nil {
			return fmt.Errorf("check test exists: %w", err)
		}
		if exists {
			return analyzer.PromptOverwrite(path, testFuncName, content)
		}
		if isMerge {
			return analyzer.MergeTestFile(path, content, imports)
		}
	}
	if isMerge {
		return analyzer.MergeTestFile(path, content, imports)
	}
	return os.WriteFile(path, content, 0644)
}

// parseMockSpec parses a --mock-from value into its components.
//
// Formats:
//
//	"UserRepository"                   → bare name, same package
//	".UserRepository"                  → dot-prefix, same package
//	"userDomain.UserRepo"              → qualifier (alias in consuming package)
//	"io/fs.FS"                         → full import path (contains '/')
//	"net/http.Handler"                 → full import path
//	"github.com/foo/bar.Doer"          → full import path (external module)
func parseMockSpec(spec string) mockSpec {
	lastDot := strings.LastIndex(spec, ".")
	if lastDot < 0 {
		// Bare name: "UserRepository"
		return mockSpec{ifaceName: spec}
	}

	prefix := spec[:lastDot]
	name := spec[lastDot+1:]

	if prefix == "" {
		// ".UserRepository" → same package
		return mockSpec{ifaceName: name}
	}

	if strings.Contains(prefix, "/") {
		// "io/fs.FS", "github.com/foo/bar.Doer" → full import path
		return mockSpec{ifaceName: name, importPath: prefix}
	}

	// "userDomain.UserRepo" or "io.Writer" → qualifier mode
	return mockSpec{qualifier: prefix, ifaceName: name}
}

func source(pkgPath string, info *analyzer.FuncInfo, rawSpec string) (*mockSource, error) {
	var result mockSource
	ms := parseMockSpec(rawSpec)
	result.ifaceName = ms.ifaceName
	result.qualifier = ms.qualifier

	params := &analyzer.InterfaceParams{
		PkgPattern:       pkgPath,
		Qualifier:        ms.qualifier,
		IfaceName:        ms.ifaceName,
		ConsumingAliases: info.ImportAliases,
		ConsumingPkgPath: info.ImportPath,
		DirectImportPath: ms.importPath,
	}

	iface, err := analyzer.LoadInterface(params)
	if err != nil && ms.importPath == "" && ms.qualifier != "" {
		// Fallback for single-segment qualifiers (e.g., "io.Writer"):
		// the qualifier might be a stdlib/package name not imported by the consuming file.
		// Retry treating qualifier as a direct import path.
		params.DirectImportPath = ms.qualifier
		params.Qualifier = ""
		iface, err = analyzer.LoadInterface(params)
	}
	if err != nil {
		return nil, fmt.Errorf("load interface %s: %w", rawSpec, err)
	}

	result.source, err = generator.GenerateMock(generator.MockGenRequest{
		Info:    iface,
		PkgName: info.Package,
	})
	if err != nil {
		return nil, fmt.Errorf("generate mock %s: %w", result.ifaceName, err)
	}

	return &result, nil
}

// runStandaloneMocks generates mock files without a consuming package context.
func runStandaloneMocks() error {
	if pkgFlag == "" {
		return fmt.Errorf("--pkg is required in standalone mock mode (no positional args)")
	}
	if outputFlag == "" {
		return fmt.Errorf("--output is required in standalone mock mode (no positional args)")
	}

	outDir := ""
	if outputFlag != "-" {
		outDir = filepath.Dir(outputFlag)
		// If --output is a directory path (ends with /), use it directly.
		if strings.HasSuffix(outputFlag, "/") || strings.HasSuffix(outputFlag, string(filepath.Separator)) {
			outDir = filepath.Clean(outputFlag)
		}
	}

	for _, rawSpec := range mockFromFlags {
		ms := parseMockSpec(rawSpec)

		params := &analyzer.InterfaceParams{
			IfaceName:        ms.ifaceName,
			Qualifier:        ms.qualifier,
			DirectImportPath: ms.importPath,
		}

		iface, err := analyzer.LoadInterface(params)
		if err != nil && ms.importPath == "" && ms.qualifier != "" {
			// Fallback: try qualifier as direct import path.
			params.DirectImportPath = ms.qualifier
			params.Qualifier = ""
			iface, err = analyzer.LoadInterface(params)
		}
		if err != nil {
			return fmt.Errorf("load interface %s: %w", rawSpec, err)
		}

		src, err := generator.GenerateMock(generator.MockGenRequest{
			Info:    iface,
			PkgName: pkgFlag,
		})
		if err != nil {
			return fmt.Errorf("generate mock %s: %w", ms.ifaceName, err)
		}

		formatted, err := analyzer.FormatCode(src)
		if err != nil {
			formatted = src
		}

		if outputFlag == "-" {
			mockFile := generator.MockFileName(ms.ifaceName)
			fmt.Printf("\n// --- mock: %s ---\n", mockFile)
			os.Stdout.Write(formatted)
			continue
		}

		mockFile := filepath.Join(outDir, generator.MockFileName(ms.ifaceName))

		// Don't overwrite existing mock files.
		if _, err := os.Stat(mockFile); err == nil {
			fmt.Fprintf(os.Stderr, "mock file %s already exists, skipping\n", mockFile)
			continue
		}

		if err := os.WriteFile(mockFile, formatted, 0644); err != nil {
			return fmt.Errorf("write mock file %s: %w", mockFile, err)
		}
		fmt.Fprintf(os.Stderr, "wrote %s\n", mockFile)
	}

	return nil
}
