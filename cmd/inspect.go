/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/padiazg/go-testgen/internal/analyzer"
	"github.com/spf13/cobra"
)

// inspectCmd represents the inspect command
var (
	inspectCmd = &cobra.Command{
		Use:   "inspect <./pkg/path> <FuncSpec>",
		Short: "Shows parsed FuncInfo as JSON for debugging",
		Args:  cobra.ExactArgs(2),
		RunE:  runInspect,
	}
)

func init() {
	rootCmd.AddCommand(inspectCmd)
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
