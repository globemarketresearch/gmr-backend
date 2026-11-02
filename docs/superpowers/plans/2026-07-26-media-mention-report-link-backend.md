# Media Mention Report Link — Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a `MediaMention` optionally reference a `Report` (`report_id` + admin-supplied `report_link_text`), and have the public API resolve that into `report_slug`/`report_title` only when the linked report is published and not deleted.

**Architecture:** Add a nullable FK column `report_id` (`ON DELETE SET NULL`) and a `report_link_text` column to `media_mentions`. Extend the `MediaMention` Go struct with those two writable fields plus two GORM read-only (`gorm:"->"`) fields, `ReportSlug`/`ReportTitle`, populated via a `LEFT JOIN reports ... WHERE status='published' AND deleted_at IS NULL` in the repository's `GetAll`/`GetByID`, mirroring the existing `category_id`/`category_name` pattern on `reports`. The handler validates `report_id` against the same published/non-deleted criteria before persisting, using a narrow `GetByID`-only interface satisfied by the existing `ReportRepository`.

**Tech Stack:** Go, Fiber, GORM (Postgres in prod, SQLite in-memory for repository tests), raw SQL for joins (existing project convention — no ORM `Preload`/`AutoMigrate` used).

## Global Constraints

- Follow this repo's existing raw-SQL-join pattern for resolved relations (see `reportRepository.GetByIDWithRelations`) — do not introduce GORM `Preload`.
- Migrations are hand-written, sequentially numbered SQL files in `migrations/`; the next number is `029`.
- New/changed Go code must build (`go build ./...`) and pass `go test ./...` before each commit.
- `report_id` is optional; `report_link_text` is optional; but if either is set on write, both must be set (paired validation) — enforced in the handler, not the DB.
- Only reports with `status = 'published' AND deleted_at IS NULL` may ever be referenced or resolved — enforced both at write-time (handler validation) and read-time (JOIN filter), independently, so a report that changes status *after* being linked stops resolving automatically.

---

### Task 1: Migration — add `report_id` and `report_link_text` to `media_mentions`

**Files:**
- Create: `migrations/029_add_report_reference_to_media_mentions.sql`

**Interfaces:**
- Produces: DB columns `media_mentions.report_id INTEGER` (nullable, FK → `reports(id)`, `ON DELETE SET NULL`) and `media_mentions.report_link_text VARCHAR(255)` (nullable), plus index `idx_media_mentions_report_id`.

- [ ] **Step 1: Write the migration file**

```sql
-- migrations/029_add_report_reference_to_media_mentions.sql

ALTER TABLE media_mentions
    ADD COLUMN report_id INTEGER REFERENCES reports(id) ON DELETE SET NULL,
    ADD COLUMN report_link_text VARCHAR(255);

CREATE INDEX idx_media_mentions_report_id ON media_mentions(report_id);
```

- [ ] **Step 2: Apply the migration against your local dev Postgres**

Run whatever this repo's existing manual migration process is (check `README.md` / `docker-compose.yml` for the psql connection string — there is no in-repo migration runner). Example if applying by hand:

```bash
psql "$DATABASE_URL" -f migrations/029_add_report_reference_to_media_mentions.sql
```

Expected: no errors; `\d media_mentions` in psql shows the two new columns and the FK constraint.

- [ ] **Step 3: Commit**

```bash
git add migrations/029_add_report_reference_to_media_mentions.sql
git commit -m "feat: add report_id and report_link_text columns to media_mentions"
```

---

### Task 2: Extend the `MediaMention` domain struct

**Files:**
- Modify: `internal/domain/mediamention/media_mention.go`

**Interfaces:**
- Consumes: nothing new.
- Produces: `MediaMention.ReportID *uint`, `MediaMention.ReportLinkText string`, `MediaMention.ReportSlug string` (read-only), `MediaMention.ReportTitle string` (read-only) — these four field names/types are relied on by Tasks 3 and 4.

- [ ] **Step 1: Edit the struct**

Replace the full file content:

```go
package mediamention

import (
	"time"
)

// MediaMention represents a citation of GMR research by an external publication
type MediaMention struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	Title          string    `json:"title" gorm:"type:varchar(255);not null"`
	Link           string    `json:"link,omitempty" gorm:"type:varchar(500)"`
	ImageURL       string    `json:"imageUrl,omitempty" gorm:"type:varchar(500)"`
	DisplayOrder   int       `json:"displayOrder" gorm:"default:0"`
	ReportID       *uint     `json:"reportId,omitempty" gorm:"index"`
	ReportLinkText string    `json:"reportLinkText,omitempty" gorm:"type:varchar(255)"`
	ReportSlug     string    `json:"reportSlug,omitempty" gorm:"->"`
	ReportTitle    string    `json:"reportTitle,omitempty" gorm:"->"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// TableName specifies the table name for GORM
