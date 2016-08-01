package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var (
	ErrNoAuth = errors.New("no authentication provided")
)

type Claims struct {
	*jwt.StandardClaims
	TokenType string
}

func ReadSharedSecret() ([]byte, error) {
	path := "."
	_, err := os.Stat("gin.secret")
	if err != nil {
		path = ""
	}

	if path == "" {
		u, err := user.Current()
		if err == nil {
			path = u.HomeDir
		}
	}

	if path == "" {
		path = os.Getenv("HOME")
	}

	filename := filepath.Join(path, "gin.secret")
	secret, err := ioutil.ReadFile(filename)

	return secret, err
}

func CreateSharedSecret() ([]byte, error) {
	key := make([]byte, 23)
	_, err := rand.Read(key)

	err = ioutil.WriteFile("gin.secret", key, 0600)
	if err != nil {
		return key, fmt.Errorf("could not write to shared secret: %v", err)
	}

	return key, nil
}

func MakeServiceToken(key []byte) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	host, err := os.Hostname()
	if err != nil {
		host = "localhost"
	}

	token.Claims = &Claims{
		&jwt.StandardClaims{
			Issuer:    "gin-repo@" + host,
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(time.Minute * 120).Unix(),
		},
		"service",
	}

	str, err := token.SignedString(key)

	if err != nil {
		return "", err
	}

	return str, nil
}
