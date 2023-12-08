package parser_test

import (
	"strings"
	"testing"

	"github.com/gamebox/gwirl/internal/parser"
)

var source = `Y#####

######X#######

###
###
######Z
`

func TestOffsetPositionBeginningOfFile(t *testing.T) {
    i := strings.IndexByte(source, 'Y')
    pos := parser.NewOffsetPosition(source, i)

    if pos.Line() != 1 {
        t.Fatalf("Expected line to be 1, got %v", pos.Line())
    }
    if pos.Column() != 0 {
        t.Fatalf("Expected column to be 0, got %v", pos.Column())
    }
}

func TestOffsetPositionMiddleOfFile(t *testing.T) {
    i := strings.IndexByte(source, 'X')
    pos := parser.NewOffsetPosition(source, i)

    if pos.Line() != 3 {
        t.Fatalf("Expected line to be 3, got %v", pos.Line())
    }
    if pos.Column() != 6 {
        t.Fatalf("Expected column to be 6, got %v", pos.Column())
    }
}

func TestOffsetPositionAlmostEndOfFile(t *testing.T) {
    i := strings.IndexByte(source, 'Z')
    pos := parser.NewOffsetPosition(source, i)

    if pos.Line() != 7 {
        t.Fatalf("Expected line to be 7, got %v", pos.Line())
    }
    if pos.Column() != 6 {
        t.Fatalf("Expected column to be 6, got %v", pos.Column())
    }
}

func TestOffsetPositionEndOfFile(t *testing.T) {
    pos := parser.NewOffsetPosition(source, len(source))

    if pos.Line() != 8 {
        t.Fatalf("Expected line to be 8, got %v", pos.Line())
    }
    if pos.Column() != 0 {
        t.Fatalf("Expected column to be 0, got %v", pos.Column())
    }
}
