package main

import (
	"log"
	"os"
)

func main() {
	flags := NewFlags()
	parserLogger := flags.Logger()
	log.SetOutput(parserLogger)
	cwd, _ := os.Getwd()
	accessor := NewRealFSAccessor(cwd)
	builder := NewBuilder(flags, accessor, parserLogger)
	err := builder.build()
	if err != nil {
		log.Fatalf("Build failed due to the following errors: %e", err)
	}
}
