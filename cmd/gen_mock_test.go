package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseMockSpec(t *testing.T) {
	tests := []struct {
		name     string
		spec     string
		expected mockSpec
	}{
		{
			name:     "bare name",
			spec:     "UserRepository",
			expected: mockSpec{ifaceName: "UserRepository"},
		},
		{
			name:     "dot-prefix same package",
			spec:     ".UserRepository",
			expected: mockSpec{ifaceName: "UserRepository"},
		},
		{
			name:     "qualifier with alias",
			spec:     "userDomain.UserRepo",
			expected: mockSpec{qualifier: "userDomain", ifaceName: "UserRepo"},
		},
		{
			name:     "single-segment qualifier (stdlib)",
			spec:     "io.Writer",
			expected: mockSpec{qualifier: "io", ifaceName: "Writer"},
		},
		{
			name:     "stdlib with slash",
			spec:     "io/fs.FS",
			expected: mockSpec{importPath: "io/fs", ifaceName: "FS"},
		},
		{
			name:     "stdlib net/http",
			spec:     "net/http.Handler",
			expected: mockSpec{importPath: "net/http", ifaceName: "Handler"},
		},
		{
			name:     "external module",
			spec:     "github.com/foo/bar.Doer",
			expected: mockSpec{importPath: "github.com/foo/bar", ifaceName: "Doer"},
		},
		{
			name:     "external module with version",
			spec:     "github.com/foo/bar/v2.Doer",
			expected: mockSpec{importPath: "github.com/foo/bar/v2", ifaceName: "Doer"},
		},
		{
			name:     "deeply nested module",
			spec:     "github.com/org/repo/internal/domain.Service",
			expected: mockSpec{importPath: "github.com/org/repo/internal/domain", ifaceName: "Service"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMockSpec(tt.spec)
			assert.Equal(t, tt.expected, result)
		})
	}
}
