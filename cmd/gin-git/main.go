package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/G-Node/gin-repo/git"
	"github.com/docopt/docopt-go"
)

func main() {
	usage := `gin git tool.

Usage:
  gin-git show-pack <pack>
  gin-git show-delta <pack> <sha1>
  gin-git cat-file <sha1>
  gin-git rev-parse <ref>
  gin-git graph-common <base> <ref>
 
  gin-git -h | --help
  gin-git --version

Options:
  -h --help     Show this screen.
  --version     Show version.
`
	args, _ := docopt.Parse(usage, nil, true, "gin-git 0.1", false)
	//fmt.Fprintf(os.Stderr, "%#v\n", args)

	repo, err := git.DiscoverRepository()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}

	if val, ok := args["rev-parse"].(bool); ok && val {
		revParse(repo, args["<ref>"].(string))
	} else if val, ok := args["show-pack"].(bool); ok && val {
		showPack(repo, args["<pack>"].(string))
	} else if val, ok := args["show-delta"].(bool); ok && val {
		showDelta(repo, args["<pack>"].(string), args["<sha1>"].(string))
	} else if oid, ok := args["<sha1>"].(string); ok {
		catFile(repo, oid)
	} else if val, ok := args["graph-common"].(bool); ok && val {
		graphCommon(repo, args["<base>"].(string), args["<ref>"].(string))
	}
}

func revParse(repo *git.Repository, refstr string) {
	ref, err := repo.OpenRef(refstr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(3)
	}

	id, err := ref.Resolve()
	var idstr string
	if err != nil {
		idstr = fmt.Sprintf("ERROR: %v", err)
	} else {
		idstr = fmt.Sprintf("%s", id)
	}

	fmt.Printf("%s\n", refstr)
	fmt.Printf(" └┬─ name: %s\n", ref.Name())
	fmt.Printf("  ├─ full: %s\n", ref.Fullname())
	fmt.Printf("  └─ SHA1: %s\n", idstr)
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
	obj.Close()
}

func showDelta(repo *git.Repository, packid string, idstr string) {
	oid, err := git.ParseSHA1(idstr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid object id: %v", err)
		os.Exit(3)
	}

	if !strings.HasPrefix(packid, "pack-") {
		packid = "pack-" + packid
	}

	path := filepath.Join(repo.Path, "objects", "pack", packid)
	idx, err := git.PackIndexOpen(path + ".idx")

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	obj, err := idx.OpenObject(oid)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	delta, ok := obj.(*git.Delta)
	if !ok {
		fmt.Fprintf(os.Stderr, "Object with %s is not Delta", oid)
		os.Exit(1)
	}

	pf := ""
	fmt.Printf("%s Delta [%d]\n", pf, delta.Size())

	if obj.Type() == git.ObjOFSDelta {
		fmt.Printf("%s   ├─ off: %v\n", pf, delta.BaseOff)
	} else {
		fmt.Printf("%s   ├─ ref: %v\n", pf, delta.BaseRef)
	}

	fmt.Printf("%s   ├─ source size: %v\n", pf, delta.SizeSource)
	fmt.Printf("%s   ├─ taget  size: %v\n", pf, delta.SizeTarget)
	fmt.Printf("%s   └┬─ Instructions\n", pf)

	for delta.NextOp() {
		op := delta.Op()
		switch op.Op {
		case git.DeltaOpCopy:
			fmt.Printf("%s    ├─ Copy: %d @ %d\n", pf, op.Size, op.Offset)
		case git.DeltaOpInsert:
			fmt.Printf("%s    ├─ Insert %d\n", pf, op.Size)
		}
	}

}

func printObject(obj git.Object, prefix string) {

	switch obj := obj.(type) {
	case *git.Commit:
		fmt.Printf("Commit [%v]\n", obj.Size())
		fmt.Printf("%s └┬─ tree:      %s\n", prefix, obj.Tree)
		for _, parent := range obj.Parent {
			fmt.Printf("%s  ├─ parent:    %s\n", prefix, parent)
		}
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
		fmt.Fprintf(os.Stderr, "Blob [%v]\n", obj.Size())
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
	idx, err := git.PackIndexOpen(path + ".idx")

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	data, err := idx.OpenPackFile()

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

		s, e := idx.FO.Bounds(i)
		for k := s; k < e; k++ {
			lead := "├─"
			pf := prefix + " │"
			if e-k < 2 {
				lead = "└─┬"
				pf = prefix + "  "
			}

			fmt.Printf("%s %s", prefix, lead)
			err := idx.ReadSHA1(&oid, k)
			if err != nil {
				fmt.Printf(" ERROR: %v\n", err)
				continue
			}

			fmt.Printf("%s\n", oid)

			off, err := idx.ReadOffset(k)
			if err != nil {
				fmt.Printf(" ERROR: %v\n", err)
				continue
			}

			obj, err := data.OpenObject(off)
			if err != nil {
				fmt.Printf(" ERROR: %v\n", err)
				continue
			}

			switch obj.Type() {
			case git.ObjCommit:
				fallthrough
			case git.ObjTree:
				fallthrough
			case git.ObjTag:
				fmt.Printf("%s └─", pf)
				printObject(obj, pf+"  ")
				obj.Close()
				continue
			}

			switch c := obj.(type) {

			case *git.Delta:
				fmt.Printf("%s └─Delta [%d, %d, %v]\n", pf, k, off, obj.Size())

				if obj.Type() == git.ObjOFSDelta {
					fmt.Printf("%s   └─ off: %v\n", pf, c.BaseOff)
				} else {
					fmt.Printf("%s   └─ ref: %v\n", pf, c.BaseRef)
				}
			default:
				fmt.Printf("%s └─ %s %d, %d, [%d]\n", pf, obj.Type(), k, off, obj.Size())

			}

			obj.Close()
		}

	}
}

func graphCommon(repo *git.Repository, basestr, refstr string) {
	baseid, err := git.ParseSHA1(basestr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid object id: %v", err)
		os.Exit(1)
	}

	refid, err := git.ParseSHA1(refstr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid object id: %v", err)
		os.Exit(1)
	}

	cg := git.NewCommitGraph(repo)
	base, err := cg.AddTip(baseid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not add tip: %v", err)
		os.Exit(2)
	}
	base.Flags = git.NodeColorRed

	ref, err := cg.AddTip(refid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not add tip: %v", err)
		os.Exit(2)
	}
	ref.Flags = git.NodeColorGreen

	err = cg.PaintDownToCommon()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building graph: %v", err)
		os.Exit(10)
	}

	fmt.Printf("digraph g1 {\n")
	q := []*git.CommitNode{base, ref}

	for len(q) != 0 {
		node := q[0]
		q = q[1:]

		if node.Flags&git.NodeFlagSeen != 0 {
			continue
		}

		fmt.Printf("%q [label=\"%.7[1]s (%d)\"];\n",
			node.ID, node.Flags&git.NodeColorWhite)

		node.Flags |= git.NodeFlagSeen

		for _, parent := range node.Parents() {
			q = append(q, parent)
			fmt.Printf("%q -> %q;\n", node.ID, parent.ID)
		}
	}

	fmt.Printf("}\n")
}
