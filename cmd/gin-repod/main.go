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

func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	defer func() {
		if r := recover(); r != nil {
			w.WriteHeader(http.StatusInternalServerError)

			buf := make([]byte, 1024*10)
			n := runtime.Stack(buf, false)
			st := string(buf[:n])

			s.log(ERROR, "'%s %s' 500, Recovered from panic!", req.Method, req.URL.Path)
			fmt.Fprintf(os.Stderr, "Panic on reqest: '%s %s': %q", req.Method, req.URL.Path, st)
		} else {
			s.log(INFO, "'%s %s'", req.Method, req.URL.Path)
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
