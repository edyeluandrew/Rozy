-- Admin seed user for Mbarara ops (OTP login with this phone)
INSERT INTO users (phone, role)
VALUES ('+256700000000', 'admin')
ON CONFLICT (phone) DO UPDATE SET role = 'admin';

ALTER TABLE operator_profiles ADD COLUMN IF NOT EXISTS nin_last4 TEXT;
