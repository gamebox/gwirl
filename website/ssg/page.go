package ssg

type VarProvider interface {
    GetBoolVar(name string) bool
    GetIntVar(name string) int
    GetFloatVar(name string) float64
    GetStringVar(name string) string
    GetVar(name string) any
}

type Site interface {
    VarProvider
    TopLevelPages() []Metadata
    SectionPages(section string) []Metadata
}

type Page interface {
    VarProvider
    Title() string
    Author() string
    Description() string
    Slug() string
    Tags() []string
    Site() Site
}
