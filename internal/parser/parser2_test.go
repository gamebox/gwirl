package parser_test


import (
	"io/fs"
	"os"
	"testing"

	"github.com/gamebox/gwirl/internal/parser"
)

func TestParseTestAll2(t *testing.T) {
    root := "./testdata/"
    filesystem := os.DirFS(root)
    template, err := fs.ReadFile(filesystem, "testAll.html.gwirl")
    if err != nil {
        t.FailNow()
    }
    p := parser.NewParser2("")
    result := p.Parse(string(template), "Test")
    expected := "****************\n* Test Component *\n****************"
    if result.Template.Comment == nil {
        t.Errorf("Expected a comment, received nil")
        t.FailNow()
    }
    if  result.Template.Comment.Text != expected {
        t.Errorf("Got comment \"%s\", expected \"%s\"", result.Template.Comment.Text, expected)
    }
    if len(result.Errors) > 0 {
        t.Errorf("Expected no errors, found %d errors", len(result.Errors))
    }
}
