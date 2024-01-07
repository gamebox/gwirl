package main

import (
	_ "embed"
	"testing"

	"github.com/gamebox/gwirl/internal/parser"
	"go.lsp.dev/protocol"
)

//go:embed testdata/test.html.gwirl
var testTemplateFile string

func TestAddImportTokens(t *testing.T) {
	tokens := make([]absToken, 0, 2)
	topImports := []parser.PosString{}
	import1 := parser.NewPosString("import \"fmt\"")
	import1.SetPos(PlainPosition{line: 1, column: 1})
	topImports = append(topImports, import1)
	template := parser.NewTemplate2(parser.NewPosString("Test"), nil, parser.NewPosString(""), topImports, []parser.TemplateTree2{})
	tokens = AddImportsTokens(&template, tokens)
	expected := []absToken{
		{0, 0, 1, protocol.SemanticTokenOperator},
		{0, 2, 6, protocol.SemanticTokenKeyword},
		{0, 7, 5, protocol.SemanticTokenString},
	}
	checkTokens(tokens, expected, t)
}

func TestAddParamsTokens(t *testing.T) {
	tokens := make([]absToken, 0, 4)
	params := parser.NewPosString("(name string, index int)")
	params.SetPos(PlainPosition{line: 1, column: 1})
	template := parser.NewTemplate2(parser.NewPosString("Test"), nil, params, []parser.PosString{}, []parser.TemplateTree2{})
	tokens = AddParamsTokens(&template, tokens)
	expected := []absToken{
		{0, 0, 1, protocol.SemanticTokenOperator},
		{0, 2, 4, protocol.SemanticTokenParameter},
		{0, 7, 6, protocol.SemanticTokenType},
		{0, 15, 5, protocol.SemanticTokenParameter},
		{0, 21, 3, protocol.SemanticTokenType},
	}
	checkTokens(tokens, expected, t)
}

func TestCreateSemanticTokensForTemplate(t *testing.T) {
	p := parser.NewParser2("")
	res := p.Parse(testTemplateFile, "Test")
	if len(res.Errors) > 0 {
		t.Fatal("Template failed to parse")
	}
	tokens := AddCommentsTokens(&res.Template, []absToken{})
	expected := []absToken{
		{0, 0, 36, protocol.SemanticTokenComment},
		{1, 0, 35, protocol.SemanticTokenComment},
		{2, 0, 36, protocol.SemanticTokenComment},
	}
	checkTokens(tokens, expected, t)
	tokens = AddParamsTokens(&res.Template, []absToken{})
	expected = []absToken{
		{3, 0, 1, protocol.SemanticTokenOperator},
		{3, 2, 12, protocol.SemanticTokenParameter},
		{3, 15, 19, protocol.SemanticTokenType},
		{3, 36, 5, protocol.SemanticTokenParameter},
		{3, 42, 3, protocol.SemanticTokenType},
	}
	checkTokens(tokens, expected, t)
	tokens = AddImportsTokens(&res.Template, []absToken{})
	expected = []absToken{
		{5, 0, 1, protocol.SemanticTokenOperator},
		{5, 1, 6, protocol.SemanticTokenKeyword},
		{5, 7, 46, protocol.SemanticTokenString},
	}
	checkTokens(tokens, expected, t)
	tokens = absTokensForContent(res.Template.Content)
}

func checkTokens(received []absToken, expected []absToken, t *testing.T) {
	if len(received) != len(expected) {
		t.Fatalf("Received %d tokens, expected %d tokens", len(received), len(expected))
	}
	for i := range received {
		if received[i].startLine != expected[i].startLine {
			t.Fatalf("Token at index %d did not match expected\n----- Expected -----\n%v\n----- Received ------\n%v", i, received[i], expected[i])
		}
	}
}

func TestSemanticTokensFromAbsTokens(t *testing.T) {
	// tokens := []absToken{
	// 	{0, 0, 36, protocol.SemanticTokenComment},
	// 	{1, 0, 35, protocol.SemanticTokenComment},
	// 	{2, 0, 36, protocol.SemanticTokenComment},
	// 	{3, 2, 12, protocol.SemanticTokenParameter},
	// }
	// res := semanticTokensDataFromAbsTokens(tokens)
	// for i := 0; i < (len(res) / 5); i += 1 {
	// 	t.Logf("Token %d: %v", i, res[5*i:5*i+4])
	// }
	// t.Logf("Result: %v", res)
}
