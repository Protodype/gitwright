package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Repo struct {
	repo   *git.Repository
	branch plumbing.ReferenceName
}

// clone clones the remote default branch into dir.
func clone(url, dir string) (*Repo, error) {
	r, err := git.PlainClone(dir, false, &git.CloneOptions{URL: url})
	if err != nil {
		return nil, err
	}
	head, err := r.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve HEAD of cloned repository: %w", err)
	}
	return &Repo{repo: r, branch: head.Name()}, nil
}

// update fetches the remote, makes the local copy match it exactly and
// reports whether any changed file is under watchPath.
func (r *Repo) update(watchPath string) (bool, error) {
	err := r.repo.Fetch(&git.FetchOptions{Force: true})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return false, fmt.Errorf("failed to fetch from remote: %w", err)
	}

	head, err := r.repo.Head()
	if err != nil {
		return false, fmt.Errorf("failed to resolve local HEAD: %w", err)
	}
	remote, err := r.repo.Reference(plumbing.NewRemoteReferenceName("origin", r.branch.Short()), true)
	if err != nil {
		return false, fmt.Errorf("failed to resolve remote branch origin/%s: %w", r.branch.Short(), err)
	}
	if head.Hash() == remote.Hash() {
		return false, nil
	}

	changed, err := r.changedPaths(head.Hash(), remote.Hash())
	if err != nil {
		return false, fmt.Errorf("failed to list files changed between %s and %s: %w", head.Hash(), remote.Hash(), err)
	}

	wt, err := r.repo.Worktree()
	if err != nil {
		return false, fmt.Errorf("failed to open worktree: %w", err)
	}
	if err := wt.Reset(&git.ResetOptions{Commit: remote.Hash(), Mode: git.HardReset}); err != nil {
		return false, fmt.Errorf("failed to reset worktree to %s: %w", remote.Hash(), err)
	}
	if err := wt.Clean(&git.CleanOptions{Dir: true}); err != nil {
		return false, fmt.Errorf("failed to remove untracked files from worktree: %w", err)
	}

	touched := false
	for _, p := range changed {
		if watchPath == "." || p == watchPath || strings.HasPrefix(p, watchPath+"/") {
			touched = true
			break
		}
	}
	return touched, nil
}

func (r *Repo) changedPaths(from, to plumbing.Hash) ([]string, error) {
	fromTree, err := r.tree(from)
	if err != nil {
		return nil, err
	}
	toTree, err := r.tree(to)
	if err != nil {
		return nil, err
	}
	changes, err := object.DiffTree(fromTree, toTree)
	if err != nil {
		return nil, fmt.Errorf("failed to diff trees: %w", err)
	}
	var paths []string
	for _, c := range changes {
		if c.From.Name != "" {
			paths = append(paths, c.From.Name)
		}
		if c.To.Name != "" {
			paths = append(paths, c.To.Name)
		}
	}
	return paths, nil
}

func (r *Repo) tree(hash plumbing.Hash) (*object.Tree, error) {
	commit, err := r.repo.CommitObject(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to read commit %s: %w", hash, err)
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to read tree of commit %s: %w", hash, err)
	}
	return tree, nil
}
