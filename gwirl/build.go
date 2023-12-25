package main

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
    "unicode"

	"github.com/gamebox/gwirl/internal/gen"
	"github.com/gamebox/gwirl/internal/parser"
)

type Builder struct {
    flags *Flags
    accessor FSAccessor
    logger io.Writer
    parser *parser.Parser2
    generator *gen.Generator
}

func NewBuilder(flags *Flags, accessor FSAccessor, logger io.Writer) *Builder {
    b := Builder{
        flags: flags,
        accessor: accessor,
        logger: logger,
    }
    p := parser.NewParser2("")
	if logger != nil {
		p.SetLogger(logger)
	}
    g := gen.NewGenerator(false)
    b.parser = &p
    b.generator = &g

    return &b
}

func capitalize(str string) string {
    runes := []rune(str)
    if len(runes) == 0 {
        return ""
    }
    runes[0] = unicode.ToUpper(runes[0])
    return string(runes)
}

func (b *Builder) Printf(format string, vals ...any) {
    if b.logger != nil {
        b.logger.Write([]byte(fmt.Sprintf(format, vals...)))
    }
}

func (b *Builder) parse(f *File) (*parser.ParseResult2, error) {
	b.Printf("Parsing %s\n", f.name)

	result := b.parser.Parse(f.content, capitalize(f.name))
	if len(result.Errors) > 0 {
		err := strings.Builder{}
		err.WriteString(fmt.Sprintf("Could not parse file %s:\n", f.name+f.filetype+".gwirl"))
		for _, e := range result.Errors {
			err.WriteString(fmt.Sprintf("%v\n", e))
		}
		return nil, errors.New(err.String())
	}
	return &result, nil
}

func (b *Builder) generate(result *parser.ParseResult2, f *File) error {
	b.Printf("Generating %s\n", f.name)

	fileWriter, fileName, err := b.accessor.CreateGwirlFile(f)
	if err != nil {
		e := errors.New(fmt.Sprintf("Failed to open go file for template: %s\nERROR: %v", f.name, err))
		return errors.Join(e, err)
	}

	err = b.generator.Generate(result.Template, f.filetype, fileWriter)
	if err != nil {
		b.accessor.Remove(fileName)
		e := errors.New(fmt.Sprintf("Could not generate a file for template: %s", f.name))
		return errors.Join(e, err)
	}

	return nil
}

func (b *Builder) build() error {
	if b.flags.clean {
		clean()
	}
	fs := b.accessor.TemplateFiles("templates", b.flags.filter.filters)
	for _, f := range fs {
		b.accessor.EnsureDirectoryExists(filepath.Join("views", f.filetype))
		result, err := b.parse(&f)
		if err != nil {
			return err
		}
		err = b.generate(result, &f)
		if err != nil {
			return err
		}
	}

    b.Printf("Completed generating %d templates", len(fs))
	return nil
}
