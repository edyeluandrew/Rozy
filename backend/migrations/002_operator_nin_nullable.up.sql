-- Allow operator registration before NIN is submitted in verification wizard
ALTER TABLE operator_profiles ALTER COLUMN nin_hash DROP NOT NULL;
