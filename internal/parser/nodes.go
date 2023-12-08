package parser

import (
	"fmt"
	"strings"
)

type Position interface {
    Column() int
    Line() int
}

type Positional interface {
    SetPos(pos Position)
}

// posString -----------------------------------------------------------------

type posString struct {
    Str string
    pos Position
}

func NewPosString(str string) posString {
    return posString{
        Str: str,
        pos: nil,
    }
}

func (ps *posString) SetPos(pos Position) {
    ps.pos = pos
}

func (ps posString) String() string {
    if ps.pos != nil {
        return fmt.Sprintf("posString{ str: \"%s\", pos: [%d,%d] }", ps.Str, ps.pos.Line(), ps.pos.Column())
    }
    return fmt.Sprintf("posString{ str: \"%s\" }", ps.Str)
}

func (ps *posString) Line() int {
    return ps.pos.Line()
}

func (ps *posString) Column() int {
    return ps.pos.Column()
}

// Value ---------------------------------------------------------------------

type Value struct {
    Ident posString
    Block GoExpPart
    column int
    line int
}

func (v *Value) Column() int {
    return v.column
}

func (v *Value) Line() int {
    return v.line
}

func (v *Value) SetPos(pos Position) {
    v.column = pos.Column()
    v.line = pos.Line()
}

func NewValue(ident posString, block GoExpPart) Value {
    return Value{
        Ident: ident,
        Block: block,
    }
}


// Def -----------------------------------------------------------------------

type Def struct {
    Name string
    Params posString
    ResultType *posString
    Code GoExpPart
    line int
    column int
}

func NewDef(name string, params posString, resultType *posString, code string) Def {
    return Def{
        Name: name,
        Params: params,
        ResultType: resultType,
        Code: NewSimple(code),
    }
}

func (d *Def) Column() int {
    return d.column
}

func (d *Def) Line() int {
    return d.line
}

// TemplateTree  -------------------------------------------------------------

type TemplateTreeType int

const (
    TTPlain TemplateTreeType = iota
    TTDisplay
    TTComment
    TTGoExp
)

type TemplateTree struct {
    Type TemplateTreeType
    Text string
    Msg string
    Parts []GoExpPart
    Transclusion *TemplateTree
    column int
    line int
}


func (tt *TemplateTree) Column() int {
    return tt.column
}

func (tt *TemplateTree) Line() int {
    return tt.line
}

func (tt *TemplateTree) SetPos(pos Position) {
    tt.column = pos.Column()
    tt.line = pos.Line()
}

func (tt TemplateTree) String() string {
    sb := strings.Builder{}
    sb.WriteString("TemplateTree{\n")
    sb.WriteString(fmt.Sprintf("\tType: %v,\n", tt.Type))
    if tt.Type == TTPlain {
        sb.WriteString(fmt.Sprintf("\tText: \"%s\"\n", tt.Text))
    }
    if tt.Type == TTDisplay {
        sb.WriteString(fmt.Sprintf("\tParts: %v\n", tt.Parts))
    }
    if tt.Type == TTComment {
        sb.WriteString(fmt.Sprintf("\tText: \"%s\"\n", tt.Text))
    }
    if tt.Type == TTGoExp {
        sb.WriteString(fmt.Sprintf("\tText: \"%s\"\n", tt.Text))
    }
    if tt.Type == TTGoExp {}
    sb.WriteString("}")
    return sb.String()
}

func NewPlain(text string) TemplateTree {
    return TemplateTree{
        Type: TTPlain,
        Text: text,
    }
}

func NewDisplay(parts []GoExpPart) TemplateTree {
    return TemplateTree{
        Type: TTDisplay,
        Parts: parts,
    }
}

func NewComment(msg string) TemplateTree {
    return TemplateTree{
        Type: TTComment,
        Text: msg,
    }
}

func NewGoExp(code string) TemplateTree {
    return TemplateTree{
        Type: TTGoExp,
        Text: code,
    }
}

func NewGoExpWithTransclusion(code string, transclusion TemplateTree) TemplateTree {
    return TemplateTree{
        Type: TTGoExp,
        Text: code,
        Transclusion: &transclusion,
    }
}

// GoExpPart --------------------------------------------------------------

type GoExpPartType int

const (
    // Should be rendered as a single line of code
    PartSimple GoExpPartType = iota
    // Content should be rendered in a literal block of code
    PartBlock
    PartRender
    PartMutiline
)

