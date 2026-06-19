-- Reverse the UPDATE for rows that don't conflict with existing data
UPDATE users SET phone_number = '23276100001' WHERE phone_number = '+23276100001' AND NOT EXISTS (SELECT 1 FROM users u2 WHERE u2.phone_number = '23276100001');
UPDATE users SET phone_number = '23276100002' WHERE phone_number = '+23276100002' AND NOT EXISTS (SELECT 1 FROM users u2 WHERE u2.phone_number = '23276100002');
UPDATE users SET phone_number = '23276100003' WHERE phone_number = '+23276100003' AND NOT EXISTS (SELECT 1 FROM users u2 WHERE u2.phone_number = '23276100003');
UPDATE users SET phone_number = '23276100004' WHERE phone_number = '+23276100004' AND NOT EXISTS (SELECT 1 FROM users u2 WHERE u2.phone_number = '23276100004');
