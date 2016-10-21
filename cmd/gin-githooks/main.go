package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/G-Node/gin-repo/auth"
	"github.com/G-Node/gin-repo/client"
	"github.com/G-Node/gin-repo/wire"
	"github.com/docopt/docopt-go"
	"io"
)

func makeServiceToken() (string, error) {

	secret, err := auth.ReadSharedSecret()

	if err != nil {
		return "", fmt.Errorf("could not load secret: %v", err)
	}

	return auth.MakeServiceToken(secret)
}

var (
	hooknameToPath = map[string]string{
		"pre-receive":  "/intern/hooks/fire",
		"update":       "/intern/hooks/fire",
		"post-receive": "/intern/hooks/fire",
	}
)

func getHook(args map[string]interface{}, inStream io.Reader) wire.GitHook {
	wd, err := os.Getwd()
	if err != nil {
		log.Panicf("Could not detect repository. Gin exited with:%v", err)
	}
	log.Printf("Repo directory is%s", wd)
	hookCall := strings.SplitAfter(os.Args[0], "/")
	log.Printf("Hook Name:%s", hookCall)

	//Reading Stdin to extract Ref info
	scanner := bufio.NewScanner(inStream)
	UpdatedRefs := []wire.RefLine{}
	for scanner.Scan() {
		if splArgs := strings.Split(scanner.Text(), " "); len(splArgs) == 3 {
			UpdatedRefs = append(UpdatedRefs, wire.RefLine{splArgs[0], splArgs[1], splArgs[2]})
		} else {
			log.Println("Sdtin not properly formatted for hook")
		}
	}
	hookArguments := args["ARGS"].([]string)
	return wire.GitHook{hookCall[len(hookCall)-1], hookArguments, wd, UpdatedRefs}
}

func sendHook(repoClient *client.Client, url string, hook wire.GitHook) bool {

	token, err := makeServiceToken()
	repoClient.AuthToken = token
	resp, err := repoClient.Call("POST", url, hook)
	if err != nil {
		log.Printf("Could not contact the repo service at: %s. Error was: %v", url, err)
		return false
	}
	if resp.StatusCode == http.StatusOK {
		log.Println("The Repo Service has approved this action")
		return true
	}
	log.Printf("Could not contact the repo Service. Response was %s", resp.Status)
	return false
}

func main() {
	usage := `gin githooks.
    Usage:
    gin-githooks [ARGS ...]
    gin-githooks -h | --help
    Options:
    -h --help                     Show this screen.


	gin hooks is called via symbolic links from git hook dircetories (eg. ln -s githooks update).
	It will process the hook type and optional provided arguments. The collected information is then passed to the repo service
	availible either locally or set by ENV(GIN_REPO_URL)
	It terminates with 0 in case everything went fine.
`
	args, err := docopt.Parse(usage, nil, true, "gin githooks 0.1", false)
	if err != nil {
		log.Fatalf("Error while parsing cmd line: %v\n", err)
		// log.Fatal is equivalent to Print() followed by a call to os.Exit(1).
		// needs to replaced once logging is done differently
	}
	log.SetOutput(os.Stderr)

	repoServiceBaseURL := os.Getenv("GIN_REPO_URL")
	if len(repoServiceBaseURL) == 0 {
		repoServiceBaseURL = "http://localhost:8082"
		log.Printf("NO GIN_REPO_URL Set; Falling back to: %s", repoServiceBaseURL)
	}

	hook := getHook(args, os.Stdin)

	url := fmt.Sprint(repoServiceBaseURL, hooknameToPath[hook.Name])
	repoClient := client.NewClient(repoServiceBaseURL)

	if ok := sendHook(repoClient, url, hook); !ok {
		log.Fatal("Hook was not accepted")
		// log.Fatal is equivalent to Print() followed by a call to os.Exit(1).
		// needs to replaced once logging is done differently
	} else {
		os.Exit(0)
	}
}
