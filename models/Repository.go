package models

import "time"

type Repository struct {
	ID       int
	Name     string
	FullName string
	Owner    string
	AddedAt  time.Time
}
