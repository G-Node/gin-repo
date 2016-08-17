package store

import (
	"net/http"

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
	return newLocalUserStore(base)
}
