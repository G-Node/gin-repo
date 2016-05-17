package git

import (
	"container/heap"
	"fmt"
)

type NodeFlag uint32

const (
	NodeColorRed NodeFlag = (1 << iota)
	NodeColorGreen
	NodeColorBlue

	NodeColorYellow = NodeColorRed | NodeColorGreen
	NodeColorWhite  = NodeColorRed | NodeColorGreen | NodeColorBlue

	NodeFlagSeen = 1 << 4
)

type CommitNode struct {
	commit  *Commit
	parents []*CommitNode
	Flags   NodeFlag
	ID      SHA1
}

func (n *CommitNode) Parents() []*CommitNode {
	return n.parents
}

type CommitGraph struct {
	tips []*CommitNode

	commits map[SHA1]*CommitNode
	repo    *Repository
}

func NewCommitGraph(repo *Repository) *CommitGraph {
	return &CommitGraph{repo: repo, commits: make(map[SHA1]*CommitNode, 0)}
}

func (c *CommitGraph) openObject(oid SHA1) (*CommitNode, error) {
	if node, ok := c.commits[oid]; ok {
		return node, nil
	}

	obj, err := c.repo.OpenObject(oid)

	if err != nil {
		return nil, err
	}

	commit, ok := obj.(*Commit)
	if !ok {
		return nil, fmt.Errorf("object [%s] not of type commit", oid)
	}

	node := &CommitNode{commit: commit, ID: oid}
	c.commits[oid] = node

	return node, nil
}

func (c *CommitGraph) AddTip(oid SHA1) (*CommitNode, error) {
	node, err := c.openObject(oid)

	if err != nil {
		return nil, err
	}

	c.tips = append(c.tips, node)
	return node, nil
}

//youngestFirst is a priority queue implemented via a 'container/heap'
//the latter is a min-heap, which nicely aligns with times in epoch
type youngestFirst []*CommitNode

func (y youngestFirst) Len() int {
	return len(y)
}

func (y youngestFirst) Less(i, j int) bool {
	// true -> i before j
	//      -> i.Date() after j.Date

	ic, jc := y[i], y[j]

	return ic.commit.Date().After(jc.commit.Date())
}

func (y youngestFirst) Swap(i, j int) {
	y[i], y[j] = y[j], y[i]
}

func (y *youngestFirst) Push(x interface{}) {
	*y = append(*y, x.(*CommitNode))
}

func (y *youngestFirst) Pop() interface{} {
	n := len(*y) - 1
	o := *y
	r := o[n]
	*y = o[0:n]
	return r
}

func (y youngestFirst) notAllWhite() bool {
	for _, node := range y {
		if node.Flags&NodeColorWhite != NodeColorWhite {
			return true
		}
	}
	return false
}

func (c *CommitGraph) PaintDownToCommon() error {

	//initialize the priority queue with the tips
	pq := make(youngestFirst, len(c.tips))

	for i, node := range c.tips {
		pq[i] = node
	}

	heap.Init(&pq)

	for pq.notAllWhite() {
		node := heap.Pop(&pq).(*CommitNode)

		flags := node.Flags
		if flags&NodeColorWhite == NodeColorYellow {
			flags |= NodeColorBlue
		}

		if len(node.parents) != len(node.commit.Parent) {
			node.parents = make([]*CommitNode, len(node.commit.Parent))
			for i, parent := range node.commit.Parent {
				var err error
				node.parents[i], err = c.openObject(parent)
				if err != nil {
					return err
				}
			}
		}

		for _, parent := range node.parents {
			if parent.Flags == flags {
				continue
			}

			parent.Flags |= flags
			heap.Push(&pq, parent)
		}
	}

	return nil
}
