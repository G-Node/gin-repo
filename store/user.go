package store

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/G-Node/gin-repo/auth"
	. "github.com/G-Node/gin-repo/common"
	"github.com/G-Node/gin-repo/ssh"
	"github.com/dgrijalva/jwt-go"
)

type UserStore struct {
	Path string

	users    map[string]*User
	key2User map[string]*User

	secret []byte
}

func (store *UserStore) loadUser(uid string) (User, error) {
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

func (store *UserStore) Setup() error {
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
			store.key2User[key.Fingerprint] = &user
			fmt.Fprintf(os.Stderr, "[D] key2User: %s <- %q\n",
				user.Uid, key.Fingerprint)
		}
	}

	store.secret, err = auth.ReadSharedSecret()
	return err
}

func (store *UserStore) LookupUserBySSH(fingerprint string) (*User, error) {

	if user, ok := store.key2User[fingerprint]; ok {
		return user, nil
	}

	return nil, fmt.Errorf("could not find user with given fingerprint")
}

func (store *UserStore) TokenForUser(uid string) (string, error) {
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

func NewUserStore(base string) (*UserStore, error) {
	store := &UserStore{Path: filepath.Join(base, "users")}

	err := store.Setup()
	if err != nil {
		return nil, err
	}

	return store, nil
}
