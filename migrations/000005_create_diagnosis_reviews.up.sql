CREATE TABLE IF NOT EXISTS diagnosis_reviews (
    id UUID PRIMARY KEY,
    diagnosis_id UUID NOT NULL REFERENCES crop_diagnoses(id) ON DELETE CASCADE,
    officer_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    review_status VARCHAR(30) NOT NULL DEFAULT 'pending',
    confirmed_condition VARCHAR(255),
    officer_comment TEXT,
    recommendation TEXT,
    urgency VARCHAR(20),
    requires_field_visit BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_diagnosis_reviews_diagnosis_id ON diagnosis_reviews(diagnosis_id);
CREATE INDEX IF NOT EXISTS idx_diagnosis_reviews_officer_id ON diagnosis_reviews(officer_id);
CREATE INDEX IF NOT EXISTS idx_diagnosis_reviews_review_status ON diagnosis_reviews(review_status);
CREATE UNIQUE INDEX IF NOT EXISTS idx_diagnosis_reviews_unique_active ON diagnosis_reviews(diagnosis_id) WHERE review_status NOT IN ('closed');
