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

// genCmd represents the gen command
var (
	configFlag    string
	outputFlag    string
	verboseFlag   bool
	styleFlag     string
	mockFromFlags []string

	genCmd = &cobra.Command{
		Use:   "gen <./pkg/path> <FuncSpec>",
		Short: "Generates test scaffolding for a function or method",
		Args:  cobra.ExactArgs(2),
		RunE:  runGen,
	}
)

func init() {
	rootCmd.AddCommand(genCmd)
	genCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "output file (default: auto-detect, use '-' for stdout)")
	genCmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "show parsed FuncInfo")
	genCmd.Flags().StringVar(&configFlag, "config", "", "path to .go-testgen.yaml config file")
	genCmd.Flags().StringVar(&styleFlag, "style", "", "test generation style: check, table, simple (default: from config or check)")
	genCmd.Flags().StringSliceVar(&mockFromFlags, "mock-from", nil, "generate mock for interface (format: qualifier.InterfaceName or .InterfaceName for same package, repeatable)")

}

func runGen(cmd *cobra.Command, args []string) error {
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

func interfaceName(mockSpec string) (string, string) {
	var qualifier, ifaceName string

	switch {
	case strings.HasPrefix(mockSpec, ".") && len(mockSpec) > 1:
		qualifier = ""
		ifaceName = mockSpec[1:]
	case strings.Contains(mockSpec, "."):
		parts := strings.SplitN(mockSpec, ".", 2)
		qualifier, ifaceName = parts[0], parts[1]
	default:
		qualifier = ""
		ifaceName = mockSpec
	}

	return qualifier, ifaceName
}

func source(pkgPath string, info *analyzer.FuncInfo, mockSpec string) (*mockSource, error) {
	var result mockSource

	result.qualifier, result.ifaceName = interfaceName(mockSpec)

	// pkgPath, result.qualifier, result.ifaceName, info.ImportAliases, info.ImportPath
	iface, err := analyzer.LoadInterface(&analyzer.InterfaceParams{
		PkgPattern:       pkgPath,
		Qualifier:        result.qualifier,
		IfaceName:        result.ifaceName,
		ConsumingAliases: info.ImportAliases,
		ConsumingPkgPath: info.ImportPath,
	})
	if err != nil {
		return nil, fmt.Errorf("load interface %s: %w", mockSpec, err)
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
