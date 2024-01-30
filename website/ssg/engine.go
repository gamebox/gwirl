package ssg

import (
	"bytes"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

type Engine struct {
    docFiles []File
    layoutTemplate func(Page, string) string
    outputDir string
    baseDir string
    markdown goldmark.Markdown
}

type LayoutTemplate func(Page, string) string

func NewEngine(docBaseDir string, layoutTemplate LayoutTemplate, outputDir string) *Engine {
    eng := Engine{ layoutTemplate: layoutTemplate, outputDir: outputDir, baseDir: docBaseDir }
    eng.markdown = goldmark.New(
        goldmark.WithExtensions(
            extension.GFM,
            meta.Meta,
        ),
        goldmark.WithParserOptions(parser.WithAutoHeadingID()),
    )
    eng.loadDocs(docBaseDir)

    return &eng
}

func (e *Engine) Generate() {
    for i := range e.docFiles {
        log.Println(e.docFiles[i].osFile.Name())
        var buff bytes.Buffer
        contents, err := os.ReadFile(e.docFiles[i].osFile.Name())
        if err != nil {
            log.Printf("Could not read file %s: %v", e.docFiles[i].osFile.Name(), err)
            continue
        }
        context := parser.NewContext()
        if err = e.markdown.Convert(contents, &buff, parser.WithContext(context)); err != nil {
            log.Printf("Could not convert markdown file %s: %v", e.docFiles[i].osFile.Name(), err)
            continue
        }
        filePath := createHtmlPath(e.docFiles[i].osFile.Name(), e.baseDir, e.outputDir)
        metadata := meta.Get(context)
        metadata["_filepath"] = filePath
        e.docFiles[i].metadata = NewMetadata(metadata)
        e.docFiles[i].source = string(contents)
        e.docFiles[i].html = buff.String()
    }
    os.RemoveAll(e.outputDir)
    e.writeFiles()
}

func (e *Engine) ensureDirectoryExists(path string) {
	_, err := os.Stat(path)
	if err != nil {
		err := os.MkdirAll(path, 0777)
		if err != nil {
			log.Fatalf("Error creating views directory: %v", err)
		}
	}

}

func (e *Engine) writeFiles() {
    pages := make([]Metadata, 0, len(e.docFiles))
    for i := range e.docFiles {
       pages = append(pages, e.docFiles[i].metadata) 
    }
    for i := range e.docFiles {
        htmlFilePath := strings.TrimSuffix(strings.Replace(e.docFiles[i].osFile.Name(), e.baseDir, e.outputDir, 1), ".md") + ".html"
        dirPath := filepath.Dir(htmlFilePath)
        e.ensureDirectoryExists(dirPath)

        page := NewFilePage(e.docFiles[i], pages)
        content := e.layoutTemplate(&page, e.docFiles[i].html)
        err := os.WriteFile(htmlFilePath, []byte(content), 0644)
        if err != nil {
            log.Fatalf("Error writing file %s: %v", htmlFilePath, err)
        }
        log.Printf("%s -> %s", e.docFiles[i].osFile.Name(), htmlFilePath)

    }
}

func (e *Engine) loadDocs(baseDir string) error {
    filenames := getMarkdownFiles(baseDir)

    files := make([]File, 0, len(filenames))

    for _, filename := range filenames {
        file, err := os.OpenFile(filename, os.O_RDONLY, 0755)
        if err != nil {
            log.Printf("Could not open \"%s\": %v", filename, err)
            return err
        }
        if file == nil {
            continue
        }
        files = append(files, File{osFile: file})
    }

    e.docFiles = files

    return nil
}

func createHtmlPath(filePath string, baseDir string, destinationDir string) string {
    return strings.TrimSuffix(strings.Replace(filePath, baseDir, destinationDir, 1), ".md") + ".html"
}

func getMarkdownFiles(baseDir string) []string {
    filenames := make([]string, 0, 10)

    var rootProcessed bool
    filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, err error) error {
        if path == baseDir && !rootProcessed {
            rootProcessed = true
            return nil
        }
        if err != nil {
            return err
        }
        if d.IsDir() {
            filenames = append(filenames, getMarkdownFiles(filepath.Join(baseDir, d.Name()))...)
            return nil
        }
        if filepath.Ext(d.Name()) == ".md" {
            filenames = append(filenames, path)
            return nil
        }
        return nil

    })

    return filenames
}
