package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func cleanDir(filetype string, d []os.DirEntry) {
    for _, entry := range d {
        if strings.HasSuffix(entry.Name(), "_gwirl.go") {
            _ = os.Remove(filepath.Join("views", filetype, entry.Name()))
        }
    }
}

func clean() {
    fmt.Println("Cleaning views directories of Gwirl files...")
    fileTypes := [4]string{"html","xml","md","txt"}
    for _, ft := range fileTypes {
        d, err := os.ReadDir(filepath.Join("views", ft))
        if err != nil {
            continue
        }
        cleanDir(ft, d)
    }
    fmt.Println("Clean!")
}

