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
	"sync"
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

type CollectJobResult struct {
	err error
}

func NewService(githubClient *github.Client, storage storage.Storage, tracker *tracker.Tracker) *Service {
	return &Service{
		githubClient: githubClient,
		storage:      storage,
		tracker:      tracker,
	}
}

func (s *Service) Collect(ctx context.Context) (CollectResult, error) {
	jobs := make(chan models.Repository)
	results := make(chan CollectJobResult)

	wg := &sync.WaitGroup{}

	workers := 5
	repos, err := s.List(ctx)
	if err != nil {
		return CollectResult{}, fmt.Errorf("failed to list repositories: %w", err)
	}
	if len(repos) == 0 {
		return CollectResult{}, nil
	}

	go func() {
		defer close(jobs)
		for _, repo := range repos {
			select {
			case jobs <- repo:
			case <-ctx.Done():
				return
			}
		}
	}()

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case job, ok := <-jobs:
					if !ok {
						return
					}

					err := s.tracker.TrackRepository(ctx, job)

					select {
					case results <- CollectJobResult{err: err}:
					case <-ctx.Done():
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var result CollectResult

	for jobResult := range results {
		if jobResult.err != nil {
			result.FailedCount++
		} else {
			result.SuccessCount++
		}
	}

	if err := ctx.Err(); err != nil {
		return result, fmt.Errorf("collecting canceled: %w", err)
	}

	return result, nil
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
