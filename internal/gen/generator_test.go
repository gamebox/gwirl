package gen

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gamebox/gwirl/internal/parser"
)

func TestGenerator(t *testing.T) {
    template := parser.NewTemplate2(
        parser.NewPosString("Testing"),
        nil,
        parser.NewPosString("(name string, index int)"),
        []string{},
        []parser.TemplateTree2{
            parser.NewTT2Plain("<div>\n\t"),
            parser.NewTT2If("if index > 0", []parser.TemplateTree2{
                parser.NewTT2Plain("\n\t\t<hr />\n\t"),
            }, nil, nil),
            parser.NewTT2Plain("\n\t<h2>"),
            parser.NewTT2GoExp("name", nil),
            parser.NewTT2Plain("</h2>\n"),
        },
        
    )

    gen := NewGenerator(false)
    writer := strings.Builder{}
    gen.Generate(template, "views", &writer)
    fmt.Println(writer.String())
}
