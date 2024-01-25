package main

import (
	"log"

	"github.com/gamebox/gwirl/website/views/html"
	"github.com/gamebox/gwirl/website/ssg"
)

func main() {
    log.Println("Generating docs...")
    engine := ssg.NewEngine("docs", html.Base, "out")

    engine.Generate()
    log.Println("Complete.")
}
