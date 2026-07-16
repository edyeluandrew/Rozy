/// API base URL.
///
/// Local override:
///   flutter run --dart-define=API_BASE_URL=http://192.168.1.5:8080/v1
///
/// Production (Render):
///   flutter run --dart-define=API_BASE_URL=https://rozy.onrender.com/v1
///
/// Defaults:
/// - Android emulator: 10.0.2.2
/// - Others: localhost (Windows desktop / iOS simulator)
abstract final class AppConfig {
  static const productionApiBaseUrl = 'https://rozy.onrender.com/v1';

  static const apiBaseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'http://localhost:8080/v1',
  );

  static const androidEmulatorBaseUrl = 'http://10.0.2.2:8080/v1';

  /// Mapbox public token — pass at run time:
  ///   flutter run --dart-define=MAPBOX_ACCESS_TOKEN=pk....
  static const mapboxToken = String.fromEnvironment('MAPBOX_ACCESS_TOKEN');

  /// Mbarara town centre
  static const defaultLat = -0.6072;
  static const defaultLng = 30.6586;
}
