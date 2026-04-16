package generator

import (
	"fmt"

	"github.com/padiazg/go-testgen/internal/config"
)

// NewForStyle returns the TestGenerator implementation for the given style.
// Passing an empty style defaults to StyleCheck.
func NewForStyle(style TestStyle, cfg *config.Config) (TestGenerator, error) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	switch style {
	case StyleCheck, "":
		return NewCheckGenerator(cfg), nil
	case StyleTable:
		return NewTableGenerator(cfg), nil
	case StyleSimple:
		return NewSimpleGenerator(cfg), nil
	default:
		return nil, fmt.Errorf("unknown test style %q", style)
	}
}
