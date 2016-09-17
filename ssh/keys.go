package ssh

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

// Key represents an SSH public key
type Key struct {
	Type    string
	Keysize int
	Comment string
	Keydata []byte
}

// ReadKeysInDir lists ssh public keys in a dir
func ReadKeysInDir(dir string) map[string]Key {
	files, _ := ioutil.ReadDir(dir)
	keys := make(map[string]Key)

	for _, f := range files {
		name := f.Name()
		path := filepath.Join(dir, name)
		ext := filepath.Ext(name)

		if !strings.HasSuffix(ext, "pub") {
			continue
		}

		var abspath string
		var err error

		if abspath, err = filepath.Abs(path); err != nil {
			abspath = path
		}

		data, err := ioutil.ReadFile(abspath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[W] Skipping %s (%v)\n", name, err)
			continue
		}

		key, err := ParseKey(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[W] Skipping %s, parse error: %v", name, err)
			continue
		}

		fingerprint, err := key.Fingerprint()

		if err != nil {
			fmt.Fprintf(os.Stderr, "[W] Skipping %s, fingerprint error: %v", name, err)
			continue
		}

		keys[fingerprint] = key
	}

	return keys
}

//ParseKey parses the public key data.
//The fingerprint is calculated via SHA256
func ParseKey(data []byte) (Key, error) {

	pub, comment, _, _, err := ssh.ParseAuthorizedKey(data)
	if err != nil {
		return Key{}, err
	}

	keydata := pub.Marshal()

	return Key{
		Type:    pub.Type(),
		Keysize: len(keydata),
		Comment: comment,
		Keydata: keydata,
	}, nil

}

//MarshalAuthorizedKey creates a string representation that can be used
//in an authorized_keys file.
func (key Key) MarshalAuthorizedKey() []byte {
	data := &bytes.Buffer{}
	data.WriteString(key.Type)
	data.WriteByte(' ')
	e := base64.NewEncoder(base64.StdEncoding, data)
	e.Write(key.Keydata)
	e.Close()
	data.WriteByte('\n')
	return data.Bytes()
}

//Fingerprint returns the SH256 has of the fingerprint
//base64 url encoded without a "SHA256:" prefix to be
//compatible with gin-auth
func (key Key) Fingerprint() (string, error) {
	sha := sha256.New()
	_, err := sha.Write(key.Keydata)
	if err != nil {
		return "", err
	}

	fingerprint := base64.RawURLEncoding.EncodeToString(sha.Sum(nil))
	return fingerprint, nil
}
