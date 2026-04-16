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
	genCmd.Flags().StringSliceVar(&mockFromFlags, "mock-from", nil, "generate mock for interface (format: qualifier.InterfaceName, repeatable)")

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

	testFuncName := analyzer.FindTestFuncName(info)

	var isMerge bool
	if outputFlag == "" {
		targetPath := analyzer.DeriveTestPath(info.SourceFile)
		if _, err := os.Stat(targetPath); err == nil {
			exists, _ := analyzer.TestExistsInFile(targetPath, testFuncName)
			isMerge = !exists
		}
	}

	// Resolve test style: flag > config > default "check".
	styleName := styleFlag
	if styleName == "" {
		styleName = cfg.TestStyle
	}
	testStyle, err := generator.ParseTestStyle(styleName)
	if err != nil {
		return fmt.Errorf("invalid --test-style: %w", err)
	}

	gen, err := generator.NewForStyle(testStyle, cfg)
	if err != nil {
		return fmt.Errorf("create generator: %w", err)
	}

	genReq := generator.GenerateRequest{Info: info, IsMerge: isMerge}
	result, err := gen.Generate(genReq)
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

	return writeOutput(formatted, outputFlag, info.SourceFile, testFuncName, isMerge)
}

func generateMocks(pkgPath string, info *analyzer.FuncInfo, outputFlag string) error {
	if len(mockFromFlags) == 0 {
		return nil
	}

	// Determine output directory for mock files.
	outDir := ""
	if outputFlag != "" && outputFlag != "-" {
		outDir = filepath.Dir(outputFlag)
	} else if info.SourceFile != "" {
		outDir = filepath.Dir(info.SourceFile)
	}
	if outDir == "" {
		return fmt.Errorf("cannot determine output directory for mock files")
	}

	for _, mockSpec := range mockFromFlags {
		parts := strings.SplitN(mockSpec, ".", 2)
		if len(parts) != 2 {
			return fmt.Errorf("--mock-from %q: expected format qualifier.InterfaceName", mockSpec)
		}
		qualifier, ifaceName := parts[0], parts[1]

		iface, err := analyzer.LoadInterface(pkgPath, qualifier, ifaceName, info.ImportAliases, info.ImportPath)
		if err != nil {
			return fmt.Errorf("load interface %s: %w", mockSpec, err)
		}

		src, err := generator.GenerateMock(generator.MockGenRequest{
			Info:    iface,
			PkgName: info.Package,
		})
		if err != nil {
			return fmt.Errorf("generate mock %s: %w", ifaceName, err)
		}

		formatted, err := analyzer.FormatCode(src)
		if err != nil {
			// Don't fail on format errors — write unformatted so user can inspect.
			formatted = src
		}

		mockFile := filepath.Join(outDir, generator.MockFileName(ifaceName))

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

func writeOutput(content []byte, outputFlag, sourceFile, testFuncName string, isMerge bool) error {
	switch outputFlag {
	case "-":
		os.Stdout.Write(content)
		return nil
	case "":
		targetPath := analyzer.DeriveTestPath(sourceFile)
		return writeToFile(targetPath, content, testFuncName, isMerge)
	default:
		return writeToFile(outputFlag, content, testFuncName, false)
	}
}

func writeToFile(path string, content []byte, testFuncName string, isMerge bool) error {
	if _, err := os.Stat(path); err == nil {
		exists, err := analyzer.TestExistsInFile(path, testFuncName)
		if err != nil {
			return fmt.Errorf("check test exists: %w", err)
		}
		if exists {
			return analyzer.PromptOverwrite(path, testFuncName, content)
		}
		if isMerge {
			return analyzer.MergeTestFile(path, content)
		}
	}
	if isMerge {
		return analyzer.MergeTestFile(path, content)
	}
	return os.WriteFile(path, content, 0644)
}
