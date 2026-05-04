-- Migration: Remove access_type column from reports table
-- Created: 2026-01-04
-- Description: Removes the access_type field as it is no longer needed for report management

-- Drop access_type column
ALTER TABLE reports DROP COLUMN IF EXISTS access_type;
