ALTER TABLE diagnosis_reviews
  DROP COLUMN IF EXISTS request_status,
  DROP COLUMN IF EXISTS site_visit_date;
