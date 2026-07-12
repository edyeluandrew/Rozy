# Rozy architecture

## System overview

```
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│  Flutter        │  │  Flutter        │  │  React Admin    │
│  Passenger App  │  │  Driver App     │  │  Dashboard      │
└────────┬────────┘  └────────┬────────┘  └────────┬────────┘
         │                    │                    │
         └────────────────────┼────────────────────┘
                              │
                    ┌─────────▼─────────┐
                    │  Go API (monolith) │
                    │  REST + WebSocket  │
                    └─────────┬─────────┘
         ┌────────────────────┼────────────────────┐
         │                    │                    │
  ┌──────▼──────┐    ┌────────▼────────┐   ┌──────▼──────┐
  │ Neon Postgres│    │ Redis GEO       │   │ Object      │
  │ + PostGIS    │    │ live operators  │   │ storage     │
  └─────────────┘    └─────────────────┘   └─────────────┘
         │
  ┌──────▼──────┐    ┌─────────────────┐
  │ Mapbox      │    │ FCM + MTN/Airtel│
  │ + Rozy POIs │    │ webhooks        │
  └─────────────┘    └─────────────────┘
```

## Backend modules

| Module | Responsibility |
|--------|----------------|
| `auth` | OTP, JWT, sessions |
| `user` | Passenger profiles |
| `operator` | Driver profiles, online/offline, one-mode rule |
| `verification` | Document submissions, NIN uniqueness, admin approval |
| `wallet` | Balance, MTN/Airtel recharge webhooks, trip fee deduction |
| `trip` | Lifecycle, state machine, PIN |
| `dispatch` | Redis GEO search, rank, `SELECT FOR UPDATE` lock |
| `location` | GPS ingest, WebSocket broadcast |
| `fare` | base + per_km, min fare, rounding, GPS final distance |
| `notification` | FCM |
| `safety` | SOS, incidents, trip share |
| `admin` | Verification queue, blocks, config |
| `places` | Mbarara POI + pin-drop support |

## Trip state machine

```
requested → searching → driver_assigned → driver_arriving → in_progress → completed
                ↓              ↓                ↓               ↓
             expired       cancelled         cancelled       cancelled / disputed
```

## Driver operator statuses

`pending_verification` · `offline` · `available` · `busy` · `wallet_blocked` · `suspended` · `expired_docs_blocked`

## Location strategy (unmapped areas)

1. **Pin drop always available** (lat/lng — no address required)
2. **Rozy `places` table** for Mbarara landmarks
3. **Mapbox geocoding** primary autocomplete
4. **OSM/Nominatim** fallback (cached, rate-limited)
5. **Fare estimate:** Mapbox route distance, or haversine × road factor
6. **Final fare:** GPS polyline distance during trip

## Revenue flow

1. Passenger pays driver **cash** at trip end (shown fare from GPS km)
2. Backend deducts **fixed Rozy fee** from driver wallet on `completed`
3. If wallet below minimum → `wallet_blocked`, cannot go online
