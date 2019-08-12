package git

import "gopkg.in/src-d/go-git.v4"

type Operations struct{}
type Interface interface {
	Clone(string, string) error
	Pull(string) error
}

func (g Operations) Clone(appCachePath, app string) error {
	_, err := git.PlainClone(appCachePath, false, &git.CloneOptions{
		URL: app,
	})
	if err != nil {
		return err
	}

	return nil
}

func (g Operations) Pull(appCachePath string) error {
	repo, err := git.PlainOpen(appCachePath)
	if err != nil {
		return err
	}

	repo.Fetch(&git.FetchOptions{
		Force: true,
	})

	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}

	err = worktree.Reset(&git.ResetOptions{
		Mode: git.HardReset,
	})
	if err != nil {
		return nil
	}

	return nil
}
