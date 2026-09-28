package main

import (
	"errors"
	"fmt"
	"log"
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
		return nil, err
	}
	return &Repo{repo: r, branch: head.Name()}, nil
}

// update fetches the remote, makes the local copy match it exactly and
// reports whether any changed file is under watchPath.
func (r *Repo) update(watchPath string) (bool, error) {
	err := r.repo.Fetch(&git.FetchOptions{Force: true})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return false, fmt.Errorf("fetch: %w", err)
	}

	head, err := r.repo.Head()
	if err != nil {
		return false, err
	}
	remote, err := r.repo.Reference(plumbing.NewRemoteReferenceName("origin", r.branch.Short()), true)
	if err != nil {
		return false, err
	}
	if head.Hash() == remote.Hash() {
		return false, nil
	}

	changed, err := r.changedPaths(head.Hash(), remote.Hash())
	if err != nil {
		return false, err
	}

	wt, err := r.repo.Worktree()
	if err != nil {
		return false, err
	}
	if err := wt.Reset(&git.ResetOptions{Commit: remote.Hash(), Mode: git.HardReset}); err != nil {
		return false, fmt.Errorf("reset: %w", err)
	}
	if err := wt.Clean(&git.CleanOptions{Dir: true}); err != nil {
		return false, fmt.Errorf("clean: %w", err)
	}

	touched := false
	for _, p := range changed {
		if watchPath == "." || p == watchPath || strings.HasPrefix(p, watchPath+"/") {
			touched = true
			break
		}
	}
	log.Printf("updated %s -> %s, watch path touched: %t", head.Hash(), remote.Hash(), touched)
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
		return nil, err
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
		return nil, err
	}
	return commit.Tree()
}
