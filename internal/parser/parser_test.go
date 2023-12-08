package parser_test

import (
	"io/fs"
	"os"
	"testing"

	"github.com/gamebox/gwirl/internal/parser"
)

func TestCommentParse(t *testing.T) {
    p := parser.NewParser("@* This is a test *@")
    comment, err := p.Comment()
    if err != nil {
        t.Fatalf("Error")
    }
    if comment.Text != " This is a test " {
        t.Fatalf("comment text was '%s'", comment.Text)
    }
}

func TestMultilineCommentParse(t *testing.T) {
    p := parser.NewParser(`@*
This is a multiline comment.
It is good.
*@`)

    comment, err := p.Comment()

    if err != nil {
        t.Fatal("error")
    }
    if comment.Text != "\nThis is a multiline comment.\nIt is good.\n" {
        t.Fatalf("comment text was '%s'", comment.Text)
    }
}

func TestLastCommentParse(t *testing.T) {
    p := parser.NewParser(`@*
Comment 1
*@
@* Comment 2 *@ @* Comment 3 *@`)
    comment, _ := p.LastComment()
    if comment == nil {
        t.FailNow()
    }
    expected := " Comment 3 "
    if comment.Text != expected {
        t.Errorf("Comment was '%s', expected '%s'", comment.Text, expected)
    }
}

func TestImportExpression(t *testing.T) {
    p := parser.NewParser("@import \"github.com/gamebox/gwirl\"\n")
    exp, _ := p.ImportExpression()
    if exp == nil {
        t.FailNow()
    }
    expected := "import \"github.com/gamebox/gwirl\""
    if exp.Code != expected {
        t.Errorf("Import was '%s', expected '%s'", exp.Code, expected)
    }
}

func TestGoBlock(t *testing.T) {
    p := parser.NewParser("@{ someGoFunc(\"Hi\") }")
    simple := p.GoBlock()
    if simple == nil {
        t.FailNow()
    }
    expected := "{ someGoFunc(\"Hi\") }"
    if simple.Code != expected {
        t.Errorf("Code was '%s', expected '%s'", simple.Code, expected)
    }
}

func TestParseTestAll(t *testing.T) {
    root := "./testdata/"
    filesystem := os.DirFS(root)
    template, err := fs.ReadFile(filesystem, "testAll.twirl.html")
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
