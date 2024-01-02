package gwirl

import (
	"fmt"
	html "html/template"
	"strings"
)

type TemplateBuilder struct {
    strings.Builder
}

// Writes the 
func WriteEscapedHTML(builder *TemplateBuilder, value interface{}) {
    builder.WriteString(html.HTMLEscapeString(fmt.Sprintf("%v", value)))
}

func WriteRawHTML(builder *TemplateBuilder, value interface{}) {
    builder.WriteString(fmt.Sprintf("%v", value))
}

