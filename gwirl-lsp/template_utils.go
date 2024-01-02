package main

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/gamebox/gwirl/internal/parser"
	lsp "go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

type PlainPosition struct {
	line   int
	column int
}

type templateEntry struct {
	name string
	path string
}

func TemplateName(fileName string) string {
	str := strings.TrimSuffix(fileName, ".html.gwirl")
	name := strings.TrimSuffix(path.Base(str), ".html.gwirl")
	return strings.Replace(name, string(name[0]), string(name[0]-32), 1)
}

func TemplateNames(fileContents map[uri.URI]string) []templateEntry {
	entries := make([]templateEntry, 0, 100)
	for filepath := range fileContents {
		name := strings.TrimSuffix(path.Base(filepath.Filename()), ".html.gwirl")
		name = strings.Replace(name, string(name[0]), string(name[0]-32), 1)
		entries = append(entries, templateEntry{
			path: filepath.Filename(),
			name: name,
		})
	}

	return entries
}

func (pos PlainPosition) Line() int {
	return pos.line
}

func (pos PlainPosition) Column() int {
	return pos.column
}

func LspPosToParserPos(pos lsp.Position) parser.Position {
	return PlainPosition{
		line:   int(pos.Line + 1),
		column: int(pos.Character),
	}
}

func ParserPosToLspPos(pos parser.Position) lsp.Position {
	return lsp.Position{
		Line:      uint32(pos.Line() - 1),
		Character: uint32(pos.Column()),
	}
}

func GetTemplateParamNames(t *parser.Template2) []string {
	tParamsStr := strings.TrimLeft(t.Params.Str, "(")
	tParamsStr = strings.TrimRight(tParamsStr, ")")
	tParamsFull := strings.Split(tParamsStr, ",")
	tParamNames := []string{}
	for _, p := range tParamsFull {
		tParamNames = append(tParamNames, strings.Split(p, " ")[0])
	}
	return tParamNames
}

func FindTemplateTreeForPosition(tts []parser.TemplateTree2, pos parser.Position) *parser.TemplateTree2 {
	var result *parser.TemplateTree2 = nil
	var previous *parser.TemplateTree2 = nil
	for idx := range tts {
		tt := tts[idx]
		nodeLineIsBeforePosLine := tt.Line() < pos.Line()
		nodeLineIsAfterPosLine := tt.Line() > pos.Line()
		nodeLineIsEqualToPosLine := tt.Line() == pos.Line()
		nodeColumnIsBeforePosColumn := tt.Column() < pos.Column()
		nodeColumnIsAfterOrEqualToPosColumn := tt.Column() >= pos.Column()
		if nodeLineIsAfterPosLine || (nodeLineIsEqualToPosLine && nodeColumnIsAfterOrEqualToPosColumn) {
			result = previous
			break
		}
		if nodeLineIsBeforePosLine || (nodeLineIsEqualToPosLine && nodeColumnIsBeforePosColumn) {
			switch tt.Type {
			case parser.TT2If, parser.TT2Else, parser.TT2ElseIf, parser.TT2GoExp, parser.TT2For:
				if tt.Children == nil || len(tt.Children) == 0 {
				} else {
					for _, childTrees := range tt.Children {
						res := FindTemplateTreeForPosition(childTrees, pos)
						if res != nil {
							return res
						}
					}
				}
			default:
			}
			previous = &tt
		}
	}
	return result
}

type absToken struct {
	startLine      uint32
	startCharacter uint32
	length         uint32
	tokenType      lsp.SemanticTokenTypes
}

func (t absToken) String() string {
	return fmt.Sprintf("#%v(%d:%d<%d>),\n", t.tokenType, t.startLine, t.startCharacter, t.length)
}

func NewAbsToken(startLine uint32, startCharacter uint32, length int, tokenType lsp.SemanticTokenTypes) absToken {
	return absToken{
		startLine:      startLine,
		startCharacter: startCharacter,
		length:         uint32(length),
		tokenType:      tokenType,
	}
}

type absTokenList []absToken

func (a absTokenList) Len() int { return len(a) }
func (a absTokenList) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a absTokenList) Less(i, j int) bool {
	if a[i].startLine < a[j].startLine {
		return true
	}
	if a[i].startLine == a[j].startLine && a[i].startCharacter < a[j].startCharacter {
		return true
	}
	return false
}

var SemanticTokenLegend = []lsp.SemanticTokenTypes{
	lsp.SemanticTokenParameter,
	lsp.SemanticTokenType,
	lsp.SemanticTokenString,
	lsp.SemanticTokenKeyword,
	lsp.SemanticTokenComment,
	lsp.SemanticTokenVariable,
	lsp.SemanticTokenOperator,
}

