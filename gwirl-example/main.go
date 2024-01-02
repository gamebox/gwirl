package main

import (
	"embed"
	"log"
	"net/http"
	"strings"

	"github.com/gamebox/gwirl/gwirl-example/model"
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

type Server struct {
	data []model.Participant
}

func (s *Server) index(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != "GET" {
		responseWriter.WriteHeader(404)
		return
	}
	content := html.Index(s.data)
	responseWriter.WriteHeader(200)
	responseWriter.Header().Add("Content-Type", "application/html")
	responseWriter.Write([]byte(content))
}

//go:embed assets
var staticFS embed.FS

func renderStatic(responseWriter http.ResponseWriter, request *http.Request) {
	path := strings.TrimPrefix(request.URL.Path, "/")
	contents, err := staticFS.ReadFile(path)
	if err != nil {
		log.Printf("Could not find static file for \"%s\"\n", path)
		responseWriter.WriteHeader(404)
		return
	}
	responseWriter.WriteHeader(200)
	responseWriter.Header().Add("Content-Type", "application/css")
	responseWriter.Write(contents)
}

func main() {
	data := []model.Participant{
		{FirstName: "Anthony", LastName: "Bullard", Email: "redacted@example.xyz", Id: "123"},
	}
	s := Server{
		data: data,
	}
	http.HandleFunc("/", s.index)
	http.HandleFunc("/assets/", renderStatic)
	http.HandleFunc("/template/", renderTemplate)
	log.Fatalf("%v", http.ListenAndServe("127.0.0.1:8080", nil))
}
