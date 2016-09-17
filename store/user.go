package store

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/G-Node/gin-repo/ssh"
)

type User struct {
	Uid  string
	Keys []ssh.Key
}

type UserStore interface {
	LookupUserBySSH(fingerprint string) (*User, error)
	TokenForUser(uid string) (string, error)
	UserForRequest(r *http.Request) (*User, error)
}

func NewUserStore(base string) (UserStore, error) {

	path := filepath.Join(base, "user.store")

	data, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return newLocalUserStore(base)
	} else if err != nil {
		return nil, err
	}

	stype := strings.Trim(string(data), "\n ")

	switch {
	case stype == "local":
		return newLocalUserStore(base)
	case strings.HasPrefix(stype, "ginauth@"):
		return newGinAuthStore(stype[8:])
	}

	return nil, fmt.Errorf("unknown store type: %v", stype)
}
