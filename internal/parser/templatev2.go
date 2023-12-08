package parser

import (
    "fmt"
    "strings"
)

type Template2 struct {
    Name posString
    Comment *TemplateTree2
    Params posString
    TopImports []string
    Content []TemplateTree2
    column int
    line int
}

func NewTemplate2(
    name posString,
    comment *TemplateTree2,
    params posString,
    topImports []string,
    content []TemplateTree2,
) Template2 {
    return Template2{
        Name: name,
        Comment: comment,
        Params: params,
        TopImports: topImports,
        Content: content,
        column: 0,
        line: 0,
    }
}

func (t *Template2) Column() int {
    return t.column
}

func (t *Template2) Line() int {
    return t.line
}

func (t *Template2) SetPos(pos Position) {
    t.column = pos.Column()
    t.line = pos.Line()
}

func (t Template2) String() string {
    sb := strings.Builder{}
    sb.WriteString("Template2 {\n")
    sb.WriteString(fmt.Sprintf("\tName: \"%s\",\n", t.Name))
    sb.WriteString(fmt.Sprintf("\tComment: %v,\n", t.Comment))
    sb.WriteString(fmt.Sprintf("\tParams = %v,\n", t.Params))
    sb.WriteString(fmt.Sprintf("\tTopImports = %v,\n", t.TopImports))
    sb.WriteString(fmt.Sprintf("\tContent = %v\n", t.Content))
    sb.WriteString(fmt.Sprintf("\tPos = (%d, %d)\n", t.line, t.column))
    sb.WriteString("}")
    return sb.String()
}
