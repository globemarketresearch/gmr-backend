-- Add report_id and report_link_text columns to media_mentions table
ALTER TABLE media_mentions
    ADD COLUMN report_id INTEGER REFERENCES reports(id) ON DELETE SET NULL,
    ADD COLUMN report_link_text VARCHAR(255);

-- Create index for querying media mentions by report
CREATE INDEX idx_media_mentions_report_id ON media_mentions(report_id);
