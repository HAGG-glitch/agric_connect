CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    full_name VARCHAR(200) NOT NULL,
    phone_number VARCHAR(30) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    district VARCHAR(100),
    preferred_language VARCHAR(20) NOT NULL DEFAULT 'english',
    role VARCHAR(20) NOT NULL DEFAULT 'farmer',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_phone_number ON users(phone_number);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
