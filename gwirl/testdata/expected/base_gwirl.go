package html

import "github.com/gamebox/gwirl/gwirl-example/flash"
import "fmt"
import (
    "github.com/gamebox/gwirl"
)


func Base(flash *flash.Flash, title string, path string, embed string) string {
    sb_ := gwirl.TemplateBuilder{}

    sb_.WriteString(`
<html lang="en" class="min-h-full">
    <head>
        <meta charset="UTF-8" />
        <meta name="viewport" content="width=device-width" />
        <link rel="icon" type="image/svg+xml" href="/logo.svg" />
        <title>`)

    sb_.WriteString(title)

    sb_.WriteString(`</title>
        <link rel="stylesheet" href="/assets/styles.css">
        <link rel="stylesheet" href="https://rsms.me/inter/inter.css">
        <style>
        html {
            --accent: 22, 163, 74;
            --tb-logo-stroke-color: rgb(var(--accent));
            color: white;
        }
        </style>
    </head>
    <body class="dark:bg-black min-h-full">
        `)

    sb_.WriteString(Nav(path))

    sb_.WriteString(`
        `)

    if  flash != nil  {
        sb_.WriteString(`
            <dialog id="flash" class="flash flash-`)

        sb_.WriteString(fmt.Sprint(flash.Type))

        sb_.WriteString(`" open>`)

        sb_.WriteString(flash.Message)

        sb_.WriteString(`</dialog>
            <script>
                document.addEventListener("DOMContentLoaded", () => {
                    const flash = document.body.querySelector("#flash");
                    setTimeout(() => {
                        flash.close();
                    }, 10 * 1000);
                });
            </script>
        `)

    }

    sb_.WriteString(`

        `)

    sb_.WriteString(embed)

    sb_.WriteString(`
        <script src="https://unpkg.com/htmx.org@1.9.10/dist/htmx.min.js"></script>
    </body>
</html>
`)

    return sb_.String()
}