func (MediaMention) TableName() string {
	return "media_mentions"
}
```

`ReportSlug`/`ReportTitle` use `gorm:"->"` (read-only) exactly like `Report.CategoryName` — GORM will never try to write them as real columns; they're populated purely by SQL column aliasing in Task 3.

- [ ] **Step 2: Verify it builds**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/domain/mediamention/media_mention.go
git commit -m "feat: add report reference fields to MediaMention struct"
```

---

### Task 3: Repository — resolve `report_slug`/`report_title` via JOIN, filtered to published+non-deleted

**Files:**
- Modify: `internal/repository/media_mention_repository.go`
- Test: `internal/repository/media_mention_repository_test.go` (new)

**Interfaces:**
- Consumes: `mediamention.MediaMention` (Task 2), `report.Report` (existing, for test fixtures — `github.com/healthcare-market-research/backend/internal/domain/report`).
- Produces: `GetAll`/`GetByID` now populate `ReportSlug`/`ReportTitle` on returned `MediaMention` values whenever `ReportID` points at a published, non-deleted report; otherwise those fields stay empty and `ReportID` itself is preserved as-is (so the admin UI can still show what's linked even if it's since been unpublished).

- [ ] **Step 1: Write the failing repository tests**

```go
// internal/repository/media_mention_repository_test.go
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
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/repository/... -run TestMediaMentionRepository -v`
Expected: FAIL — `ReportSlug`/`ReportTitle` undefined or empty (struct/repo not yet updated), or compile error if Task 2 wasn't done first.

- [ ] **Step 3: Rewrite `GetAll` and `GetByID` to use the raw-SQL LEFT JOIN**

Replace the full file content of `internal/repository/media_mention_repository.go`:

```go
package repository

import (
	"github.com/healthcare-market-research/backend/internal/domain/mediamention"
	"gorm.io/gorm"
)

type MediaMentionRepository interface {
	GetAll(page, limit int, search string) ([]mediamention.MediaMention, int64, error)
	GetByID(id uint) (*mediamention.MediaMention, error)
	Create(mention *mediamention.MediaMention) error
	Update(mention *mediamention.MediaMention) error
	Delete(id uint) error
}

type mediaMentionRepository struct {
	db *gorm.DB
}

func NewMediaMentionRepository(db *gorm.DB) MediaMentionRepository {
	return &mediaMentionRepository{db: db}
}

const mediaMentionReportJoinSQL = `
	SELECT m.*, r.slug as report_slug, r.title as report_title
	FROM media_mentions m
	LEFT JOIN reports r ON m.report_id = r.id AND r.status = 'published' AND r.deleted_at IS NULL
`

func (r *mediaMentionRepository) GetAll(page, limit int, search string) ([]mediamention.MediaMention, int64, error) {
	var mentions []mediamention.MediaMention
	var total int64

	offset := (page - 1) * limit

	countQuery := r.db.Model(&mediamention.MediaMention{})
	if search != "" {
		countQuery = countQuery.Where("title ILIKE ?", "%"+search+"%")
	}
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	querySQL := mediaMentionReportJoinSQL
	args := []interface{}{}
	if search != "" {
		querySQL += " WHERE m.title ILIKE ?"
		args = append(args, "%"+search+"%")
	}
	querySQL += " ORDER BY m.display_order ASC, m.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	err := r.db.Raw(querySQL, args...).Scan(&mentions).Error
	return mentions, total, err
}

func (r *mediaMentionRepository) GetByID(id uint) (*mediamention.MediaMention, error) {
	var mention mediamention.MediaMention
	querySQL := mediaMentionReportJoinSQL + " WHERE m.id = ?"

	err := r.db.Raw(querySQL, id).Scan(&mention).Error
	if err != nil {
		return nil, err
	}
	if mention.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &mention, nil
}

func (r *mediaMentionRepository) Create(mention *mediamention.MediaMention) error {
	return r.db.Create(mention).Error
}

func (r *mediaMentionRepository) Update(mention *mediamention.MediaMention) error {
	return r.db.Save(mention).Error
}

func (r *mediaMentionRepository) Delete(id uint) error {
	return r.db.Delete(&mediamention.MediaMention{}, id).Error
}
```

