-- Idempotent provider transaction IDs for wallet webhooks
CREATE UNIQUE INDEX IF NOT EXISTS idx_wallet_recharges_external_tx
  ON wallet_recharges (external_tx_id)
  WHERE external_tx_id IS NOT NULL;
