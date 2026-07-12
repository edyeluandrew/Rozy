import 'api_client.dart';
import '../models/trip.dart';
import '../models/place.dart';

class TripApi {
  TripApi(this._client);

  final ApiClient _client;

  Future<FareEstimate> estimate({
    required double pickupLat,
    required double pickupLng,
    required double destLat,
    required double destLng,
    required String rideType,
  }) async {
    final data = await _client.post('/fare/estimate', body: {
      'pickup': {'lat': pickupLat, 'lng': pickupLng},
      'dest': {'lat': destLat, 'lng': destLng},
      'ride_type': rideType,
    });
    return FareEstimate.fromJson(data);
  }

  Future<Trip> requestTrip({
    required double pickupLat,
    required double pickupLng,
    required double destLat,
    required double destLng,
    required String rideType,
    String? pickupLandmark,
    String? destLandmark,
  }) async {
    final data = await _client.post('/trips', auth: true, body: {
      'pickup': {'lat': pickupLat, 'lng': pickupLng},
      'dest': {'lat': destLat, 'lng': destLng},
      'ride_type': rideType,
      if (pickupLandmark != null) 'pickup_landmark': pickupLandmark,
      if (destLandmark != null) 'dest_landmark': destLandmark,
    });
    return Trip.fromJson(data);
  }

  Future<Trip?> activeTrip() async {
    final data = await _client.get('/trips/active', auth: true);
    if (data['active'] != true) return null;
    return Trip.fromJson(data['trip'] as Map<String, dynamic>);
  }

  Future<Trip> getTrip(String id) async {
    final data = await _client.get('/trips/$id', auth: true);
    return Trip.fromJson(data);
  }

  Future<void> cancelTrip(String id) async {
    await _client.post('/trips/$id/cancel', auth: true);
  }

  Future<List<Place>> searchPlaces(String query) async {
    final data = await _client.get('/places/search?q=${Uri.encodeComponent(query)}');
    final list = data['places'] as List<dynamic>? ?? [];
    return list.map((e) => Place.fromJson(e as Map<String, dynamic>)).toList();
  }
}
