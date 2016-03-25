package common

import (
	"github.com/G-Node/gin-repo/ssh"
)

type User struct {
	Uid  string
	Keys []ssh.Key
}
