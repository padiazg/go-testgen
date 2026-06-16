/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"errors"

	"github.com/padiazg/go-testgen/internal/gencases"
	"github.com/padiazg/go-testgen/internal/spec"
	"github.com/spf13/cobra"
)

var (
	gcDryRun  bool
	gcOutput  string
	gcForce   bool
	gcNoHints bool
	gcVerbose bool

	genCasesCmd = &cobra.Command{
		Use:   "gen-cases <spec-file>",
		Short: "[EXPERIMENTAL!] Materialize test cases from a .testspec.yaml into a _test.go file",
		Args:  cobra.ExactArgs(1),
		RunE:  runGenCases,
	}
)

func init() {
	rootCmd.AddCommand(genCasesCmd)
	genCasesCmd.Flags().BoolVar(&gcDryRun, "dry-run", false, "print generated code to stdout without modifying any file")
	genCasesCmd.Flags().StringVarP(&gcOutput, "output", "o", "", "override the output _test.go path")
	genCasesCmd.Flags().BoolVar(&gcForce, "force", false, "replace existing entries instead of skipping them")
	genCasesCmd.Flags().BoolVar(&gcNoHints, "no-hints", false, "omit // ai-hint: comments from output")
	genCasesCmd.Flags().BoolVarP(&gcVerbose, "verbose", "v", false, "print a summary of generated/skipped entries")
}

func runGenCases(cmd *cobra.Command, args []string) error {
	if !cmd.Parent().Flags().Changed("experimental") {
		return errors.New("gen-cases is experimental: re-run with --experimental flag")
	}

	s, err := spec.ParseFile(args[0])
	if err != nil {
		return err
	}

	opts := gencases.Options{
		DryRun:  gcDryRun,
		Output:  gcOutput,
		Force:   gcForce,
		NoHints: gcNoHints,
		Verbose: gcVerbose,
	}
	return gencases.Run(s, opts)
}
