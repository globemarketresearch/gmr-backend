-- Set the reports ID sequence to start from 1001
-- This ensures new reports will have IDs starting from 1001

-- Only set the sequence if it's currently below 1001
SELECT setval('reports_id_seq', 1001, false);

-- The 'false' parameter means the next value will be 1001
-- If the sequence is already at or above 1001, this won't decrease it
