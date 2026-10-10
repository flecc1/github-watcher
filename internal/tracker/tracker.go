package tracker

import (
	"github-watcher/internal/github"
	"github-watcher/internal/storage"
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
