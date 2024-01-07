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

//go:embed testdata/testAll_gwirl.go
var testAll string

type SimplePosition struct {
    line int
    column int
}
func (p SimplePosition) Line() int {
    return p.line
}
func (p SimplePosition) Column() int {
    return p.column
}

func ptr(t parser.TemplateTree2) *parser.TemplateTree2 {
	return &t
}

func withPos(t *parser.TemplateTree2, pos SimplePosition) parser.TemplateTree2 {
    t.SetPos(pos)
    return *t
}

var tests = []struct {
	filename string
	template parser.Template2
	expected string
}{
	{
		"testdata/simple_gwirl.go",
		parser.NewTemplate2(
			parser.NewPosString("Testing"),
			nil,
			parser.NewPosString("(name string, index int)"),
			[]parser.PosString{},
			[]parser.TemplateTree2{
				parser.NewTT2Plain("<div>\n\t"),
				parser.NewTT2If("if index > 0", []parser.TemplateTree2{
					parser.NewTT2Plain("\n\t\t<hr />\n\t"),
				}, nil, nil),
				parser.NewTT2Plain("\n\t<h2>"),
				parser.NewTT2GoExp("name", true, nil),
				parser.NewTT2Plain("</h2>\n"),
			},
		),
		simple,
	},
	{
		"testdata/testAll_gwirl.go",
		parser.NewTemplate2(
			parser.NewPosString("TestAll"),
			ptr(parser.NewTT2BlockComment("*****************\n* Test Component *\n*****************")),
			parser.NewPosString("(name string, index int)"),
			[]parser.PosString{},
			[]parser.TemplateTree2{
				parser.NewTT2Plain("<div "),
				parser.NewTT2If(" index == 0 ", []parser.TemplateTree2{
					parser.NewTT2Plain(` class="first" `),
				}, []parser.TemplateTree2{}, nil),
				parser.NewTT2Plain(">\n    "),
				parser.NewTT2If(" index > 0 ", []parser.TemplateTree2{
					parser.NewTT2Plain("\n        <hr />\n    "),
				}, []parser.TemplateTree2{}, nil),
				parser.NewTT2Plain("\n    "),
				parser.NewTT2If(` name == "Jeff" `, []parser.TemplateTree2{
					parser.NewTT2Plain("\n        <h2>JEFF</h2>\n    "),
				}, []parser.TemplateTree2{
					parser.NewTT2ElseIf(` name == "Sue" `, []parser.TemplateTree2{
						parser.NewTT2Plain("\n        <h2>SuE</h2>\n    "),
					}),
					parser.NewTT2ElseIf(` name == "Bob" `, []parser.TemplateTree2{
						parser.NewTT2Plain("\n        <h2>B.O.B.</h2>\n    "),
					}),
				}, ptr(parser.NewTT2Else([]parser.TemplateTree2{
					parser.NewTT2Plain("\n        <h2>"),
					parser.NewTT2GoExp("name", false, [][]parser.TemplateTree2{}),
					parser.NewTT2Plain("</h2>\n    "),
				}))),
                parser.NewTT2Plain("\n\n    "),
                withPos(ptr(parser.NewTT2GoExp(`Card("title")`, false, [][]parser.TemplateTree2{
                    {
                        parser.NewTT2Plain("\n        <p>This is content in the card</p>\n    "),
                    },
                    {
                        parser.NewTT2Plain("\n        <button>Card action</button>\n    "),
                    },
                })), SimplePosition{20, 5}),
				parser.NewTT2Plain("\n</div>\n"),
			},
		),
		testAll,
	},
}

func TestGenerator(t *testing.T) {
	for _, test := range tests {
		res := t.Run(test.filename, func(t *testing.T) {
			gen := NewGenerator(false)
			writer := strings.Builder{}
			gen.Generate(test.template, "views", &writer)
			if writer.String() != test.expected {
				edits := myers.ComputeEdits(span.URI(test.filename), test.expected, writer.String())
				diff := gotextdiff.ToUnified("expected", "received", test.expected, edits)
				t.Fatalf("Generated template did not match golden:\n%s", diff)
			}
		})
		if !res {
			t.Fail()
		}
	}
}
