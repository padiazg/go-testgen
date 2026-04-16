package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewForStyle(t *testing.T) {
	tests := []struct {
		style    TestStyle
		wantType string
		wantErr  bool
	}{
		{StyleCheck, "*generator.CheckGenerator", false},
		{StyleTable, "*generator.TableGenerator", false},
		{StyleSimple, "*generator.SimpleGenerator", false},
		{"", "*generator.CheckGenerator", false},
		{"bad", "", true},
	}
	for _, tt := range tests {
		t.Run(string(tt.style), func(t *testing.T) {
			gen, err := NewForStyle(tt.style, nil)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, gen)
		})
	}
}

func TestNewForStyle_DefaultIsCheck(t *testing.T) {
	gen, err := NewForStyle("", nil)
	require.NoError(t, err)
	_, ok := gen.(*CheckGenerator)
	assert.True(t, ok, "expected *CheckGenerator for empty style")
}
