import 'api_client.dart';
import '../models/operator_profile.dart';
import '../models/trip.dart';

class OperatorApi {
  OperatorApi(this._client);

  final ApiClient _client;

  Future<OperatorProfile> register(String rideType) async {
    final data = await _client.post(
      '/operator/register',
      auth: true,
      body: {'ride_type': rideType},
    );
    return OperatorProfile.fromJson(data['operator'] as Map<String, dynamic>);
  }

  Future<OperatorProfileStatus> profileStatus() async {
    final data = await _client.get('/operator/profile', auth: true);
    final registered = data['registered'] as bool? ?? false;
    if (!registered) {
      return OperatorProfileStatus.notRegistered();
    }
    return OperatorProfileStatus.registered(
      OperatorProfile.fromJson(data['operator'] as Map<String, dynamic>),
    );
  }

  Future<OperatorProfile> goOnline({double lat = -0.6072, double lng = 30.6586}) async {
    final data = await _client.post(
      '/operator/online',
      auth: true,
      body: {'lat': lat, 'lng': lng},
    );
    return OperatorProfile.fromJson(data['operator'] as Map<String, dynamic>);
  }

  Future<OperatorProfile> goOffline() async {
    final data = await _client.post('/operator/offline', auth: true);
    return OperatorProfile.fromJson(data['operator'] as Map<String, dynamic>);
  }

  Future<void> updateLocation({required double lat, required double lng}) async {
    await _client.post(
      '/operator/location',
      auth: true,
      body: {'lat': lat, 'lng': lng},
    );
  }

  Future<Map<String, dynamic>?> incomingTrip() async {
    final data = await _client.get('/operator/trips/incoming', auth: true);
    return data['trip'] as Map<String, dynamic>?;
  }

  Future<void> acceptTrip(String tripId) async {
    await _client.post('/operator/trips/$tripId/accept', auth: true);
  }

  Future<void> rejectTrip(String tripId) async {
    await _client.post('/operator/trips/$tripId/reject', auth: true);
  }

  Future<Trip?> activeTrip() async {
    final data = await _client.get('/operator/trips/active', auth: true);
    if (data['active'] != true) return null;
    return Trip.fromJson(data['trip'] as Map<String, dynamic>);
  }

  Future<void> markArrived(String tripId) async {
    await _client.post('/operator/trips/$tripId/arrived', auth: true);
  }

  Future<void> startTrip(String tripId, String pin) async {
    await _client.post('/operator/trips/$tripId/start', auth: true, body: {'pin': pin});
  }

  Future<TripCompleteResult> completeTrip(String tripId) async {
    final data = await _client.post('/operator/trips/$tripId/complete', auth: true);
    return TripCompleteResult.fromJson(data);
  }
}

class OperatorProfileStatus {
  const OperatorProfileStatus._({required this.registered, this.profile});

  factory OperatorProfileStatus.notRegistered() =>
      const OperatorProfileStatus._(registered: false);

  factory OperatorProfileStatus.registered(OperatorProfile profile) =>
      OperatorProfileStatus._(registered: true, profile: profile);

  final bool registered;
  final OperatorProfile? profile;
}
