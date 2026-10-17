package storage

import (
	"context"
	"errors"
	models2 "github-watcher/internal/models"
)

type Storage interface {
	AddRepository(ctx context.Context, fullName string) (models2.Repository, error)
	DeleteRepository(ctx context.Context, fullName string) error
	ListRepositories(ctx context.Context) ([]models2.Repository, error)
	SaveSnapshot(ctx context.Context, snapshot models2.RepositorySnapshot) error
	ListSnapshots(ctx context.Context, repositoryID int) ([]models2.RepositorySnapshot, error)
}

var ErrRepositoryAlreadyTracked = errors.New("repository already tracked")
var ErrRepositoryNotFound = errors.New("repository not found")
