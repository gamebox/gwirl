package parser_test

import (
    "testing"

    "github.com/gamebox/gwirl/internal/parser"
)

var safeExpressionTests = []ParsingTest{
    {"simple expression", "@(foobar)a", parser.NewTT2GoExpSafe("foobar", false)},
    {"complex expression", "@(foo.bar)a", parser.NewTT2GoExpSafe("foo.bar", false)},
    {"complex method with chaining", "@(foo.bar().something.else)a", parser.NewTT2GoExpSafe("foo.bar().something.else", false)},
    {"complex method with params with chaining", "@(foo.bar(param1, param2).something.else)a", parser.NewTT2GoExpSafe("foo.bar(param1, param2).something.else", false)},
    {"complex method with literal params with chaining", "@(foo.bar(\"hello\", 123).something.else)a", parser.NewTT2GoExpSafe("foo.bar(\"hello\", 123).something.else", false)},
    {"escaped simple expression", "@!(foobar)a", parser.NewTT2GoExpSafe("foobar", true)},
    {"escaped complex expression", "@!(foo.bar)a", parser.NewTT2GoExpSafe("foo.bar", true)},
    {"escaped complex method with chaining", "@!(foo.bar().something.else)a", parser.NewTT2GoExpSafe("foo.bar().something.else", true)},
    {"escaped complex method with params with chaining", "@!(foo.bar(param1, param2).something.else)a", parser.NewTT2GoExpSafe("foo.bar(param1, param2).something.else", true)},
    {"escaped complex method with literal params with chaining", "@!(foo.bar(\"hello\", 123).something.else)a", parser.NewTT2GoExpSafe("foo.bar(\"hello\", 123).something.else", true)},
}

func TestParseSafeExpression(t *testing.T) {
    runParserTest(safeExpressionTests, t, func(p *parser.Parser2) *parser.TemplateTree2 {
        return p.SafeExpression()
    }, "")
}
