package html

import "github.com/gamebox/gwirl/gwirl-example/flash"
import (
    "strings"

    "github.com/gamebox/gwirl"
)


func Base(flash *flash.Flash, title string, path string, embed string) string {
    sb_ := strings.Builder{}

    sb_.WriteString(`
<html lang="en" class="min-h-full">
    <head>
        <meta charset="UTF-8" />
        <meta name="description" content="Astro description" />
        <meta name="viewport" content="width=device-width" />
        <link rel="icon" type="image/svg+xml" href="/logo.svg" />
        <title>`)

    gwirl.WriteEscapedHTML(&sb_, title)

    sb_.WriteString(`</title>
        <link rel="stylesheet" href="/assets/timebox.css">
        <link rel="stylesheet" href="https://rsms.me/inter/inter.css">
        <style>
        html {
            --accent: 22, 163, 74;
            --tb-logo-stroke-color: rgb(var(--accent));
        `)

    return sb_.String()
}
