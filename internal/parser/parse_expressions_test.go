package parser_test

import (
	"testing"

	"github.com/gamebox/gwirl/internal/parser"
)

var expressionTests = []ParsingTest{
    {"simple expression", "@foobar\"", parser.NewTT2GoExp("foobar", false, noChildren)},
    {"simple method", "@foobar()\"", parser.NewTT2GoExp("foobar()", false, noChildren)},
    {"complex expression", "@foo.bar\"", parser.NewTT2GoExp("foo.bar", false, noChildren)},
    {"complex method", "@foo.bar()\"", parser.NewTT2GoExp("foo.bar()", false, noChildren)},
    {"complex method with params", "@foo.bar(param1, param2)\"", parser.NewTT2GoExp("foo.bar(param1, param2)", false, noChildren)},
    {"complex method with literal params", "@foo.bar(\"hello\", 123)\"", parser.NewTT2GoExp("foo.bar(\"hello\", 123)", false, noChildren)},
    {"complex method with struct literal param", "@foo.bar(MyStruct{something, else}, 123)\"", parser.NewTT2GoExp("foo.bar(MyStruct{something, else}, 123)", false, noChildren)},
    {"complex method with chaining", "@foo.bar().something.else\"", parser.NewTT2GoExp("foo.bar().something.else", false, noChildren)},
    {"complex method with params with chaining", "@foo.bar(param1, param2).something.else\"", parser.NewTT2GoExp("foo.bar(param1, param2).something.else", false, noChildren)},
    {"complex method with literal params with chaining", "@foo.bar(\"hello\", 123).something.else\"", parser.NewTT2GoExp("foo.bar(\"hello\", 123).something.else", false, noChildren)},
   
    // Transclusion tests
    {
        "simple method with transclusion",
        "@foobar() {\n\t<div>Hello</div>\n}",
        parser.NewTT2GoExp(
            "foobar()",
            false,
            simpleTransclusionChildren,
        ),
    },

    {
        "complex method with transclusion",
        "@foo.bar() {\n\t<div>Hello</div>\n}",
        parser.NewTT2GoExp(
            "foo.bar()",
            false,
            simpleTransclusionChildren,
        ),
    },

    {
        "complex method with param with transclusion",
        "@foo.bar(param1, param2) {\n\t<div>Hello</div>\n}",
        parser.NewTT2GoExp(
            "foo.bar(param1, param2)",
            false,
            simpleTransclusionChildren,
        ),
    },

    {
        "complex method with literal params with transclusion",
        "@foo.bar(\"hello\", 123) {\n\t<div>Hello</div>\n}",
        parser.NewTT2GoExp(
            "foo.bar(\"hello\", 123)",
            false,
            simpleTransclusionChildren,
        ),
    },
}

func TestExpressionParsing(t *testing.T) {
    runParserTest(expressionTests, t, func (p *parser.Parser2) *parser.TemplateTree2 {
        return p.Expression()
    },"")
}

