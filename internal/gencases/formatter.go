package gencases

import (
	"fmt"
	"go/format"
	"os"
)

// Format formats Go source bytes using go/format.
func Format(src []byte) ([]byte, error) {
	out, err := format.Source(src)
	if err != nil {
		return nil, fmt.Errorf("format: %w", err)
	}
	return out, nil
}

// WriteFile writes content to path, or prints to stdout on dry-run.
func WriteFile(path string, content []byte, dryRun bool) error {
	if dryRun {
		os.Stdout.Write(content)
		return nil
	}
	return os.WriteFile(path, content, 0644)
}
