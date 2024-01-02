package html

import (
    "github.com/gamebox/gwirl"
)


func Layout(content string) string {
    sb_ := gwirl.TemplateBuilder{}

    if  content == ""  {
        sb_.WriteString(`
    <html></html>
`)

    }

    sb_.WriteString(`
<html>
    <head>
        <title>`)

    gwirl.WriteEscapedHTML(&sb_, content)

    sb_.WriteString(`</title>
    </head>
    <body hx-boost="true">
        <nav class="nav"></nav>
        <main>`)

    sb_.WriteString(content)

    sb_.WriteString(`</main>
        <footer></footer>
        <script src="https://unpkg.com/htmx.org@1.9.10/dist/htmx.min.js"></script>
    </body>
</html>
`)

    return sb_.String()
}