type GoExpPart struct {
    Type GoExpPartType
    Code string
    Whitespace string
    Args *posString
    Content *[]TemplateTree
    column int
    line int
}

func (part *GoExpPart) Column() int {
    return part.column
}

func (part *GoExpPart) Line() int {
    return part.line
}

func (part *GoExpPart) SetPos(pos Position) {
    part.column = pos.Column()
    part.line = pos.Line()
}

func (part GoExpPart) String() string {
    sb := strings.Builder{}
    sb.WriteString("GoExpPart{ ")
    if part.Type == PartSimple {
        sb.WriteString("Type: Simple, ")
        sb.WriteString(fmt.Sprintf("Code: \"%s\"", part.Code))
    }
    if part.Type == PartRender {
        sb.WriteString("Type: Render, ")
        sb.WriteString(fmt.Sprintf("Code: \"%s\"", part.Code))
    }
    if part.Type == PartBlock {
        sb.WriteString("Type: Block, ")
        sb.WriteString(fmt.Sprintf("Whitespace: \"%s\", ", part.Whitespace))
        sb.WriteString(fmt.Sprintf("Content: %s, ", part.Content))
        
    }
    if part.Type == PartMutiline {
        sb.WriteString("Type: Multiline, ")
        sb.WriteString(fmt.Sprintf("Code: %s, ", part.Code))
    }
    sb.WriteString(" }")
    return sb.String()
}

func NewSimple(code string) GoExpPart {
    return GoExpPart{
        Type: PartSimple,
        Code: code,
    }
}

func NewBlock(whitespace string, args *posString, content *[]TemplateTree) GoExpPart {
    return GoExpPart{
        Type: PartBlock,
        Whitespace: whitespace,
        Args: args,
        Content: content,
    }
}

func (part *GoExpPart) Join(other *GoExpPart) *GoExpPart {
    var result *GoExpPart = nil
    if part.Type == PartSimple && other.Type == PartSimple {
        p := GoExpPart{
            Type: PartSimple,
            Code: part.Code + other.Code,
            column: part.column,
            line: part.line,
        }

        result = &p
    }
    return result
}

// Constructor ---------------------------------------------------------------

type Constructor struct {
    Comment *TemplateTree
    Params posString
}

func NewConstructor(comment *TemplateTree, params posString) Constructor {
    ctr :=  Constructor{
        Comment: comment,
        Params: params,
    }

    return ctr
}

func (c Constructor) String() string {
    sb := strings.Builder{}
    sb.WriteString("Constructor {\n")
    if c.Comment != nil {
        sb.WriteString(fmt.Sprintf("\tComment = %v\n", c.Comment))
    }
    sb.WriteString(fmt.Sprintf("\tParams = %v\n", c.Params))
    sb.WriteString("}\n")
    return sb.String()
}

// Template ------------------------------------------------------------------

type Template struct {
    Name posString
    Constructor *Constructor
    Comment *TemplateTree
    Params posString
    TopImports []GoExpPart
    Imports []GoExpPart
    Defs []Def
    Sub []Template
    Content []TemplateTree
    column int
    line int
}

func NewTemplate(
    name posString,
    constructor *Constructor,
    comment *TemplateTree,
    params posString,
    topImports []GoExpPart,
    imports []GoExpPart,
    defs []Def,
    sub []Template,
    content []TemplateTree,
) Template {
    return Template{
        Name: name,
        Constructor: constructor,
        Comment: comment,
        Params: params,
        TopImports: topImports,
        Imports: imports,
        Defs: defs,
        Sub: sub,
        Content: content,
        column: 0,
        line: 0,
    }
}

func (t *Template) Column() int {
    return t.column
}

func (t *Template) Line() int {
    return t.line
}

func (t *Template) SetPos(pos Position) {
    t.column = pos.Column()
    t.line = pos.Line()
}

func (t Template) String() string {
    sb := strings.Builder{}
    sb.WriteString("Template {\n")
    sb.WriteString(fmt.Sprintf("\tName: \"%s\",\n", t.Name))
    sb.WriteString(fmt.Sprintf("\tComment: %v,\n", t.Comment))
    sb.WriteString(fmt.Sprintf("\tParams = %v,\n", t.Params))
    sb.WriteString(fmt.Sprintf("\tContent = %v\n", t.Content))
    sb.WriteString("}")
    return sb.String()
}
