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
	part1, part2, err := parseName(fullName)
	if err != nil {
		return models.Repository{}, err
	}
	repoInfo, err := s.githubClient.GetRepository(ctx, part1, part2)
	if err != nil {
		return models.Repository{}, fmt.Errorf("failed to fetch repository: %w", err)
	}

	repo, err := s.storage.AddRepository(ctx, repoInfo.FullName)
	if err != nil {
		return models.Repository{}, fmt.Errorf("failed to add repository to storage: %w", err)
	}
	return repo, nil
}

func (s *Service) List(ctx context.Context) ([]models.Repository, error) {
	return s.storage.ListRepositories(ctx)
}

func parseName(fullName string) (string, string, error) {
	parts := strings.Split(fullName, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", errors.New("invalid repository name")
	}
	return parts[0], parts[1], nil
}
