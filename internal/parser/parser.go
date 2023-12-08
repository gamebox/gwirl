package parser

import (
	"errors"
	"log"
	"strings"
)

/*
 * The GoTwirl Parser implements this  grammar that removes some back-tracking within the
 * 'mixed' non-terminal. It is defined as follows:
 * {{{
 *   parser : comment? whitespace? ('@' parentheses)? templateContent
 *   templateContent : (importExpression | localDef | template | mixed)*
 *   templateDeclaration : '@' identifier parentheses*
 *   localDef : templateDeclaration (' ' | '\t')* '=' (' ' | '\t') goBlock
 *   template : templateDeclaration (' ' | '\t')* '=' (' ' | '\t') '{' templateContent '}'
 *   mixed : (comment | scalaBlockDisplayed | forExpression | ifExpression | matchExpOrSafeExpOrExpr | caseExpression | plain) | ('{' mixed* '}')
 *   matchExpOrSafeExpOrExpr : (expression | safeExpression) (whitespaceNoBreak 'match' block)?
 *   scalaBlockDisplayed : scalaBlock
 *   scalaBlockChained : scalaBlock
 *   scalaBlock : '@' brackets
 *   importExpression : '@' 'import ' .* '\r'? '\n'
 *   forExpression : '@' "for" parentheses block
 *   simpleExpr : methodCall expressionPart*
 *   complexExpr : parentheses
 *   safeExpression : '@' parentheses
 *   ifExpression : '@' "if" parentheses expressionPart (elseIfCall)* elseCall?
 *   elseCall : whitespaceNoBreak? "else" whitespaceNoBreak? expressionPart
 *   elseIfCall : whitespaceNoBreak? "else if" parentheses whitespaceNoBreak? expressionPart
 *   chainedMethods : ('.' methodCall)+
 *   expressionPart : chainedMethods | block | (whitespaceNoBreak scalaBlockChained) | parentheses
 *   expression : '@' methodCall expressionPart*
 *   methodCall : identifier parentheses?
 *   block : whitespaceNoBreak? '{' mixed* '}'
 *   brackets : '{' (brackets | [^'}'])* '}'
 *   comment : '@*' [^'*@']* '*@'
 *   parentheses : '(' (parentheses | [^')'])* ')'
 *   squareBrackets : '[' (squareBrackets | [^']'])* ']'
 *   plain : ('@@' | '@}' | ([^'@'] [^'{' '}']))+
 *   whitespaceNoBreak : [' ' '\t']+
 *   identifier : javaIdentStart javaIdentPart* // see java docs for what these two rules mean
 * }}}
 */
type Parser struct {
  input input
  errorStack []posString
}

func (p *Parser) accept(str string) {
    length := len(str)
    if !p.input.isPastEOF(length) && p.input.matches(str) {
        p.input.advance(length)
        return
    }
}

func (p *Parser) check(pred stringPredicate) bool {
    if !p.input.isEOF() && pred(p.input.apply(0)) {
        p.input.advance(1)
        return true
    }
    return false
}

func (p *Parser) checkStr(str string) bool {
    length := len(str)
    if !p.input.isPastEOF(length) && p.input.matches(str) {
        p.input.advance(length)
        return true
    }
    return false
}

func (p *Parser) error(message string, offset int) {
    posString := NewPosString(message)
    p.position(&posString, offset)
    p.errorStack = append(p.errorStack, posString) 
}

func (p *Parser) any(length int) string {
    if p.input.isEOF() {
        return ""
    } else {
        s := p.input.apply(length)
        p.input.advance(length)
        return s
    }
}

func (p *Parser) anyUntil(f func(string) bool, inclusive bool) string {
    sb := strings.Builder{}
    for !p.input.isEOF() && !f(p.input.apply(1)) {
        sb.WriteString(p.any(1))
    }
    if inclusive && !p.input.isEOF() {
        sb.WriteString(p.any(1))
    }
    return sb.String()
}

