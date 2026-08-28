package main

import (
    "sync"

    "github.com/gamebox/gwirl/internal/parser"
)

type TemplateCacheItem struct {
    text string
    version int
    template parser.Template2
    goFileText string
}

type TemplateCache struct {
    sync.Map
}

func NewTemplateCache() TemplateCache {
    return TemplateCache{ sync.Map{} }
}


