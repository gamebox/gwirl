package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/gamebox/gwirl/internal/gen"
	"github.com/gamebox/gwirl/internal/parser"
)

type File struct {
    name string
    filetype string
    content string
    generated *string
}

func getTemplateFiles(templateDir string, filters []string) []File {
    entries, err := os.ReadDir(templateDir)
    files := []File{}
    if err != nil {
        return files
    }
    for _, dir := range entries {
        if dir.IsDir() {
            subEntries := getTemplateFiles(filepath.Join(templateDir, dir.Name()), filters)
            files = append(files, subEntries...) 
        } else if strings.HasSuffix(dir.Name(), ".twirl.html") && matchesFilter(dir.Name(), filters) {
            fileContent, err := os.ReadFile(filepath.Join(templateDir, dir.Name()))
            if err != nil {
                continue
            }
            files = append(files, File{
                name: strings.Split(dir.Name(), ".")[0],
                content: string(fileContent),
                generated: nil,
                filetype: "html",
            })
        }
    }
    return files
}

func ensureDirectoryExists(path string) {
    _, err := os.Stat(path)
    if err != nil {
        err := os.MkdirAll(path, 0755)
        if err != nil {
            log.Fatalf("Error creating views directory: %v", err)
        }
    }
} 

func capitalize(str string) string {
    runes := []rune(str)
    runes[0] = unicode.ToUpper(runes[0])
    return string(runes)
}

var filterFlag = "--filter"

func fileFilter(flags []string) []string {
    collectingFiles := false
    result := []string{}
    for _, flag := range flags {
        if flag == filterFlag {
            collectingFiles = true
            continue
        }
        if collectingFiles {
            if strings.HasPrefix(flag, "--") {
                break
            }
            result = append(result, flag)
        }
    }
    return result
}

func matchesFilter(name string, filters []string) bool {
    if len(filters) == 0 {
        return true
    }
    for _, filter := range filters {
        log.Printf("\"%s\" has suffix \"%s\"?", name, filter)
        if strings.HasSuffix(name, filter) {
            return true
        }
    }
    return false
}

var logFileFlag = "--logTo"

func loggerFromFlags(flags []string) io.Writer {
    for i, flag := range flags {
        if flag == logFileFlag && len(flags) > i {
            writerName := flags[i + 1]
            if writerName == "stdout" {
                log.Printf("Will log parsing output to stdout\n")
                return os.Stdout
            }
            log.Printf("Will log parsing output to file: %s\n", writerName)
            file, err := os.OpenFile(writerName, os.O_RDWR|os.O_CREATE, 0o777)
            file.Seek(0, 0)
            file.Truncate(0)
            if err != nil {
                log.Fatalf("Could not open log file %s: %v", writerName, err)
            }
            return file
        }
    }
    return nil
}

func main() {
    filters := fileFilter(os.Args)
    g := gen.NewGenerator(false)
    fs := getTemplateFiles("templates", filters)
    parserLogger := loggerFromFlags(os.Args)
    p := parser.NewParser2("")
    if parserLogger != nil {
        p.SetLogger(parserLogger)
    }

    ensureDirectoryExists(filepath.Join("views", "html"))
    for _, f := range fs {
        log.Printf("Parsing %s\n", f.name)

        result := p.Parse(f.content, capitalize(f.name))
        if len(result.Errors) > 0 {
            log.Printf("Could not parse file %s:\n", f.name+".twirl."+f.filetype)
            for _, e := range result.Errors {
                log.Printf("%v\n", e)
            }
            log.Fatalln("FAILED")
        }

        fileWriter, err := os.Create(filepath.Join("views", f.filetype, f.name + "_twirl.go"))
        if err != nil {
            log.Fatalf("Failed to open go file for template: %s\nERROR: %v", f.name, err)
        }

        if parserLogger != nil {
            l, err := parserLogger.Write([]byte(fmt.Sprintf("%v", result.Template)))
            if err != nil || l == 0 {
                fmt.Printf("Something is wrong with the logger")
            }
        } else {
            fmt.Printf("%v", result.Template)
        }

        log.Printf("Generating %s\n", f.name)
        err = g.Generate(result.Template, f.filetype, fileWriter)
        if err != nil {
            os.Remove(fileWriter.Name())
            log.Fatalf("Could not generate a file for template: %s", f.name)
        }
    }

    log.Printf("Completed generating %d templates", len(fs))
}
