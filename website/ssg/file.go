package ssg

import "os"

type File struct {
    source string
    html string
    metadata Metadata
    osFile *os.File
}
