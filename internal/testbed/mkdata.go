package testbed

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

//findContrib tries to find gin-repo/contrib, needs to be
//called with a current work directory below gin-repo/
func findContrib() (string, error) {
	dir, err := os.Getwd()

	if err != nil {
		return "", nil
	}

	for {
		cd := filepath.Join(dir, "contrib")
		fmt.Fprintf(os.Stderr, "checking %q\n", cd)
		_, err := os.Stat(cd)

		if err == nil {
			return cd, nil
		} else if dir == "." || dir == "/" {
			return "", fmt.Errorf("Could not find %q in hierarchy", "contrib")
		}

		dir = filepath.Dir(dir)
	}
}

//MkData creates test data the temporary directory
//under tmpdir. Returns the path to that newly
//create temporary directory
func MkData(tmpdir string) (string, error) {
	datadir := filepath.Join(tmpdir, "grd-data")
	err := os.MkdirAll(datadir, 0755)

	if err != nil {
		return "", fmt.Errorf("could not create tmp dir: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "[D] using : %q\n", datadir)
	contrib, err := findContrib()
	if err != nil {
		return "", fmt.Errorf("could not find contrib dir!\n")
	}

	mkdata := filepath.Join(contrib, "mkdata.py")
	datafile := filepath.Join(contrib, "data.yml")

	cmd := exec.Command(mkdata, datafile)
	cmd.Dir = datadir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()

	if err != nil {
		return "", fmt.Errorf("could not run mkdata: %v\n", err)
	}

	return datadir, nil
}
