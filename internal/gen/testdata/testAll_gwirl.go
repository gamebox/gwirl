package views

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
    `)

    if  name == "Jeff"  {
        sb_.WriteString(`
        <h2>JEFF</h2>
    `)

    } else if  name == "Sue"  {
        sb_.WriteString(`
        <h2>SuE</h2>
    `)

    } else if  name == "Bob"  {
        sb_.WriteString(`
        <h2>B.O.B.</h2>
    `)

    } else {
        sb_.WriteString(`
        <h2>`)

        gwirl.WriteRawHTML(&sb_, name)

        sb_.WriteString(`</h2>
    `)

    }

    sb_.WriteString(`

    `)

    var transclusion__20__5__0 string
    {
        sb_ := gwirl.TemplateBuilder{}
        sb_.WriteString(`
        <p>This is content in the card</p>
    `)

        transclusion__20__5__0 = sb_.String()
    }
    var transclusion__20__5__1 string
    {
        sb_ := gwirl.TemplateBuilder{}
        sb_.WriteString(`
        <button>Card action</button>
    `)

        transclusion__20__5__1 = sb_.String()
    }
    gwirl.WriteRawHTML(&sb_, Card("title", transclusion__20__5__0, transclusion__20__5__1))


    sb_.WriteString(`
</div>
`)

    return sb_.String()
}
