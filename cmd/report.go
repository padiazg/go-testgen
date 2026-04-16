/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/padiazg/go-testgen/internal/analyzer"
	"github.com/padiazg/go-testgen/internal/generator"
	"github.com/spf13/cobra"
)

var (
	formatFlag string // "text" | "table" | "json"

	// reportCmd represents the report command
	reportCmd = &cobra.Command{
		Use:   "report <pkg>",
		Short: "Show test/mock status and gen suggestions for all exported functions",
		Args:  cobra.ExactArgs(1),
		RunE:  runReport,
	}
)

func init() {
	rootCmd.AddCommand(reportCmd)
	reportCmd.Flags().StringVar(&formatFlag, "format", "text", "output format: text, table, json")
}

func runReport(cmd *cobra.Command, args []string) error {
	pkgPattern := args[0]

	result, err := analyzer.ScanPackage(pkgPattern, true)
	if err != nil {
		return fmt.Errorf("scan package: %w", err)
	}

	// Populate suggested style for each function.
	for i := range result.Funcs {
		result.Funcs[i].SuggestedStyle = generator.SuggestStyle(&result.Funcs[i]).String()
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

	t.AppendHeader(table.Row{"Function", "Signature", "Test", "Style", "Interface Deps", "Mocks"})
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, Align: text.AlignLeft, WidthMax: 30},
		{Number: 2, Align: text.AlignLeft, WidthMax: 50},
		{Number: 3, Align: text.AlignCenter, WidthMax: 4},
		{Number: 4, Align: text.AlignLeft, WidthMax: 8},
		{Number: 5, Align: text.AlignLeft, WidthMax: 28},
		{Number: 6, Align: text.AlignLeft, WidthMax: 28},
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
			fn.SuggestedStyle,
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
		cmd := "  go-testgen gen " + pkgPattern + " " + fn.FuncSpec
		if fn.SuggestedStyle != "" && fn.SuggestedStyle != "check" {
			cmd += " --test-style " + fn.SuggestedStyle
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
			cmd := "go-testgen gen " + pkgPattern + " " + fn.FuncSpec
			if fn.SuggestedStyle != "" && fn.SuggestedStyle != "check" {
				cmd += " --test-style " + fn.SuggestedStyle
			}
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
