package views

import (
    "github.com/gamebox/gwirl"
)


func Testing(name string, index int) string {
    sb_ := gwirl.TemplateBuilder{}

    sb_.WriteString(`<div>
	`)

    if if index > 0 {
        sb_.WriteString(`
		<hr />
	`)

    }

    sb_.WriteString(`
	<h2>`)

    gwirl.WriteEscapedHTML(&sb_, name)

    sb_.WriteString(`</h2>
`)

    return sb_.String()
}
