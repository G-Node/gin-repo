package main

import (
	"bytes"
	"flag"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/fsouza/go-dockerclient"
)

var aliceKey []byte

func TestSSHLogin(t *testing.T) {

	signer, err := ssh.ParsePrivateKey(aliceKey)
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

func BailOut(dkr *docker.Client, container string, res int) {
	if err := dkr.RemoveContainer(docker.RemoveContainerOptions{
		ID:    container,
		Force: true,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "cannot remove container: %s", err)
	}

	os.Exit(res)
}

func ExtractPrivateKey(dkr *docker.Client, container string, user string) ([]byte, error) {
	path := fmt.Sprintf("/data/users/%[1]s/%[1]s.ssh.key", user)
	exec, err := dkr.CreateExec(docker.CreateExecOptions{
		Cmd:       []string{"cat", path},
		Container: container,
		//Tty:       false,

		AttachStdout: true,
		AttachStderr: true,
	})

	if err != nil {
		return nil, fmt.Errorf("could not create exec: %v\n", err)
	}

	var o bytes.Buffer
	var e bytes.Buffer

	err = dkr.StartExec(exec.ID, docker.StartExecOptions{
		OutputStream: &o,
		ErrorStream:  &e,
		//Tty:          false,
	})

	if err != nil {
		return nil, fmt.Errorf("could not start exec container: %v\n", err)
	}

	return o.Bytes(), nil
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
		BailOut(dkr, c.ID, 1)
	}

	done = time.Now().Add(5 * time.Second)
	for time.Now().Before(done) {
		c, err = dkr.InspectContainer(c.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot inspect docker: %s", err)
			BailOut(dkr, c.ID, 1)
		} else if c.State.Running {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if !c.State.Running {
		fmt.Fprintf(os.Stderr, "Timeout while waiting for container to start: %v\n", c.State)
		BailOut(dkr, c.ID, 1)
	}

	//wait until the container is up and can serve http request
	//i.e. the data creation script is done in the container
	hcl := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequest("GET", "http://localhost:8082", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not create http request: %v\n", err)
		os.Exit(1)
	}

	done = time.Now().Add(20 * time.Second)
	for time.Now().Before(done) {
		_, err = hcl.Do(req)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	aliceKey, err = ExtractPrivateKey(dkr, c.ID, "alice")
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get priv key for alice: %v", err)
		BailOut(dkr, c.ID, 1)
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

	BailOut(dkr, c.ID, 0)
}
