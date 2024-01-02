package html

import (
    "github.com/gamebox/gwirl"
)


func Fun(msg string) string {
    sb_ := gwirl.TemplateBuilder{}

    sb_.WriteString(`<section>
    <strong>`)

    gwirl.WriteEscapedHTML(&sb_, msg)

    sb_.WriteString(`</strong>
</section>
`)

    return sb_.String()
}