Note: `Raw(...).Scan(...)` does not return `gorm.ErrRecordNotFound` on zero rows the way `First` does — the explicit `mention.ID == 0` check in `GetByID` replicates that behavior so callers (service/handler) that check `err.Error() == "record not found"` keep working unchanged.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/repository/... -run TestMediaMentionRepository -v`
Expected: PASS (all 5 tests).

- [ ] **Step 5: Run the full repository test suite to check for regressions**

Run: `go test ./internal/repository/... -v`
Expected: PASS (no pre-existing tests broken).

- [ ] **Step 6: Commit**

```bash
git add internal/repository/media_mention_repository.go internal/repository/media_mention_repository_test.go
git commit -m "feat: resolve media mention report link via published-only JOIN"
```

---

### Task 4: Handler — validate `reportId` on write, support partial update, wire into `main.go`

**Files:**
- Modify: `internal/handler/media_mention_handler.go`
- Modify: `cmd/api/main.go`
- Test: `internal/handler/media_mention_handler_test.go` (new)

**Interfaces:**
- Consumes: `repository.ReportRepository.GetByID(id uint) (*report.Report, error)` (existing method, used structurally — see narrow interface below) — `report.Report.Status string`, `report.Report.DeletedAt *time.Time` (existing fields).
- Produces: `NewMediaMentionHandler(service service.MediaMentionService, reportLookup ReportLookup) *MediaMentionHandler` — signature changes, `main.go` call site must be updated in this task.

- [ ] **Step 1: Write the failing handler tests**

```go
// internal/handler/media_mention_handler_test.go
package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/domain/mediamention"
	"github.com/healthcare-market-research/backend/internal/domain/report"
	"github.com/healthcare-market-research/backend/pkg/response"
)

type mockMediaMentionService struct {
	getAllFunc  func(page, limit int, search string) ([]mediamention.MediaMention, int64, error)
	getByIDFunc func(id uint) (*mediamention.MediaMention, error)
	createFunc  func(mention *mediamention.MediaMention) error
	updateFunc  func(id uint, mention *mediamention.MediaMention) error
	deleteFunc  func(id uint) error
}

func (m *mockMediaMentionService) GetAll(page, limit int, search string) ([]mediamention.MediaMention, int64, error) {
	if m.getAllFunc != nil {
		return m.getAllFunc(page, limit, search)
	}
	return []mediamention.MediaMention{}, 0, nil
}
func (m *mockMediaMentionService) GetByID(id uint) (*mediamention.MediaMention, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(id)
	}
	return &mediamention.MediaMention{ID: id, Title: "Test"}, nil
}
func (m *mockMediaMentionService) Create(mention *mediamention.MediaMention) error {
	if m.createFunc != nil {
		return m.createFunc(mention)
	}
	return nil
}
func (m *mockMediaMentionService) Update(id uint, mention *mediamention.MediaMention) error {
	if m.updateFunc != nil {
		return m.updateFunc(id, mention)
	}
	return nil
}
func (m *mockMediaMentionService) Delete(id uint) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(id)
	}
	return nil
}
func (m *mockMediaMentionService) UploadImage(id uint, file interface{ Filename() string }) (*mediamention.MediaMention, error) {
	return nil, nil
}
func (m *mockMediaMentionService) DeleteImage(id uint) error { return nil }

type mockReportLookup struct {
	getByIDFunc func(id uint) (*report.Report, error)
}

func (m *mockReportLookup) GetByID(id uint) (*report.Report, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(id)
	}
	return &report.Report{ID: id, Title: "VFX Market Report", Slug: "vfx-market-report", Status: "published"}, nil
}

