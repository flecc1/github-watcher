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

	query := `INSERT INTO repositories (owner, name, full_name) VALUES ($1, $2, $3)
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

func (s *Store) ListRepositories(ctx context.Context) ([]models.Repository, error) {
	query := `SELECT id, owner, name, full_name, added_at FROM repositories
ORDER BY added_at DESC, id DESC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	repositories := make([]models.Repository, 0)

	for rows.Next() {
		var repo models.Repository
		if err := rows.Scan(&repo.ID, &repo.Owner, &repo.Name, &repo.FullName, &repo.AddedAt); err != nil {
			return nil, fmt.Errorf("failed to scan repository: %w", err)
		}
		repositories = append(repositories, repo)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate repositories: %w", err)
	}
	return repositories, nil
}

func (s *Store) DeleteRepository(ctx context.Context, fullName string) error {
	query := `DELETE FROM repositories WHERE LOWER(full_name) = LOWER($1)`
	result, err := s.db.ExecContext(ctx, query, strings.TrimSpace(fullName))
	if err != nil {
		return fmt.Errorf("failed to delete repository %s: %w", fullName, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrRepositoryNotFound
	}
	return nil
}

func (s *Store) SaveSnapshot(ctx context.Context, snapshot models.RepositorySnapshot) error {
	query := `INSERT INTO repository_snapshots (
                                  repository_id,
                                  stars, 
                                  forks, 
                                  has_readme, 
                                  open_issues, 
                                  open_pull_requests,
                                  commits,
                                  commits_last_7_days,
                                  last_commit_at
                                  )
                                  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
                                  `

	_, err := s.db.ExecContext(
		ctx,
		query,
		snapshot.RepositoryID,
		snapshot.Stars,
		snapshot.Forks,
		snapshot.HasReadme,
		snapshot.OpenIssues,
		snapshot.OpenPullRequests,
		snapshot.Commits,
		snapshot.CommitsLast7Days,
		snapshot.LastCommitAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save snapshot for repository %d: %w", snapshot.RepositoryID, err)
	}
	return nil
}

func (s *Store) ListSnapshots(ctx context.Context, repositoryID int) ([]models.RepositorySnapshot, error) {
	query := `SELECT
id, repository_id, stars, forks, has_readme, open_issues,
open_pull_requests, collected_at, commits, commits_last_7_days,
last_commit_at
FROM repository_snapshots
WHERE repository_id = $1
ORDER BY collected_at DESC, id DESC`

	rows, err := s.db.QueryContext(ctx, query, repositoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots for repository %d: %w", repositoryID, err)
	}
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	snapshots := make([]models.RepositorySnapshot, 0)

	for rows.Next() {
		var snapshot models.RepositorySnapshot

		if err := rows.Scan(
			&snapshot.ID,
			&snapshot.RepositoryID,
			&snapshot.Stars,
			&snapshot.Forks,
			&snapshot.HasReadme,
			&snapshot.OpenIssues,
			&snapshot.OpenPullRequests,
			&snapshot.CollectedAt,
			&snapshot.Commits,
			&snapshot.CommitsLast7Days,
			&snapshot.LastCommitAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan snapshot for repository %d: %w", repositoryID, err)
		}
		snapshots = append(snapshots, snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate snapshots for repository %d: %w", repositoryID, err)
	}
	return snapshots, nil
}
