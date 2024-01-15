package parser_test

import (
	"os"
	"testing"

	"github.com/gamebox/gwirl/internal/parser"
)

type ParsingTest struct {
    name string
    input string
    expected parser.TemplateTree2
}
var noChildren = [][]parser.TemplateTree2{}
var simpleTransclusionChildren = [][]parser.TemplateTree2{
    {
        parser.NewTT2Plain("\n\t<div>Hello</div>\n"),
    },
}

func compareTrees(a parser.TemplateTree2, b parser.TemplateTree2, t *testing.T) {
    if a.Text != b.Text {
        t.Fatalf("Expected \"%s\" but got \"%s\"", a.Text, b.Text)
    }
    if a.Type != b.Type {
        t.Fatalf("Expected type %d but got %d", a.Type, b.Type)
    }
    if a.Metadata != b.Metadata {
        t.Fatalf("Expected metadata %d but got %d", a.Metadata, b.Metadata)
    }
    if a.Children == nil && b.Children != nil {
        t.Fatal("Expected the children to be nil")
    }
    if a.Children != nil && b.Children == nil {
        t.Fatal("Expected the children to not be nil")
    }
    if len(a.Children) != len(b.Children) {
        t.Fatalf("Expected %d children, got %d children", len(a.Children), len(b.Children)) 
    }
    for i := range a.Children {
        aChildTree, bChildtree := a.Children[i], b.Children[i]
        if aChildTree == nil && bChildtree != nil {
            t.Fatal("Expected the children to be nil")
        }
        if aChildTree != nil && bChildtree == nil {
            t.Fatal("Expected the children to not be nil")
        }
        if len(aChildTree) != len(bChildtree) {
            t.Fatalf("Expected child to have %d trees, got %d trees", len(a.Children), len(b.Children)) 
        }
        for childIdx := range aChildTree {
            compareTrees(aChildTree[childIdx], bChildtree[childIdx], t)
        }
    }
}

func runParserTest(tests []ParsingTest, t *testing.T, parseFn func(*parser.Parser2) *parser.TemplateTree2, debug string) {
    for i := range tests {
        test := tests[i]
        if debug != "" && debug != test.name {
            continue
        }
        success := t.Run(test.name, func(t *testing.T) {
            p := parser.NewParser2(test.input)
            if debug != "" {
                p.SetLogger(os.Stdout)
            }
            res := parseFn(&p)
            if res == nil {
                t.Fatal("Expected a result, got nil")
            }
            compareTrees(test.expected, *res, t)
        })
        if !success {
            t.FailNow()
        }
    }
}
