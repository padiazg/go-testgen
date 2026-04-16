package generator

// TestGenerator is the common interface for all test generation styles.
type TestGenerator interface {
	Generate(req GenerateRequest) (*GenerateResult, error)
}
