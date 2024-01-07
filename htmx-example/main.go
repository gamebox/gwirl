package main

import (
    "log"
    "net/http"

    "github.com/gamebox/gwirl/htmx-example/views/html"
)

type Server struct {
    count int
}


func (s *Server) renderIndex(rw http.ResponseWriter, req *http.Request) {
    rw.Write([]byte(html.Index(s.count)))
    rw.WriteHeader(200)
}

func (s *Server) renderCount(rw http.ResponseWriter, req *http.Request) {
    s.count += 1
    rw.Write([]byte(html.Counter(s.count)))
    rw.WriteHeader(200)
}

func main() {
    log.Println("Starting up...")
    handler := http.NewServeMux()
    s := Server{0}
    handler.HandleFunc("/", s.renderIndex)
    handler.HandleFunc("/count", s.renderCount)
    log.Fatalf("%v", http.ListenAndServe(":3000", handler))
}
