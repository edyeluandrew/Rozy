import 'dart:async';

import 'package:flutter/material.dart';
import 'package:geolocator/geolocator.dart';

import '../api/operator_api.dart';
import 'app_services.dart';

/// Sends driver GPS to the API while on an active trip.
class DriverLocationTracker {
  DriverLocationTracker(this._operator);

  final OperatorApi _operator;
  Timer? _timer;
  StreamSubscription<Position>? _sub;

  Future<void> start() async {
    await stop();
    final enabled = await _ensurePermission();
    if (!enabled) return;

    _pushCurrent();
    _timer = Timer.periodic(const Duration(seconds: 5), (_) => _pushCurrent());
    _sub = Geolocator.getPositionStream(
      locationSettings: const LocationSettings(
        accuracy: LocationAccuracy.high,
        distanceFilter: 15,
      ),
    ).listen((pos) {
      _operator.updateLocation(lat: pos.latitude, lng: pos.longitude);
    });
  }

  Future<void> stop() async {
    _timer?.cancel();
    _timer = null;
    await _sub?.cancel();
    _sub = null;
  }

  Future<void> _pushCurrent() async {
    try {
      final pos = await Geolocator.getCurrentPosition(
        locationSettings: const LocationSettings(accuracy: LocationAccuracy.high),
      );
      await _operator.updateLocation(lat: pos.latitude, lng: pos.longitude);
    } catch (_) {}
  }

  Future<bool> _ensurePermission() async {
    var perm = await Geolocator.checkPermission();
    if (perm == LocationPermission.denied) {
      perm = await Geolocator.requestPermission();
    }
    return perm == LocationPermission.always || perm == LocationPermission.whileInUse;
  }
}

/// Singleton-style access for driver screens.
DriverLocationTracker get driverLocationTracker =>
    DriverLocationTracker(AppServices.live.operator);
