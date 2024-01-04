# Gwirl

![ci status](https://github.com/gamebox/gwirl/actions/workflows/master.yml/badge.svg)  [![Go Reference](https://pkg.go.dev/badge/github.com/gamebox/gwirl.svg)](https://pkg.go.dev/github.com/gamebox/gwirl)

A typesafe templating language for Go projects, inspired by Scala Play's Twirl
template engine.

```html
@(name string, activities []Activity, showBorders bool)

<section @if showBorders {class="b-w-sm b-"}>
    <h2>@name</h2>

    <ul>
    @for _, activity := range activities {
        <li>@activity.Label</li>
    }
    </ul>
</section>
```

Similar to other great templating languages like [Templ](https://templ.guide),
Gwirl is fully type-safe and doesn't require you to learn a lot of special rules
or do a lot of configuration.  Just drop some html into a `.[ext].gwirl` file
into a `templates` directory at the root of your project and run `gwirl`.  Gwirl
will generate performant Go code in a `views` package that you can then call anywhere
that you can import that package.  It can be used in more than just HTML;
including XML, Markdown, and plain text.  The only special character is `@`,
which means if it is used in your content, you must escape it as `@@`.
Similarly, you can escape opening `{` as `@{` and closing `}`s as `@}` if you
are for instancing building a Go struct or slice in a if, for, Go block, or Go
expression.

## Getting started

First, you should install the gwirl tool using `go get`:

```
go install github.com/gamebox/gwirl/gwirl
```

Then in the root of your project create a directory for your templates:

```
mkdir templates
```

Then create your first Gwirl file:

```
touch templates/hello.html.gwirl
```

Open that file and put in the following content:

```html
@(name string)

<h1>Hello @name!</h1>
```

You can then render this out using a net/http handler, your web framework of
choice, or even just a fmt.Print, like so:

### Simple usage
```go
package main

import (
    "fmt"

    "myproject.com/myproject/views/html"
)

func main() {
    fmt.Println(views.Hello("World"))
}
```

Then just run the following commands the first time:

```
> gwirl
> go mod tidy
> go run .
```

When you edit or add a template, just run `gwirl` and then `go run .`.

You can also feel free to add a go:generate comment to the top of the file
holding your `main()` function and then use `go generate` to generate the go
files along with any other generated files you need.

```go
package main
//go:generate gwirl

func main() {
    // ...
}
```

## Supported features

### Template parameters

```gwirl
@(name string, activities []Activity, showBorders bool)
```

Just specify the parameters to your template like you would for a function in
Go.  No return type is needed as they all return a `string`.

### Go blocks

```gwirl
@{
    someValue := pkg.SomeHelperFunction(var1, var2)
}
```

Use an arbitrary block of Go code that will be ran everytime your template is
rendered.  Any variables created here are block scoped and available for use in
the rendered content.

### Go expressions
```gwirl
<h1>@name</h1>
<h2>@pkg.SomeFunction(var1)</h2>
@!userContent
```

Use an arbitrary Go expression (simple expressions only) and that content will
be rendered as a string directly into the content using the `Stringer`
interface. Use the `@!` syntax to render content with escaping that is
appropriate to the content's filetype.

### Imports

```gwirl
@import "yoururl.com/example/pkg"
@import (
    "fmt"
    h "example.com/stringhelpers/helpers"
)
```

Use Go import statements the same as you would in Go to bring in needed helpers
and other functionality.

### If/Else if/Else statements

```html
<section @if showBorders {class="b-w-sm b-"}>
```

Conditionally render content using `@if` the same way you would in Go, anything
between the `@if` and the `{` on the same line can contain any code you would
use for a condition in a Go `if`.

### For statements

```html
<ul>
@for _, activity := range activities {
    <li>@activity.Label</li>
}
</ul>
```

Render out arrays and slices of content using `for`, again, exactly the same
way you would in Go.  Use initializer/condition/change syntax, or range syntax.
(You can also use forever and for while-type syntax at your own peril :-)).
