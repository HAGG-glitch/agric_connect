-- Reddit-style reviews: allow multiple reviews per diagnosis, farmers accept/reject
DROP INDEX IF EXISTS idx_diagnosis_reviews_unique_active;

ALTER TABLE diagnosis_reviews
    ADD COLUMN IF NOT EXISTS is_accepted BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS is_hidden   BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_diagnosis_reviews_is_accepted ON diagnosis_reviews(is_accepted);
CREATE INDEX IF NOT EXISTS idx_diagnosis_reviews_is_hidden   ON diagnosis_reviews(is_hidden);
