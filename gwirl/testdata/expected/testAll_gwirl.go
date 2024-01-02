package html

import (
    "github.com/gamebox/gwirl"
)


func TestAll(name string, index int) string {
    sb_ := gwirl.TemplateBuilder{}

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

    sb_.WriteString(name)

    sb_.WriteString(`</h2>
</div>
`)

    return sb_.String()
}
