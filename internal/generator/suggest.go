package generator

import "github.com/padiazg/go-testgen/internal/analyzer"

// SuggestStyle recommends a test generation style for a function based on
// its signature characteristics and interface dependencies.
//
// Heuristics (evaluated top to bottom, first match wins):
//   - simple: no return values, OR context param + many interface deps (>=2)
//   - check:  pointer/slice result (hard to compare as whole value),
//     method with interface deps, OR multiple non-error return values,
//     OR returns interface (interfaces are hard to compare)
//   - table:  everything else (pure functions, scalar types, no interface deps)
func SuggestStyle(summary *analyzer.FuncSummary) TestStyle {
	numIfaceDeps := len(summary.InterfaceDeps)

	// No return values at all: standalone test makes more sense.
	if summary.NumResults == 0 {
		return StyleSimple
	}

	// Context param + many interface deps: integration-style, standalone test.
	if summary.HasContext && numIfaceDeps >= 2 {
		return StyleSimple
	}

	// Pointer or slice result: comparing the whole value is impractical,
	// check-functions let you assert individual fields/elements.
	if summary.HasPointerResult || summary.HasSliceResult {
		return StyleCheck
	}

	// Returns interface: interfaces are hard to compare as whole values,
	// check-functions allow asserting individual aspects.
	if summary.ReturnsInterface {
		return StyleCheck
	}

	// Method with interface deps: complex setup, use check-functions.
	if summary.IsMethod && numIfaceDeps > 0 {
		return StyleCheck
	}

	// Multiple non-error return values: hard to predict upfront, use check-functions.
	nonErrResults := summary.NumResults
	if summary.HasError {
		nonErrResults--
	}
	if nonErrResults > 1 {
		return StyleCheck
	}

	// Default: scalar return signature, suitable for table+want.
	return StyleTable
}
