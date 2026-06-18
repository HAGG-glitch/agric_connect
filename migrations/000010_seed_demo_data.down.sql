-- Remove seeded demo data
DELETE FROM market_prices WHERE source IS NOT NULL;

DELETE FROM users WHERE phone_number IN (
  '23276100001', '23276100002', '23276100003', '23276100004'
);
