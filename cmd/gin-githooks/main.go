package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/docopt/docopt-go"
)

type gitHook struct {
	Name     string   `json:"name,omitempty"`
	Args     []string `json:"args,omitempty"`
	RepoPath string   `json:"repopath,omitempty"`
}

func main() {
	usage := `gin githooks.
    Usage:
    gin-githooks [ARGS ...]
    gin-githooks -h | --help
    Options:
    -h --help                     Show this screen.
	
	gin hooks is supposed to be called via symbolic links from git hook dircetories (eg. ln -s githooks update).
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
		log.Panicf("Could not detect the repository. Gin exitetd with:%v", err)
	}
	log.Printf("Repo dir is%s", wd)

	hooknameToPath := map[string]string{
		"pre-receive":  "/intern/hooks/pre-receive",
		"update":       "/intern/hooks/update",
		"post-receive": "/intern/hooks/post-receive",
	}

	repoServicebaseURL := os.Getenv("GIN_REPOURI")
	if len(repoServicebaseURL) == 0 {
		repoServicebaseURL = "http://localhost"
		log.Printf("NO GIN_REPOURI Set; Falling back to: %s", repoServicebaseURL)
	}

	hookCall := strings.SplitAfter(os.Args[0], "/")
	log.Printf("Hook Name:%s", hookCall)
	url := fmt.Sprint(repoServicebaseURL, hooknameToPath[hookCall[len(hookCall)-1]])

	hookArguments := args["ARGS"].([]string)
	hookJ, hErr := json.Marshal(gitHook{hookCall[len(hookCall)-1], hookArguments, wd})
	if hErr != nil {
		log.Println(hErr)
	} else {
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(hookJ))
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("Could not contact the repo service at: %s. Error was: %v", url, err)
		}
		if resp.StatusCode == http.StatusOK {
			log.Println("The Repo Service has approved this action")
			os.Exit(0)
		}
		log.Fatalf("Could not contact the repo Service. Response was %s", resp.Status)
	}
}
