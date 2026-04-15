package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/padiazg/testgen/internal/analyzer"
	"github.com/padiazg/testgen/internal/config"
	"github.com/padiazg/testgen/internal/generator"
	"github.com/spf13/cobra"
)

var (
	outputFlag    string
	verboseFlag   bool
	styleFlag     string
	mockFromFlags []string
	formatFlag    string // "text" | "table" | "json"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "testgen",
		Short: "Generates unit test scaffolding in padiazg style",
	}

	var genCmd = &cobra.Command{
		Use:   "gen <./pkg/path> <FuncSpec>",
		Short: "Generates test scaffolding for a function or method",
		Args:  cobra.ExactArgs(2),
		RunE:  runGen,
	}

	var inspectCmd = &cobra.Command{
		Use:   "inspect <./pkg/path> <FuncSpec>",
		Short: "Shows parsed FuncInfo as JSON for debugging",
		Args:  cobra.ExactArgs(2),
		RunE:  runInspect,
	}

	genCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "output file (default: auto-detect, use '-' for stdout)")
	genCmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "show parsed FuncInfo")
	genCmd.Flags().StringVar(&styleFlag, "style", "", "path to .testgen.yaml config file")
	genCmd.Flags().StringSliceVar(&mockFromFlags, "mock-from", nil, "generate mock for interface (format: qualifier.InterfaceName, repeatable)")

	inspectCmd.Flags().StringVar(&styleFlag, "style", "", "path to .testgen.yaml config file")

	var reportCmd = &cobra.Command{
		Use:   "report <pkg>",
		Short: "Show test/mock status and gen suggestions for all exported functions",
		Args:  cobra.ExactArgs(1),
		RunE:  runReport,
	}
	reportCmd.Flags().StringVar(&formatFlag, "format", "text", "output format: text, table, json")

	rootCmd.AddCommand(genCmd)
	rootCmd.AddCommand(inspectCmd)
	rootCmd.AddCommand(reportCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runGen(cmd *cobra.Command, args []string) error {
	pkgPath := args[0]
	funcSpec := args[1]

	cfg, err := config.Load(styleFlag)
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

	gen := generator.New(cfg)
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

func runInspect(cmd *cobra.Command, args []string) error {
	pkgPath := args[0]
	funcSpec := args[1]

	info, err := analyzer.Load(pkgPath, funcSpec)
	if err != nil {
		return fmt.Errorf("analyze: %w", err)
	}

	b, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	os.Stdout.Write(b)
	os.Stdout.Write([]byte("\n"))
	return nil
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

func runReport(cmd *cobra.Command, args []string) error {
	pkgPattern := args[0]

	result, err := analyzer.ScanPackage(pkgPattern)
	if err != nil {
		return fmt.Errorf("scan package: %w", err)
	}

	switch formatFlag {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	case "table":
		printTableReport(result, pkgPattern)
	default:
		printReport(result, pkgPattern)
	}
	return nil
}

func printTableReport(result *analyzer.ScanResult, pkgPattern string) {
	fmt.Printf("Package: %s\n", result.ImportPath)
	fmt.Printf("Source:  %s\n\n", result.SourceDir)

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleLight)
	t.Style().Options.SeparateRows = true

	t.AppendHeader(table.Row{"Function", "Signature", "Test", "Interface Deps", "Mocks"})
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, Align: text.AlignLeft, WidthMax: 30},
		{Number: 2, Align: text.AlignLeft, WidthMax: 55},
		{Number: 3, Align: text.AlignCenter, WidthMax: 4},
		{Number: 4, Align: text.AlignLeft, WidthMax: 30},
		{Number: 5, Align: text.AlignLeft, WidthMax: 30},
	})

	for _, fn := range result.Funcs {
		testStatus := "✗"
		if fn.TestExists {
			testStatus = "✓"
		}

		var depLines, mockLines []string
		for _, dep := range fn.InterfaceDeps {
			depLines = append(depLines, dep.MockFrom)
			mockStatus := "✗ "
			if dep.MockExists {
				mockStatus = "✓ "
			}
			mockLines = append(mockLines, mockStatus+dep.MockFile)
		}

		t.AppendRow(table.Row{
			fn.FuncSpec,
			fn.Signature,
			testStatus,
			strings.Join(depLines, "\n"),
			strings.Join(mockLines, "\n"),
		})
	}
	t.Render()

	// Suggestions for untested functions.
	var suggestions []string
	for _, fn := range result.Funcs {
		if fn.TestExists {
			continue
		}
		cmd := "  testgen gen " + pkgPattern + " " + fn.FuncSpec
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
	if len(suggestions) > 0 {
		fmt.Println("\nSuggestions:")
		for _, s := range suggestions {
			fmt.Println(s)
		}
	}
}

func printReport(result *analyzer.ScanResult, pkgPattern string) {
	fmt.Printf("Package: %s\n", result.ImportPath)
	fmt.Printf("Source:  %s\n\n", result.SourceDir)

	if len(result.Funcs) == 0 {
		fmt.Println("  No exported functions found.")
		return
	}

	for _, fn := range result.Funcs {
		status := "✗"
		if fn.TestExists {
			status = "✓"
		}
		fmt.Printf("  %s  %s\n", status, fn.TestFuncName)
		fmt.Printf("       %s\n", fn.Signature)

		if len(fn.InterfaceDeps) > 0 {
			fmt.Println("       Interface deps:")
			for _, dep := range fn.InterfaceDeps {
				mockStatus := "✗ (missing)"
				if dep.MockExists {
					mockStatus = "✓"
				}
				fmt.Printf("         %s   %s  %s\n", dep.MockFrom, dep.MockFile, mockStatus)
			}
		}

		if !fn.TestExists {
			// Build the suggested gen command.
			cmd := "testgen gen " + pkgPattern + " " + fn.FuncSpec
			var mockArgs []string
			for _, dep := range fn.InterfaceDeps {
				if !dep.MockExists {
					mockArgs = append(mockArgs, "--mock-from "+dep.MockFrom)
				}
			}
			if len(mockArgs) == 0 {
				fmt.Printf("       Suggest: %s\n", cmd)
			} else if len(mockArgs) == 1 {
				fmt.Printf("       Suggest: %s %s\n", cmd, mockArgs[0])
			} else {
				fmt.Printf("       Suggest: %s \\\n", cmd)
				for i, m := range mockArgs {
					if i < len(mockArgs)-1 {
						fmt.Printf("                  %s \\\n", m)
					} else {
						fmt.Printf("                  %s\n", m)
					}
				}
			}
		}

		fmt.Println()
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
