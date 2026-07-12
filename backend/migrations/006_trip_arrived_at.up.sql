-- Driver at-pickup timestamp (status stays driver_arriving until PIN start)
ALTER TABLE trips ADD COLUMN IF NOT EXISTS arrived_at TIMESTAMPTZ;
