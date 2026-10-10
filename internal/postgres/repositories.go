package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github-watcher/internal/storage"
	"github-watcher/models"
	"strings"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) AddRepository(ctx context.Context, fullName string) (models.Repository, error) {
	fullName = strings.TrimSpace(fullName)

	parts := strings.Split(fullName, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return models.Repository{}, fmt.Errorf("invalid repository name %s: expected owner/repository", fullName)
	}

	owner := strings.TrimSpace(parts[0])
	name := strings.TrimSpace(parts[1])
	fullName = fmt.Sprintf("%s/%s", owner, name)

	query := `INSERT INTO repositories (owner, name, fullName) VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING RETURNING id, owner, full_name, name, added_at`

	var repo models.Repository

	err := s.db.QueryRowContext(ctx, query, owner, name, fullName).Scan(
		&repo.ID,
		&repo.Owner,
		&repo.FullName,
		&repo.Name,
		&repo.AddedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Repository{}, storage.ErrRepositoryAlreadyTracked
		}
		return models.Repository{}, fmt.Errorf("failed to add repository: %s: %w", fullName, err)
	}
	return repo, nil
}
