package html

import (
    "strings"

    "github.com/gamebox/gwirl"
)


func TestAll(name string, index int) string {
    sb_ := strings.Builder{}

    sb_.WriteString(`<div `)

    if  index == 0  {
        sb_.WriteString(` class="first" `)

    }

    sb_.WriteString(`>
    `)

    if  index > 0  {
        sb_.WriteString(`
        <hr />
    `)

    }

    sb_.WriteString(`
    <h2>`)

    gwirl.WriteEscapedHTML(&sb_, name)

    sb_.WriteString(`</h2>
</div>
`)

    return sb_.String()
}
