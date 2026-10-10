package storage

import (
	"context"
	"errors"
	"github-watcher/models"
)

type Storage interface {
	AddRepository(ctx context.Context, fullName string) (models.Repository, error)
	DeleteRepository(ctx context.Context, fullName string) error
	ListRepositories(ctx context.Context) ([]models.Repository, error)
	SaveSnapshot(ctx context.Context, snapshot models.RepositorySnapshot) error
	ListSnapshots(ctx context.Context, repositoryID int) ([]models.RepositorySnapshot, error)
}

var ErrRepositoryAlreadyTracked = errors.New("repository already tracked")
