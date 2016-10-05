package store

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/G-Node/gin-repo/auth"
	"github.com/G-Node/gin-repo/ssh"
)

type GinAuthStore struct {
	URL string
}

func close(b io.ReadCloser) {
	err := b.Close()
	if err != nil {
		fmt.Println("Error during cleanup:", err)
	}
}

func (store *GinAuthStore) LookupUserBySSH(fingerprint string) (*User, error) {

	q := &url.Values{}
	q.Set("fingerprint", fingerprint)
	address := fmt.Sprintf("%s/api/keys?%s", store.URL, q.Encode())
	res, err := http.Get(address)

	if err != nil {
		return nil, err
	}
	defer close(res.Body)

	b, err := ioutil.ReadAll(res.Body)

	var acc struct {
		Login       string `json:"login"`
		Fingerprint string `json:"fingerprint"`
		Key         string `json:"key"`
	}
	err = json.Unmarshal(b, &acc)
	if err != nil {
		return nil, err
	}

	key, err := ssh.ParseKey([]byte(acc.Key))
	if err != nil {
		return nil, err
	}

	user := &User{Uid: acc.Login, Keys: []ssh.Key{key}}
	return user, nil
}

func (store *GinAuthStore) TokenForUser(uid string) (string, error) {
	return "", fmt.Errorf("Not implemented :-(")
}

func (store *GinAuthStore) UserForRequest(r *http.Request) (*User, error) {
	header := r.Header.Get("Authorization")

	if header == "" {
		return nil, auth.ErrNoAuth
	} else if !strings.HasPrefix(header, "Bearer ") {
		return nil, fmt.Errorf("Invalid auth type: %q", header)
	}

	token := strings.Trim(header[6:], " ")

	address := fmt.Sprintf("%s/oauth/validate/%s", store.URL, token)
	res, err := http.Get(address)

	if err != nil {
		return nil, err
	}
	defer close(res.Body)

	b, err := ioutil.ReadAll(res.Body)

	var acc struct {
		Login string `json:"login"`
	}

	err = json.Unmarshal(b, &acc)
	if err != nil {
		return nil, err
	}

	user := &User{Uid: acc.Login}
	return user, nil
}

func newGinAuthStore(url string) (*GinAuthStore, error) {
	store := &GinAuthStore{URL: url}
	return store, nil
}
