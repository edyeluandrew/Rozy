# Rozy API (MVP)

Base URL: `https://api.rozy.app/v1` (dev: `http://localhost:8080/v1`)

Auth: `Authorization: Bearer <jwt>`

---

## Auth

| Method | Path | Description |
|--------|------|-------------|
| POST | `/auth/otp/request` | `{ "phone": "+256..." }` |
| POST | `/auth/otp/verify` | `{ "phone", "code" }` → JWT |
| POST | `/auth/refresh` | Refresh token |

---

## Passenger

| Method | Path | Description |
|--------|------|-------------|
| GET | `/passenger/profile` | Profile |
| PATCH | `/passenger/profile` | Update name, photo |
| GET | `/places/search?q=` | POI + geocode results |
| POST | `/fare/estimate` | `{ pickup, dest, ride_type }` → estimate |
| POST | `/trips` | Request ride |
| GET | `/trips/:id` | Trip detail + driver snapshot |
| POST | `/trips/:id/cancel` | Cancel |
| GET | `/trips/history` | Past trips |

---

## Driver / operator

| Method | Path | Description |
|--------|------|-------------|
| POST | `/operator/register` | `{ ride_type: "boda"\|"car_basic"\|"car_xl" }` — **once per account** |
| GET | `/operator/profile` | `{ registered: bool, operator? }` |
| POST | `/operator/verification/submit` | Upload docs metadata + storage keys |
| GET | `/operator/wallet` | Balance + min required |
| POST | `/operator/wallet/recharge` | `{ amount, provider: "mtn"\|"airtel" }` |
| POST | `/operator/online` | Go online (checks wallet + verified) |
| POST | `/operator/offline` | Go offline |
| POST | `/operator/location` | `{ lat, lng, heading?, speed? }` |
| GET | `/operator/trips/incoming` | Pending request (if any) |
| POST | `/operator/trips/:id/accept` | Accept |
| POST | `/operator/trips/:id/reject` | Reject |
| POST | `/operator/trips/:id/arrived` | At pickup |
| POST | `/operator/trips/:id/start` | `{ pin }` verify & start |
| POST | `/operator/trips/:id/complete` | Complete trip |

---

## Wallet webhooks (server-to-server)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/webhooks/mtn` | MTN MoMo payment callback |
| POST | `/webhooks/airtel` | Airtel Money callback |

Idempotent on provider transaction ID.

---

## Safety

| Method | Path | Description |
|--------|------|-------------|
| POST | `/trips/:id/share` | `{ phone }` trip share link |
| POST | `/trips/:id/sos` | Trigger SOS |
| POST | `/trips/:id/incident` | Report issue |

---

## Admin

| Method | Path | Description |
|--------|------|-------------|
| GET | `/admin/verification/queue` | Pending submissions |
| POST | `/admin/verification/:id/approve` | Approve |
| POST | `/admin/verification/:id/reject` | `{ reason }` |
| GET | `/admin/trips/active` | Live trips |
| POST | `/admin/operators/:id/suspend` | Suspend |
| GET/PATCH | `/admin/fare-rules` | Mbarara fare config |
| GET/POST | `/admin/places` | POI management |

---

## WebSocket

Connect: `wss://api.rozy.app/v1/ws?token=<jwt>`

### Client → server

| Event | Payload |
|-------|---------|
| `location:update` | `{ lat, lng, trip_id? }` |
| `trip:join` | `{ trip_id }` |

### Server → client

| Event | Payload |
|-------|---------|
| `trip:assigned` | trip + driver snapshot |
| `trip:status` | `{ trip_id, status }` |
| `trip:driver_location` | `{ lat, lng, eta_seconds? }` |
| `operator:ride_request` | incoming trip for driver |
| `wallet:updated` | `{ balance }` |
| `notification` | generic push payload |

---

## Trip statuses (API values)

`requested` · `searching` · `driver_assigned` · `driver_arriving` · `in_progress` · `completed` · `cancelled` · `expired` · `disputed`

## Ride types

`boda` · `car_basic` · `car_xl`