func subUint32(x uint32, y uint32) uint32 {
	if y > x {
		return 0
	}
	return x - y
}

func semanticTokensDataFromAbsTokens(absTokens []absToken) []uint32 {
	tokens := make([]uint32, len(absTokens)*5)
	var prevPos *lsp.Position
	sort.Sort(absTokenList(absTokens))
	for i, t := range absTokens {
		var line, char uint32
		if prevPos == nil {
			line = uint32(t.startLine)
			char = uint32(t.startCharacter)
		} else if t.startLine == prevPos.Line {
			line = 0
			char = subUint32(t.startCharacter, prevPos.Character)
		} else {
			line = subUint32(t.startLine, prevPos.Line)
			char = t.startCharacter
		}
		var tokenType uint32 = 0
		var idx uint32 = 0
		for idx < uint32(len(SemanticTokenLegend)) {
			if SemanticTokenLegend[idx] == t.tokenType {
				tokenType = idx
				break
			}
			idx += 1
		}
		tokens[5*i] = line
		tokens[5*i+1] = char
		tokens[5*i+2] = uint32(t.length)
		tokens[5*i+3] = tokenType
		tokens[5*i+4] = 0
		if prevPos == nil {
			prevPos = &lsp.Position{Line: t.startLine, Character: t.startCharacter}
		} else {
			prevPos.Line = t.startLine
			prevPos.Character = t.startCharacter
		}
	}
	return tokens
}

func absTokensForChildren(children [][]parser.TemplateTree2) []absToken {
	tokens := make([]absToken, 0, len(children)*2)

	for _, child := range children {
		ts := absTokensForContent(child)
		tokens = append(tokens, ts...)
	}

	return tokens
}
func absTokensForContent(tt []parser.TemplateTree2) []absToken {
	tokens := make([]absToken, 0, len(tt)*2)

	for _, t := range tt {
		lsppos := ParserPosToLspPos(&t)
		startLine := lsppos.Line
		startCol := lsppos.Character
		switch t.Type {
		case parser.TT2If:
			length := 2
			atToken := NewAbsToken(startLine, startCol-1, 1, lsp.SemanticTokenOperator)
			token := NewAbsToken(startLine, startCol, length, lsp.SemanticTokenKeyword)
			tokens = append(tokens, atToken, token)
			if t.Children != nil {
				blockTokens := absTokensForChildren(t.Children)
				tokens = append(tokens, blockTokens...)
			}
		case parser.TT2For:
			length := 3
			atToken := NewAbsToken(startLine, startCol-1, 1, lsp.SemanticTokenOperator)
			token := NewAbsToken(startLine, startCol, length, lsp.SemanticTokenKeyword)
			tokens = append(tokens, atToken, token)
			if t.Children == nil {
				continue
			}
			blockTokens := absTokensForChildren(t.Children)
			tokens = append(tokens, blockTokens...)
		case parser.TT2Else:
			length := 4
			atToken := NewAbsToken(startLine, startCol-1, 1, lsp.SemanticTokenOperator)
			token := NewAbsToken(startLine, startCol, length, lsp.SemanticTokenKeyword)
			tokens = append(tokens, atToken, token)
			if t.Children == nil {
				continue
			}
			blockTokens := absTokensForChildren(t.Children)
			tokens = append(tokens, blockTokens...)
		case parser.TT2GoExp:
			length := len(t.Text)
			var atToken absToken
			if t.Metadata.Has(parser.TTMDEscape) {
				atToken = NewAbsToken(startLine, startCol-2, 2, lsp.SemanticTokenOperator)
			} else {
				atToken = NewAbsToken(startLine, startCol-1, 1, lsp.SemanticTokenOperator)
			}
			token := NewAbsToken(startLine, startCol, length, lsp.SemanticTokenParameter)
			tokens = append(tokens, atToken, token)
			if t.Children == nil {
				continue
			}
			blockTokens := absTokensForChildren(t.Children)
			tokens = append(tokens, blockTokens...)
		case parser.TT2ElseIf:
			length := 7
			atToken := NewAbsToken(startLine, startCol-1, 1, lsp.SemanticTokenOperator)
			token := NewAbsToken(startLine, startCol, length, lsp.SemanticTokenKeyword)
			tokens = append(tokens, atToken, token)
			if t.Children == nil {
				continue
			}
			blockTokens := absTokensForChildren(t.Children)
			tokens = append(tokens, blockTokens...)
		case parser.TT2BlockComment:
			startCol = subUint32(lsppos.Character, 3)
			lines := strings.Split(t.Text, "\n")
			for i, l := range lines {
				var length int
				if i == 0 || i == (len(lines)-1) {
					length = len(l) + 3
				} else if len(lines) == 1 {
					length = len(l) + 6
				} else {
					length = len(l)
				}
				if i > 0 {
					startCol = 0
				}
				token := NewAbsToken(startLine+uint32(i), startCol, length, lsp.SemanticTokenComment)
				tokens = append(tokens, token)
			}
		case parser.TT2LineComment:
		}
	}

	return tokens
}

