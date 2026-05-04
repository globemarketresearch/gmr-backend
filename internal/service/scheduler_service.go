package service

import (
	"context"
	"time"

	"github.com/healthcare-market-research/backend/internal/repository"
	"github.com/healthcare-market-research/backend/pkg/logger"
)

type SchedulerService interface {
	Start(ctx context.Context)
	Stop()
}

type schedulerService struct {
	reportRepo       repository.ReportRepository
	blogRepo         repository.BlogRepository
	pressReleaseRepo repository.PressReleaseRepository
	ticker           *time.Ticker
	stopCh           chan struct{}
}

func NewSchedulerService(
	reportRepo repository.ReportRepository,
	blogRepo repository.BlogRepository,
	pressReleaseRepo repository.PressReleaseRepository,
) SchedulerService {
	return &schedulerService{
		reportRepo:       reportRepo,
		blogRepo:         blogRepo,
		pressReleaseRepo: pressReleaseRepo,
		stopCh:           make(chan struct{}),
	}
}

func (s *schedulerService) Start(ctx context.Context) {
	s.ticker = time.NewTicker(1 * time.Minute)
	logger.Info("Scheduled publishing service started")

	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.processScheduledPublishes()
			case <-s.stopCh:
				logger.Info("Scheduled publishing service stopped")
				return
			case <-ctx.Done():
				logger.Info("Scheduled publishing service context cancelled")
				return
			}
		}
	}()
}

func (s *schedulerService) processScheduledPublishes() {
	now := time.Now()

	if err := s.reportRepo.PublishScheduled(now); err != nil {
		logger.Error("Failed to publish scheduled reports", "error", err)
	}

	if err := s.blogRepo.PublishScheduled(now); err != nil {
		logger.Error("Failed to publish scheduled blogs", "error", err)
	}

	if err := s.pressReleaseRepo.PublishScheduled(now); err != nil {
		logger.Error("Failed to publish scheduled press releases", "error", err)
	}
}

func (s *schedulerService) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	close(s.stopCh)
}
