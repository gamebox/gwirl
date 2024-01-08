package gen

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/gamebox/gwirl/internal/parser"
)

type IndentStyle = int

const (
	ISSpaces = iota
	ISTabs
)

var (
	tabIndent   []byte = []byte("\t")
	spaceIndent []byte = []byte("    ")
	newline     []byte = []byte("\n")
)

type Generator struct {
	indentLevel int
	indentStyle IndentStyle
	writer      io.Writer
}

func NewGenerator(useTabs bool) Generator {
	g := Generator{}
	if useTabs {
		g.indentStyle = ISTabs
	} else {
		g.indentStyle = ISSpaces
	}
	return g
}

func (G *Generator) GenTemplateTree(tree parser.TemplateTree2) error {
	switch tree.Type {
	case parser.TT2Plain:
		G.write("sb_.WriteString(`")
		G.writeNoIndent(tree.Text)
		G.writeNoIndent("`)")
		G.newlines()
	case parser.TT2BlockComment:
		G.write("/")
		for i, line := range strings.Split(tree.Text, "\n") {
			if i == 0 {
				G.writeNoIndent(line)
			}
			G.write(line)
		}
		G.writeNoIndent("/")
		G.newlines()
	case parser.TT2GoBlock:
		cleanedText := strings.TrimLeft(tree.Text, "{")
		cleanedText = strings.TrimLeft(cleanedText, "\n")
		cleanedText = strings.TrimRight(cleanedText, "}")
		cleanedText = strings.TrimLeft(cleanedText, "\n")
		for _, line := range strings.Split(cleanedText, "\n") {
			G.write(strings.TrimLeft(line, " \t"))
			G.write("\n")
		}
		G.newlines()
	case parser.TT2If:
		G.write("if ")
		G.writeNoIndent(tree.Text)
		G.writeNoIndent(" {\n")
		G.indent()
		// Content of main block in tree.Children[0]
		if len(tree.Children) > 0 {
			for _, child := range tree.Children[0] {
				G.GenTemplateTree(child)
			}
		}
		G.dedent()
		G.write("}")
		// Else ifs in tree.Children[1]
		if len(tree.Children) > 1 {
			for _, elseIf := range tree.Children[1] {
				G.GenTemplateTree(elseIf)
			}
		}
		// Else is tree.Children[2][0]
		if len(tree.Children) > 2 && len(tree.Children[2]) > 0 {
			G.GenTemplateTree(tree.Children[2][0])
		}
		G.newlines()
	case parser.TT2ElseIf:
		G.writeNoIndent(" else if ")
		G.writeNoIndent(tree.Text)
		G.writeNoIndent(" {\n")
		G.indent()
		if len(tree.Children) > 0 {
			for _, child := range tree.Children[0] {
				G.GenTemplateTree(child)
			}
		}
		G.dedent()
		G.write("}")
	case parser.TT2Else:
		G.writeNoIndent(" else {\n")
		G.indent()
		if len(tree.Children) > 0 {
			for _, child := range tree.Children[0] {
				G.GenTemplateTree(child)
			}
		}
		G.dedent()
		G.write("}")
	case parser.TT2For:
		G.write("for ")
		G.writeNoIndent(tree.Text)
		G.writeNoIndent(" {\n")
		G.indent()
		// Content of main block in tree.Children[0]
		if len(tree.Children) > 0 {
			for _, child := range tree.Children[0] {
				G.GenTemplateTree(child)
			}
		}
		G.dedent()
		G.write("}")
		G.newlines()
	case parser.TT2GoExp:
		if len(tree.Children) > 0 {
            transclusionParams := strings.Builder{}
            for i, transclusion := range tree.Children {
                varName := fmt.Sprintf("transclusion__%d__%d__%d", tree.Line(), tree.Column(), i)
                if i > 0 {
                    transclusionParams.WriteString(", ")
                }
                transclusionParams.WriteString(varName)
                G.write("var ")
                G.writeNoIndent(varName)
                G.writeNoIndent(" string\n")
                G.write("{\n")
                G.indent()
                G.write("sb_ := gwirl.TemplateBuilder{}\n")
                for _, child := range transclusion {
                    G.GenTemplateTree(child)
                }
                G.write(varName)
                G.writeNoIndent(" = sb_.String()\n")
                G.dedent()
                G.write("}\n")
            }
			if tree.Metadata.Has(parser.TTMDEscape) {
				fmt.Printf("GoExp %v\n", tree)
				G.write("gwirl.WriteEscapedHTML(&sb_, ")
			} else {
				G.write("gwirl.WriteRawHTML(&sb_, ")
			}
            transclusionParamsStr := transclusionParams.String()
			if strings.HasSuffix(tree.Text, "()") {
				text, _ := strings.CutSuffix(tree.Text, ")")
				text = text + transclusionParamsStr + ")"
				G.writeNoIndent(text)
			} else if strings.HasSuffix(tree.Text, ")") {
				text, _ := strings.CutSuffix(tree.Text, ")")
				text = text + ", " + transclusionParamsStr + ")"
				G.writeNoIndent(text)
			} else {
				return errors.New("Transclusion can only occur with a method call")
			}
			G.writeNoIndent(")\n")
		} else {
			if tree.Metadata.Has(parser.TTMDEscape) {
				G.write("gwirl.WriteEscapedHTML(&sb_, ")
			} else {
				G.write("gwirl.WriteRawHTML(&sb_, ")
			}
			G.writeNoIndent(tree.Text)
			G.writeNoIndent(")")
		}
		G.newlines()
	}
	return nil
}

func (G *Generator) write(str string) {
	indent := 0
	for indent < G.indentLevel {
		if G.indentStyle == ISTabs {
			G.writer.Write(tabIndent)
		} else {
			G.writer.Write(spaceIndent)
		}
		indent += 1
	}
	G.writer.Write([]byte(str))
}

func (G *Generator) writeln(str string) {
	G.write(str)
	G.writer.Write(newline)
}

func (G *Generator) writeNoIndent(str string) {
	G.writer.Write([]byte(str))
}

func (G *Generator) newlines() {
	G.writer.Write([]byte("\n\n"))
}

func (G *Generator) indent() {
	G.indentLevel += 1
}

func (G *Generator) dedent() {
	G.indentLevel -= 1
}

func (G *Generator) Generate(template parser.Template2, pkg string, writer io.Writer) error {
	G.writer = writer

	pkgLine := fmt.Sprintf("package %s\n\n", pkg)
	G.write(pkgLine)

	// Write imports
	for _, i := range template.TopImports {
		G.write(i.Str)
		G.write("\n")
	}
	G.writeln("import (")
	G.indent()
	G.writeln("\"github.com/gamebox/gwirl\"")
	G.dedent()
	G.writeln(")")

	G.newlines()

	// Write comment as doc comment

	// Write Template boilerplate start
	funcStart := fmt.Sprintf("func %s%s string {\n", template.Name.Str, template.Params.Str)
	G.write(funcStart)

	G.indent()
	G.write("sb_ := gwirl.TemplateBuilder{}")
	G.newlines()

	// Write content
	for _, tree := range template.Content {
		G.GenTemplateTree(tree)
	}

	G.write("return sb_.String()\n")
	G.dedent()

	// Write Template boilerplate end
	G.writeln("}")

	return nil
}
