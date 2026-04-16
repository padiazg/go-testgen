package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTestStyle(t *testing.T) {
	tests := []struct {
		input   string
		want    TestStyle
		wantErr bool
	}{
		{"check", StyleCheck, false},
		{"table", StyleTable, false},
		{"simple", StyleSimple, false},
		{"", StyleCheck, false},
		{"unknown", "", true},
		{"CHECK", "", true},
		{"Table", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseTestStyle(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTestStyleString(t *testing.T) {
	assert.Equal(t, "check", StyleCheck.String())
	assert.Equal(t, "table", StyleTable.String())
	assert.Equal(t, "simple", StyleSimple.String())
}
