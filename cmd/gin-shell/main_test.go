package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestWorks(t *testing.T) {
	t.Logf("Seems working!")
}

func FindContrib() (string, error) {
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

func TestMain(m *testing.M) {
	flag.Parse()

	if testing.Short() {
		fmt.Fprintf(os.Stderr, "test relying on data/docker skipped\n")
		os.Exit(0)
	}

	tmpdir := os.TempDir()
	datadir := filepath.Join(tmpdir, "grd-data")
	err := os.MkdirAll(datadir, 0755)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not create tmp dir: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "[D] using : %q\n", datadir)
	contrib, err := FindContrib()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not find contrib dir!\n")
		os.Exit(1)
	}

	mkdata := filepath.Join(contrib, "mkdata.py")
	datafile := filepath.Join(contrib, "data.yml")

	cmd := exec.Command(mkdata, datafile)
	cmd.Dir = datadir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not run mkdata: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}
