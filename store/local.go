package store

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"

	"github.com/G-Node/gin-repo/auth"
	"github.com/G-Node/gin-repo/ssh"
)

type LocalUserStore struct {
	Path string

	users    map[string]*User
	key2User map[string]*User

	secret []byte
}

func (store *LocalUserStore) loadUser(uid string) (User, error) {
	base := filepath.Join(store.Path, uid)

	keyMap := ssh.ReadKeysInDir(base)

	keys := make([]ssh.Key, len(keyMap))
	i := 0
	for _, v := range keyMap {
		keys[i] = v
		i++
	}

	user := User{Uid: uid, Keys: keys}
	return user, nil
}

func (store *LocalUserStore) setup() error {
	fmt.Fprintf(os.Stderr, "User Store Init [%s]\n\n", store.Path)
	dir, err := os.Open(store.Path)

	if err != nil {
		return err
	}

	entires, err := dir.Readdir(-1)

	if err != nil {
		return err
	}

	store.users = make(map[string]*User)
	store.key2User = make(map[string]*User)
	for _, fi := range entires {
		fmt.Fprintf(os.Stderr, "%s\n", fi.Name())

		if !fi.IsDir() {
			continue
		}

		user, err := store.loadUser(fi.Name())
		if err != nil {
			continue
		}

		store.users[user.Uid] = &user
		for _, key := range user.Keys {
			//TODO: check if fingerprint is in index already?

			fingerprint, err := key.Fingerprint()
			if err != nil {
				fmt.Fprintf(os.Stderr, "[W] fingerprint generation error: %v\n", err)
				continue
			}

			store.key2User[fingerprint] = &user
			fmt.Fprintf(os.Stderr, "[D] key2User: %s <- %q\n",
				user.Uid, fingerprint)
		}
	}

	store.secret, err = auth.ReadSharedSecret()
	return err
}

func (store *LocalUserStore) LookupUserBySSH(fingerprint string) (*User, error) {

	if user, ok := store.key2User[fingerprint]; ok {
		return user, nil
	}

	return nil, fmt.Errorf("could not find user with given fingerprint")
}

func (store *LocalUserStore) TokenForUser(uid string) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	host, err := os.Hostname()
	if err != nil {
		host = "localhost"
	}

	token.Claims = &auth.Claims{
		StandardClaims: &jwt.StandardClaims{
			Issuer:    "gin-repo@" + host,
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(time.Minute * 120).Unix(), //FIXME: hardcoded exp time
			Subject:   uid,
		},
		TokenType: "user",
	}

	str, err := token.SignedString(store.secret)

	if err != nil {
		return "", err
	}

	return str, nil
}

func (store *LocalUserStore) UserForRequest(r *http.Request) (*User, error) {
	header := r.Header.Get("Authorization")

	if header == "" {
		return nil, auth.ErrNoAuth
	} else if !strings.HasPrefix(header, "Bearer ") {
		return nil, fmt.Errorf("Invalid auth type: %q", header)
	}

	token, err := jwt.ParseWithClaims(strings.Trim(header[6:], " "), &auth.Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Wrong signing method: %v", token.Header["alg"])
		}
		return store.secret, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*auth.Claims)

	if !ok || claims.TokenType != "user" {
		return nil, fmt.Errorf("Invalid token")
	}

	user, ok := store.users[claims.Subject]

	if !ok {
		return nil, fmt.Errorf("Invalid or unknown user")
	}

	return user, nil
}

func newLocalUserStore(base string) (*LocalUserStore, error) {
	store := &LocalUserStore{Path: filepath.Join(base, "users")}

	err := store.setup()
	if err != nil {
		return nil, err
	}

	return store, nil
}
