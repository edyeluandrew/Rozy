class DriverSnapshot {
  const DriverSnapshot({
    required this.id,
    required this.name,
    required this.plate,
    required this.rideType,
    required this.lat,
    required this.lng,
    this.locationUpdatedAt,
  });

  final String id;
  final String name;
  final String plate;
  final String rideType;
  final double lat;
  final double lng;
  final String? locationUpdatedAt;

  factory DriverSnapshot.fromJson(Map<String, dynamic> json) {
    return DriverSnapshot(
      id: json['id'] as String,
      name: json['name'] as String,
      plate: json['plate'] as String? ?? '',
      rideType: json['ride_type'] as String,
      lat: (json['lat'] as num).toDouble(),
      lng: (json['lng'] as num).toDouble(),
      locationUpdatedAt: json['location_updated_at'] as String?,
    );
  }

  bool get isBoda => rideType == 'boda';

  DriverSnapshot copyWith({
    String? id,
    String? name,
    String? plate,
    String? rideType,
    double? lat,
    double? lng,
    String? locationUpdatedAt,
  }) {
    return DriverSnapshot(
      id: id ?? this.id,
      name: name ?? this.name,
      plate: plate ?? this.plate,
      rideType: rideType ?? this.rideType,
      lat: lat ?? this.lat,
      lng: lng ?? this.lng,
      locationUpdatedAt: locationUpdatedAt ?? this.locationUpdatedAt,
    );
  }
}

class Trip {
  const Trip({
    required this.id,
    required this.status,
    required this.rideType,
    this.estimatedFare,
    this.finalFare,
    this.rozyFee,
    this.tripPin,
    this.pickupLandmark,
    this.destLandmark,
    this.arrivedAt,
    required this.pickupLat,
    required this.pickupLng,
    required this.destLat,
    required this.destLng,
    this.estimatedKm,
    this.driver,
    this.driverDistanceKm,
    this.driverEtaMinutes,
  });

  final String id;
  final String status;
  final String rideType;
  final int? estimatedFare;
  final int? finalFare;
  final int? rozyFee;
  final String? tripPin;
  final String? pickupLandmark;
  final String? destLandmark;
  final String? arrivedAt;
  final double pickupLat;
  final double pickupLng;
  final double destLat;
  final double destLng;
  final double? estimatedKm;
  final DriverSnapshot? driver;
  final double? driverDistanceKm;
  final int? driverEtaMinutes;

  factory Trip.fromJson(Map<String, dynamic> json) {
    DriverSnapshot? driver;
    final driverJson = json['driver'];
    if (driverJson is Map<String, dynamic>) {
      driver = DriverSnapshot.fromJson(driverJson);
    }
    return Trip(
      id: json['id'] as String,
      status: json['status'] as String,
      rideType: json['ride_type'] as String,
      estimatedFare: (json['estimated_fare'] as num?)?.toInt(),
      finalFare: (json['final_fare'] as num?)?.toInt(),
      rozyFee: (json['rozy_fee'] as num?)?.toInt(),
      tripPin: json['trip_pin'] as String?,
      pickupLandmark: json['pickup_landmark'] as String?,
      destLandmark: json['dest_landmark'] as String?,
      arrivedAt: json['arrived_at'] as String?,
      pickupLat: (json['pickup_lat'] as num?)?.toDouble() ?? -0.6072,
      pickupLng: (json['pickup_lng'] as num?)?.toDouble() ?? 30.6586,
      destLat: (json['dest_lat'] as num?)?.toDouble() ?? -0.601,
      destLng: (json['dest_lng'] as num?)?.toDouble() ?? 30.649,
      estimatedKm: (json['estimated_km'] as num?)?.toDouble(),
      driver: driver,
      driverDistanceKm: (json['driver_distance_km'] as num?)?.toDouble(),
      driverEtaMinutes: (json['driver_eta_minutes'] as num?)?.toInt(),
    );
  }

