# Rozy

Local ride-hailing for **Mbarara**, Western Uganda.

Cash rides. Drivers earn the fare; Rozy takes a fixed fee from the **driver wallet**. Built for boda and car operators first.

## Stack

| Layer | Tech |
|-------|------|
| Passenger & driver apps | Flutter |
| API | Go (Chi) |
| Database | Neon PostgreSQL + PostGIS |
| Dispatch | Redis GEO (Postgres fallback) |
| Admin | React + Vite |

## Repo layout

```
backend/   Go API, migrations, wallet webhooks, WebSockets
mobile/    Flutter — passenger (`main_passenger.dart`) & driver (`main_driver.dart`)
admin/     Verification queue, active trips, operators
docs/      Architecture, API, screens, brand
```

## Quick start

### 1. Backend

```bash
cd backend
cp .env.example .env   # fill DATABASE_URL and secrets locally — never commit .env
go mod download
go run ./cmd/api
```

Migrations run on startup when `AUTO_MIGRATE=true`.

Local tester: [http://localhost:8080/dev](http://localhost:8080/dev)

### 2. Admin

```bash
cd admin
npm install
npm run dev
# http://localhost:5173
```

Use a seeded admin account from your local DB (see `docs/` / `/dev` page). OTP is printed in the API console when `SMS_PROVIDER=console` — not stored in this README.

### 3. Mobile

```bash
cd mobile
flutter pub get

# Local API (desktop / simulator)
flutter run -t lib/main_passenger.dart
flutter run -t lib/main_driver.dart

# Physical Android phone (same Wi‑Fi as your PC)
flutter run -t lib/main_driver.dart --dart-define=API_BASE_URL=http://YOUR_PC_LAN_IP:8080/v1
```

Mapbox (optional): pass `--dart-define=MAPBOX_ACCESS_TOKEN=pk....` at run time. Do not commit tokens.

## Secrets policy

- **Commit:** `.env.example` (placeholders only)
- **Never commit:** `.env`, API keys, DB passwords, JWT secrets, MoMo/Airtel secrets, Mapbox tokens
- Put real values only in your local `.env` or CI secrets

## Docs

- [Architecture](docs/ARCHITECTURE.md)
- [API & WebSockets](docs/API.md)
- [Screens](docs/SCREENS.md)
- [Brand](docs/BRAND.md)
- [Database](docs/DATABASE.md)

## MVP rules

- Mbarara pilot first
- Cash to driver; Rozy fee from driver wallet
- One driver account = one mode (boda **or** car)
- One NIN = one driver account
- Fare: base + per km by ride type
