package gen

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/gamebox/gwirl/internal/parser"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
)

//go:embed testdata/simple_gwirl.go
var simple string

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
            parser.NewTT2GoExp("name", true, nil),
            parser.NewTT2Plain("</h2>\n"),
        },
        
    )

    gen := NewGenerator(false)
    writer := strings.Builder{}
    gen.Generate(template, "views", &writer)
    if writer.String() != simple {
        edits := myers.ComputeEdits(span.URI("testdata/simple_gwirl.go"), simple, writer.String())
        diff := gotextdiff.ToUnified("expected", "received", simple, edits)
        t.Fatalf("Generated template did not match golden:\n%s", diff)
    }
}
