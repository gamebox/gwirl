package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type File struct {
    name string
    filetype string
    content string
}

type FSAccessor interface {
    // Collects a slice of all template files found within the directory specified.
    // The directory should be relative to a root directory, which may be the
    // current working directory or a configurable root directory depending on
    // the implementation.
    TemplateFiles(dir string, filters []string) []File
    // Creates the "_gwirl.go" file that is executable in the Go program. Where
    // it will be created is based on the name and the filetype.  The base
    // directory will be determined by joining the root directory with the 
    // segment "views".
    CreateGwirlFile(f *File) (io.Writer, string, error)
    // Will create the given subdirectory in the root directory if it does not
    // exist.
    EnsureDirectoryExists(name string)
    // Will remove the file at the path determined by joining the root directory
    // with the path given.
    Remove(name string) error
}

type RealFSAccessor struct {
    rootDir string
}

func NewRealFSAccessor(rootDir string) *RealFSAccessor {
    a := RealFSAccessor{ rootDir: rootDir }
    return &a
}

func (a *RealFSAccessor) Remove(name string) error {
    return os.Remove(filepath.Join(a.rootDir, name))
}

func (a *RealFSAccessor) EnsureDirectoryExists(path string) {
    _, err := os.Stat(filepath.Join(a.rootDir, path))
    if err != nil {
        err := os.MkdirAll(filepath.Join(a.rootDir,path), 0755)
        if err != nil {
            log.Fatalf("Error creating views directory: %v", err)
        }
    }
} 

func (a *RealFSAccessor) TemplateFiles(templateDir string, filters []string) []File {
    return templateFiles(filepath.Join(a.rootDir, templateDir), filters)
}

func templateFiles(templateDir string, filters []string) []File {
    entries, err := os.ReadDir(templateDir)
    files := make([]File, 0, len(entries))
    if err != nil {
        return files
    }
    for _, dir := range entries {
        if dir.IsDir() {
            subEntries := templateFiles(filepath.Join(templateDir, dir.Name()), filters)
            files = append(files, subEntries...) 
        } else if strings.HasSuffix(dir.Name(), ".gwirl") && matchesFilter(dir.Name(), filters) {
            fileContent, err := os.ReadFile(filepath.Join(templateDir, dir.Name()))
            if err != nil {
                continue
            }
            filenameSegments := strings.Split(filepath.Base(dir.Name()), ".")
            fileType := filenameSegments[len(filenameSegments) - 2]
            if fileType != "html" && fileType != "xml" && fileType != "md" && fileType != "txt" {
                fileType = "txt"
            }
            files = append(files, File{
                name: strings.Split(dir.Name(), ".")[0],
                content: string(fileContent),
                filetype: fileType,
            })
        }
    }
    return files
}

func (a *RealFSAccessor) CreateGwirlFile(f *File) (io.Writer, string, error) {
    fileName := filepath.Join(a.rootDir, "views", f.filetype, f.name+"_gwirl.go")
	fileWriter, err := os.Create(fileName)
	if err != nil {
		e := errors.New(fmt.Sprintf("Failed to open go file for template: %s\nERROR: %v", f.name, err))
		return nil, fileName, errors.Join(e, err)
	}
    return fileWriter, fileName, nil 
}

func matchesFilter(name string, filters []string) bool {
    if len(filters) == 0 {
        return true
    }
    for _, filter := range filters {
        log.Printf("\"%s\" has prefix \"%s\"?", name, filter)
        if strings.HasPrefix(name, filter) {
            log.Println("YUP")
            return true
        }
    }
    return false
}