func (p *Parser) anyUntilStr(stop string, inclusive bool) string {
    sb := strings.Builder{}
    stopLength := len(stop)
    for !p.input.isPastEOF(stopLength) && !p.input.matches(stop) {
        sb.WriteString(p.any(1))
    }
    if inclusive && !p.input.isPastEOF(stopLength) {
        sb.WriteString(p.any(stopLength))
    }
    return sb.String()
}

func several[T any](p *Parser, parser func(p *Parser) *T, provided []*T) []*T {
    parsed := parser(p)
    for parsed != nil {
        provided = append(provided, parsed)
        parsed = parser(p)
    }
    return provided
}

func (p *Parser) position(positional Positional, offset int) {
    if positional == nil {
        return
    }
    positional.SetPos(NewOffsetPosition(p.input.source(), offset))
}

func (p *Parser) recursiveTag(prefix string, suffix string, allowStringLiterals bool) *string {
    if p.checkStr(prefix) {
        stack := 1
        sb := strings.Builder{}
        sb.WriteString(prefix)
        for stack > 0 {
            if p.checkStr(prefix) {
                stack = stack + 1
                sb.WriteString(prefix)
            } else if p.checkStr(suffix) {
                stack = stack - 1
                sb.WriteString(suffix)
            } else if p.input.isEOF() {
                stack = 0
            } else if allowStringLiterals {
                s, err := p.stringLiteral("\"", "\\")
                if err != nil {
                    sb.WriteString(p.any(1))
                } else {
                    sb.WriteString(s)
                }
            } else {
                sb.WriteString(p.any(1))
            }
        }
        tag := sb.String()
        return &tag
    }
    return nil 

}

func (p *Parser) stringLiteral(quote string, escape string) (string, error) {
    if p.checkStr(quote) {
        within := true
        sb := strings.Builder{}
        sb.WriteString(quote)
        for within {
            if p.checkStr(quote) {
                sb.WriteString(quote)
                within = false
            } else if p.checkStr(escape) {
                sb.WriteString(escape)
                if p.checkStr(quote) {
                    sb.WriteString(quote)
                } else if p.checkStr(escape) {
                    sb.WriteString(escape)
                }
            } else if p.input.isEOF() {
                within = false
            } else {
                sb.WriteString(p.any(1))
            }
        }
        return sb.String(), nil
    }
    return "", errors.New("Quote not found")
}

func (p *Parser) parentheses() *string {
    return p.recursiveTag("(", ")", true)
}

func (p *Parser) squareBrackets() *string {
    return p.recursiveTag("[", "]", false)
}

func (p *Parser) whitespaceNoBreak() *string {
    result := p.anyUntil(func(c string) bool {
        return c != " " && c != "\t" 
    }, false)
    return &result
}

func (p *Parser) whitespace() (string, error) {
    return p.anyUntil(func(c string) bool {
        return c[0] > 32
    }, false), nil
}

func (p *Parser) identifier() (string, error) {
    if !p.input.isEOF() && isGoIdentifierStart(p.input.apply(1)[0]) {
        log.Printf("getting identifier\n")
        return p.anyUntil(func(c string) bool {
            log.Println("Identifier anyUntil")
            return !isGoIdentifierPart(c[0])
        }, false), nil
    } 
    return "", errors.New("Not an identifier")
}

func (p *Parser) Comment() (*TemplateTree, error) {
    pos := p.input.offset()
    if p.checkStr("@*") {
        text := p.anyUntilStr("*@", false)
        p.accept("*@")
        comment := NewComment(text)
        p.position(&comment, pos)
        return &comment, nil
    }
    return nil, errors.New("Not a comment")
}

func (p *Parser) LastComment() (*TemplateTree, error) {
    var last *TemplateTree
    last = nil
    for true {
        p.whitespace()
        next, _ := p.Comment()
        if next == nil {
            return last, nil
        }
        last = next
    }
    return last, nil
}

func (p *Parser) GoBlock() *GoExpPart {
    if p.checkStr("@{") {
        p.input.regress(1)
        pos := p.input.offset()
        b := p.Brackets()
        if b == "" {
            return nil
        }
        simple := NewSimple(b)
        p.position(&simple, pos)
        return &simple 
    }
    return nil
}

