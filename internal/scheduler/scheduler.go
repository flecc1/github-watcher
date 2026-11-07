package scheduler

import (
	"context"
	"github-watcher/internal/logger"
	"github-watcher/internal/service"
	"log/slog"
	"time"
)

type Scheduler struct {
	repositoryService *service.Service
	interval          time.Duration
}

func NewScheduler(repositoryService *service.Service, interval time.Duration) *Scheduler {
	return &Scheduler{
		repositoryService: repositoryService,
		interval:          interval,
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	s.collect(ctx)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.collect(ctx)
		}
	}
}

func (s *Scheduler) collect(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	result, err := s.repositoryService.Collect(ctx)
	if err != nil {
		slog.Error("scheduled collection failed", logger.Err(err))
		return
	}
	slog.Info("scheduled collection finished",
		"success", result.SuccessCount, "failed", result.FailedCount)
}
