package html

import (
    "strings"

    "github.com/gamebox/gwirl"
)


func UseOther(names []string) string {
    sb_ := strings.Builder{}

    sb_.WriteString(`<div>
`)

    for  i, name := range names  {
        sb_.WriteString(`
    `)

        gwirl.WriteEscapedHTML(&sb_, TestAll(name, i))

        sb_.WriteString(`
`)

    }

    sb_.WriteString(`
</div>
`)

    return sb_.String()
}