func (p *Parser) Brackets() string {
    result := p.recursiveTag("{", "}", false)
    if result != nil && !strings.HasSuffix(*result, "}") {
        result = nil
    }
    return *result
}

// TODO: Write a parser for go import block statement
func (p *Parser) ImportExpression() (*GoExpPart, error) {
    pos := p.input.offset()
    if p.checkStr("@import ") {
        content:= strings.TrimSpace(p.anyUntilStr("\n", true))
        simple := NewSimple("import " + content)
        p.position(&simple, pos + 1)
        return &simple, nil
    }
    return nil, errors.New("No import expression")
}

func (p *Parser) simpleParens() *GoExpPart {
    pos := p.input.offset()
    parens := p.parentheses()
    if parens != nil {
        s := NewSimple(*parens)
        p.position(&s, pos)
        return &s
    }
    return nil
}

func (p *Parser) goBlockChained() *GoExpPart {
    return nil
}

func (p *Parser) wsThenGoBlockChained() *GoExpPart {
    reset := p.input.offset()
    p.whitespaceNoBreak()
    chained := p.goBlockChained()
    if chained == nil {
        p.input.regressTo(reset)
    }
    return chained
}

func (p *Parser) methodCall() string {
    name, _ := p.identifier()
    log.Printf("methodCall name: \"%s\"", name)
    if name != "" {
        sb := strings.Builder{}
        sb.WriteString(name)
        parens := p.parentheses()
        if parens != nil {
            sb.WriteString(*parens)
        }
        name = sb.String()
        log.Printf("full methodCall name: \"%s\"\n", name)
    }

    return name
}

func (p *Parser) chainedMethods() *GoExpPart {
    var exp *GoExpPart = nil
    pos := p.input.offset()
    if p.checkStr(".") {
        sb := strings.Builder{}
        sb.WriteString(".")
        done := false
        matchMethodCall := true
        for !done {
            if matchMethodCall {
                method := p.methodCall()
                if method != "" {
                    sb.WriteString(method)
                }
            } else {
                if p.checkStr(".") {
                    sb.WriteString(".")
                } else {
                    done = true
                }
            }
            matchMethodCall = !matchMethodCall
        }
        s := NewSimple(sb.String())
        exp = &s
        p.position(exp, pos)
    }
    return exp
}

func (p *Parser) expression() *TemplateTree {
    log.Println("expression")
    var result *TemplateTree = nil
    if !p.checkStr("@") {
        return result
    } 

    pos := p.input.offset()
    call := p.methodCall()
    if call == "" {
        p.input.regressTo(pos - 1)
        return result
    }
    
    code := p.chainedMethods()
    var combinedPart GoExpPart
    if code == nil {
        combinedPart = NewSimple(call)
    } else {
        callPart := NewSimple(call)
        combined := callPart.Join(code)
        combinedPart = *combined
    }


    blk := p.block()
    if blk != nil {
        t := NewGoExpWithTransclusion(combinedPart.Code, NewDisplay([]GoExpPart{*blk}))
        result = &t
    } else {
        t := NewGoExp(combinedPart.Code)
        result = &t
    }

    return result
}

func (p *Parser) block() *GoExpPart {
    var result *GoExpPart = nil
    pos := p.input.offset()
    ws := p.whitespaceNoBreak()
    if p.checkStr("{") {
        flatMixeds := []TemplateTree{}
        mixeds := []*[]TemplateTree{}
        mixeds = several[[]TemplateTree](p, func (p *Parser) *[]TemplateTree {
            res := p.Mixed()
            if len(res) == 0 {
                return nil
            }
            return &res
        }, mixeds) 
        p.accept("}")
        for _, m := range mixeds {
            flatMixeds = append(flatMixeds, *m...)
        }
        blk := NewBlock(*ws, nil, &flatMixeds)
        p.position(&blk, pos)
        result = &blk
    } else {
        p.input.regressTo(pos)
    }
    return result
}

