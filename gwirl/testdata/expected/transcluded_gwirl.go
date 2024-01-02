package html

import (
    "github.com/gamebox/gwirl"
)


func Transcluded(name string, index int) string {
    sb_ := gwirl.TemplateBuilder{}

    var foo string    
    if index % 2 == 0 {    
    foo = "even"    
    } else {    
    foo = "odd"    
    }    
        


    sb_.WriteString(`

`)

    var transclusion__12__1 string
    {
        sb_ := gwirl.TemplateBuilder{}
        sb_.WriteString(`
    <div>
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
        <h3>`)

        sb_.WriteString(foo)

        sb_.WriteString(`</h3>
        <script>
            document.body.addEventListener("load", () => {`)

        transclusion__12__1 = sb_.String()
    }
    sb_.WriteString(Layout(transclusion__12__1))


    sb_.WriteString(`)
        </script>
    </div>
`)

    return sb_.String()
}
