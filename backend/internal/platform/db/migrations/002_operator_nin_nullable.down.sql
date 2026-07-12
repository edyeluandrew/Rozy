-- Revert only if no NULL nin_hash rows exist
ALTER TABLE operator_profiles ALTER COLUMN nin_hash SET NOT NULL;
