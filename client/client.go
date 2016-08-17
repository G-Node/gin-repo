package client

import (
	"net/http"
	"net/url"

	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/G-Node/gin-repo/store"
	"github.com/G-Node/gin-repo/wire"
)

type Client struct {
	Address   string
	AuthToken string
	web       *http.Client
}

func NewClient(address string) *Client {
	return &Client{Address: address, web: &http.Client{}}
}

func (client *Client) Call(method string, url string, v interface{}) (*http.Response, error) {

	var body io.Reader

	if v != nil {
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("client: json error: ")
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("client: request creation failed: %v", err)
	}

	if body != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	if token := client.AuthToken; token != "" {
		req.Header.Add("Authorization", "Bearer "+token)
	}

	return client.web.Do(req)
}

func (client *Client) LookupUserByFingerprint(fingerprint string) (*store.User, error) {

	params := url.Values{}
	params.Add("key", fingerprint)
	url := fmt.Sprintf("%s/intern/user/lookup?%s", client.Address, params.Encode())

	res, err := client.Call("GET", url, nil)

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

	var user store.User
	if err = json.Unmarshal(body, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (client *Client) RepoAccess(path string, uid string) (string, bool, error) {

	query := wire.RepoAccessQuery{Path: path, User: uid}
	url := fmt.Sprintf("%s/intern/repos/access", client.Address)

	res, err := client.Call("POST", url, &query)
	if err != nil {
		return "", false, err
	} else if status := res.StatusCode; status != 200 {
		return "", false, fmt.Errorf("Server returned non-OK status: %d", status)
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return "", false, err
	}

	var info wire.RepoAccessInfo
	if err = json.Unmarshal(body, &info); err != nil {
		return "", false, err
	}

	return info.Path, info.Push, nil
}
