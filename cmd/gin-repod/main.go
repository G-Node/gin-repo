package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/G-Node/gin-repo/auth"
	"github.com/G-Node/gin-repo/store"
	"github.com/dgrijalva/jwt-go"
	"github.com/docopt/docopt-go"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type Server struct {
	http.Server
	Root *mux.Router

	srvKey []byte

	users store.UserStore
	repos *store.RepoStore
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

		claims := token.Claims.(*auth.Claims)

		if claims.TokenType != "service" {
			s.log(WARN, "Got token without or non-service type")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	s.Root.ServeHTTP(w, req)
}

func (s *Server) getAuthToken(r *http.Request) (*jwt.Token, error) {
	header := r.Header.Get("Authorization")

	if header == "" {
		return nil, auth.ErrNoAuth
	} else if !strings.HasPrefix(header, "Bearer ") {
		return nil, fmt.Errorf("Invalid auth type: %q", header)
	}

	return jwt.ParseWithClaims(strings.Trim(header[6:], " "), &auth.Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Wrong signing method: %v", token.Header["alg"])
		}
		return s.srvKey, nil
	})
}

func (s *Server) SetupServiceSecret() error {

	secret, err := auth.ReadSharedSecret()

	if err != nil {
		secret, err = auth.CreateSharedSecret()

		if err != nil {
			panic(fmt.Errorf("Could not create shared secret: %v", err))
		}

		path := "gin.secret"
		err = ioutil.WriteFile(path, secret, 0600)
		if err != nil {
			s.log(PANIC, "Could not write to gin.secret: %v", err)
			os.Exit(10)
		}

		if abspath, err := filepath.Abs(path); err == nil {
			path = abspath
		}

		s.log(DEBUG, "Wrote shared secret to %q", path)
	}

	s.srvKey = secret

	if val := os.Getenv("GIN_REPO_DEBUGTOKEN"); val != "" {
		str, err := auth.MakeServiceToken(s.srvKey)

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

	s.repos, err = store.NewRepoStore(dir)

	if err != nil {
		s.log(PANIC, "Could not setup repo store: %v", err)
		os.Exit(12)
	}

	repos, err := s.repos.ListRepos()
	if err != nil {
		s.log(PANIC, "Could not read repo store: %v", err)
		os.Exit(13)
	}

	s.log(DEBUG, "repos detected:")
	for _, repo := range repos {
		s.log(DEBUG, "- [%s]", repo)
		public, err := s.repos.GetRepoVisibility(repo)
		if err != nil {
			s.log(WARN, " - visibility error: %v", err)
		} else {
			s.log(DEBUG, " - public: %v", public)
		}

		access, err := s.repos.ListSharedAccess(repo)

		if err != nil {
			s.log(WARN, " - shared access error: %v", err)
		} else if len(access) > 0 {
			s.log(DEBUG, " - sharing:")
			for k, v := range access {
				s.log(DEBUG, "   %q: %s", k, v)
			}
		}
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

func NewServer(addr string) *Server {
	s := &Server{Server: http.Server{Addr: addr}, Root: mux.NewRouter()}
	s.Handler = s
	return s
}

func main() {
	usage := `gin repo daemon.

Usage:
  gin-repod [--listen=<address>]
  gin-repod make-token <user>
  gin-repod -h | --help
  gin-repod --version


Options:
  -h --help            Show this screen.
  --version            Show version.
  --listen=<address>   Address to listen on [default: :8082]
  `

	args, err := docopt.Parse(usage, nil, true, "gin repod 0.1a", false)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while parsing command line: %v\n", err)
		os.Exit(-1)
	}

	fmt.Println(args)

	s := NewServer(args["--listen"].(string))
	s.SetupRoutes()
	s.SetupServiceSecret()
	s.SetupStores()

	s.Handler = handlers.CORS(
		handlers.AllowedHeaders([]string{"Authorization", "Content-Type"}),
		handlers.AllowedMethods([]string{"GET", "PUT", "POST", "DELETE", "PATCH"}),
	)(s.Handler)

	// this call might never return if there actually was
	// a command line "command"
	s.handleCommands(args)

	s.ListenAndServe()
}
