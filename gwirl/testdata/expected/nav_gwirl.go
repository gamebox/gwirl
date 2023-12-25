package html

import (
    "strings"

    "github.com/gamebox/gwirl"
)


func Nav(path string) string {
    sb_ := strings.Builder{}

    routes := []struct {    
    label string    
    path string    
    }{}    
        


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

        gwirl.WriteEscapedHTML(&sb_, route.label)

        sb_.WriteString(`</li>
        `)

    }

    sb_.WriteString(`
    </ul>
</nav>
`)

    return sb_.String()
}
