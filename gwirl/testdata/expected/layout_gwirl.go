package html

import (
    "strings"

    "github.com/gamebox/gwirl"
)


func Layout(content string) string {
    sb_ := strings.Builder{}

    if  content == ""  {
        sb_.WriteString(`
    <html></html>
`)

    }

    sb_.WriteString(`
<html>
    <head>
        `)

        


    sb_.WriteString(`
        <title>`)

    gwirl.WriteEscapedHTML(&sb_, content)

    sb_.WriteString(`!</title>
    </head>
    <body>
        <nav class="nav"></nav>
        <main>`)

    gwirl.WriteEscapedHTML(&sb_, content)

    sb_.WriteString(`</main>
        <footer></footer>
    </body>
</html>
`)

    return sb_.String()
}
