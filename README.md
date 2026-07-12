# Rozy

Local ride-hailing platform for Mbarara, Western Uganda.

**Stack:** Flutter (passenger + driver apps) · Go backend · Neon PostgreSQL · Redis GEO · React admin

## Apps

| App | Folder | Users |
|-----|--------|-------|
| Rozy | `mobile/` (passenger flavor) | Passengers |
| Rozy Driver | `mobile/` (driver flavor) | Boda riders & car drivers |
| Admin | `admin/` | Operations & verification |

## Quick start

### Backend

```bash
cd backend
cp .env.example .env   # add Neon DATABASE_URL, Redis, etc.
go mod download
go run ./cmd/api
```

Apply migrations to Neon:

```bash
# Automatic on API startup (AUTO_MIGRATE=true), or manually:
migrate -path ./migrations -database "$DATABASE_URL" up
```

**Browser API tester:** start the server and open [http://localhost:8080/dev](http://localhost:8080/dev) to test every endpoint from the browser.

### Admin dashboard

```bash
cd admin
npm install
npm run dev
# → http://localhost:5173
```

Admin login: phone **+256700000000** · OTP in API console · approve/reject in Verification queue.

### Mobile

Install [Flutter](https://docs.flutter.dev/get-started/install), then:

```bash
cd mobile
flutter pub get

# Windows / iOS simulator (API on localhost)
flutter run -t lib/main_passenger.dart
flutter run -t lib/main_driver.dart

# Android emulator (API via 10.0.2.2)
flutter run -t lib/main_passenger.dart --dart-define=API_BASE_URL=http://10.0.2.2:8080/v1
flutter run -t lib/main_driver.dart --dart-define=API_BASE_URL=http://10.0.2.2:8080/v1

# Physical device (use your PC LAN IP)
flutter run --dart-define=API_BASE_URL=http://192.168.x.x:8080/v1 -t lib/main_passenger.dart

# Mapbox map (passenger app)
flutter run -t lib/main_passenger.dart --dart-define=MAPBOX_ACCESS_TOKEN=pk.your_token_here
```

## Docs

- [Architecture](docs/ARCHITECTURE.md)
- [API & WebSocket events](docs/API.md)
- [Screen lists](docs/SCREENS.md)
- [Brand & colors](docs/BRAND.md)

## MVP rules (locked)

- Mbarara pilot first
- Cash to driver; Rozy fee from driver wallet
- One driver account = one vehicle mode (boda **or** car)
- One NIN = one driver account
- Fare: base fee + per km per ride category
