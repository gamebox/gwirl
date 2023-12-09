# Gwirl

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
@GwirlRaw(userContent)
```

Use an arbitrary Go expression (simple expressions only) and that content will
be escaped appropriately and rendered as a string directly into the content
using the `Stringer` interface. Use the `@GwirlRaw` function to render content
as-is with no escaping.

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
