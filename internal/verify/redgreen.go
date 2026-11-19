package verify

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aschepis/maslow/internal/evidence"
	"github.com/aschepis/maslow/internal/gitrepo"
	"github.com/aschepis/maslow/internal/runner"
	"github.com/aschepis/maslow/internal/spec"
)

// baseTree is a throwaway checkout of the base commit for overlay contracts to run in.
type baseTree struct {
	sha string
	// tmp holds the worktree; dir is the worktree path matching the current
	// directory's place in the repository.
	tmp, root, dir string
	err            error
}

// checkoutBase adds a worktree at sha. A failure is kept on the tree rather
// than returned, so each overlay contract reports it as an error.
func checkoutBase(sha string) *baseTree {
	tree := &baseTree{sha: sha}
	prefix, err := gitrepo.Prefix("")
	if err != nil {
		tree.err = err
		return tree
	}
	tree.tmp, err = os.MkdirTemp("", "maslow-base-")
	if err != nil {
		tree.err = err
		return tree
	}
	tree.root = filepath.Join(tree.tmp, "tree")
	if err := gitrepo.AddWorktree("", tree.root, sha); err != nil {
		tree.err = err
		return tree
	}
	tree.dir = filepath.Join(tree.root, prefix)
	return tree
}

func (b *baseTree) remove() {
	if b.root != "" && b.err == nil {
		_ = gitrepo.RemoveWorktree("", b.root)
	}
	if b.tmp != "" {
		_ = os.RemoveAll(b.tmp)
	}
}

// judgeRedGreen runs contracts on the base tree and folds each base result
// into the matching head result.
func judgeRedGreen(contracts []spec.Contract, head []evidence.ContractResult, base *baseTree) {
	var baseResults []evidence.ContractResult
	if base.err == nil {
		baseResults = runner.RunContractsWith(contracts, runner.ContractOptions{Dir: base.dir})
	}
	for i := range head {
		run := &evidence.BaseRun{SHA: base.sha}
		if base.err != nil {
			run.Status = "error"
			run.Error = fmt.Sprintf("checking out base %s: %v", base.sha, base.err)
		} else {
			b := baseResults[i]
			run.Status, run.Error, run.FailedStep = b.Status, b.Error, b.FailedStep
		}
		head[i].HeadStatus = head[i].Status
		head[i].Base = run
		head[i].Status = redGreenStatus(head[i].Status, run.Status)
	}
}

// redGreenStatus passes a contract only if it fails on the base and passes on
// the head. A base run that could not be judged cannot prove the contract red.
func redGreenStatus(head, base string) string {
	switch {
	case head != "pass":
		return head
	case base == "pass":
		return evidence.StatusNotRed
	case base == "fail":
		return "pass"
	default:
		return "error"
	}
}
