import 'package:shared_preferences/shared_preferences.dart';

class SessionStorage {
  static const _tokenKey = 'rozy_token';
  static const _phoneKey = 'rozy_phone';
  static const _roleKey = 'rozy_role';

  Future<void> saveSession({
    required String token,
    required String phone,
    required String role,
  }) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_tokenKey, token);
    await prefs.setString(_phoneKey, phone);
    await prefs.setString(_roleKey, role);
  }

  Future<String?> getToken() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getString(_tokenKey);
  }

  Future<void> clear() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_tokenKey);
    await prefs.remove(_phoneKey);
    await prefs.remove(_roleKey);
    await prefs.remove(_tripPinKey);
  }

  static const _tripPinKey = 'rozy_active_trip_pin';

  Future<void> saveTripPin(String pin) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_tripPinKey, pin);
  }

  Future<String?> getTripPin() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getString(_tripPinKey);
  }

  Future<void> clearTripPin() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_tripPinKey);
  }
}
