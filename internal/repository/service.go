package repository

import (
	"context"
	"errors"
	"fmt"
	"github-watcher/internal/github"
	"github-watcher/internal/storage"
	"github-watcher/models"
	"strings"
)

type Service struct {
	githubClient *github.Client
	storage      storage.Storage
}

func NewService(githubClient *github.Client, storage storage.Storage) *Service {
	return &Service{
		githubClient: githubClient,
		storage:      storage,
	}
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