func (p *Parser) expressionPart(blockArgsAllowed bool) *GoExpPart {
    x := p.chainedMethods()
    if x != nil {
        return x
    }
    x = p.block()
    if x != nil {
        return x
    }
    x = p.wsThenGoBlockChained()
    if x != nil {
        return x
    }
    return p.simpleParens()
}

func (p *Parser) ifOrForDeclaration() *GoExpPart {
    result := p.anyUntilStr("{", false)
    s := NewSimple(result)
    return &s
}

func (p *Parser) forExpression() *TemplateTree {
    result := []GoExpPart{}
    pos := p.input.offset()
    var positional *GoExpPart = nil
    if p.checkStr("@for") {
        condition := p.ifOrForDeclaration()
        if (condition != nil && condition.Code != "") {
            s := NewSimple("for" + condition.Code)
            blk := p.expressionPart(true)
            log.Printf("Got blk %v", blk)
            if blk != nil {
                positional = &s
                result = append(result, s)
                result = append(result, *blk)
            }
        }
    }

    if len(result) == 0 {
        p.input.regressTo(pos)
        return nil
    }
    p.position(positional, pos + 1)
    display := NewDisplay(result)
    return &display
}

func (p *Parser) ifExpression() *TemplateTree {
    result := []GoExpPart{}
    // defaultElse := NewSimple(" else {} ")
    pos := p.input.offset()
    var positional *GoExpPart = nil
    if p.checkStr("@if") {
        condition := p.ifOrForDeclaration()
        if (condition != nil && condition.Code != "") {
            s := NewSimple("if" + condition.Code)
            blk := p.expressionPart(true)
            log.Printf("Got blk %v", blk)
            if blk != nil {
                positional = &s
                result = append(result, s)
                result = append(result, *blk)

                // TODO: Get elseIfs

                elseCallPart := p.elseCall()
                if elseCallPart != nil {
                    result = append(result, *elseCallPart...)
                }
            }
        }
    }

    if len(result) == 0 {
        p.input.regressTo(pos)
        return nil
    }
    p.position(positional, pos + 1)
    display := NewDisplay(result)
    return &display
}

func (p *Parser) elseCall() *[]GoExpPart {
    reset := p.input.offset()
    p.whitespaceNoBreak()
    if p.checkStr("else") {
        p.whitespaceNoBreak()
        blk := p.expressionPart(true)
        if blk != nil {
            parts := []GoExpPart{ NewSimple("else"), *blk }
            return &parts
        }
        return nil
    }
    p.input.regressTo(reset)
    return nil
}

func (p *Parser) plainSingle() string {
    if p.checkStr("@@") {
        return "@"
    }
    if p.checkStr("@}") {
        return "}"
    }
    if p.input.isEOF() {
        return ""
    }
    next := p.input.apply(1)
    if  next != "@" && next != "}" {
        return p.any(1)
    }
    return ""
}

func (p *Parser) plain() *TemplateTree {
    pos := p.input.offset()
    var result *TemplateTree = nil
    part := p.plainSingle()
    if part != "" {
        sb := strings.Builder{}
        for part != "" {
            sb.WriteString(part)
            part = p.plainSingle()
        }
        plain := NewPlain(sb.String())
        p.position(&plain, pos)
        result = &plain
    }

    return result
}

func (p *Parser) mixedOpt1() []TemplateTree {
    pos := p.input.offset()
    log.Printf("mixedOpt1: trying comment @ %d", pos)
    comment, _ := p.Comment()
    if comment != nil {
        return []TemplateTree{ *comment }
    }
    log.Printf("mixedOpt1: trying block @ %d", pos)
    block := p.GoBlock()
    if block != nil {
        return []TemplateTree { NewDisplay([]GoExpPart{*block}) }
    }
    log.Printf("mixedOpt1: trying for @ %d", pos)
    forExp := p.forExpression()
    if forExp != nil {
        return []TemplateTree { *forExp }
    }
    log.Printf("mixedOpt1: trying if @ %d", pos)
    ifExp := p.ifExpression()
    if ifExp != nil {
        return []TemplateTree { *ifExp }
    }
    log.Printf("mixedOpt1: trying plain @ %d", pos)
    plain := p.plain()
    if plain != nil {
        log.Printf("mixedOpt1: got plain: %v", plain)
        return []TemplateTree { *plain }
    }
    log.Printf("mixedOpt1: giving up @ %d", pos)
    return nil
}

