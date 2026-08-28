package pathutils

import (
	"path/filepath"
	"strings"
)

type PathConfig struct {
  TemplateBase string
  DestBase string
}

func NewPathConfig(templateBase string, destBase string) PathConfig {
    return PathConfig{templateBase, destBase}
}


func (c PathConfig) GoPath(gwirlPath string) string {
    if filepath.Ext(gwirlPath) != ".gwirl" {
        return ""
    }
    templateName := filepath.Base(gwirlPath)
    templateName, fullExtension, found := strings.Cut(templateName, ".")
    if !found {
        return ""
    }
    filetype, gwirl, found := strings.Cut(fullExtension, ".")
    if !found || filetype == "gwirl" || gwirl != "gwirl" {
        return ""
    }
    filename := templateName+"_gwirl.go"

    dest := filepath.Dir(gwirlPath)
    dest, found = strings.CutSuffix(dest, c.TemplateBase)
    if !found {
        return ""
    }
    return filepath.Join(dest, c.DestBase, filetype, filename)
}

func (c PathConfig) GwirlPath(goPath string) string {
    if filepath.Ext(goPath) != ".go" {
        println("Didn't have go extension")
        return ""
    }
    templateName := filepath.Base(goPath)
    templateName, found := strings.CutSuffix(templateName, "_gwirl.go")
    if !found || templateName == "" {
        println("didn't end in _gwirl.go")
        return ""
    }
    segments := strings.Split(filepath.Dir(goPath), "/")
    if len(segments) == 0 {
        return ""
    }
    filetype := segments[len(segments) - 1]
    filename := templateName+"."+filetype+".gwirl"
    // TODO: pop off the filetype segment
    dest := filepath.Dir(filepath.Dir(goPath))
    dest, found = strings.CutSuffix(dest, c.DestBase)
    if !found {
        println("didn't find the destBase")
        return ""
    }
    return filepath.Join(dest, c.TemplateBase, filename)
}
