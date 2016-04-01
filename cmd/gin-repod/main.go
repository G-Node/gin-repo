package main

import (
	"fmt"
	"net/http"

	"github.com/docopt/docopt-go"

	"github.com/gorilla/mux"
	"os"
	"runtime"
)

type Server struct {
	http.Server

	Root *mux.Router
}

type LogLevel int

const (
	PANIC LogLevel = iota
	ERROR
	WARN
	INFO
	DEBUG
)

func (s *Server) log(lvl LogLevel, msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
}

type statusResponseWriter struct {
	http.ResponseWriter
	status        int
	headerWritten bool
}

func (w *statusResponseWriter) WriteHeader(status int) {
	if !w.headerWritten {
		w.status = status
	}

	w.ResponseWriter.WriteHeader(status)
}

func (s *Server) ServeHTTP(original http.ResponseWriter, req *http.Request) {

	w := &statusResponseWriter{original, 200, false}

	defer func() {
		if r := recover(); r != nil {
			w.WriteHeader(http.StatusInternalServerError)

			buf := make([]byte, 1024*10)
			n := runtime.Stack(buf, false)
			st := string(buf[:n])

			s.log(ERROR, "'%s %s' %d, Recovered from panic!", req.Method, req.URL.Path, w.status)
			fmt.Fprintf(os.Stderr, "Panic on reqest: '%s %s': %s", req.Method, req.URL.Path, st)
		} else {
			s.log(INFO, "'%s %s' %d", req.Method, req.URL.Path, w.status)
		}
	}()

	s.Root.ServeHTTP(w, req)
}

func (s *Server) ListenAndServe() error {
	s.log(INFO, "Listening on %s", s.Addr)
	err := s.Server.ListenAndServe()
	if err != nil {
		s.log(ERROR, "ListenAndServe() error: %v", err)
	}
	return err
}

func NewServer(addr string) *Server {
	s := &Server{Server: http.Server{Addr: addr}, Root: mux.NewRouter()}
	s.Handler = s
	return s
}

func main() {
	usage := `gin repo daemon.

Usage:
  gin-repod
  gin-repod -h | --help
  gin-repod --version

Options:
  -h --help     Show this screen.
  `

	args, _ := docopt.Parse(usage, nil, true, "gin repod 0.1a", false)
	fmt.Println(args)

	s := NewServer(":8888")
	r := s.Root

	r.HandleFunc("/intern/user/lookup", lookupUser).Methods("GET")
	r.HandleFunc("/intern/repos/access", repoAccess).Methods("POST")

	r.HandleFunc("/users/{user}/repos", createRepo).Methods("POST")

	s.ListenAndServe()
}