func (p *Parser) mixedOpt2() []TemplateTree {
    log.Printf("mixedOpt2: trying expression")
    exp := p.expression()
    if exp != nil {
        log.Printf("expression was not null: %v\n", exp)
        return []TemplateTree{ *exp }
    }
    log.Println("mixedOpt2: giving up")
    return nil
}

func (p *Parser) Mixed() []TemplateTree {
    opt1 := p.mixedOpt1()
    if opt1 == nil {
        return p.mixedOpt2()
    }
    return opt1
}

func (p *Parser) TemplateContent() ([]GoExpPart, []Def, []Template, []TemplateTree) {
    imports := make([]GoExpPart, 0, 0)
    localDefs := make([]Def, 0, 0)
    templates := make([]Template, 0, 0)
    mixeds := make([]TemplateTree, 0, 0)

    done := false

    for !done {
        impExp, _ := p.ImportExpression()
        if impExp != nil {
            imports = append(imports, *impExp)
            continue
        }
        // local Def
        // Template
        mix := p.Mixed()
        if mix != nil {
            mixeds = append(mixeds, mix...)
            continue
        }
        pos := p.input.offset()
        if p.checkStr("@") {
            p.error("Invalid '@' symbol", pos)
        } else {
            done = true
        }
    }

    return imports, localDefs, templates, mixeds
}

type ParseResult struct{
    Template Template
    Input input
    Errors []posString
}

func (p *Parser) constructorArgs() *posString {
    if p.checkStr("@(") {
        p.input.regress(1)
        pos := p.input.offset()
        args := p.templateArgs()
        if args != nil {
            ps := NewPosString(*args)
            p.position(&ps, pos)
            return &ps
        }
        return nil
    }
    return nil
}

func (p *Parser) parseConstructorAndArgComment() (*Constructor, *TemplateTree) {
    p.whitespace()
    argsComment, _ := p.LastComment()
    p.whitespace()
    ctr := NewConstructor(nil, NewPosString("()"))
    return &ctr, argsComment
}

func (p *Parser) extraImports() []GoExpPart {
    return []GoExpPart{}
}

func (p *Parser) templateArgs() *string {
    return p.parentheses()
} 

func (p *Parser) maybeTemplateArgs() *posString {
    if p.checkStr("@(") {
        p.input.regress(1)
        pos := p.input.offset()
        args := p.templateArgs()
        log.Printf("Args is %v\n", args)
        if args != nil {
            ps := NewPosString(*args)
            p.position(&ps, pos)
            result := ps
            p.checkStr("\n")
            return &result
        }
        return nil
    }
    return nil
}

func (p *Parser) Parse(source string, name string) ParseResult {
    p.input.reset(source)
    p.errorStack = make([]posString, 0, 0)

    topImports := p.extraImports()
    ctr, comment := p.parseConstructorAndArgComment()
    args := p.maybeTemplateArgs()
    imports, localDefs, templates, mixeds := p.TemplateContent()
    var templateArgs posString
    if args == nil {
        templateArgs = NewPosString("()")
    } else {
        templateArgs = *args
    }

    template := NewTemplate(
        NewPosString(name),
        ctr,
        comment,
        templateArgs,
        topImports,
        imports,
        localDefs,
        templates,
        mixeds,
    )

    if len(p.errorStack) > 0 {
        log.Printf("Errors found while parsing\n")
    }
    for _, e := range p.errorStack {
        log.Printf("Error: %v\n", e)
    }

    return ParseResult{ template, p.input, p.errorStack }
}

func NewParser(source string) Parser {
    in := input{}
    in.reset(source)
    return Parser {
        input: in,
        errorStack: make([]posString, 0),
    }
}

