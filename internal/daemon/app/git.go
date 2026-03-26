package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	gogitconfig "github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/sirupsen/logrus"
)

type GitService struct {
	cacheDir string
	logger   *logrus.Logger
}

func NewGitService(logger *logrus.Logger) (*GitService, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}
	cacheDir := filepath.Join(home, ".cache", "agach")
	return &GitService{
		cacheDir: cacheDir,
		logger:   logger,
	}, nil
}

func (s *GitService) GetWorktreePath(projectID string) string {
	return filepath.Join(s.cacheDir, projectID)
}

func (s *GitService) EnsureWorktree(ctx context.Context, projectID, gitURL, mainBranch string) (string, error) {
	if gitURL == "" {
		return "", errors.New("git URL is required")
	}

	path := s.GetWorktreePath(projectID)

	_, err := git.PlainOpen(path)
	if err != nil {
		if !errors.Is(err, git.ErrRepositoryNotExists) {
			return "", fmt.Errorf("open repo: %w", err)
		}
		if cloneErr := s.clone(ctx, path, gitURL, mainBranch); cloneErr != nil {
			return "", fmt.Errorf("clone: %w", cloneErr)
		}
		return path, nil
	}

	if err := s.fetchAndPull(ctx, path, mainBranch); err != nil {
		return "", fmt.Errorf("fetch and pull: %w", err)
	}
	return path, nil
}

// CreateSessionWorktree ensures the main repo is up-to-date (fetch --all + pull),
// then creates a detached git worktree for the given session ID.
// The worktree is placed under ~/.cache/agach/<projectID>-sessions/<sessionID>.
func (s *GitService) CreateSessionWorktree(ctx context.Context, projectID, sessionID, gitURL, mainBranch string) (string, error) {
	if gitURL == "" {
		return "", errors.New("git URL is required")
	}

	// Ensure main repo exists and is up-to-date
	mainPath := s.GetWorktreePath(projectID)
	if _, err := git.PlainOpen(mainPath); err != nil {
		if !errors.Is(err, git.ErrRepositoryNotExists) {
			return "", fmt.Errorf("open repo: %w", err)
		}
		if err := s.clone(ctx, mainPath, gitURL, mainBranch); err != nil {
			return "", fmt.Errorf("clone: %w", err)
		}
	} else {
		if err := s.fetchAndPull(ctx, mainPath, mainBranch); err != nil {
			return "", fmt.Errorf("fetch and pull: %w", err)
		}
	}

	// Create a dedicated worktree for this session
	wtPath := filepath.Join(s.cacheDir, projectID+"-sessions", sessionID)
	if err := os.MkdirAll(filepath.Dir(wtPath), 0700); err != nil {
		return "", fmt.Errorf("create session dir: %w", err)
	}

	cmd := execCommandContext(ctx, "git", "worktree", "add", "--detach", wtPath, "HEAD")
	cmd.Dir = mainPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git worktree add: %s: %w", string(out), err)
	}

	s.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"session_id": sessionID,
		"worktree":   wtPath,
	}).Info("Created session worktree")

	return wtPath, nil
}

// RemoveSessionWorktree removes a session worktree and prunes it from the main repo.
func (s *GitService) RemoveSessionWorktree(ctx context.Context, projectID, sessionID string) error {
	wtPath := filepath.Join(s.cacheDir, projectID+"-sessions", sessionID)
	mainPath := s.GetWorktreePath(projectID)

	// Remove the worktree directory
	if err := os.RemoveAll(wtPath); err != nil {
		s.logger.WithError(err).WithField("path", wtPath).Warn("failed to remove worktree directory")
	}

	// Prune stale worktree references
	cmd := execCommandContext(ctx, "git", "worktree", "prune")
	cmd.Dir = mainPath
	if out, err := cmd.CombinedOutput(); err != nil {
		s.logger.WithError(err).WithField("output", string(out)).Warn("git worktree prune failed")
	}

	s.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"session_id": sessionID,
	}).Info("Removed session worktree")

	return nil
}

// execCommandContext is a variable so tests can override it.
var execCommandContext = exec.CommandContext

func (s *GitService) clone(ctx context.Context, path, gitURL, mainBranch string) error {
	if err := os.MkdirAll(path, 0700); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	auth, _ := s.getAuth(gitURL)

	opts := &git.CloneOptions{
		URL:  gitURL,
		Auth: auth,
	}
	if mainBranch != "" {
		opts.ReferenceName = gogitconfig.NewBranchReferenceName(mainBranch)
	}

	s.logger.WithFields(logrus.Fields{
		"url":  gitURL,
		"path": path,
	}).Info("Cloning repository")

	_, err := git.PlainCloneContext(ctx, path, false, opts)
	return err
}

func (s *GitService) fetchAndPull(ctx context.Context, path, mainBranch string) error {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return fmt.Errorf("open repo: %w", err)
	}

	remote, err := repo.Remote("origin")
	if err != nil {
		return fmt.Errorf("get remote: %w", err)
	}

	remoteURLs := remote.Config().URLs
	var remoteURL string
	if len(remoteURLs) > 0 {
		remoteURL = remoteURLs[0]
	}

	auth, _ := s.getAuth(remoteURL)

	fetchErr := repo.FetchContext(ctx, &git.FetchOptions{
		RemoteName: "origin",
		Auth:       auth,
	})
	if fetchErr != nil && !errors.Is(fetchErr, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("fetch: %w", fetchErr)
	}

	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("get worktree: %w", err)
	}

	pullOpts := &git.PullOptions{
		RemoteName: "origin",
		Auth:       auth,
		Force:      true,
	}
	if mainBranch != "" {
		pullOpts.ReferenceName = gogitconfig.NewBranchReferenceName(mainBranch)
	}

	pullErr := w.PullContext(ctx, pullOpts)
	if pullErr != nil && !errors.Is(pullErr, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("pull: %w", pullErr)
	}

	return nil
}

func (s *GitService) getAuth(gitURL string) (transport.AuthMethod, error) {
	if strings.HasPrefix(gitURL, "git@") || strings.HasPrefix(gitURL, "ssh://") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		keyPath := filepath.Join(home, ".ssh", "id_rsa")
		if _, err := os.Stat(keyPath); os.IsNotExist(err) {
			keyPath = filepath.Join(home, ".ssh", "id_ed25519")
		}
		auth, err := gitssh.NewPublicKeysFromFile("git", keyPath, "")
		if err != nil {
			return nil, fmt.Errorf("load ssh key: %w", err)
		}
		return auth, nil
	}

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return &githttp.BasicAuth{
			Username: "x-access-token",
			Password: token,
		}, nil
	}

	return nil, nil
}
