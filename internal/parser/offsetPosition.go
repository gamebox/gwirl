package parser

import (
	"fmt"
	"strings"
)

type OffsetPosition struct {
	column int
	line   int
	source string
}

func (op OffsetPosition) Column() int {
	return op.column
}

func (op OffsetPosition) Line() int {
	return op.line
}

func (op OffsetPosition) String() string {
	return fmt.Sprintf("[%d:%d]", op.line, op.column)
}

func NewOffsetPosition(source string, offset int) OffsetPosition {
	bytes := []byte(source)
	lines := 1
	column := 0
	left := offset
	for _, byte := range bytes {
		if left == 0 {
			break
		}
		if byte == '\n' {
			lines += 1
			column = 0
		} else {
			column += 1
		}
		left -= 1
	}
	sourceLines := strings.Split(source, "\n")
	return OffsetPosition{
		line:   lines,
		column: column,
		source: sourceLines[lines-1],
	}
}
