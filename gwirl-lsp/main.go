package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

func logFileFallback() io.Writer {
	logFile, err := os.OpenFile(os.DevNull, os.O_APPEND|os.O_RDONLY, 0755)
	if err != nil {
		log.Println("Will log to stderr")
		return os.Stderr
	}
	log.Println("Logging disabled")
	return logFile
}

func createLogFile() io.Writer {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Println("Cannot find cache directory.  Ensure that $HOME is set.")
		return logFileFallback()
	}
	logpath := fmt.Sprintf(filepath.Join(cacheDir, "gwirl-lsp"))
	_, err = os.Stat(logpath)
	if err != nil {
		err = os.Mkdir(logpath, 0755)
	}
	if err != nil {
		log.Println("Could not create log file")
		return logFileFallback()
	}
	logFilePath := filepath.Join(logpath, "gwirl-lsp.log.txt")
	logFile, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o777)
	if err != nil {
		log.Printf("Could not open log file: %v", err)
		return logFileFallback()
	}
	log.Printf("Logging to %s", logFilePath)
	return logFile
}

func main() {
	ctx := context.Background()
	reader := bufio.NewReader(os.Stdin)
	reader = bufio.NewReaderSize(reader, 40*1024)
	logFile := createLogFile()
	server := NewGwirlLspServer(os.Stdout, logFile, reader, ctx)

	server.Handle()
}
