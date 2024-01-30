package ssg

type SiteMetadata struct {
    pages []Metadata
}

func (site *SiteMetadata) TopLevelPages() []Metadata {
    return site.pages
}

func (site *SiteMetadata) SectionPages(section string) []Metadata {
    return site.pages
}

func (site *SiteMetadata) GetBoolVar(name string) bool {
   return false 
}

func (site *SiteMetadata) GetIntVar(name string) int {
    return 0
}

func (site *SiteMetadata) GetFloatVar(name string) float64 {
    return 0
}

func (site *SiteMetadata) GetStringVar(name string) string {
    return ""
}

func (site *SiteMetadata) GetVar(name string) any {
    return nil
}
