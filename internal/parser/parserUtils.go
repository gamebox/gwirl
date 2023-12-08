package parser

import (
    "unicode"
)

type stringPredicate func(string) bool

func isGoIdentifierStart(c byte) bool {
    return unicode.IsLetter(rune(c))
}

func isGoIdentifierPart(c byte) bool {
    return isGoIdentifierStart(c) || unicode.IsNumber(rune(c))
}

