-- Expand form_submissions.category column to accommodate 'request-customization' (22 chars)
ALTER TABLE form_submissions ALTER COLUMN category TYPE varchar(30);
