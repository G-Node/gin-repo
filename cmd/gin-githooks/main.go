package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"bufio"
	"github.com/G-Node/gin-repo/auth"
	"github.com/G-Node/gin-repo/client"
	"github.com/G-Node/gin-repo/wire"
	"github.com/docopt/docopt-go"
)

func makeServiceToken() (string, error) {

	secret, err := auth.ReadSharedSecret()

	if err != nil {
		return "", fmt.Errorf("could not load secret: %v", err)
	}

	return auth.MakeServiceToken(secret)
}

const (
	hooknameToPath = map[string]string{
		"pre-receive":  "/intern/hooks/fire",
		"update":       "/intern/hooks/fire",
		"post-receive": "/intern/hooks/fire",
	}
)

func main() {
	usage := `gin githooks.
    Usage:
    gin-githooks [ARGS ...]
    gin-githooks -h | --help
    Options:
    -h --help                     Show this screen.


	gin hooks is called via symbolic links from git hook dircetories (eg. ln -s githooks update).
	It will process the hook type and optional provided arguments. The collected information is then passed to the repo service
	availible either locally or set by ENV(GIN_REPOURI)
	It terminates with 0 in case everything went fine.
`
	args, err := docopt.Parse(usage, nil, true, "gin githooks 0.1", false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while parsing cmd line: %v\n", err)
		os.Exit(-1)
	}
	log.SetOutput(os.Stderr)

	wd, err := os.Getwd()
	if err != nil {
		log.Panicf("Could not detect the repository. Gin exited with:%v", err)
	}
	log.Printf("Repo directory is%s", wd)

	repoServiceBaseURL := os.Getenv("GIN_REPOURI")
	if len(repoServiceBaseURL) == 0 {
		repoServiceBaseURL = "http://localhost"
		log.Printf("NO GIN_REPOURI Set; Falling back to: %s", repoServiceBaseURL)
	}

	hookCall := strings.SplitAfter(os.Args[0], "/")
	log.Printf("Hook Name:%s", hookCall)

	//Reading Stdin to extract Ref info
	scanner := bufio.NewScanner(os.Stdin)
	UpdatedRefs := []RefLine{}
	for scanner.Scan() {
		if splArgs := strings.Split(scanner.Text(), " "); len(splArgs) == 3 {
			UpdatedRefs = append(UpdatedRefs, wire.RefLine{splArgs[0], splArgs[1], splArgs[2]})
		} else {
			log.Println("Sdtin not properly formatted for hook")
		}
	}
	url := fmt.Sprint(repoServiceBaseURL, hooknameToPath[hookCall[len(hookCall)-1]])
	hookArguments := args["ARGS"].([]string)
	hook := wire.GitHook{hookCall[len(hookCall)-1], hookArguments, wd, UpdatedRefs}
	repoClient := client.NewClient(repoServiceBaseURL)
	token, err := makeServiceToken()
	repoClient.AuthToken = token
	resp, err := repoClient.Call("POST", url, hook)
	if err != nil {
		log.Fatalf("Could not contact the repo service at: %s. Error was: %v", url, err)
	}
	if resp.StatusCode == http.StatusOK {
		log.Println("The Repo Service has approved this action")
		os.Exit(0)
	}
	log.Fatalf("Could not contact the repo Service. Response was %s", resp.Status)
}
