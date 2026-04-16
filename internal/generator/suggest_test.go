package generator

import (
	"testing"

	"github.com/padiazg/go-testgen/internal/analyzer"
	"github.com/stretchr/testify/assert"
)

func TestSuggestStyle(t *testing.T) {
	tests := []struct {
		name    string
		summary analyzer.FuncSummary
		want    TestStyle
	}{
		{
			name: "no return values → simple",
			summary: analyzer.FuncSummary{
				NumResults: 0,
			},
			want: StyleSimple,
		},
		{
			name: "context + many interface deps → simple",
			summary: analyzer.FuncSummary{
				HasContext:    true,
				NumResults:    1,
				HasError:      true,
				InterfaceDeps: []analyzer.InterfaceDep{{}, {}},
			},
			want: StyleSimple,
		},
		{
			name: "context + one interface dep → check (not simple)",
			summary: analyzer.FuncSummary{
				HasContext:    true,
				IsMethod:      true,
				NumResults:    1,
				HasError:      true,
				InterfaceDeps: []analyzer.InterfaceDep{{}},
			},
			want: StyleCheck,
		},
		{
			name: "method with interface deps → check",
			summary: analyzer.FuncSummary{
				IsMethod:      true,
				NumResults:    1,
				InterfaceDeps: []analyzer.InterfaceDep{{}},
			},
			want: StyleCheck,
		},
		{
			name: "pointer result → check",
			summary: analyzer.FuncSummary{
				NumResults:       1,
				HasPointerResult: true,
			},
			want: StyleCheck,
		},
		{
			name: "slice result → check",
			summary: analyzer.FuncSummary{
				NumResults:     2,
				HasError:       true,
				HasSliceResult: true,
			},
			want: StyleCheck,
		},
		{
			name: "pointer result + error → check (e.g. NewClient() *Client)",
			summary: analyzer.FuncSummary{
				NumResults:       2,
				HasError:         true,
				HasPointerResult: true,
			},
			want: StyleCheck,
		},
		{
			name: "multiple non-error results → check",
			summary: analyzer.FuncSummary{
				NumResults: 3,
				HasError:   true, // 2 non-error results
			},
			want: StyleCheck,
		},
		{
			name: "pure function, single scalar result → table",
			summary: analyzer.FuncSummary{
				NumResults: 1,
			},
			want: StyleTable,
		},
		{
			name: "pure function, scalar result + error → table",
			summary: analyzer.FuncSummary{
				NumResults: 2,
				HasError:   true,
			},
			want: StyleTable,
		},
		{
			name: "method no interface deps, single scalar result → table",
			summary: analyzer.FuncSummary{
				IsMethod:   true,
				NumResults: 1,
			},
			want: StyleTable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SuggestStyle(&tt.summary)
			assert.Equal(t, tt.want, got)
		})
	}
}
