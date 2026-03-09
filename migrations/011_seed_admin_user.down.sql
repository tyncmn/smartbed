-- Rollback 011: remove seeded default admin user

DELETE FROM users
WHERE email = 'admin@smartbed.local';
