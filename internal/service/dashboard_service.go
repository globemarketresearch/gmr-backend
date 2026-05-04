package service

import (
	"fmt"
	"time"

	"github.com/healthcare-market-research/backend/internal/cache"
	"github.com/healthcare-market-research/backend/internal/domain/dashboard"
	"github.com/healthcare-market-research/backend/internal/repository"
)

// DashboardService defines the interface for dashboard operations
type DashboardService interface {
	GetStats(userRole string) (*dashboard.DashboardStats, error)
	GetRecentActivity(limit int) (*dashboard.ActivityResponse, error)
}

type dashboardService struct {
	dashboardRepo     repository.DashboardRepository
	reportRepo        repository.ReportRepository
	blogRepo          repository.BlogRepository
	pressReleaseRepo  repository.PressReleaseRepository
	userRepo          repository.UserRepository
	formRepo          repository.FormRepository
	auditRepo         repository.AuditRepository
}

// NewDashboardService creates a new dashboard service instance
func NewDashboardService(
	dashboardRepo repository.DashboardRepository,
	reportRepo repository.ReportRepository,
	blogRepo repository.BlogRepository,
	pressReleaseRepo repository.PressReleaseRepository,
	userRepo repository.UserRepository,
	formRepo repository.FormRepository,
	auditRepo repository.AuditRepository,
) DashboardService {
	return &dashboardService{
		dashboardRepo:    dashboardRepo,
		reportRepo:       reportRepo,
		blogRepo:         blogRepo,
		pressReleaseRepo: pressReleaseRepo,
		userRepo:         userRepo,
		formRepo:         formRepo,
		auditRepo:        auditRepo,
	}
}

// GetStats retrieves dashboard statistics with role-based filtering and caching
func (s *dashboardService) GetStats(userRole string) (*dashboard.DashboardStats, error) {
	cacheKey := fmt.Sprintf("dashboard:stats:%s", userRole)
	var stats dashboard.DashboardStats

	err := cache.GetOrSet(cacheKey, &stats, 10*time.Minute, func() (interface{}, error) {
		result := &dashboard.DashboardStats{}

		// Get report stats
		reportStats, err := s.dashboardRepo.GetReportStats(userRole)
		if err != nil {
			return nil, fmt.Errorf("failed to get report stats: %w", err)
		}

		// Get top performing reports
		topReports, err := s.dashboardRepo.GetTopPerformingReports(5)
		if err == nil {
			reportStats.TopPerformers = topReports
		}

		result.Reports = reportStats

		// Get blog stats
		blogStats, err := s.dashboardRepo.GetBlogStats()
		if err != nil {
			return nil, fmt.Errorf("failed to get blog stats: %w", err)
		}
		result.Blogs = blogStats

		// Get press release stats
		pressReleaseStats, err := s.dashboardRepo.GetPressReleaseStats()
		if err != nil {
			return nil, fmt.Errorf("failed to get press release stats: %w", err)
		}
		result.PressReleases = pressReleaseStats

		// Get user stats (admin only)
		if userRole == "admin" {
			userStats, err := s.dashboardRepo.GetUserStats()
			if err != nil {
				return nil, fmt.Errorf("failed to get user stats: %w", err)
			}
			result.Users = userStats
		}

		// Get lead stats
		leadStats, err := s.dashboardRepo.GetLeadStats()
		if err != nil {
			return nil, fmt.Errorf("failed to get lead stats: %w", err)
		}
		result.Leads = leadStats

		// Get content creation stats
		contentStats, err := s.dashboardRepo.GetContentCreationStats()
		if err != nil {
			return nil, fmt.Errorf("failed to get content creation stats: %w", err)
		}
		result.ContentCreation = contentStats

		// Get performance stats (top reports and categories)
		topCategories, err := s.dashboardRepo.GetTopCategories(5)
		if err == nil && len(topCategories) > 0 {
			result.Performance = &dashboard.PerformanceStats{
				TopReports:    topReports,
				TopCategories: topCategories,
			}
		}

		return result, nil
	})

	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// GetRecentActivity retrieves recent activity feed with caching
func (s *dashboardService) GetRecentActivity(limit int) (*dashboard.ActivityResponse, error) {
	// Enforce max limit
	if limit > 50 {
		limit = 50
	}
	if limit <= 0 {
		limit = 10
	}

	cacheKey := fmt.Sprintf("dashboard:activity:%d", limit)
	var response dashboard.ActivityResponse

	err := cache.GetOrSet(cacheKey, &response, 5*time.Minute, func() (interface{}, error) {
		activities, err := s.dashboardRepo.GetRecentActivities(limit)
		if err != nil {
			return nil, fmt.Errorf("failed to get recent activities: %w", err)
		}

		return &dashboard.ActivityResponse{
			Activities: activities,
			Total:      int64(len(activities)),
		}, nil
	})

	if err != nil {
		return nil, err
	}

	return &response, nil
}
