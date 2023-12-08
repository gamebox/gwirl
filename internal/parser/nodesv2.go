package parser

import (
	"fmt"
	"strings"
)

type TemplateTree2Type = int

const (
    TT2GoBlock TemplateTree2Type = iota
    TT2Plain
    TT2If
    TT2ElseIf
    TT2Else
    TT2For
    TT2GoExp
    TT2BlockComment
    TT2LineComment
)

type TemplateTree2 struct {
    Type TemplateTree2Type
    Text string
    Children [][]TemplateTree2
    line int
    column int
}

func (tt *TemplateTree2) Line() int {
    return tt.line
}

func (tt *TemplateTree2) Column() int {
    return tt.column
}

func (tt *TemplateTree2) SetPos(pos Position) {
    tt.line = pos.Line()
    tt.column = pos.Column()
}

func NewTT2GoBlock(content string) TemplateTree2 {
    return TemplateTree2{
        Type: TT2GoBlock,
        Text: content,
    }
}

func NewTT2Plain(content string) TemplateTree2 {
    return TemplateTree2{
        Type: TT2Plain,
        Text: content,
    }
}

func NewTT2If(condition string, content []TemplateTree2, elseIfTree []TemplateTree2, elseTree *TemplateTree2) TemplateTree2 {
    children := [][]TemplateTree2{ content, elseIfTree }
    if elseTree != nil {
        children = append(children, []TemplateTree2{ *elseTree })
    }
    return TemplateTree2{
        Type: TT2If,
        Text: condition,
        Children: children,
    }
}

func NewTT2Else(content []TemplateTree2) TemplateTree2 {
    return TemplateTree2{
        Type: TT2Else,
        Children: [][]TemplateTree2{ content },
    }
}

func NewTT2For(initialization string, blk []TemplateTree2) TemplateTree2 {
    return TemplateTree2{
        Type: TT2For,
        Text: initialization,
        Children: [][]TemplateTree2{ blk },
    }
}

func NewTT2GoExp(content string, transclusions []TemplateTree2) TemplateTree2 {
    children := [][]TemplateTree2{}
    if len(transclusions) > 0 {
        children = append(children, transclusions)
    }
    return TemplateTree2{
        Type: TT2GoExp,
        Text: content,
        Children: children,
    }
}
func NewTT2BlockComment(content string) TemplateTree2 {
    return TemplateTree2{
        Type: TT2BlockComment,
        Text: content,
    }
}

func NewTT2LineComment(content string) TemplateTree2 {
    return TemplateTree2{
        Type: TT2LineComment,
        Text: content,
    }
}

func (tt TemplateTree2) String() string {
    sb := strings.Builder{}
    switch tt.Type {
    case TT2GoBlock:
        sb.WriteString(fmt.Sprintf("GoBlock(\"%s\")", tt.Text))
    case TT2Plain:
        sb.WriteString(fmt.Sprintf("Plain(\"%s\")", tt.Text))
    case TT2GoExp:
        sb.WriteString(fmt.Sprintf("GoExp(\"%s\", { %v })", tt.Text, tt.Children))
    case TT2If:
        sb.WriteString(fmt.Sprintf("GoIf(\"%s\", %v)", tt.Text, tt.Children))
    case TT2ElseIf:
        sb.WriteString(fmt.Sprintf("GoElseIf(\"%s\", %v)", tt.Text, tt.Children))
    case TT2Else:
        sb.WriteString(fmt.Sprintf("GoElse(%v)", tt.Children))
    case TT2BlockComment:
        sb.WriteString(fmt.Sprintf("GoComment(\"%s\")", tt.Text))
    }
    sb.WriteString(fmt.Sprintf("@[%d,%d]", tt.line, tt.column))
    return sb.String()
}
