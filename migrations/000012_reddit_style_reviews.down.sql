DROP INDEX IF EXISTS idx_diagnosis_reviews_is_accepted;
DROP INDEX IF EXISTS idx_diagnosis_reviews_is_hidden;

ALTER TABLE diagnosis_reviews
    DROP COLUMN IF EXISTS is_accepted,
    DROP COLUMN IF EXISTS is_hidden;

CREATE UNIQUE INDEX IF NOT EXISTS idx_diagnosis_reviews_unique_active ON diagnosis_reviews(diagnosis_id) WHERE review_status NOT IN ('closed');
