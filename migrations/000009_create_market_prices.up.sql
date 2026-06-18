CREATE TABLE IF NOT EXISTS market_prices (
    id UUID PRIMARY KEY,
    commodity VARCHAR(100) NOT NULL,
    market_name VARCHAR(200) NOT NULL,
    district VARCHAR(100) NOT NULL,
    price NUMERIC(12,2) NOT NULL,
    currency VARCHAR(10) NOT NULL DEFAULT 'SLE',
    unit VARCHAR(50) NOT NULL,
    source VARCHAR(200),
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_market_prices_commodity ON market_prices(commodity);
CREATE INDEX IF NOT EXISTS idx_market_prices_district ON market_prices(district);
CREATE INDEX IF NOT EXISTS idx_market_prices_created_at ON market_prices(created_at);
