-- Operator last known location for dispatch fallback when Redis unavailable
ALTER TABLE operator_profiles ADD COLUMN IF NOT EXISTS last_lat DOUBLE PRECISION;
ALTER TABLE operator_profiles ADD COLUMN IF NOT EXISTS last_lng DOUBLE PRECISION;
ALTER TABLE operator_profiles ADD COLUMN IF NOT EXISTS last_location_at TIMESTAMPTZ;
