-- Migration: Create redirects table
-- Run order: 022

CREATE TABLE IF NOT EXISTS redirects (
  id              BIGSERIAL PRIMARY KEY,
  source_url      VARCHAR(500) NOT NULL,
  destination_url VARCHAR(500),                          -- NULL allowed for 410/451 (no destination)
  redirect_type   INTEGER NOT NULL DEFAULT 301,
  is_enabled      BOOLEAN NOT NULL DEFAULT TRUE,
  hit_count       BIGINT  NOT NULL DEFAULT 0,
  notes           VARCHAR(500),
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  deleted_at      TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_redirects_source_url
  ON redirects(source_url)
  WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_redirects_enabled
  ON redirects(is_enabled)
  WHERE deleted_at IS NULL;

ALTER TABLE redirects
  ADD CONSTRAINT chk_redirect_type
  CHECK (redirect_type IN (301, 302, 307, 308, 410, 451));
