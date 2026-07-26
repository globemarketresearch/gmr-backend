package repository

import (
	"testing"

	"github.com/healthcare-market-research/backend/internal/domain/mediamention"
	"github.com/healthcare-market-research/backend/internal/domain/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupMediaMentionTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&report.Report{}, &mediamention.MediaMention{})
	require.NoError(t, err)

	return db
}

func createTestReportWithStatus(t *testing.T, db *gorm.DB, status string) uint {
	rep := &report.Report{
		Title:  "VFX Market Report",
		Slug:   "vfx-market-report",
		Status: status,
	}
	require.NoError(t, db.Create(rep).Error)
	return rep.ID
}

func TestMediaMentionRepository_GetByID_ResolvesPublishedReportLink(t *testing.T) {
	db := setupMediaMentionTestDB(t)
	repo := NewMediaMentionRepository(db)
	reportID := createTestReportWithStatus(t, db, "published")

	mention := &mediamention.MediaMention{
		Title:          "Forbes",
		ReportID:       &reportID,
		ReportLinkText: "Global VFX Market",
	}
	require.NoError(t, db.Create(mention).Error)

	found, err := repo.GetByID(mention.ID)
	require.NoError(t, err)
	assert.Equal(t, "vfx-market-report", found.ReportSlug)
	assert.Equal(t, "VFX Market Report", found.ReportTitle)
}

func TestMediaMentionRepository_GetByID_HidesDraftReportLink(t *testing.T) {
	db := setupMediaMentionTestDB(t)
	repo := NewMediaMentionRepository(db)
	reportID := createTestReportWithStatus(t, db, "draft")

	mention := &mediamention.MediaMention{
		Title:          "Forbes",
		ReportID:       &reportID,
		ReportLinkText: "Global VFX Market",
	}
	require.NoError(t, db.Create(mention).Error)

	found, err := repo.GetByID(mention.ID)
	require.NoError(t, err)
	assert.Empty(t, found.ReportSlug, "draft report should not resolve a slug")
	assert.NotNil(t, found.ReportID, "report_id itself should still be preserved")
}

func TestMediaMentionRepository_GetByID_HidesSoftDeletedReportLink(t *testing.T) {
	db := setupMediaMentionTestDB(t)
	repo := NewMediaMentionRepository(db)
	reportID := createTestReportWithStatus(t, db, "published")
	require.NoError(t, db.Model(&report.Report{}).Where("id = ?", reportID).
		Update("deleted_at", "2026-01-01 00:00:00").Error)

	mention := &mediamention.MediaMention{
		Title:          "Forbes",
		ReportID:       &reportID,
		ReportLinkText: "Global VFX Market",
	}
	require.NoError(t, db.Create(mention).Error)

	found, err := repo.GetByID(mention.ID)
	require.NoError(t, err)
	assert.Empty(t, found.ReportSlug, "soft-deleted report should not resolve a slug")
}

func TestMediaMentionRepository_GetByID_NoReportLinked(t *testing.T) {
	db := setupMediaMentionTestDB(t)
	repo := NewMediaMentionRepository(db)

	mention := &mediamention.MediaMention{Title: "Forbes"}
	require.NoError(t, db.Create(mention).Error)

	found, err := repo.GetByID(mention.ID)
	require.NoError(t, err)
	assert.Empty(t, found.ReportSlug)
	assert.Nil(t, found.ReportID)
}

func TestMediaMentionRepository_GetAll_ResolvesReportLink(t *testing.T) {
	db := setupMediaMentionTestDB(t)
	repo := NewMediaMentionRepository(db)
	reportID := createTestReportWithStatus(t, db, "published")

	require.NoError(t, db.Create(&mediamention.MediaMention{
		Title: "Forbes", ReportID: &reportID, ReportLinkText: "Global VFX Market",
	}).Error)
	require.NoError(t, db.Create(&mediamention.MediaMention{Title: "Bloomberg"}).Error)

	mentions, total, err := repo.GetAll(1, 20, "")
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, mentions, 2)

	byTitle := map[string]mediamention.MediaMention{}
	for _, m := range mentions {
		byTitle[m.Title] = m
	}
	assert.Equal(t, "vfx-market-report", byTitle["Forbes"].ReportSlug)
	assert.Empty(t, byTitle["Bloomberg"].ReportSlug)
}
