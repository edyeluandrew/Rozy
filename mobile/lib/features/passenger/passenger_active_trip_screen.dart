import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../core/api/api_client.dart';
import '../../core/models/trip.dart';
import '../../core/services/app_services.dart';
import '../../core/services/rozy_realtime.dart';
import '../../core/theme/rozy_colors.dart';
import '../../core/theme/rozy_theme.dart';
import 'active_trip_map.dart';
import 'ride_request_screen.dart';

class PassengerActiveTripScreen extends StatefulWidget {
  const PassengerActiveTripScreen({
    super.key,
    required this.initialTrip,
    this.tripPin,
  });

  final Trip initialTrip;
  final String? tripPin;

  @override
  State<PassengerActiveTripScreen> createState() => _PassengerActiveTripScreenState();
}

class _PassengerActiveTripScreenState extends State<PassengerActiveTripScreen> {
  late Trip _trip;
  String? _pin;
  bool _busy = false;
  String? _error;
  RozyRealtime? _realtime;

  @override
  void initState() {
    super.initState();
    _trip = widget.initialTrip;
    _pin = widget.tripPin;
    _loadPin();
    _connectRealtime();
    if (_trip.isActive) {
      _refresh(full: true);
    }
  }

  Future<void> _loadPin() async {
    if (_pin != null) return;
    final saved = await AppServices.live.session.getTripPin();
    if (mounted && saved != null) setState(() => _pin = saved);
  }

  Future<void> _connectRealtime() async {
    final token = await AppServices.live.session.getToken();
    if (token == null || !mounted) return;
    _realtime?.disconnect();
    _realtime = RozyRealtime(token: token);
    _realtime!.connect(_onRealtimeEvent);
  }

  void _onRealtimeEvent(String event, Map<String, dynamic> payload) {
    if (!mounted) return;
    final tripId = payload['trip_id'] as String?;
    if (tripId != null && tripId != _trip.id) return;

    switch (event) {
      case 'trip:status':
      case 'trip:assigned':
        final status = payload['status'] as String?;
        if (status != null) {
          setState(() => _trip = _trip.copyWith(status: status));
        }
        _refresh(full: true);
      case 'trip:driver_location':
        final lat = payload['lat'];
        final lng = payload['lng'];
        if (lat is num && lng is num && _trip.driver != null) {
          final driver = _trip.driver!.copyWith(
            lat: lat.toDouble(),
            lng: lng.toDouble(),
            locationUpdatedAt: DateTime.now().toUtc().toIso8601String(),
          );
          setState(() {
            _trip = _trip.copyWith(
              driver: driver,
              driverDistanceKm: _distanceKm(driver.lat, driver.lng),
              driverEtaMinutes: _etaMinutes(_distanceKm(driver.lat, driver.lng)),
            );
          });
        } else {
          _refresh(full: true);
        }
    }
  }

  double? _distanceKm(double driverLat, double driverLng) {
    final targetLat = _trip.status == 'in_progress' ? _trip.destLat : _trip.pickupLat;
    final targetLng = _trip.status == 'in_progress' ? _trip.destLng : _trip.pickupLng;
    const r = 6371.0;
    final dLat = _deg2rad(targetLat - driverLat);
    final dLng = _deg2rad(targetLng - driverLng);
    final a = math.sin(dLat / 2) * math.sin(dLat / 2) +
        math.cos(_deg2rad(driverLat)) *
            math.cos(_deg2rad(targetLat)) *
            math.sin(dLng / 2) *
            math.sin(dLng / 2);
    final c = 2 * math.atan2(math.sqrt(a), math.sqrt(1 - a));
    return (r * c * 10).round() / 10;
  }

  double _deg2rad(double deg) => deg * math.pi / 180;

  int? _etaMinutes(double? km) {
    if (km == null) return null;
    if (km <= 0.05) return 1;
    final speed = _trip.rideType == 'boda' ? 22.0 : 20.0;
    return math.max(1, (km / speed * 60).ceil());
  }

