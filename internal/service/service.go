package service

import (
	"context"
	"errors"
	"fmt"
	"github-watcher/internal/github"
	"github-watcher/internal/models"
	"github-watcher/internal/storage"
	"github-watcher/internal/tracker"
	"strings"
)

type Service struct {
	githubClient *github.Client
	storage      storage.Storage
	tracker      *tracker.Tracker
}

type CollectResult struct {
	SuccessCount int
	FailedCount  int
}

func NewService(githubClient *github.Client, storage storage.Storage, tracker *tracker.Tracker) *Service {
	return &Service{
		githubClient: githubClient,
		storage:      storage,
		tracker:      tracker,
	}
}

func (s *Service) Collect(ctx context.Context) (CollectResult, error) {

}

func (s *Service) Add(ctx context.Context, fullName string) (models.Repository, error) {
	owner, repo, err := parseName(fullName)
	if err != nil {
		return models.Repository{}, err
	}
	repoInfo, err := s.githubClient.GetRepository(ctx, owner, repo)
	if err != nil {
		return models.Repository{}, fmt.Errorf("failed to fetch repository: %w", err)
	}

	repository, err := s.storage.AddRepository(ctx, repoInfo.FullName)
	if err != nil {
		return models.Repository{}, fmt.Errorf("failed to add repository to storage: %w", err)
	}
	return repository, nil
}

func (s *Service) List(ctx context.Context) ([]models.Repository, error) {
	return s.storage.ListRepositories(ctx)
}

func (s *Service) Delete(ctx context.Context, fullName string) error {
	owner, repo, err := parseName(fullName)
	if err != nil {
		return err
	}

	checkedName := owner + "/" + repo
	err = s.storage.DeleteRepository(ctx, checkedName)
	if err != nil {
		if errors.Is(err, storage.ErrRepositoryNotFound) {
			return fmt.Errorf("repository %s not found", checkedName)
		}
		return fmt.Errorf("failed to delete repository %s: %w", checkedName, err)
	}
	return nil
}

func parseName(fullName string) (string, string, error) {
	fullName = strings.TrimSpace(fullName)
	parts := strings.Split(fullName, "/")

	if len(parts) != 2 {
		return "", "", errors.New("invalid repository name")
	}

	owner := strings.TrimSpace(parts[0])
	name := strings.TrimSpace(parts[1])

	if owner == "" || name == "" {
		return "", "", errors.New("invalid repository name")
	}

	return owner, name, nil
}
