package main

import (
	"bufio"
	"context"
	"log"
	"os"
)

func main() {
	ctx := context.Background()
	reader := bufio.NewReader(os.Stdin)
    reader = bufio.NewReaderSize(reader, 40 * 1024)
	logFile, err := os.OpenFile("/Users/anthonybullard/Desktop/gwirl-lsp.log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o777)
	if err != nil {
		log.Fatalf("Could not open log file: %v", err)
	}
	server := NewGwirlLspServer(os.Stdout, logFile, reader, ctx)

    server.Handle()
}
