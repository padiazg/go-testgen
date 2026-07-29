package spec

import (
	"fmt"
	"go/format"
	"os"
)

// format formats Go source bytes using go/format.
func formatSource(src []byte) ([]byte, error) {
	out, err := format.Source(src)
	if err != nil {
		return nil, fmt.Errorf("format: %w", err)
	}
	return out, nil
}

// writeFile writes content to path, or prints to stdout on dry-run.
func writeFile(path string, content []byte, dryRun bool) error {
	if dryRun {
		_, _ = os.Stdout.Write(content)
		return nil
	}
	return os.WriteFile(path, content, 0644)
}
