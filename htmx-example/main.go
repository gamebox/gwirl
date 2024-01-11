package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gamebox/gwirl/htmx-example/todo"
	"github.com/gamebox/gwirl/htmx-example/views/html"
)

type Server struct {
	count  int
	nextId int
	tl     todo.TodoList
	filter todo.Filter
}

func (s *Server) renderIndex(rw http.ResponseWriter, req *http.Request) {
	log.Println("renderIndex")
	if req.URL.Path != "/" {
		rw.WriteHeader(404)
	}
	rw.Write([]byte(html.Index(s.count)))
}

func (s *Server) renderCount(rw http.ResponseWriter, req *http.Request) {
	log.Println("renderCount")
	s.count += 1
	rw.Write([]byte(html.Counter(s.count)))
}

func (s *Server) renderTodo(rw http.ResponseWriter, req *http.Request) {
	log.Println("renderTodo")
	switch req.Method {
	case "GET":
		filter := filterFromRequest(req)
		s.filter = filter
		content := html.Todo(s.tl.GetByFilter(s.filter), s.filter, s.tl.CompletedRemain())
		rw.Write([]byte(content))
	case "POST":
		text := req.FormValue("newTodo")
		if text == "" {
			return
		}
		firstTodo := len(s.tl.GetByFilter(todo.DefaultFilter())) == 0
		todo := s.tl.AddTodo(text)
		var content string
		if firstTodo {
			content = fmt.Sprintf(
				"%s\n%s",
				html.MainSection(s.tl.GetByFilter(s.filter), s.filter, s.tl.CompletedRemain()),
				html.TodoInput(false),
			)
		} else {
			content = fmt.Sprintf(
				"%s\n%s",
				html.TodoItem(todo),
				html.TodoInput(false),
			)
		}
		rw.Header().Add("HX-TRIGGER", "todos-updated")
		rw.Write([]byte(content))
		rw.WriteHeader(201)
	case "PATCH":
		queryParams := req.URL.Query()
		action := queryParams.Get("action")
		if action == "" {
			rw.WriteHeader(400)
			return
		}
		id := queryParams.Get("id")
		if id == "" {
			rw.WriteHeader(400)
			return
		}
		idInt, err := strconv.Atoi(id)
		if err != nil {
			rw.WriteHeader(400)
			return
		}
		switch action {
		case "edit":
			s.tl.ToggleEditing(idInt)
		case "update":
			req.ParseForm()
			text := req.FormValue("text")
			if text == "" {
				s.tl.RemoveTodo(idInt)
			}
			s.tl.UpdateText(idInt, text)
			s.tl.ToggleEditing(idInt)
		case "complete":
			s.tl.ToggleCompleted(idInt)
		default:
			rw.WriteHeader(400)
			return
		}
		t := s.tl.TodoById(idInt)
		if t == nil {
			rw.WriteHeader(500)
			return
		}
		content := html.TodoItem(*t)
		rw.Write([]byte(content))
	case "DELETE":
		queryParams := req.URL.Query()
		id := queryParams.Get("id")
		if id == "" {
			rw.WriteHeader(400)
			return
		}
		idInt, err := strconv.Atoi(id)
		if err != nil {
			rw.WriteHeader(400)
			return
		}
		s.tl.RemoveTodo(idInt)
		rw.Header().Add("HX-TRIGGER", "todos-updated")
	default:
		req.ParseForm()
		log.Printf("METHOD: %s; FORM: %v", req.Method, req.Form)
		rw.WriteHeader(404)
	}
}

func (s *Server) updateTodoFilter(rw http.ResponseWriter, req *http.Request) {
	queryParams := req.URL.Query()
	rawFilter := queryParams.Get("filter")
	filter := todo.ParseFilter(rawFilter)
	s.filter = filter

	content := html.Filters(filter)
	rw.Header().Add("HX-TRIGGER", "filter-updated")
	rw.Write([]byte(content))
}

func (s *Server) renderTodoList(rw http.ResponseWriter, req *http.Request) {
	todos := s.tl.GetByFilter(s.filter)
	content := html.TodoList(todos)
	rw.Write([]byte(content))
}

func filterFromRequest(req *http.Request) todo.Filter {
	anchor := req.URL.Query().Get("filter")
	log.Printf("anchor is \"%s\"", anchor)
	if anchor == "" {
		anchor = "all"
	}
	return todo.ParseFilter(anchor)
}

func (s *Server) renderTodoCount(rw http.ResponseWriter, req *http.Request) {
	content := fmt.Sprintf("%d", len(s.tl.GetByFilter(s.filter)))
	rw.Write([]byte(content))
}

func (s *Server) serveAssets(rw http.ResponseWriter, req *http.Request) {
	log.Println("serveAssets")
	if !strings.HasPrefix(req.URL.Path, "/assets") {
		log.Printf("URL path was \"%s\"", req.URL.Path)
		rw.WriteHeader(404)
		return
	}
	http.ServeFile(rw, req, "."+req.URL.Path)
}

func (s *Server) clearCompleted(rw http.ResponseWriter, req *http.Request) {
	if req.Method != "DELETE" {
		rw.WriteHeader(404)
		return
	}
	s.tl.ClearCompleted()
	todos := s.tl.GetByFilter(s.filter)
	content := html.TodoList(todos)
	rw.Header().Add("HX-TRIGGER", "todos-updated")
	rw.Write([]byte(content))
}

func main() {
	log.Println("Starting up...")
	handler := http.NewServeMux()
	s := Server{0, 0, todo.NewTodoList(), todo.DefaultFilter()}
	handler.HandleFunc("/", s.renderIndex)
	handler.HandleFunc("/completed", s.clearCompleted)
	handler.HandleFunc("/count", s.renderCount)
	handler.HandleFunc("/todo", s.renderTodo)
	handler.HandleFunc("/todocount", s.renderTodoCount)
	handler.HandleFunc("/todofilter", s.updateTodoFilter)
	handler.HandleFunc("/todos", s.renderTodoList)
	handler.HandleFunc("/assets/", s.serveAssets)
	log.Fatalf("%v", http.ListenAndServe(":3000", handler))
}
