package git

import (
	"gopkg.in/src-d/go-git.v4"
)

// Operations contains the behaviors of the internal git package
type Operations struct{}

// Interface defines the behaviors of the internal git package
type Interface interface {
	Clone(string, string) error
	Pull(string) error
}

// Clone clones a repo locally, returns an error if it fails
func (g Operations) Clone(appCachePath, app string) error {
	_, err := git.PlainClone(appCachePath, false, &git.CloneOptions{
		URL: app,
	})
	if err != nil {
		return err
	}

	return nil
}

// Pull will update the changes for the provided repo or fails
func (g Operations) Pull(appCachePath string) error {
	repo, err := git.PlainOpen(appCachePath)
	if err != nil {
		return err
	}

	err = repo.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		Force:      true,
	})
	if err != nil {
		switch e := err.Error(); e {
		case git.NoErrAlreadyUpToDate.Error():
			break
		default:
			return err
		}
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}

	err = worktree.Reset(&git.ResetOptions{
		Mode: git.HardReset,
	})
	if err != nil {
		return err
	}

	err = worktree.Pull(&git.PullOptions{
		Force: true,
	})
	if err != nil {
		return err
	}

	return nil
}
