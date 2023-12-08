package main

import (
	"log"
	"net/http"
	"strings"
    "github.com/gamebox/gwirl/gwirl-example/views/html"
)


func renderTemplate(responseWriter http.ResponseWriter, request *http.Request) {
    template := strings.TrimPrefix(request.URL.Path, "/template/")
    var content string
    switch template {
    case "transcluded":
        content = html.Transcluded("Bob", 0)
    case "layout":
        names := []string{"Bob", "Joe", "Sue", "Your Mom"}
        content = html.Layout(html.UseOther(names))
    }
    if content == "" {
        responseWriter.WriteHeader(404)
        responseWriter.Header().Add("Content-Type", "application/html")
        responseWriter.Write([]byte("Not found"))
        return
    }
    responseWriter.WriteHeader(200)
    responseWriter.Header().Add("Content-Type", "text/plain")
    responseWriter.Write([]byte(content))
}


func main() {
    http.HandleFunc("/template/", renderTemplate)
    log.Fatalf("%v", http.ListenAndServe("127.0.0.1:8080", nil))
}
