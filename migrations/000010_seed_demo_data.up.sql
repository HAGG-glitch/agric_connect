-- Seed demo data for MVP demonstration
-- Password for all demo users: demo123

INSERT INTO users (id, full_name, phone_number, district, preferred_language, role, password_hash, is_active, created_at, updated_at)
SELECT gen_random_uuid(), 'Admin User', '+23276100001', 'Western Area Urban', 'english', 'admin', '$2a$10$syvtYjgisY5ioxy3v9UHLOoddeID9Ok0RMU23tpwY7oZIGQUsFCt.', true, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM users WHERE phone_number = '+23276100001' AND role = 'admin');

INSERT INTO users (id, full_name, phone_number, district, preferred_language, role, password_hash, is_active, created_at, updated_at)
SELECT gen_random_uuid(), 'Fatmata Kamara', '+23276100002', 'Bombali', 'english', 'officer', '$2a$10$syvtYjgisY5ioxy3v9UHLOoddeID9Ok0RMU23tpwY7oZIGQUsFCt.', true, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM users WHERE phone_number = '+23276100002');

INSERT INTO users (id, full_name, phone_number, district, preferred_language, role, password_hash, is_active, created_at, updated_at)
SELECT gen_random_uuid(), 'Amadu Sesay', '+23276100003', 'Kenema', 'krio', 'officer', '$2a$10$syvtYjgisY5ioxy3v9UHLOoddeID9Ok0RMU23tpwY7oZIGQUsFCt.', true, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM users WHERE phone_number = '+23276100003');

INSERT INTO users (id, full_name, phone_number, district, preferred_language, role, password_hash, is_active, created_at, updated_at)
SELECT gen_random_uuid(), 'Demo Farmer', '+23276100004', 'Port Loko', 'english', 'farmer', '$2a$10$syvtYjgisY5ioxy3v9UHLOoddeID9Ok0RMU23tpwY7oZIGQUsFCt.', true, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM users WHERE phone_number = '+23276100004');

-- Seed market prices (sample data for MVP)
INSERT INTO market_prices (id, commodity, market_name, district, price, currency, unit, source, is_verified, created_at, updated_at)
SELECT gen_random_uuid(), 'rice', 'Central Market', 'Western Area Urban', 28000, 'SLE', '50kg bag', 'Ministry of Agriculture', true, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM market_prices WHERE commodity = 'rice' AND district = 'Western Area Urban' AND market_name = 'Central Market');

INSERT INTO market_prices (id, commodity, market_name, district, price, currency, unit, source, is_verified, created_at, updated_at)
SELECT gen_random_uuid(), 'cassava', 'Lungi Market', 'Port Loko', 5000, 'SLE', 'pile', 'Field Survey', true, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM market_prices WHERE commodity = 'cassava' AND district = 'Port Loko' AND market_name = 'Lungi Market');

INSERT INTO market_prices (id, commodity, market_name, district, price, currency, unit, source, is_verified, created_at, updated_at)
SELECT gen_random_uuid(), 'groundnut', 'Bo Market', 'Bo', 1500, 'SLE', 'cup', 'Market Report', true, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM market_prices WHERE commodity = 'groundnut' AND district = 'Bo' AND market_name = 'Bo Market');

INSERT INTO market_prices (id, commodity, market_name, district, price, currency, unit, source, is_verified, created_at, updated_at)
SELECT gen_random_uuid(), 'palm oil', 'Makeni Market', 'Bombali', 12000, 'SLE', 'gallon', 'Ministry of Agriculture', true, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM market_prices WHERE commodity = 'palm oil' AND district = 'Bombali' AND market_name = 'Makeni Market');

INSERT INTO market_prices (id, commodity, market_name, district, price, currency, unit, source, is_verified, created_at, updated_at)
SELECT gen_random_uuid(), 'cocoa', 'Kenema Market', 'Kenema', 8000, 'SLE', 'kg', 'Field Survey', true, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM market_prices WHERE commodity = 'cocoa' AND district = 'Kenema' AND market_name = 'Kenema Market');

INSERT INTO market_prices (id, commodity, market_name, district, price, currency, unit, source, is_verified, created_at, updated_at)
SELECT gen_random_uuid(), 'rice', 'Kailahun Market', 'Kailahun', 26000, 'SLE', '50kg bag', 'Ministry of Agriculture', true, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM market_prices WHERE commodity = 'rice' AND district = 'Kailahun' AND market_name = 'Kailahun Market');

INSERT INTO market_prices (id, commodity, market_name, district, price, currency, unit, source, is_verified, created_at, updated_at)
SELECT gen_random_uuid(), 'cassava', 'Waterloo Market', 'Western Area Rural', 4500, 'SLE', 'pile', 'Market Report', true, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM market_prices WHERE commodity = 'cassava' AND district = 'Western Area Rural' AND market_name = 'Waterloo Market');

INSERT INTO market_prices (id, commodity, market_name, district, price, currency, unit, source, is_verified, created_at, updated_at)
SELECT gen_random_uuid(), 'maize', 'Kono Market', 'Kono', 3500, 'SLE', 'kg', 'Field Survey', false, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM market_prices WHERE commodity = 'maize' AND district = 'Kono' AND market_name = 'Kono Market');