  String get statusLabel {
    switch (status) {
      case 'searching':
        return 'Finding a driver…';
      case 'driver_assigned':
        return 'Driver assigned';
      case 'driver_arriving':
        return arrivedAt != null ? 'Driver at pickup' : 'Driver on the way';
      case 'in_progress':
        return 'Trip in progress';
      case 'completed':
        return 'Trip completed';
      case 'cancelled':
        return 'Cancelled';
      default:
        return status;
    }
  }

  String get rideTypeLabel {
    switch (rideType) {
      case 'boda':
        return 'Rozy Boda';
      case 'car_basic':
        return 'Rozy Car Basic';
      case 'car_xl':
        return 'Rozy Car XL';
      default:
        return rideType;
    }
  }

  String? get trackingSubtitle {
    if (driver == null) return null;
    if (status == 'searching') return null;
    if (driverDistanceKm != null && driverEtaMinutes != null) {
      final dest = status == 'in_progress' ? 'destination' : 'pickup';
      return '${driverDistanceKm!.toStringAsFixed(1)} km to $dest · ~$driverEtaMinutes min';
    }
    if (driver != null) {
      return '${driver!.name} · ${driver!.plate}';
    }
    return null;
  }

  bool get isActive =>
      status == 'searching' ||
      status == 'driver_assigned' ||
      status == 'driver_arriving' ||
      status == 'in_progress';

  bool get showLiveMap =>
      driver != null &&
      (status == 'driver_assigned' ||
          status == 'driver_arriving' ||
          status == 'in_progress');

  Trip copyWith({
    String? id,
    String? status,
    String? rideType,
    int? estimatedFare,
    int? finalFare,
    int? rozyFee,
    String? tripPin,
    String? pickupLandmark,
    String? destLandmark,
    String? arrivedAt,
    double? pickupLat,
    double? pickupLng,
    double? destLat,
    double? destLng,
    double? estimatedKm,
    DriverSnapshot? driver,
    double? driverDistanceKm,
    int? driverEtaMinutes,
  }) {
    return Trip(
      id: id ?? this.id,
      status: status ?? this.status,
      rideType: rideType ?? this.rideType,
      estimatedFare: estimatedFare ?? this.estimatedFare,
      finalFare: finalFare ?? this.finalFare,
      rozyFee: rozyFee ?? this.rozyFee,
      tripPin: tripPin ?? this.tripPin,
      pickupLandmark: pickupLandmark ?? this.pickupLandmark,
      destLandmark: destLandmark ?? this.destLandmark,
      arrivedAt: arrivedAt ?? this.arrivedAt,
      pickupLat: pickupLat ?? this.pickupLat,
      pickupLng: pickupLng ?? this.pickupLng,
      destLat: destLat ?? this.destLat,
      destLng: destLng ?? this.destLng,
      estimatedKm: estimatedKm ?? this.estimatedKm,
      driver: driver ?? this.driver,
      driverDistanceKm: driverDistanceKm ?? this.driverDistanceKm,
      driverEtaMinutes: driverEtaMinutes ?? this.driverEtaMinutes,
    );
  }
}

class FareEstimate {
  const FareEstimate({required this.distanceKm, required this.estimatedFare});

  final double distanceKm;
  final int estimatedFare;

  factory FareEstimate.fromJson(Map<String, dynamic> json) {
    return FareEstimate(
      distanceKm: (json['distance_km'] as num).toDouble(),
      estimatedFare: (json['estimated_fare'] as num).toInt(),
    );
  }
}

class TripCompleteResult {
  const TripCompleteResult({
    required this.tripId,
    required this.status,
    required this.finalFare,
    required this.rozyFee,
    required this.walletBalance,
  });

  final String tripId;
  final String status;
  final int finalFare;
  final int rozyFee;
  final int walletBalance;

  factory TripCompleteResult.fromJson(Map<String, dynamic> json) {
    return TripCompleteResult(
      tripId: json['trip_id'] as String,
      status: json['status'] as String,
      finalFare: (json['final_fare'] as num).toInt(),
      rozyFee: (json['rozy_fee'] as num).toInt(),
      walletBalance: (json['wallet_balance'] as num).toInt(),
    );
  }
}
