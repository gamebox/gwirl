package ssg

type FilePage struct {
	file File
	site SiteMetadata
}

func NewFilePage(file File, pages []Metadata) FilePage {
	return FilePage{file: file, site: SiteMetadata{pages}}
}

func (fp *FilePage) GetBoolVar(name string) bool {
	return false
}

func (fp *FilePage) GetIntVar(name string) int {
	return 0
}

func (fp *FilePage) GetFloatVar(name string) float64 {
	return 0
}

func (fp *FilePage) GetStringVar(name string) string {
	return ""
}

func (fp *FilePage) GetVar(name string) any {
	return nil
}

func (fp *FilePage) Title() string {
	return fp.file.metadata.Title
}
func (fp *FilePage) Author() string {
	return fp.file.metadata.Author
}
func (fp *FilePage) Description() string {
	return fp.file.metadata.Description
}
func (fp *FilePage) Slug() string {
	return fp.file.metadata.Slug
}
func (fp *FilePage) Tags() []string {
	return fp.file.metadata.Tags
}
func (fp *FilePage) Site() Site {
	return &fp.site
}
