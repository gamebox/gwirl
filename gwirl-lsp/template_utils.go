package main

import (
	"log"
	"path"
	"strings"

	"github.com/gamebox/gwirl/internal/parser"
	lsp "go.lsp.dev/protocol"
)

type PlainPosition struct {
	line   int
	column int
}

type templateEntry struct {
    name string
    path string
}

func TemplateNames(fileContents map[string]string) []templateEntry {
    entries := make([]templateEntry, 0, 100)
    for filepath := range fileContents {
        name :=  strings.TrimSuffix(path.Base(filepath), ".gwirl.html")
        name = strings.TrimSuffix(name, ".twirl.html")
        name = strings.Replace(name, string(name[0]), string(name[0] - 32), 1)
        entries = append(entries, templateEntry{
            path: filepath,
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
    log.Printf("Pos is at [ %d, %d ]", pos.Line(), pos.Column())
    for idx := range tts {
        tt := tts[idx]
        log.Printf("Checking %v", tt)
        nodeLineIsBeforePosLine := tt.Line() < pos.Line()
        nodeLineIsAfterPosLine := tt.Line() > pos.Line()
        nodeLineIsEqualToPosLine := tt.Line() == pos.Line()
        nodeColumnIsBeforePosColumn := tt.Column() < pos.Column()
        nodeColumnIsAfterOrEqualToPosColumn := tt.Column() >= pos.Column()
        if nodeLineIsAfterPosLine || (nodeLineIsEqualToPosLine && nodeColumnIsAfterOrEqualToPosColumn ) {
            log.Print("Is after the position, will return %v", previous)
            result = previous
            break
        }
        if nodeLineIsBeforePosLine || (nodeLineIsEqualToPosLine && nodeColumnIsBeforePosColumn ) {
            log.Printf("Is before the position, type is %d", tt.Type)
            switch tt.Type {
            case parser.TT2If, parser.TT2Else, parser.TT2ElseIf, parser.TT2GoExp, parser.TT2For:
                if tt.Children == nil || len(tt.Children) == 0 {
                    log.Print("Has no children")
                } else {
                    log.Print("Will dive into children")
                    for _, childTrees := range tt.Children {
                      res := FindTemplateTreeForPosition(childTrees, pos)
                      if res != nil {
                          return res
                      }
                    }
                }
            default:
            }
            log.Printf("Setting previous to %p", &tt)
            previous = &tt
        }
	}
    if result != nil {
        log.Printf("Returning %v", result)
    } else {
        log.Print("Return NO RESULT")
    }
	return result
}
