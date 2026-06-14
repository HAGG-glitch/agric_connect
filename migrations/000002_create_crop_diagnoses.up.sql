CREATE TABLE IF NOT EXISTS crop_diagnoses (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,

    crop VARCHAR(100) NOT NULL,
    district VARCHAR(100),
    preferred_language VARCHAR(20) NOT NULL DEFAULT 'english',

    plant_part VARCHAR(100),
    symptom_description TEXT NOT NULL,
    symptoms_started_at DATE,
    affected_percentage NUMERIC(5,2),

    recent_weather TEXT,
    fertiliser_history TEXT,
    pesticide_history TEXT,

    image_storage_path TEXT NOT NULL,
    image_original_name VARCHAR(255),
    image_content_type VARCHAR(100) NOT NULL,
    image_size_bytes BIGINT NOT NULL,
    image_sha256 VARCHAR(64),

    probable_condition VARCHAR(255),
    confidence NUMERIC(5,2),
    confidence_label VARCHAR(20),
    description TEXT,

    observed_signs JSONB NOT NULL DEFAULT '[]'::jsonb,
    possible_alternatives JSONB NOT NULL DEFAULT '[]'::jsonb,
    recommended_actions JSONB NOT NULL DEFAULT '[]'::jsonb,
    prevention_tips JSONB NOT NULL DEFAULT '[]'::jsonb,

    urgency VARCHAR(20),
    requires_expert_review BOOLEAN NOT NULL DEFAULT TRUE,
    disclaimer TEXT,

    raw_ai_result JSONB,
    model VARCHAR(150),

    status VARCHAR(30) NOT NULL DEFAULT 'processing',
    error_message TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_crop_diagnoses_user_id ON crop_diagnoses(user_id);
CREATE INDEX IF NOT EXISTS idx_crop_diagnoses_created_at ON crop_diagnoses(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_crop_diagnoses_crop ON crop_diagnoses(crop);
CREATE INDEX IF NOT EXISTS idx_crop_diagnoses_condition ON crop_diagnoses(probable_condition);
CREATE INDEX IF NOT EXISTS idx_crop_diagnoses_status ON crop_diagnoses(status);
