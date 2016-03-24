package ssh

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Key represents an SSH public key
type Key struct {
	Fingerprint string
	Keysize     string
	Comment     string
	Keydata     []byte
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

		out, err := exec.Command("ssh-keygen", "-l", "-f"+abspath).CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[W] Skipping %s (%v:%s)\n", name, err, string(out))
			continue
		}

		components := strings.Split(string(out), " ")
		if len(components) != 4 {
			fmt.Fprintf(os.Stderr, "[W] Skipping %s", name)
			continue
		}

		key := Key{Fingerprint: components[1],
			Keysize: components[0],
			Comment: components[2]}

		data, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[W] Skipping %s", name)
			continue
		}

		key.Keydata = data
		keys[key.Fingerprint] = key
	}

	return keys
}