func TestMediaMentionHandler_Create_WithPublishedReport_Succeeds(t *testing.T) {
	mockService := &mockMediaMentionService{}
	mockReports := &mockReportLookup{}
	h := NewMediaMentionHandler(mockService, mockReports)

	app := fiber.New()
	app.Post("/media-mentions", h.Create)

	body, _ := json.Marshal(map[string]interface{}{
		"title":          "Forbes",
		"reportId":       1,
		"reportLinkText": "Global VFX Market",
	})
	req := httptest.NewRequest("POST", "/media-mentions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
}

func TestMediaMentionHandler_Create_WithUnpublishedReport_Rejected(t *testing.T) {
	mockService := &mockMediaMentionService{}
	mockReports := &mockReportLookup{
		getByIDFunc: func(id uint) (*report.Report, error) {
			return &report.Report{ID: id, Status: "draft"}, nil
		},
	}
	h := NewMediaMentionHandler(mockService, mockReports)

	app := fiber.New()
	app.Post("/media-mentions", h.Create)

	body, _ := json.Marshal(map[string]interface{}{
		"title":          "Forbes",
		"reportId":       1,
		"reportLinkText": "Global VFX Market",
	})
	req := httptest.NewRequest("POST", "/media-mentions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}

	var result response.Response
	bodyBytes, _ := io.ReadAll(resp.Body)
	json.Unmarshal(bodyBytes, &result)
	if result.Success {
		t.Error("expected success=false")
	}
}

func TestMediaMentionHandler_Create_ReportIDWithoutLinkText_Rejected(t *testing.T) {
	h := NewMediaMentionHandler(&mockMediaMentionService{}, &mockReportLookup{})

	app := fiber.New()
	app.Post("/media-mentions", h.Create)

	body, _ := json.Marshal(map[string]interface{}{
		"title":    "Forbes",
		"reportId": 1,
	})
	req := httptest.NewRequest("POST", "/media-mentions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestMediaMentionHandler_Create_NoReportFields_Succeeds(t *testing.T) {
	h := NewMediaMentionHandler(&mockMediaMentionService{}, &mockReportLookup{})

	app := fiber.New()
	app.Post("/media-mentions", h.Create)

	body, _ := json.Marshal(map[string]interface{}{"title": "Forbes"})
	req := httptest.NewRequest("POST", "/media-mentions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
}

func TestMediaMentionHandler_Update_UnlinksReportWhenExplicitlyNulled(t *testing.T) {
	existing := &mediamention.MediaMention{
		ID: 1, Title: "Forbes",
		ReportID: func() *uint { id := uint(1); return &id }(),
		ReportLinkText: "Old Caption",
	}
	var savedMention *mediamention.MediaMention
	mockService := &mockMediaMentionService{
		getByIDFunc: func(id uint) (*mediamention.MediaMention, error) { return existing, nil },
		updateFunc: func(id uint, mention *mediamention.MediaMention) error {
			savedMention = mention
			return nil
		},
	}
	h := NewMediaMentionHandler(mockService, &mockReportLookup{})

	app := fiber.New()
	app.Put("/media-mentions/:id", h.Update)

	body, _ := json.Marshal(map[string]interface{}{
		"reportId":       nil,
		"reportLinkText": "",
	})
	req := httptest.NewRequest("PUT", "/media-mentions/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if savedMention == nil {
		t.Fatal("expected Update to be called")
	}
	if savedMention.ReportID != nil {
		t.Error("expected ReportID to be cleared")
	}
}

var _ = time.Now
var _ = errors.New
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/handler/... -run TestMediaMentionHandler -v`
Expected: FAIL — compile error, since `NewMediaMentionHandler` doesn't yet accept a second argument and `ReportID`/`ReportLinkText` mismatches don't yet exist on the request path.

- [ ] **Step 3: Update the handler**

Edit `internal/handler/media_mention_handler.go`. Add imports and the `ReportLookup` interface, change the struct/constructor, and add validation to `Create` and `Update`:

```go
package handler

import (
	"fmt"
	"math"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/healthcare-market-research/backend/internal/domain/mediamention"
	"github.com/healthcare-market-research/backend/internal/domain/report"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/pkg/response"
	"github.com/healthcare-market-research/backend/pkg/validation"
)

// ReportLookup is the minimal capability MediaMentionHandler needs from
// ReportRepository: looking up a report by ID to validate a reportId
// reference before persisting it. Satisfied structurally by
// repository.ReportRepository — no adapter needed at the call site.
type ReportLookup interface {
	GetByID(id uint) (*report.Report, error)
}

type MediaMentionHandler struct {
	service      service.MediaMentionService
	reportLookup ReportLookup
}

func NewMediaMentionHandler(service service.MediaMentionService, reportLookup ReportLookup) *MediaMentionHandler {
	return &MediaMentionHandler{service: service, reportLookup: reportLookup}
}

// validateReportReference checks that reportID (if present) points at a
// published, non-deleted report, and that reportID/reportLinkText are
// either both set or both empty.
func (h *MediaMentionHandler) validateReportReference(reportID *uint, reportLinkText string) error {
	if reportID == nil && reportLinkText == "" {
		return nil
	}
	if reportID == nil || reportLinkText == "" {
		return fmt.Errorf("reportId and reportLinkText must be set together")
	}

	rep, err := h.reportLookup.GetByID(*reportID)
	if err != nil || rep.Status != "published" || rep.DeletedAt != nil {
		return fmt.Errorf("selected report not found or not published")
	}
	return nil
}
```

Then update `Create` (insert the validation call right after the existing link-URL check, before `h.service.Create`):

```go
	if req.Link != "" {
		if err := validation.ValidateURL(req.Link); err != nil {
			return response.BadRequest(c, "Invalid link URL: "+err.Error())
		}
	}

	if err := h.validateReportReference(req.ReportID, req.ReportLinkText); err != nil {
		return response.BadRequest(c, err.Error())
	}

	if err := h.service.Create(&req); err != nil {
```

Then update `Update`'s `bodyMap` block to handle `reportId`/`reportLinkText` and re-validate the merged result:

```go
	bodyMap := make(map[string]interface{})
	if err := c.BodyParser(&bodyMap); err == nil {
		if _, ok := bodyMap["link"]; ok {
			if req.Link != "" {
				if err := validation.ValidateURL(req.Link); err != nil {
					return response.BadRequest(c, "Invalid link URL: "+err.Error())
				}
			}
			existing.Link = req.Link
		}
		if _, ok := bodyMap["displayOrder"]; ok {
			existing.DisplayOrder = req.DisplayOrder
		}
		if _, ok := bodyMap["reportId"]; ok {
			existing.ReportID = req.ReportID
		}
		if _, ok := bodyMap["reportLinkText"]; ok {
			existing.ReportLinkText = req.ReportLinkText
		}
	}

	if err := h.validateReportReference(existing.ReportID, existing.ReportLinkText); err != nil {
		return response.BadRequest(c, err.Error())
	}

	if err := h.service.Update(uint(id), existing); err != nil {
```

- [ ] **Step 4: Wire the new dependency into `main.go`**

In `cmd/api/main.go`, change line 175 from:

```go
	mediaMentionHandler := handler.NewMediaMentionHandler(mediaMentionService)
```

to:

```go
	mediaMentionHandler := handler.NewMediaMentionHandler(mediaMentionService, reportRepo)
```

`reportRepo` (created at line 125, `repository.NewReportRepository(...)`) already satisfies `handler.ReportLookup` structurally since it implements `GetByID(id uint) (*report.Report, error)`.

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./internal/handler/... -run TestMediaMentionHandler -v`
Expected: PASS (all 5 tests).

- [ ] **Step 6: Build the whole project**

Run: `go build ./...`
Expected: no errors, confirms `main.go` wiring compiles.

- [ ] **Step 7: Run the full test suite**

Run: `go test ./...`
Expected: PASS, no regressions in other handlers/services.

- [ ] **Step 8: Commit**

```bash
git add internal/handler/media_mention_handler.go internal/handler/media_mention_handler_test.go cmd/api/main.go
git commit -m "feat: validate media mention report references against published reports"
```

---

### Task 5: Update Swagger docs for the changed request/response shape

**Files:**
- Modify: `internal/handler/media_mention_handler.go` (doc comments only — already touched in Task 4, this is a follow-up pass)

**Interfaces:**
- Consumes: nothing new.
- Produces: regenerated `docs/swagger.json`/`docs/swagger.yaml`/`docs/docs.go` (if this repo has a swag-generation command) reflecting `reportId`, `reportLinkText`, `reportSlug`, `reportTitle` on the `MediaMention` schema.

- [ ] **Step 1: Check for a swag generation command**

Run: `grep -r "swag init" README.md Makefile 2>/dev/null` (or check `go.mod` for `github.com/swaggo/swag`)

- [ ] **Step 2: Regenerate docs if a generator exists**

If found, run it (typically `swag init -g cmd/api/main.go -o docs`). If no generator is wired into this repo, skip this task's remaining steps — the doc comments added in Task 4 (`@Param mediaMention body ...`) are sufficient for now, and note in your final report that `/media-mentions` swagger paths were already absent before this change (confirmed during research) so this is not a regression.

- [ ] **Step 3: Commit if docs were regenerated**

```bash
git add docs/
git commit -m "docs: regenerate swagger for media mention report reference fields"
```

---

## Final Verification

- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] Manually hit `POST /api/v1/media-mentions` with a valid `reportId` of a published report and confirm `201` with `reportSlug`/`reportTitle` populated on the subsequent `GET`
- [ ] Manually hit it with a `reportId` of a draft report and confirm `400`
