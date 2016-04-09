package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"os/exec"

	"github.com/G-Node/gin-repo/git"
	"github.com/docopt/docopt-go"
)

func main() {
	usage := `gin git tool.

Usage:
  gin-git show-pack <pack>
  gin-git cat-file <sha1>
 
  gin-git -h | --help
  gin-git --version

Options:
  -h --help     Show this screen.
  --version     Show version.
`
	args, _ := docopt.Parse(usage, nil, true, "gin-git 0.1", false)
	//fmt.Fprintf(os.Stderr, "%#v\n", args)

	repo, err := discoverRepository()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(2)
	}

	if path, ok := args["<pack>"].(string); ok {
		showPack(repo, path)
	} else if oid, ok := args["<sha1>"].(string); ok {

		catFile(repo, oid)
	}
}

func discoverRepository() (*git.Repository, error) {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	data, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	path := strings.Trim(string(data), "\n ")
	return &git.Repository{Path: path}, nil
}

func catFile(repo *git.Repository, idstr string) {
	id, err := git.ParseSHA1(idstr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid object id: %v", err)
		os.Exit(3)
	}

	obj, err := repo.OpenObject(id)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	printObject(obj, "")
}

func printObject(obj git.Object, prefix string) {

	switch obj := obj.(type) {
	case *git.Commit:
		fmt.Printf("Commit [%v]\n", obj.Size())
		fmt.Printf("%s └┬─ tree:      %s\n", prefix, obj.Tree)
		fmt.Printf("%s  ├─ parent:    %s\n", prefix, obj.Parent)
		fmt.Printf("%s  ├─ author:    %s\n", prefix, obj.Author)
		fmt.Printf("%s  ├─ committer: %s\n", prefix, obj.Committer)
		fmt.Printf("%s  └─ message:   [%.40s...]\n", prefix, obj.Message)
	case *git.Tree:
		fmt.Printf("Tree [%v]\n", obj.Size())

		for obj.Next() {
			entry := obj.Entry()
			fmt.Printf("%s ├─ %08o %-7s %s %s\n", prefix, entry.Mode, entry.Type, entry.ID, entry.Name)
		}

		if err := obj.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "%sERROR: %v", prefix, err)
		}
	case *git.Blob:
		fmt.Printf("Blob [%v]\n", obj.Size())
		_, err := io.Copy(os.Stdout, obj)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sERROR: %v", prefix, err)
		}

	case *git.Tag:
		fmt.Printf("Tag [%v]\n", obj.Size())
		fmt.Printf("%s └┬─ object:    %s\n", prefix, obj.Object)
		fmt.Printf("%s  ├─ type:      %v\n", prefix, obj.ObjType)
		fmt.Printf("%s  ├─ tagger:    %s\n", prefix, obj.Tagger)
		fmt.Printf("%s  └─ message:   [%.40s...]\n", prefix, obj.Message)

	default:
		fmt.Printf("%s%v [%v]\n", prefix, obj.Type(), obj.Size())
	}

}

func showPack(repo *git.Repository, packid string) {
	if !strings.HasPrefix(packid, "pack-") {
		packid = "pack-" + packid
	}

	path := filepath.Join(repo.Path, "objects", "pack", packid)
	pack, err := git.OpenPack(path)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for i := byte(0); i < 255; i++ {
		lead, prefix := "├─", "│"
		if i == 254 {
			lead, prefix = "└─", " "
		}
		fmt.Printf("%s[%02x]\n", lead, i)

		var oid git.SHA1

		s, e := pack.Index.FO.Bounds(i)
		for k := s; k < e; k++ {
			lead := "├─"
			pf := prefix + " │"
			if e-k < 2 {
				lead = "└─┬"
				pf = prefix + "  "
			}

			fmt.Printf("%s %s", prefix, lead)
			err := pack.Index.ReadSHA1(&oid, k)
			if err != nil {
				fmt.Printf(" ERROR: %v\n", err)
				continue
			}

			fmt.Printf("%s\n", oid)

			off, err := pack.Index.ReadOffset(k)
			if err != nil {
				fmt.Printf(" ERROR: %v\n", err)
				continue
			}

			obj, err := pack.Data.ReadPackObject(off)
			if err != nil {
				fmt.Printf(" ERROR: %v\n", err)
				continue
			}

			switch c := obj.(type) {

			case *git.Commit:
				fmt.Printf("%s └─", pf)
				printObject(obj, pf+"  ")
			case *git.DeltaOfs:
				fmt.Printf("%s └─Delta OFS [%d, %d, %v]\n", pf, k, off, obj.Size())
				fmt.Printf("%s   └─ offset: %v\n", pf, c.Offset)

			case *git.DeltaRef:
				fmt.Printf("%s └─Delta Ref [%d, %d, %v]\n", pf, k, off, obj.Size())
				fmt.Printf("%s   └─ ref: %v\n", pf, c.Base)

			default:
				fmt.Printf("%s └─ %s %d, %d, [%d]\n", pf, obj.Type(), k, off, obj.Size())

			}
		}

	}

}
