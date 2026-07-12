-- Rozy MVP schema
-- Requires: PostgreSQL 15+ with PostGIS on Neon

CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Enums
CREATE TYPE user_role AS ENUM ('passenger', 'driver', 'admin');
CREATE TYPE operator_type AS ENUM ('boda', 'car');
CREATE TYPE ride_type AS ENUM ('boda', 'car_basic', 'car_xl');
CREATE TYPE operator_status AS ENUM (
  'pending_verification',
  'offline',
  'available',
  'busy',
  'wallet_blocked',
  'suspended',
  'expired_docs_blocked'
);
CREATE TYPE verification_status AS ENUM ('pending', 'approved', 'rejected', 'resubmit');
CREATE TYPE trip_status AS ENUM (
  'requested',
  'searching',
  'driver_assigned',
  'driver_arriving',
  'in_progress',
  'completed',
  'cancelled',
  'expired',
  'disputed'
);
CREATE TYPE wallet_tx_type AS ENUM ('trip_fee', 'recharge', 'adjustment', 'refund');
CREATE TYPE recharge_status AS ENUM ('pending', 'completed', 'failed', 'expired');
CREATE TYPE payment_provider AS ENUM ('mtn', 'airtel', 'admin_manual');
CREATE TYPE car_category AS ENUM ('basic', 'xl');
CREATE TYPE doc_type AS ENUM (
  'nin_front', 'nin_back', 'selfie', 'permit', 'logbook',
  'insurance', 'vehicle_front', 'vehicle_side', 'plate_closeup', 'interior'
);
CREATE TYPE incident_type AS ENUM ('sos', 'route_deviation', 'long_stop', 'wrong_vehicle', 'off_app', 'other');

-- Cities
CREATE TABLE cities (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  center GEOGRAPHY(POINT, 4326),
  is_active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO cities (name, slug, center) VALUES
  ('Mbarara', 'mbarara', ST_SetSRID(ST_MakePoint(30.6586, -0.6072), 4326)::geography);

-- Users
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  phone TEXT NOT NULL UNIQUE,
  role user_role NOT NULL DEFAULT 'passenger',
  is_active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE passenger_profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  full_name TEXT,
  photo_url TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One driver user → one operator profile
CREATE TABLE operator_profiles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
  operator_type operator_type NOT NULL,
  ride_type ride_type NOT NULL,
  status operator_status NOT NULL DEFAULT 'pending_verification',
  full_name TEXT,
  photo_url TEXT,
  nin_hash TEXT NOT NULL, -- SHA-256 of NIN; one NIN → one account
  rating_avg NUMERIC(3,2) NOT NULL DEFAULT 5.00,
  total_trips INT NOT NULL DEFAULT 0,
  wallet_balance BIGINT NOT NULL DEFAULT 0, -- UGX integer
  wallet_min_balance BIGINT NOT NULL DEFAULT 5000,
  city_id UUID REFERENCES cities(id),
  last_online_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT operator_ride_type_match CHECK (
    (operator_type = 'boda' AND ride_type = 'boda') OR
    (operator_type = 'car' AND ride_type IN ('car_basic', 'car_xl'))
  )
);

CREATE UNIQUE INDEX idx_operator_nin_hash ON operator_profiles(nin_hash);

