package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTypeToString_ChannelDirections(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"bidi channel", "chan int", "chan int"},
		{"recv-only channel", "<-chan int", "<-chan int"},
		{"send-only channel", "chan<- int", "chan<- int"},
		{"channel of string", "chan string", "chan string"},
		{"channel of pointer", "chan *int", "chan *int"},
		{"recv channel of pointer", "<-chan *int", "<-chan *int"},
		{"channel of slice", "chan []string", "chan []string"},
		{"recv channel of slice", "<-chan []string", "<-chan []string"},
		{"send channel of slice", "chan<- []string", "chan<- []string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			code := "package p\ntype T " + tt.input
			f, err := parser.ParseFile(fset, "test.go", code, 0)
			assert.NoError(t, err)

			var typeSpec *ast.TypeSpec
			for _, decl := range f.Decls {
				if gd, ok := decl.(*ast.GenDecl); ok {
					if ts, ok := gd.Specs[0].(*ast.TypeSpec); ok {
						typeSpec = ts
						break
					}
				}
			}
			assert.NotNil(t, typeSpec)

			got := typeToString(typeSpec.Type)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractTypePrefix_ChannelDirections(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"bidi channel", "chan int", "chan "},
		{"recv-only channel", "<-chan int", "<-chan "},
		{"send-only channel", "chan<- int", "chan<- "},
		{"channel of slice", "chan []int", "chan []"},
		{"channel of pointer", "chan *int", "chan *"},
		{"recv of slice pointer", "<-chan []*int", "<-chan []*"},
		{"send of slice pointer", "chan<- []*int", "chan<- []*"},
		{"recv of slice", "<-chan []int", "<-chan []"},
		{"send of slice", "chan<- []int", "chan<- []"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			code := "package p\ntype T " + tt.input
			f, err := parser.ParseFile(fset, "test.go", code, 0)
			assert.NoError(t, err)

			var typeSpec *ast.TypeSpec
			for _, decl := range f.Decls {
				if gd, ok := decl.(*ast.GenDecl); ok {
					if ts, ok := gd.Specs[0].(*ast.TypeSpec); ok {
						typeSpec = ts
						break
					}
				}
			}
			assert.NotNil(t, typeSpec)

			got := extractTypePrefix(typeSpec.Type)
			assert.Equal(t, tt.want, got)
		})
	}
}
