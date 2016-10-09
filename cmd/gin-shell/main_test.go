package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/fsouza/go-dockerclient"
)

func TestSSHLogin(t *testing.T) {
	key, err := ioutil.ReadFile("/tmp/grd-data/users/alice/alice.ssh.key")
	if err != nil {
		t.Fatalf("unable to read private key: %v", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		t.Fatalf("unable to parse private key: %v", err)
	}

	config := &ssh.ClientConfig{
		User: "git",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
	}

	done := time.Now().Add(5 * time.Second)
	var client *ssh.Client
	for time.Now().Before(done) {
		client, err = ssh.Dial("tcp", "localhost:2345", config)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if err != nil {
		t.Fatalf("failed to dial: " + err.Error())
	}

	session, err := client.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session: " + err.Error())
	}
	defer session.Close()

	// Once a Session is created, you can execute a single command on
	// the remote side using the Run method.
	var o bytes.Buffer
	var e bytes.Buffer
	session.Stdout = &o
	session.Stderr = &e

	err = session.Run("/usr/bin/whoami")
	if err == nil {
		t.Errorf("stderr: [%q], stdout: [%q]", e.String(), o.String())
		t.Fatalf("successfully executed. Should not be possible!")
	}

	fmt.Fprintf(os.Stderr, "SSH: %q\n", e.String())
}

func TestMain(m *testing.M) {
	flag.Parse()

	if testing.Short() {
		fmt.Fprintf(os.Stderr, "test relying on data/docker skipped\n")
		os.Exit(0)
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
		//Binds: []string{fmt.Sprintf("%s:/data", datadir)},
	}

	opts := docker.CreateContainerOptions{
		Config: &docker.Config{
			Image:        "gin-repod",
			ExposedPorts: map[docker.Port]struct{}{"22/tcp": {}, "8082/tcp": {}},
			Env:          []string{"GRD_GENDATA=1"},
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

	if !c.State.Running {
		fmt.Fprintf(os.Stderr, "Timeout while waiting for container to start: %v\n", c.State)
		goto remove_container
	}

	//Now lets get on with the tests
	res = m.Run()

	if res != 0 {
		fmt.Fprintf(os.Stdout, "Container logs:\n")
		err = dkr.Logs(docker.LogsOptions{
			Container:    c.ID,
			OutputStream: os.Stdout,
			ErrorStream:  os.Stderr,
			Stdout:       true,
			Stderr:       true,

			Follow: false,
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "could not get container logs: %s", err)
		}
		fmt.Fprintf(os.Stdout, "---- EOF ----\n")
	}

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
