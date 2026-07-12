# Rozy Mobile

Flutter apps for **Rozy** (passenger) and **Rozy Driver**.

## Prerequisites

Install [Flutter SDK](https://docs.flutter.dev/get-started/install).

## Run on a physical Android phone

Phone and PC must be on the **same Wi‑Fi**. Use your PC's LAN IP (not `localhost`).

```bash
# Find PC IP: ipconfig  (look for Wi-Fi IPv4, e.g. 192.168.1.39)

flutter pub get
flutter devices   # phone should appear when USB debugging is on

# Driver app
flutter run -t lib/main_driver.dart -d <device-id> --dart-define=API_BASE_URL=http://192.168.1.39:8080/v1

# Passenger app (second phone, or reinstall after changing -t)
flutter run -t lib/main_passenger.dart -d <device-id> --dart-define=API_BASE_URL=http://192.168.1.39:8080/v1
```

### Phone setup
1. **Developer options** → enable **USB debugging**
2. Connect USB → allow debugging on phone
3. Install **Android Studio** → SDK Manager → install **Android SDK Command-line Tools**
4. In cmd: `set ANDROID_HOME=C:\Users\edyel\AppData\Local\Android\sdk`
5. Run: `flutter doctor --android-licenses` (accept all)

### Test logins
- Driver: `+256700000081` / OTP `123456`
- Passenger: `+256700000082` / OTP `123456`

## Run (desktop / emulator)

```bash
flutter pub get
flutter run -t lib/main_passenger.dart
flutter run -t lib/main_driver.dart
```

## Brand

Colors and theme: `lib/core/theme/rozy_colors.dart`, `rozy_theme.dart`.

See [../docs/BRAND.md](../docs/BRAND.md).
