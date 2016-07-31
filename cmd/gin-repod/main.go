package main

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/G-Node/gin-repo/store"
	"github.com/dgrijalva/jwt-go"
	"github.com/docopt/docopt-go"
	"github.com/gorilla/mux"
)

type Server struct {
	http.Server
	Root *mux.Router

	srvKey []byte

	users *store.UserStore
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

	// this should most like be done differently, in a middleware maybe
	//  good enough for now though
	if strings.HasPrefix(path.Clean(req.URL.Path), "/intern") {
		token, err := s.getAuthToken(req)
		if err != nil {
			s.log(WARN, "Got invalid token: %v", err)
			w.WriteHeader(http.StatusForbidden)
			return
		}

		if role, ok := token.Claims["role"].(string); !ok || (role != "service" && role != "debug") {
			s.log(WARN, "Got token without or non-service role")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	s.Root.ServeHTTP(w, req)
}

func (s *Server) getAuthToken(r *http.Request) (*jwt.Token, error) {
	auth := r.Header.Get("Authorization")

	if auth == "" {
		return nil, fmt.Errorf("No auth header")
	} else if !strings.HasPrefix(auth, "Bearer ") {
		return nil, fmt.Errorf("Invalid auth type: %q", auth)
	}

	return jwt.Parse(strings.Trim(auth[6:], " "), func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Wrong signing method: %v", token.Header["alg"])
		}
		return s.srvKey, nil
	})
}

func (s *Server) createSharedSecret() {
	s.srvKey = make([]byte, 23)
	_, err := rand.Read(s.srvKey)

	if err != nil {
		s.log(PANIC, "Could not obtain random bytes for secret: %v", err)
		os.Exit(10)
	}

	path := "gin.secret"
	err = ioutil.WriteFile(path, s.srvKey, 0600)
	if err != nil {
		s.log(PANIC, "Could not write to gin.secret: %v", err)
		os.Exit(10)
	}

	if abspath, err := filepath.Abs(path); err == nil {
		path = abspath
	}

	s.log(DEBUG, "Wrote shared secret to %q", path)
}

func (s *Server) readSharedSecret() error {
	path := "gin.secret"
	data, err := ioutil.ReadFile(path)

	if err != nil {
		return err
	}

	s.srvKey = data
	return nil
}

func (s *Server) SetupServiceSecret() error {

	err := s.readSharedSecret()

	if err != nil {
		s.log(DEBUG, "Creating new shared secret")
		s.createSharedSecret()
	}

	if val := os.Getenv("GIN_REPO_DEBUGTOKEN"); val != "" {
		token := jwt.New(jwt.SigningMethodHS256)

		host, err := os.Hostname()
		if err != nil {
			host = "localhost"
		}

		token.Claims["iss"] = "gin-repo@" + host
		token.Claims["iat"] = time.Now().Unix()
		token.Claims["exp"] = time.Now().Add(time.Minute * 120).Unix()
		token.Claims["role"] = "debug"

		str, err := token.SignedString(s.srvKey)

		if err != nil {
			s.log(PANIC, "Could not make debug auth token")
		}
		s.log(DEBUG, "Token: [%s]", str)

		err = ioutil.WriteFile("token", []byte(str), 0600)

		if err != nil {
			s.log(WARN, "Could not write debug token: %v", err)
		}
	}

	return nil
}

func (s *Server) SetupStores() {
	var err error
	dir := os.Getenv("GIN_REPO_DIR")

	if dir == "" {
		dir = "."
	}

	s.users, err = store.NewUserStore(dir)

	if err != nil {
		s.log(PANIC, "Could not setup user store: %v", err)
		os.Exit(11)
	}

}

func (s *Server) ListenAndServe() error {
	s.log(INFO, "Listening on %s", s.Addr)
	err := s.Server.ListenAndServe()
	if err != nil {
		s.log(ERROR, "ListenAndServe() error: %v", err)
	}
	return err
}

func (s *Server) repoDir(user string) string {
	dir := os.Getenv("GIN_REPO_DIR")

	if dir == "" {
		dir = "."
	}

	return filepath.Join(dir, user)
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

	r.HandleFunc("/intern/user/lookup", s.lookupUser).Methods("GET")
	r.HandleFunc("/intern/repos/access", s.repoAccess).Methods("POST")

	r.HandleFunc("/users/{user}/repos", createRepo).Methods("POST")
	r.HandleFunc("/users/{user}/repos", s.listRepos).Methods("GET")

	r.HandleFunc("/users/{user}/repos/{repo}/branches/{branch}", s.getBranch).Methods("GET")

	r.HandleFunc("/users/{user}/repos/{repo}/objects/{object}", s.getObject).Methods("GET")

	s.SetupServiceSecret()
	s.SetupStores()
	s.ListenAndServe()
}
