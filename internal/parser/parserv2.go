package parser

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

/*
 * The Gwirl Parser2 implements this  grammar that removes some back-tracking within the
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
type Parser2 struct {
	input      input
	errorStack []PosString
	logger     io.Writer
}

func (p *Parser2) SetLogger(logger io.Writer) {
	p.logger = logger
}

func (p *Parser2) log(message string) {
	if p.logger != nil {
		p.logger.Write([]byte("[PARSER]: "))
		p.logger.Write([]byte(message))
		p.logger.Write([]byte("\n"))
	}
}

func (p *Parser2) logf(format string, other ...any) {
	p.log(fmt.Sprintf(format, other...))
}

func (p *Parser2) accept(str string) bool {
	length := len(str)
	if !p.input.isPastEOF(length) && p.input.matches(str) {
		p.input.advance(length)
	} else if p.input.isPastEOF(length) {
		return false
	}
	return true
}

func (p *Parser2) check(pred stringPredicate) bool {
	if !p.input.isEOF() && pred(p.input.apply(0)) {
		p.input.advance(1)
		return true
	}
	return false
}

func (p *Parser2) checkStr(str string) bool {
	length := len(str)
	if !p.input.isPastEOF(length) && p.input.matches(str) {
		p.input.advance(length)
		return true
	}
	return false
}

func (p *Parser2) error(message string, offset int) {
	posString := NewPosString(message)
	p.logf("Error offset is %d", offset)
	p.position(&posString, offset)
	p.errorStack = append(p.errorStack, posString)
}

func (p *Parser2) any(length int) string {
	if p.input.isEOF() {
		return ""
	} else {
		s := p.input.apply(length)
		p.input.advance(length)
		return s
	}
}

func (p *Parser2) anyUntil(f func(string) bool, inclusive bool) string {
	sb := strings.Builder{}
	for !p.input.isEOF() && !f(p.input.apply(1)) {
		sb.WriteString(p.any(1))
	}
	if inclusive && !p.input.isEOF() {
		sb.WriteString(p.any(1))
	}
	return sb.String()
}

func (p *Parser2) anyUntilStr(stop string, inclusive bool) string {
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

func (p *Parser2) position(positional Positional, offset int) {
	if positional == nil {
		return
	}
	positional.SetPos(NewOffsetPosition(p.input.source(), offset))
}

func (p *Parser2) recursiveTag(prefix string, suffix string, allowStringLiterals bool) *string {
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
				p.error(fmt.Sprintf("Expected '%s', got end of file", suffix), p.input.offset())
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

func (p *Parser2) stringLiteral(quote string, escape string) (string, error) {
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

func several2[T any](p *Parser2, parser func(p *Parser2) *T, provided []*T) []*T {
	parsed := parser(p)
	for parsed != nil {
		provided = append(provided, parsed)
		parsed = parser(p)
	}
	return provided
}

func (p *Parser2) parentheses() *string {
	return p.recursiveTag("(", ")", true)
}

func (p *Parser2) squareBrackets() *string {
	return p.recursiveTag("[", "]", false)
}

func (p *Parser2) whitespaceNoBreak() *string {
	result := p.anyUntil(func(c string) bool {
		return c != " " && c != "\t"
	}, false)
	return &result
}

func (p *Parser2) whitespace() (string, error) {
	return p.anyUntil(func(c string) bool {
		return c[0] > 32
	}, false), nil
}

func (p *Parser2) identifier() (string, error) {
	if !p.input.isEOF() && isGoIdentifierStart(p.input.apply(1)[0]) {
		p.log("getting identifier\n")
		return p.anyUntil(func(c string) bool {
			p.log("Identifier anyUntil")
			return !isGoIdentifierPart(c[0])
		}, false), nil
	}
	return "", errors.New("Not an identifier")
}

func (p *Parser2) Comment() *TemplateTree2 {
	pos := p.input.offset()
	if p.checkStr("@*") {
		text := p.anyUntilStr("*@", false)
		p.accept("*@")
		comment := NewTT2BlockComment(text)
		p.position(&comment, pos)
		return &comment
	}
	return nil
}

func (p *Parser2) LastComment() *TemplateTree2 {
	var last *TemplateTree2
	last = nil
	for true {
		p.whitespace()
		next := p.Comment()
		if next == nil {
			return last
		}
		last = next
	}
	return last
}

func (p *Parser2) GoBlock() *TemplateTree2 {
	if p.checkStr("@{") {
		p.input.regress(1)
		pos := p.input.offset()
		b := p.Brackets()
		if b == "" {
			return nil
		}
		blk := NewTT2GoBlock(b)
		p.position(&blk, pos)
		return &blk
	}
	return nil
}

func (p *Parser2) Brackets() string {
	result := p.recursiveTag("{", "}", false)
	if result != nil && !strings.HasSuffix(*result, "}") {
		result = nil
	}
	if result == nil {
		return ""
	}
	return *result
}

// TODO: Write a parser for go import block statement
func (p *Parser2) ImportExpression() *PosString {
	start := p.input.offset()
	if p.checkStr("@import ") {
		content := strings.TrimSpace(p.anyUntilStr("\n", true))
		ps := NewPosString("import " + content)
		p.position(&ps, start+1)
		return &ps
	}
	return nil
}

func (p *Parser2) methodCall() string {
	name, _ := p.identifier()
	p.logf("methodCall name: \"%s\"", name)
	if name != "" {
		sb := strings.Builder{}
		sb.WriteString(name)
		parens := p.parentheses()
		if parens != nil {
			sb.WriteString(*parens)
		}
		name = sb.String()
		p.logf("full methodCall name: \"%s\"\n", name)
	}

	return name
}

func (p *Parser2) chainedMethods() string {
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
		return sb.String()
	}
	return ""
}

func (p *Parser2) expression() *TemplateTree2 {
	p.log("expression")
	if !p.checkStr("@") {
		return nil
	}

	escape := p.checkStr("!")

	pos := p.input.offset()
	call := p.methodCall()
	if call == "" {
		p.input.regressTo(pos - 1)
		return nil
	}

	code := p.chainedMethods()
	combinedExpression := call
	if code != "" {
		combinedExpression += code
	}

	if !strings.HasSuffix(combinedExpression, ")") {
		t := NewTT2GoExp(combinedExpression, escape, []TemplateTree2{})
		p.position(&t, pos)
		return &t
	}

	blk := p.block()
	if blk != nil {
		t := NewTT2GoExp(combinedExpression, escape, *blk)
		p.position(&t, pos)
		return &t
	} else {
		t := NewTT2GoExp(combinedExpression, escape, []TemplateTree2{})
		p.position(&t, pos)
		return &t
	}
}

func (p *Parser2) block() *[]TemplateTree2 {
	var result *[]TemplateTree2 = nil
	pos := p.input.offset()
	p.whitespaceNoBreak()
	if p.checkStr("{") {
		mixeds := []*TemplateTree2{}
		mixeds = several2[TemplateTree2](p, func(p *Parser2) *TemplateTree2 {
			res := p.Mixed()
			if res == nil {
				return nil
			}
			return res
		}, mixeds)

		accepted := p.accept("}")
		if !accepted {
			p.error(fmt.Sprintf("Expected '}', found end of file"), p.input.offset())
		}
		flatMixed := []TemplateTree2{}
		for _, m := range mixeds {
			if m != nil {
				flatMixed = append(flatMixed, *m)
			}
		}
		result = &flatMixed
	} else {
		p.input.regressTo(pos)
	}
	return result
}

func (p *Parser2) expressionPart(blockArgsAllowed bool) *[]TemplateTree2 {
	return p.block()
}

func (p *Parser2) ifOrForDeclaration() string {
	return p.anyUntilStr("{", false)
}

func (p *Parser2) forExpression() *TemplateTree2 {
	var result *TemplateTree2 = nil
	pos := p.input.offset()
	if p.checkStr("@for") {
		condition := p.ifOrForDeclaration()
		if condition != "" {
			blk := p.expressionPart(true)
			if blk != nil {
				s := NewTT2For(condition, *blk)
				result = &s
			}
		}
	}

	if result == nil {
		p.input.regressTo(pos)
		return nil
	}
	p.position(result, pos+1)
	return result
}

func (p *Parser2) elseIfs() []TemplateTree2 {
    trees := []TemplateTree2{}
    for {
        pos := p.input.offset()
        p.whitespaceNoBreak()
        if p.checkStr("@else if") {
            condition := p.ifOrForDeclaration()
            if condition == "" {
                p.error("No condition found for else if", p.input.offset())
                break
            }
            blk := p.expressionPart(true)
            if blk == nil {
                p.error("Empty block for else if", p.input.offset())
                break
            }
            tree := NewTT2ElseIf(condition, *blk)
            trees = append(trees, tree)
        } else {
            p.input.regressTo(pos)
            break
        }
    }
    return trees
}

func (p *Parser2) ifExpression() *TemplateTree2 {
	var result *TemplateTree2 = nil
	pos := p.input.offset()
	if p.checkStr("@if") {
		condition := p.ifOrForDeclaration()
		if condition != "" {
			var elseIfTrees []TemplateTree2
			var elseTree *TemplateTree2
			blk := p.expressionPart(true)
			p.logf("Got blk %v", blk)
			if blk != nil {
				// TODO: Get elseIfs
                elseIfTrees = p.elseIfs()
				elseTree = p.elseCall()

				ifTree := NewTT2If(condition, *blk, elseIfTrees, elseTree)
				result = &ifTree
			}
		}
	}

	if result == nil {
		p.input.regressTo(pos)
		return nil
	}
	p.position(result, pos+1)
	return result
}

func (p *Parser2) elseCall() *TemplateTree2 {
	reset := p.input.offset()
	p.whitespaceNoBreak()
	if p.checkStr("@else") {
		p.whitespaceNoBreak()
		blk := p.expressionPart(true)
		if blk != nil {
			t := NewTT2Else(*blk)
			return &t
		}
		return nil
	}
	p.input.regressTo(reset)
	return nil
}

func (p *Parser2) plainSingle() string {
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
	if next != "@" && next != "}" {
		return p.any(1)
	}
	return ""
}

func (p *Parser2) plain() *TemplateTree2 {
	pos := p.input.offset()
	var result *TemplateTree2 = nil
	part := p.plainSingle()
	if part != "" {
		sb := strings.Builder{}
		for part != "" {
			sb.WriteString(part)
			part = p.plainSingle()
		}
		plain := NewTT2Plain(sb.String())
		p.position(&plain, pos)
		result = &plain
	}

	return result
}

func (p *Parser2) Mixed() *TemplateTree2 {
	pos := p.input.offset()
	p.logf("mixedOpt1: trying comment @ %d", pos)
	comment := p.Comment()
	if comment != nil {
		return comment
	}
	p.logf("mixedOpt1: trying block @ %d", pos)
	block := p.GoBlock()
	if block != nil {
		return block
	}
	p.logf("mixedOpt1: trying for @ %d", pos)
	forExp := p.forExpression()
	if forExp != nil {
		return forExp
	}
	p.logf("mixedOpt1: trying if @ %d", pos)
	ifExp := p.ifExpression()
	if ifExp != nil {
		return ifExp
	}
	p.logf("mixedOpt1: trying plain @ %d", pos)
	plain := p.plain()
	if plain != nil {
		p.logf("mixedOpt1: got plain: %v", plain)
		return plain
	}
	p.logf("mixedOpt1: trying expression")
	exp := p.expression()
	if exp != nil {
		p.logf("expression was not null: %v\n", exp)
		return exp
	}
	p.log("mixedOpt1: giving up")
	return nil
}

func (p *Parser2) TopImports() []PosString {
	imports := make([]PosString, 0, 0)
	done := false
	p.whitespace()
	for !done {
		impExp := p.ImportExpression()
		if impExp != nil {
			imports = append(imports, *impExp)
			continue
		}
		done = true
	}
	return imports
}

func (p *Parser2) TemplateContent() []TemplateTree2 {
	mixeds := []TemplateTree2{}

	done := false

	for !done {
		mix := p.Mixed()
		if mix != nil {
			mixeds = append(mixeds, *mix)
			continue
		}
		pos := p.input.offset()
		if p.checkStr("@") {
			p.error("Invalid '@' symbol", pos)
		} else {
			done = true
		}
	}

	return mixeds
}

type ParseResult2 struct {
	Template Template2
	Input    input
	Errors   []PosString
}

func (p *Parser2) constructorArgs() *PosString {
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

func (p *Parser2) parseConstructorAndArgComment() (*Constructor, *TemplateTree2) {
	p.whitespace()
	argsComment := p.LastComment()
	p.whitespace()
	ctr := NewConstructor(nil, NewPosString("()"))
	return &ctr, argsComment
}

func (p *Parser2) templateArgs() *string {
	return p.parentheses()
}

func (p *Parser2) maybeTemplateArgs() *PosString {
	if p.checkStr("@(") {
		p.input.regress(1)
		pos := p.input.offset()
		args := p.templateArgs()
		p.logf("Args is %v\n", args)
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

func (p *Parser2) Parse(source string, name string) ParseResult2 {
	p.input.reset(source)
	p.errorStack = make([]PosString, 0, 0)

	_, comment := p.parseConstructorAndArgComment()
	args := p.maybeTemplateArgs()
	p.log("Looking for top imports")
	topImports := p.TopImports()
	p.logf("TopImports, %v", topImports)
	mixeds := p.TemplateContent()
	var templateArgs PosString
	if args == nil {
		templateArgs = NewPosString("()")
	} else {
		templateArgs = *args
	}

	template := NewTemplate2(
		NewPosString(name),
		comment,
		templateArgs,
		topImports,
		mixeds,
	)

	if len(p.errorStack) > 0 {
		p.logf("Errors found while parsing\n")
	}
	for _, e := range p.errorStack {
		p.logf("Error: %v\n", e)
	}

	return ParseResult2{template, p.input, p.errorStack}
}

func NewParser2(source string) Parser2 {
	in := input{}
	in.reset(source)
	return Parser2{
		input:      in,
		errorStack: make([]PosString, 0),
	}
}
