
package gwirl

import (
	"fmt"
	html "html/template"
	"strings"
)

// Writes the 
func WriteEscapedHTML(builder *strings.Builder, value interface{}) {
    builder.WriteString(html.HTMLEscapeString(fmt.Sprintf("%v", value)))
}

func WriteRawHTML(builder *strings.Builder, value interface{}) {
    builder.WriteString(fmt.Sprintf("%v", value))
}

