package scan

import (
	"sync"

	ts "github.com/odvcencio/gotreesitter"
)

// gotreesitter's Node.Parent() isn't reliable: for some children (e.g. a
// JS else_clause's statement_block) it returns a detached supertype node
// ("statement") whose own Parent() is nil, so an upward climb stops
// short of the enclosing function. Children() is reliable, so each parsed
// tree's real parent links are recorded top-down and every upward climb
// goes through parentOf instead.
var (
	parentMu    sync.Mutex
	parentIndex map[*ts.Node]*ts.Node
)

// indexParents records root's tree's parent links. The caller holds
// parentMu for as long as the index is in use (see Scan).
func indexParents(root *ts.Node) {
	parentIndex = map[*ts.Node]*ts.Node{}
	var rec func(n *ts.Node)
	rec = func(n *ts.Node) {
		for _, c := range n.Children() {
			if c != nil {
				parentIndex[c] = n
				rec(c)
			}
		}
	}
	rec(root)
}

// parentOf returns n's parent in the indexed tree (nil for its root), or
// n.Parent() when no tree is indexed.
func parentOf(n *ts.Node) *ts.Node {
	if n == nil {
		return nil
	}
	if parentIndex == nil {
		return n.Parent()
	}
	return parentIndex[n]
}
