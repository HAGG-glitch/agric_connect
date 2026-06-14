-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Conversations table
CREATE TABLE IF NOT EXISTS ai_conversations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    title VARCHAR(200) NOT NULL,
    preferred_language VARCHAR(20) NOT NULL DEFAULT 'english',
    district VARCHAR(100),
    crop VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON ai_conversations(user_id);
CREATE INDEX IF NOT EXISTS idx_conversations_updated_at ON ai_conversations(updated_at DESC);

-- Messages table
CREATE TABLE IF NOT EXISTS ai_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    language VARCHAR(20),
    model VARCHAR(150),
    input_tokens INTEGER,
    output_tokens INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON ai_messages(conversation_id);

-- Agricultural documents table
CREATE TABLE IF NOT EXISTS agricultural_documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    crop VARCHAR(100),
    category VARCHAR(100) NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    language VARCHAR(20) NOT NULL DEFAULT 'english',
    source TEXT,
    reviewed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agri_docs_crop ON agricultural_documents(crop);
CREATE INDEX IF NOT EXISTS idx_agri_docs_category ON agricultural_documents(category);

-- Weather cache table
CREATE TABLE IF NOT EXISTS weather_cache (
    district VARCHAR(100) PRIMARY KEY,
    response JSONB NOT NULL,
    fetched_at TIMESTAMPTZ NOT NULL
);
