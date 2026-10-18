package models

import "time"

type RepositorySnapshot struct {
	ID               int
	RepositoryID     int
	Stars            int
	Forks            int
	HasReadme        bool
	OpenIssues       int
	OpenPullRequests int
	CollectedAt      time.Time
	CommitsLast7Days int
	LastCommitAt     *time.Time
}
