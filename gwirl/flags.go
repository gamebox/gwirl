package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

type Flags struct {
	logger string
	clean  bool
	filter Filters
}

type Filters struct {
	filters []string
}

func (f *Filters) String() string {
	sb := strings.Builder{}
	for _, s := range f.filters {
		sb.WriteString(s)
	}

	return sb.String()
}

func (f *Filters) Set(s string) error {
	f.filters = append(f.filters, strings.TrimSpace(s))
	return nil
}

func (flags *Flags) Logger() io.Writer {
	var writerName string
	if flags.logger == "stdout" {
		fmt.Printf("Will log parsing output to stdout\n")
		return os.Stdout
	}
	if flags.logger == "" {
		fmt.Printf("No log output\n")
		writerName = os.DevNull
	} else {
		writerName = flags.logger
		fmt.Printf("Will log parsing output to file: %s\n", writerName)
	}
	file, err := os.OpenFile(writerName, os.O_RDWR|os.O_CREATE, 0o777)
	file.Seek(0, 0)
	file.Truncate(0)
	if err != nil {
		log.Fatalf("Could not open log file %s: %v", writerName, err)
	}
	return file
}

func NewFlags() *Flags {
	flags := Flags{}
	var filters Filters = Filters{filters: make([]string, 0, 10)}
	flag.Var(&filters, "filter", "Filter the templates that are generated")
	logger := flag.String("logTo", "", "A file to output logs to.  Use \"stdout\" to have the logs just be printed to stdout")
	clean := flag.Bool("clean", false, "Clean Gwirl output")

	flag.Parse()
	flags.filter = filters
	if logger != nil {
		fmt.Printf("Logger is %s", *logger)
		flags.logger = *logger
	}
	if clean != nil {
		flags.clean = *clean
	}
	return &flags
}
