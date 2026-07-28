# Industry News module — design spec

Date: 2026-07-28

## Summary

Add a third content module, "Industry News", alongside the existing PR (`press_release`) and Statistics (`blog`) modules, across `gmr-backend`, `gmr-admin`, and `gmr`. It is a brand-new backend domain (not a rebrand of an existing table, unlike Statistics/`blog`). Two behaviors distinguish it from a plain copy of PR:

1. New articles default their author to a seeded "Globe Market Research" author record.
2. The admin add/edit form gets an image gallery manager identical in behavior to the Report module's (`report-images-manager.tsx`), backed by a new `industry_news_images` table and the existing Cloudflare R2 upload service.

## Reference implementations

- `press_release` domain (`gmr-backend/internal/domain/press_release/press_release.go`, migration `014_create_press_releases_table.sql`) — direct schema/layer template.
- `blog` domain (`gmr-backend/internal/domain/blog/blog.go`, migration `013_create_blogs_table.sql`) — same shape, cross-checked for field parity.
- `report_images` (migration `010_create_report_images_table.sql`, `internal/service/report_image_service.go`, `internal/repository/report_image_repository.go`, `internal/handler/report_image_handler.go`) — template for the image gallery.
- `gmr-admin/components/reports/report-images-manager.tsx` + `gmr-admin/lib/api/report-images.ts` — template for the admin gallery UI.
- `internal/service/cloudflare_images_service.go` — existing R2 upload service, reused as-is (no new storage code).

## 1. Backend (`gmr-backend`)

### 1.1 `industry_news` table and domain

New migration `0XX_create_industry_news_table.sql`, copied from `013_create_blogs_table.sql` / `014_create_press_releases_table.sql`. Columns:

- `id`, `title varchar(200) not null`, `slug varchar(250) unique not null`
- `excerpt varchar(500) not null`, `content text not null`
- `category_id → categories(id)`, `tags varchar(500)`
- `author_id → authors(id)`
- `status varchar(20) default 'draft'` (draft / pending_review / published, matching `PressReleaseStatus`/`BlogStatus`)
- `publish_date`, `scheduled_publish_enabled boolean default false`
- `location varchar(255)`
- `metadata jsonb` (MetaTitle, MetaDescription, Keywords — same shape as `PressReleaseMetadata`)
- `internal_links jsonb`
- `reviewed_by → authors(id) nullable`, `reviewed_at`
- `deleted_at` (soft delete), `created_at`, `updated_at`

Go struct: `internal/domain/industry_news/industry_news.go`, GORM tags matching `PressRelease` field-for-field (including `CreateIndustryNewsRequest`, `UpdateIndustryNewsRequest`, `GetIndustryNewsQuery`, `IndustryNewsListResponse`, `IndustryNewsResponse` DTOs). Reuses existing `category.Category` and `author.Author` domain types — no new FK targets.

### 1.2 Default author seed

Same migration (or a follow-up one) inserts an `authors` row with `name = 'Globe Market Research'` guarded by `INSERT INTO authors (...) SELECT ... WHERE NOT EXISTS (SELECT 1 FROM authors WHERE name = 'Globe Market Research')`, so it's idempotent and safe to run alongside existing author data.

### 1.3 Layers

- `internal/repository/industry_news_repository.go`
- `internal/service/industry_news_service.go`
- `internal/handler/industry_news_handler.go`

All copied from the `press_release` equivalents, including soft-delete/restore, submit-review/publish/unpublish/schedule/cancel-schedule transitions.

### 1.4 Image gallery

- Migration for `industry_news_images` table, copied from `010_create_report_images_table.sql`: `id, industry_news_id → industry_news(id), image_url, title, is_active, uploaded_by, created_at, updated_at`.
- `internal/repository/industry_news_image_repository.go`, `internal/service/industry_news_image_service.go` (calls the existing `cloudflare_images_service.go` `Upload()` — no changes to that service), `internal/handler/industry_news_image_handler.go` (mirrors `report_image_handler.go`'s `UploadImage()`).

### 1.5 Routes (`cmd/api/main.go`)

```
GET/POST   /v1/industry-news
GET        /v1/industry-news/slug/:slug
GET/PUT/DELETE /v1/industry-news/:id
PATCH      /v1/industry-news/:id/submit-review
PATCH      /v1/industry-news/:id/publish
PATCH      /v1/industry-news/:id/unpublish
PATCH      /v1/industry-news/:id/soft-delete
PATCH      /v1/industry-news/:id/restore
PATCH      /v1/industry-news/:id/schedule
PATCH      /v1/industry-news/:id/cancel-schedule
POST       /v1/industry-news/:id/images        (admin/editor only, multipart "image" + optional "title")
GET        /v1/categories/:slug/industry-news
GET        /v1/sitemap/industry-news
```

## 2. Admin (`gmr-admin`)

- Routes: `app/(dashboard)/industry-news/page.tsx` (list), `.../new/page.tsx`, `.../[id]/page.tsx` (edit), `.../[id]/preview/`, `.../trash/page.tsx` — copied from `press-releases/`.
- Components: `components/industry-news/industry-news-form.tsx`, `-list.tsx`, `-filters.tsx`, `industry-news-images-manager.tsx` (copy of `report-images-manager.tsx`, retargeted at `/v1/industry-news/:id/images`, using the same upload/gallery/copy-URL-to-clipboard UX for pasting into the TipTap content editor).
- `industry-news-form.tsx` reuses `AuthorSelector` (`components/statistics/author-selector.tsx`) exactly as PR/Statistics do. On **create only**, once the authors list loads, the form finds the author named "Globe Market Research" and defaults `authorId` to its id (instead of `0`). Still user-editable.
- Data layer: `hooks/use-industry-news.ts`, `hooks/use-industry-news-list.ts`, `lib/api/industry-news.ts` (direct `/v1/industry-news*` client, no rebrand indirection like Statistics→blog), `lib/types/industry-news.ts`.
- Sidebar nav: add "Industry News" entry next to Press Releases / Statistics.

## 3. Frontend (`gmr`)

- Routes: `app/industry-news/page.tsx` (list) + `loading.tsx`, `app/industry-news/[slug]/page.tsx` (detail) + `loading.tsx`. Single consistent plural path for both list and detail (deliberately avoiding the singular/plural inconsistency in PR's `/press-releases` vs `/press-release/[slug]`).
- Components: `components/industry-news/IndustryNewsCard.tsx`, `IndustryNewsListCard.tsx`, `IndustryNewsListingClient.tsx`, `RelatedIndustryNewsSection.tsx` — copied and renamed from the PR equivalents.
- Author display: same `AuthorHoverCard` component/pattern as PR (`components/authors/AuthorHoverCard.tsx`). Since `authorDetails` will be populated from a real author row, the plain-string fallback path stays as a safety net only.
- Nav/menu/sitemap: add "Industry News" link wherever PR/Statistics currently appear.

## Out of scope

- No changes to the Cloudflare R2 upload service itself.
- No changes to existing PR/Statistics/Report modules.
- No new `categories` scoping — Industry News reuses the existing shared `categories` table, same as PR and Statistics.
