CREATE TABLE IF NOT EXISTS transcription_feedback (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    language_hint VARCHAR(20) NOT NULL,
    rating VARCHAR(30) NOT NULL,
    correction_length INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_transcription_feedback_user_id ON transcription_feedback(user_id);
