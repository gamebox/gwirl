package html

import (
    "github.com/gamebox/gwirl"
)


func UseOther(names []string) string {
    sb_ := gwirl.TemplateBuilder{}

    sb_.WriteString(`<div>
`)

    for  i, name := range names  {
        sb_.WriteString(`
    `)

        sb_.WriteString(TestAll(name, i))

        sb_.WriteString(`
`)

    }

    sb_.WriteString(`
</div>
`)

    return sb_.String()
}
