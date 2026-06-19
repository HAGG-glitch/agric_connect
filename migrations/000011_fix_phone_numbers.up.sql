-- Fix phone numbers in seed data to include +232 prefix
-- NormalizePhone prepends +232, so bare numbers like "23276100001" become "+23223276100001"
-- If a +232-version row already exists (e.g. from re-registration), delete the stale bare-number row.

DELETE FROM users WHERE phone_number = '23276100001' AND EXISTS (SELECT 1 FROM users u2 WHERE u2.phone_number = '+23276100001');
DELETE FROM users WHERE phone_number = '23276100002' AND EXISTS (SELECT 1 FROM users u2 WHERE u2.phone_number = '+23276100002');
DELETE FROM users WHERE phone_number = '23276100003' AND EXISTS (SELECT 1 FROM users u2 WHERE u2.phone_number = '+23276100003');
DELETE FROM users WHERE phone_number = '23276100004' AND EXISTS (SELECT 1 FROM users u2 WHERE u2.phone_number = '+23276100004');

UPDATE users SET phone_number = '+23276100001' WHERE phone_number = '23276100001';
UPDATE users SET phone_number = '+23276100002' WHERE phone_number = '23276100002';
UPDATE users SET phone_number = '+23276100003' WHERE phone_number = '23276100003';
UPDATE users SET phone_number = '+23276100004' WHERE phone_number = '23276100004';
