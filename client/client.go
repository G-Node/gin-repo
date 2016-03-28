package client

import (
	"net/http"
	"net/url"

	"bytes"
	"encoding/json"
	"fmt"
	. "github.com/G-Node/gin-repo/common"
	"github.com/G-Node/gin-repo/wire"
	"io/ioutil"
)

type Client struct {
	address string
	web     *http.Client
}

func NewClient(address string) *Client {
	return &Client{address: address, web: &http.Client{}}
}

func (client *Client) LookupUserByFingerprint(fingerprint string) (*User, error) {

	params := url.Values{}
	params.Add("key", fingerprint)
	url := fmt.Sprintf("%s/intern/user/lookup?%s", client.address, params.Encode())

	res, err := client.web.Get(url)
	if err != nil {
		return nil, err
	} else if status := res.StatusCode; status != 200 {
		return nil, fmt.Errorf("Server returned non-OK status: %d", status)
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return nil, err
	}

	var user User
	if err = json.Unmarshal(body, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (client *Client) RepoAccess(path string, uid string, method string) (string, error) {

	query := wire.RepoAccessQuery{Path: path, User: uid, Method: method}

	data, err := json.Marshal(&query)

	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/intern/repos/access", client.address)
	res, err := client.web.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return "", err
	} else if status := res.StatusCode; status != 200 {
		return "", fmt.Errorf("Server returned non-OK status: %d", status)
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return "", err
	}

	return string(body), nil
}