  @override
  void dispose() {
    _realtime?.disconnect();
    super.dispose();
  }

  Future<void> _refresh({bool full = false}) async {
    if (_busy) return;
    try {
      if (_trip.isActive) {
        final active = await AppServices.live.trip.activeTrip();
        if (!mounted) return;
        if (active != null) {
          setState(() => _trip = active);
          return;
        }
      }
      final done = await AppServices.live.trip.getTrip(_trip.id);
      if (!mounted) return;
      setState(() => _trip = done);
      if (done.status == 'completed') {
        await AppServices.live.session.clearTripPin();
      }
    } catch (_) {}
  }

  Future<void> _cancel() async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await AppServices.live.trip.cancelTrip(_trip.id);
      await AppServices.live.session.clearTripPin();
      if (!mounted) return;
      Navigator.of(context).pushReplacement(
        MaterialPageRoute(builder: (_) => const RideRequestScreen()),
      );
    } on ApiException catch (e) {
      if (mounted) setState(() => _error = e.message);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  void _done() {
    Navigator.of(context).pushReplacement(
      MaterialPageRoute(builder: (_) => const RideRequestScreen()),
    );
  }

  @override
  Widget build(BuildContext context) {
    final completed = _trip.status == 'completed';
    final canCancel = _trip.status != 'in_progress' && !completed;
    final driver = _trip.driver;

    return Scaffold(
      body: Stack(
        children: [
          Positioned.fill(
            child: _trip.showLiveMap
                ? ActiveTripMap(trip: _trip)
                : ActiveTripMap(
                    trip: Trip(
                      id: _trip.id,
                      status: _trip.status,
                      rideType: _trip.rideType,
                      pickupLat: _trip.pickupLat,
                      pickupLng: _trip.pickupLng,
                      destLat: _trip.destLat,
                      destLng: _trip.destLng,
                      pickupLandmark: _trip.pickupLandmark,
                      destLandmark: _trip.destLandmark,
                    ),
                  ),
          ),
          SafeArea(
            child: Align(
              alignment: Alignment.topCenter,
              child: Padding(
                padding: const EdgeInsets.all(12),
                child: Row(
                  children: [
                    if (canCancel)
                      CircleAvatar(
                        backgroundColor: RozyColors.cream,
                        child: IconButton(
                          icon: const Icon(Icons.close, color: RozyColors.charcoal),
                          onPressed: _busy ? null : _cancel,
                        ),
                      ),
                    const Spacer(),
                    if (_trip.isActive)
                      CircleAvatar(
                        backgroundColor: RozyColors.cream,
                        child: IconButton(
                          icon: const Icon(Icons.refresh, color: RozyColors.charcoal),
                          onPressed: _busy ? null : () => _refresh(full: true),
                        ),
                      ),
                  ],
                ),
              ),
            ),
          ),
          DraggableScrollableSheet(
            initialChildSize: 0.32,
            minChildSize: 0.22,
            maxChildSize: 0.55,
            builder: (context, scrollController) {
              return Container(
                decoration: const BoxDecoration(
                  color: RozyColors.cream,
                  borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
                  boxShadow: [
                    BoxShadow(
                      color: Colors.black26,
                      blurRadius: 12,
                      offset: Offset(0, -4),
                    ),
                  ],
                ),
                child: ListView(
                  controller: scrollController,
                  padding: const EdgeInsets.fromLTRB(20, 12, 20, 24),
                  children: [
                    Center(
                      child: Container(
                        width: 40,
                        height: 4,
                        decoration: BoxDecoration(
                          color: RozyColors.border,
                          borderRadius: BorderRadius.circular(2),
                        ),
                      ),
                    ),
                    const SizedBox(height: 16),
                    if (driver != null) ...[
                      Row(
                        children: [
                          CircleAvatar(
                            radius: 28,
                            backgroundColor: RozyColors.charcoal,
                            child: Icon(
                              driver.isBoda ? Icons.two_wheeler : Icons.directions_car_filled,
                              color: RozyColors.gold,
                              size: 28,
                            ),
                          ),
                          const SizedBox(width: 14),
                          Expanded(
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Text(
                                  driver.name,
                                  style: Theme.of(context).textTheme.titleMedium?.copyWith(
                                        fontWeight: FontWeight.bold,
                                      ),
                                ),
                                Text(
                                  '${_trip.rideTypeLabel}${driver.plate.isNotEmpty ? ' · ${driver.plate}' : ''}',
                                  style: Theme.of(context).textTheme.bodySmall?.copyWith(
                                        color: RozyColors.grey,
                                      ),
                                ),
                              ],
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 14),
                    ],
                    Text(
                      _trip.statusLabel,
                      style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                            fontWeight: FontWeight.bold,
                            color: RozyColors.darkGold,
                          ),
                    ),
                    if (_trip.trackingSubtitle != null) ...[
                      const SizedBox(height: 6),
                      Text(
                        _trip.trackingSubtitle!,
                        style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                              color: RozyColors.charcoal,
                            ),
                      ),
                    ],
                    const SizedBox(height: 12),
                    Container(
                      padding: const EdgeInsets.all(14),
                      decoration: RozyTheme.premiumCardDecoration,
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            completed
                                ? 'UGX ${_trip.finalFare ?? _trip.estimatedFare ?? 0}'
                                : 'Est. UGX ${_trip.estimatedFare ?? 0}',
                            style: Theme.of(context).textTheme.titleLarge?.copyWith(
                                  color: RozyColors.cream,
                                  fontWeight: FontWeight.bold,
                                ),
                          ),
                          if (_trip.estimatedKm != null)
                            Text(
                              'Trip distance ~${_trip.estimatedKm!.toStringAsFixed(1)} km',
                              style: TextStyle(color: RozyColors.grey.withValues(alpha: 0.95)),
                            ),
                          if (_trip.pickupLandmark != null)
                            Padding(
                              padding: const EdgeInsets.only(top: 6),
                              child: Text(
                                'Pickup: ${_trip.pickupLandmark}',
                                style: TextStyle(color: RozyColors.grey.withValues(alpha: 0.95)),
                              ),
                            ),
                        ],
                      ),
                    ),
                    if (!completed && _pin != null) ...[
                      const SizedBox(height: 12),
                      Card(
                        color: RozyColors.beige,
                        child: Padding(
                          padding: const EdgeInsets.all(16),
                          child: Row(
                            children: [
                              Expanded(
                                child: Column(
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: [
                                    const Text('Your PIN'),
                                    Text(
                                      _pin!,
                                      style: Theme.of(context).textTheme.headlineMedium?.copyWith(
                                            letterSpacing: 8,
                                            fontWeight: FontWeight.bold,
                                          ),
                                    ),
                                  ],
                                ),
                              ),
                              IconButton(
                                onPressed: () {
                                  Clipboard.setData(ClipboardData(text: _pin!));
                                  ScaffoldMessenger.of(context).showSnackBar(
                                    const SnackBar(content: Text('PIN copied')),
                                  );
                                },
                                icon: const Icon(Icons.copy),
                              ),
                            ],
                          ),
                        ),
                      ),
                    ],
                    if (_error != null) ...[
                      const SizedBox(height: 8),
                      Text(_error!, style: const TextStyle(color: Colors.red)),
                    ],
                    const SizedBox(height: 12),
                    if (completed)
                      ElevatedButton(onPressed: _done, child: const Text('Book another ride'))
                    else if (canCancel)
                      OutlinedButton(
                        onPressed: _busy ? null : _cancel,
                        child: const Text('Cancel ride'),
                      ),
                  ],
                ),
              );
            },
          ),
        ],
      ),
    );
  }
}
