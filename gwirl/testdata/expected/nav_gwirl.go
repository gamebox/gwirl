package html

import (
    "github.com/gamebox/gwirl"
)


func Nav(path string) string {
    sb_ := gwirl.TemplateBuilder{}

    routes := []struct {    
    label string    
    path string    
    }{    
    { "Home", "/" },    
    }    
        


    sb_.WriteString(`
<nav class="navbar">
    <ul>
        `)

    for  _, route := range routes  {
        sb_.WriteString(`
        <li `)

        if  route.path == path  {
            sb_.WriteString(` class="selected" `)

        }

        sb_.WriteString(`>`)

        sb_.WriteString(route.label)

        sb_.WriteString(`</li>
        `)

    }

    sb_.WriteString(`
    </ul>
</nav>
`)

    return sb_.String()
}