func AddParamsTokens(t *parser.Template2, absTokens []absToken) []absToken {
	ps := strings.Split(strings.Trim(t.Params.Str, "()"), ", ")
	lsppos := ParserPosToLspPos(&t.Params)
	paramsLine := lsppos.Line
	paramsColumn := lsppos.Character + 1
	atToken := NewAbsToken(paramsLine, subUint32(lsppos.Character, 1), 1, lsp.SemanticTokenOperator)
	absTokens = append(absTokens, atToken)
	for _, p := range ps {
		parts := strings.SplitN(p, " ", 2)
		if len(parts) != 2 {
			continue
		}
		nameT := NewAbsToken(paramsLine, paramsColumn, len(parts[0]), lsp.SemanticTokenParameter)
		typeT := NewAbsToken(paramsLine, paramsColumn+uint32(len(parts[0]))+1, len(parts[1]), lsp.SemanticTokenType)
		absTokens = append(absTokens, nameT, typeT)
		paramsColumn += uint32(len(p)) + 2
	}
	return absTokens
}

func AddImportsTokens(t *parser.Template2, absTokens []absToken) []absToken {
	for _, imp := range t.TopImports {
		lsppos := ParserPosToLspPos(&imp)
		startLine := lsppos.Line
		startCol := lsppos.Character
		length := 6
		atToken := NewAbsToken(startLine, subUint32(startCol, 1), 1, lsp.SemanticTokenOperator)
		impT := NewAbsToken(startLine, startCol, length, lsp.SemanticTokenKeyword)
		startCol += 7
		str, _ := strings.CutPrefix(imp.Str, "import ")
		if strings.HasPrefix(str, "\"") {
			pkgT := NewAbsToken(startLine, startCol, len(str), lsp.SemanticTokenString)
			absTokens = append(absTokens, atToken, impT, pkgT)
		} else {
			absTokens = append(absTokens, atToken, impT)
		}
	}
	return absTokens
}

func AddCommentsTokens(t *parser.Template2, absTokens []absToken) []absToken {
	if t.Comment != nil {
		commentTs := absTokensForContent([]parser.TemplateTree2{*t.Comment})
		absTokens = append(absTokens, commentTs...)
	}
	return absTokens
}

func CreateTemplateSemanticTokens(t *parser.Template2) *lsp.SemanticTokens {
	absTokens := make([]absToken, 0, 50)

	absTokens = AddCommentsTokens(t, absTokens)
	absTokens = AddParamsTokens(t, absTokens)
	absTokens = AddImportsTokens(t, absTokens)
	tokens := absTokensForContent(t.Content)
	absTokens = append(absTokens, tokens...)

	return &lsp.SemanticTokens{
		Data: semanticTokensDataFromAbsTokens(absTokens),
	}
}

func splitOnSet(s string, splitSet string) []string {
	res := []string{}
	str := s
	for str != "" {
		minIndex := len(str)
		for _, setRune := range splitSet {
			i := strings.IndexRune(str, setRune)
			if i > -1 && i < minIndex {
				minIndex = i
			}
		}
		res = append(res, str[:minIndex])
		if minIndex == len(str) {
			str = ""
		} else {
			str = str[minIndex+1:]
		}
	}

	return res
}

func FindWordAtPosition(template *parser.Template2, position lsp.Position) string {
	pos := LspPosToParserPos(position)
	t := FindTemplateTreeForPosition(template.Content, pos)
	if t == nil {
		return ""
	}
	switch t.Type {
	// At the current moment we will only return a word for the first segment of a GoExp.
	case parser.TT2GoExp:
		words := splitOnSet(t.Text, ".()")
		if len(words) == 0 {
			break
		}
		word := words[0]
		if pos.Line() != t.Line() && pos.Column() > (t.Column()+len(word)) {
			break
		}
		return word
	}
	return ""
}
