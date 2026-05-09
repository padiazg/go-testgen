package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQualifiedTypeName_ChannelPrefixes(t *testing.T) {
	tests := []struct {
		name       string
		typeName   string
		pkg        string
		want       string
	}{
		{"recv-only of pointer", "<-chan *Foo", "pkg", "<-chan *pkg.Foo"},
		{"send-only of named", "chan<- Foo", "pkg", "chan<- pkg.Foo"},
		{"bidi of slice", "chan []Foo", "pkg", "chan []pkg.Foo"},
		{"recv-only of slice pointer", "<-chan []*Foo", "pkg", "<-chan []*pkg.Foo"},
		{"send-only of slice pointer", "chan<- []*Foo", "pkg", "chan<- []*pkg.Foo"},
		{"recv-only of slice", "<-chan []Foo", "pkg", "<-chan []pkg.Foo"},
		{"send-only of slice", "chan<- []Foo", "pkg", "chan<- []pkg.Foo"},
		{"bidi channel", "chan Foo", "pkg", "chan pkg.Foo"},
		{"bidi of pointer", "chan *Foo", "pkg", "chan *pkg.Foo"},
		{"no prefix", "Foo", "pkg", "pkg.Foo"},
		{"pointer prefix", "*Foo", "pkg", "*pkg.Foo"},
		{"slice prefix", "[]Foo", "pkg", "[]pkg.Foo"},
		{"slice pointer prefix", "[]*Foo", "pkg", "[]*pkg.Foo"},
		{"empty qualifier", "chan Foo", "", "chan Foo"},
		{"bidi of MyType", "chan MyType", "pkg", "chan pkg.MyType"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := qualifiedTypeName(tt.typeName, tt.pkg)
			assert.Equal(t, tt.want, got)
		})
	}
}
