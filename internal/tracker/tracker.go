package tracker

import (
	"context"
	"fmt"
	"github-watcher/internal/github"
	"github-watcher/internal/storage"
	"github-watcher/models"
)

type Tracker struct {
	githubClient *github.Client
	storage      storage.Storage
}

func NewTracker(githubClient *github.Client, storage storage.Storage) *Tracker {
	return &Tracker{
		githubClient: githubClient,
		storage:      storage,
	}
}

func (t *Tracker) TrackRepository(ctx context.Context, repo models.Repository) error {
	info, err := t.githubClient.GetRepository(ctx, repo.Owner, repo.Name)
	if err != nil {
		return fmt.Errorf("failed to get repository info: %w", err)
	}

	hasReadme, err := t.githubClient.HasReadme(ctx, repo.Owner, repo.Name)
	if err != nil {
		return fmt.Errorf("failed to check if repository readme exists: %w", err)
	}

	openPullRequests, err := t.githubClient.GetOpenPullRequests(ctx, repo.Owner, repo.Name)
	if err != nil {
		return fmt.Errorf("failed to get open pull requests: %w", err)
	}

	commitsLast7Days, err := t.githubClient.GetCommitsLast7Days(ctx, repo.Owner, repo.Name)
	if err != nil {
		return fmt.Errorf("failed to get commits for the last 7 days: %w", err)
	}

	lastCommitAt, err := t.githubClient.GetLastCommitAt(ctx, repo.Owner, repo.Name)
	if err != nil {
		return fmt.Errorf("failed to get last commit date: %w", err)
	}

	snapshot := models.RepositorySnapshot{
		RepositoryID:     repo.ID,
		Stars:            info.Stars,
		Forks:            info.Forks,
		HasReadme:        hasReadme,
		OpenIssues:       info.OpenIssues - openPullRequests,
		OpenPullRequests: openPullRequests,
		CommitsLast7Days: commitsLast7Days,
		LastCommitAt:     lastCommitAt,
	}

	if err := t.storage.SaveSnapshot(ctx, snapshot); err != nil {
		return fmt.Errorf("failed to save repository snapshot: %w", err)
	}
	return nil
}
