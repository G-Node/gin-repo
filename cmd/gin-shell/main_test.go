package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsouza/go-dockerclient"
)

func TestWorks(t *testing.T) {
	time.Sleep(10 * time.Second)
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

	tmpdir := "/tmp"
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

	// now the container fun!
	dkr, err := docker.NewClientFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not connect to docker: %v\n", err)
		os.Exit(1)
	}

	hcfg := &docker.HostConfig{
		PortBindings: map[docker.Port][]docker.PortBinding{
			"22/tcp": {
				{HostIP: "0.0.0.0", HostPort: "2345"},
			},
			"8082/tcp": {
				{HostIP: "0.0.0.0", HostPort: "8082"},
			},
		},
		Binds: []string{fmt.Sprintf("%s:/data", datadir)},
	}

	opts := docker.CreateContainerOptions{
		Config: &docker.Config{
			Image:        "gin-repod",
			ExposedPorts: map[docker.Port]struct{}{"22/tcp": {}, "8082/tcp": {}},
			//Volumes:      map[string]struct{}{"/data": {}},
		},
		HostConfig: hcfg,
	}

	c, err := dkr.CreateContainer(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not create container: %v\n", err)
		os.Exit(1)
	}

	res := 1
	var done time.Time
	fmt.Fprintf(os.Stderr, "Container id: %q\n", c.ID)

	err = dkr.StartContainer(c.ID, hcfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot start Docker container: %s", err)
		goto remove_container
	}

	done = time.Now().Add(5 * time.Second)
	for time.Now().Before(done) {
		c, err = dkr.InspectContainer(c.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot inspect docker: %s", err)
			goto remove_container
		} else if c.State.Running {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Fprintf(os.Stderr, "Container: %v\n", c)

	if !c.State.Running {
		fmt.Fprintf(os.Stderr, "Timeout while waiting for container to start: %v\n", c.State)
		goto remove_container
	}

	res = m.Run()

	//tear down
	if err = dkr.StopContainer(c.ID, 2000); err != nil {
		fmt.Fprintf(os.Stderr, "cannot stop container: %s", err)
	}

remove_container:
	if err := dkr.RemoveContainer(docker.RemoveContainerOptions{
		ID:    c.ID,
		Force: true,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "cannot remove container: %s", err)
	}

	os.Exit(res)
}
