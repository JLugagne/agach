package app

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func newTestGitService(t *testing.T) *GitService {
	t.Helper()
	tmpDir := t.TempDir()
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return &GitService{
		cacheDir: tmpDir,
		logger:   logger,
	}
}

func TestGitService_EnsureWorktree_Clone(t *testing.T) {
	bareDir := t.TempDir()

	bare, err := git.PlainInit(bareDir, true)
	require.NoError(t, err)

	workDir := t.TempDir()
	work, err := git.PlainInit(workDir, false)
	require.NoError(t, err)

	_, err = work.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{bareDir},
	})
	require.NoError(t, err)

	w, err := work.Worktree()
	require.NoError(t, err)

	testFile := filepath.Join(workDir, "readme.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("hello"), 0600))

	_, err = w.Add("readme.txt")
	require.NoError(t, err)

	sig := &object.Signature{
		Name:  "Test",
		Email: "test@example.com",
		When:  time.Now(),
	}
	_, err = w.Commit("initial commit", &git.CommitOptions{Author: sig, Committer: sig})
	require.NoError(t, err)

	err = work.Push(&git.PushOptions{RemoteName: "origin"})
	require.NoError(t, err)

	_ = bare

	svc := newTestGitService(t)
	ctx := context.Background()

	path, err := svc.EnsureWorktree(ctx, "proj-clone-test", bareDir, "master")
	require.NoError(t, err)
	require.DirExists(t, path)

	content, err := os.ReadFile(filepath.Join(path, "readme.txt"))
	require.NoError(t, err)
	require.Equal(t, "hello", string(content))

	path2, err := svc.EnsureWorktree(ctx, "proj-clone-test", bareDir, "master")
	require.NoError(t, err)
	require.Equal(t, path, path2)
}

func TestGitService_EnsureWorktree_NoURL(t *testing.T) {
	svc := newTestGitService(t)
	_, err := svc.EnsureWorktree(context.Background(), "proj-no-url", "", "main")
	require.Error(t, err)
	require.Contains(t, err.Error(), "git URL is required")
}