CREATE TABLE boda_details (
  operator_profile_id UUID PRIMARY KEY REFERENCES operator_profiles(id) ON DELETE CASCADE,
  plate TEXT NOT NULL,
  bike_make TEXT,
  bike_model TEXT,
  bike_color TEXT,
  permit_number TEXT NOT NULL,
  permit_expiry DATE NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE car_details (
  operator_profile_id UUID PRIMARY KEY REFERENCES operator_profiles(id) ON DELETE CASCADE,
  plate TEXT NOT NULL,
  category car_category NOT NULL,
  capacity INT NOT NULL CHECK (capacity BETWEEN 1 AND 8),
  make TEXT,
  model TEXT,
  color TEXT,
  permit_number TEXT NOT NULL,
  permit_expiry DATE NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT car_category_ride_type CHECK (
    (category = 'basic') OR (category = 'xl')
  )
);

CREATE UNIQUE INDEX idx_boda_plate ON boda_details(plate);
CREATE UNIQUE INDEX idx_car_plate ON car_details(plate);

-- Verification
CREATE TABLE verification_submissions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  operator_profile_id UUID NOT NULL REFERENCES operator_profiles(id) ON DELETE CASCADE,
  status verification_status NOT NULL DEFAULT 'pending',
  rejection_reason TEXT,
  reviewed_by UUID REFERENCES users(id),
  reviewed_at TIMESTAMPTZ,
  submitted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE documents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  submission_id UUID NOT NULL REFERENCES verification_submissions(id) ON DELETE CASCADE,
  doc_type doc_type NOT NULL,
  storage_key TEXT NOT NULL,
  sha256_hash TEXT NOT NULL,
  mime_type TEXT,
  expires_at DATE,
  uploaded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE verified_operator_snapshots (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  operator_profile_id UUID NOT NULL REFERENCES operator_profiles(id) ON DELETE CASCADE,
  submission_id UUID NOT NULL REFERENCES verification_submissions(id),
  legal_name TEXT NOT NULL,
  nin_last4 TEXT NOT NULL,
  plate TEXT NOT NULL,
  ride_type ride_type NOT NULL,
  face_photo_key TEXT NOT NULL,
  vehicle_photo_keys TEXT[] NOT NULL DEFAULT '{}',
  permit_expiry DATE NOT NULL,
  insurance_expiry DATE NOT NULL,
  approved_by UUID NOT NULL REFERENCES users(id),
  approved_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Fare rules (per city + ride type)
CREATE TABLE fare_rules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  city_id UUID NOT NULL REFERENCES cities(id) ON DELETE CASCADE,
  ride_type ride_type NOT NULL,
  base_fare BIGINT NOT NULL,
  per_km_rate BIGINT NOT NULL,
  min_fare BIGINT NOT NULL,
  rozy_fee_fixed BIGINT NOT NULL,
  round_to BIGINT NOT NULL DEFAULT 500,
  road_factor_fallback NUMERIC(4,2) NOT NULL DEFAULT 1.30,
  min_billable_km NUMERIC(6,3) NOT NULL DEFAULT 0.5,
  is_active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (city_id, ride_type)
);

-- Seed Mbarara fare placeholders (tune after field survey)
INSERT INTO fare_rules (city_id, ride_type, base_fare, per_km_rate, min_fare, rozy_fee_fixed, road_factor_fallback)
SELECT c.id, rt.ride_type, rt.base_fare, rt.per_km, rt.min_fare, rt.rozy_fee, rt.road_factor
FROM cities c
CROSS JOIN (VALUES
  ('boda'::ride_type, 1500, 500, 2000, 400, 1.25),
  ('car_basic'::ride_type, 3000, 1200, 4000, 1000, 1.35),
  ('car_xl'::ride_type, 5000, 1800, 6000, 1500, 1.35)
) AS rt(ride_type, base_fare, per_km, min_fare, rozy_fee, road_factor)
WHERE c.slug = 'mbarara';

-- Places (Mbarara POIs + pin-drop support)
CREATE TABLE places (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  city_id UUID NOT NULL REFERENCES cities(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  landmark_note TEXT,
  location GEOGRAPHY(POINT, 4326) NOT NULL,
  category TEXT,
  is_active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_places_location ON places USING GIST (location);

-- Trips
CREATE TABLE trips (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  passenger_id UUID NOT NULL REFERENCES users(id),
  operator_profile_id UUID REFERENCES operator_profiles(id),
  city_id UUID NOT NULL REFERENCES cities(id),
  ride_type ride_type NOT NULL,
  status trip_status NOT NULL DEFAULT 'requested',
  pickup GEOGRAPHY(POINT, 4326) NOT NULL,
  pickup_address TEXT,
  pickup_landmark TEXT,
  destination GEOGRAPHY(POINT, 4326) NOT NULL,
  dest_address TEXT,
  dest_landmark TEXT,
  estimated_fare BIGINT,
  final_fare BIGINT,
  estimated_km NUMERIC(8,3),
  actual_km NUMERIC(8,3),
  rozy_fee BIGINT,
  trip_pin_hash TEXT,
  cancel_reason TEXT,
  cancelled_by UUID REFERENCES users(id),
  requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  assigned_at TIMESTAMPTZ,
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  cancelled_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_trips_passenger ON trips(passenger_id, created_at DESC);
CREATE INDEX idx_trips_operator ON trips(operator_profile_id, created_at DESC);
CREATE INDEX idx_trips_status ON trips(status) WHERE status IN ('searching', 'driver_assigned', 'driver_arriving', 'in_progress');

CREATE TABLE trip_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  from_status trip_status,
  to_status trip_status NOT NULL,
  actor_id UUID REFERENCES users(id),
  metadata JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE trip_locations (
  id BIGSERIAL PRIMARY KEY,
  trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  operator_profile_id UUID NOT NULL REFERENCES operator_profiles(id),
  location GEOGRAPHY(POINT, 4326) NOT NULL,
  speed NUMERIC(6,2),
  heading NUMERIC(5,2),
  recorded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_trip_locations_trip ON trip_locations(trip_id, recorded_at);

-- Wallet
CREATE TABLE wallet_transactions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  operator_profile_id UUID NOT NULL REFERENCES operator_profiles(id),
  tx_type wallet_tx_type NOT NULL,
  amount BIGINT NOT NULL,
  balance_after BIGINT NOT NULL,
  trip_id UUID REFERENCES trips(id),
  reference TEXT NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_wallet_tx_reference ON wallet_transactions(reference);

CREATE TABLE wallet_recharges (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  operator_profile_id UUID NOT NULL REFERENCES operator_profiles(id),
  amount BIGINT NOT NULL,
  provider payment_provider NOT NULL,
  status recharge_status NOT NULL DEFAULT 'pending',
  external_tx_id TEXT,
  idempotency_key TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Safety
CREATE TABLE safety_incidents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  trip_id UUID NOT NULL REFERENCES trips(id),
  reporter_id UUID NOT NULL REFERENCES users(id),
  incident_type incident_type NOT NULL,
  location GEOGRAPHY(POINT, 4326),
  description TEXT,
  status TEXT NOT NULL DEFAULT 'open',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE trip_shares (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  shared_with_phone TEXT NOT NULL,
  share_token TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Auth helpers
CREATE TABLE otp_codes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  phone TEXT NOT NULL,
  code_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_otp_phone ON otp_codes(phone, created_at DESC);

CREATE TABLE fcm_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token TEXT NOT NULL,
  platform TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, token)
);

-- App config key-value
CREATE TABLE app_config (
  key TEXT PRIMARY KEY,
  value JSONB NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO app_config (key, value) VALUES
  ('dispatch.search_radius_km', '3'),
  ('dispatch.search_expand_km', '8'),
  ('dispatch.accept_timeout_seconds', '30'),
  ('wallet.default_min_balance', '5000');

-- updated_at trigger
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_updated_at BEFORE UPDATE ON users
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER operator_profiles_updated_at BEFORE UPDATE ON operator_profiles
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trips_updated_at BEFORE UPDATE ON trips
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER fare_rules_updated_at BEFORE UPDATE ON fare_rules
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
