import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:latlong2/latlong.dart';

import '../../core/config/app_config.dart';
import '../../core/models/trip.dart';
import '../../core/theme/rozy_colors.dart';

/// Live trip map: pickup, destination, and moving driver marker.
class ActiveTripMap extends StatefulWidget {
  const ActiveTripMap({
    super.key,
    required this.trip,
  });

  final Trip trip;

  @override
  State<ActiveTripMap> createState() => _ActiveTripMapState();
}

class _ActiveTripMapState extends State<ActiveTripMap> {
  final _mapController = MapController();
  LatLng? _lastDriverPoint;

  @override
  void didUpdateWidget(covariant ActiveTripMap oldWidget) {
    super.didUpdateWidget(oldWidget);
    final driver = widget.trip.driver;
    if (driver != null) {
      final point = LatLng(driver.lat, driver.lng);
      if (_lastDriverPoint == null ||
          _lastDriverPoint!.latitude != point.latitude ||
          _lastDriverPoint!.longitude != point.longitude) {
        _lastDriverPoint = point;
        _fitBounds();
      }
    }
  }

  void _fitBounds() {
    final points = <LatLng>[
      LatLng(widget.trip.pickupLat, widget.trip.pickupLng),
      LatLng(widget.trip.destLat, widget.trip.destLng),
    ];
    final driver = widget.trip.driver;
    if (driver != null) {
      points.add(LatLng(driver.lat, driver.lng));
    }
    if (points.length < 2) return;

    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      try {
        _mapController.fitCamera(
          CameraFit.coordinates(
            coordinates: points,
            padding: const EdgeInsets.all(56),
          ),
        );
      } catch (_) {}
    });
  }

  @override
  Widget build(BuildContext context) {
    final token = AppConfig.mapboxToken;
    final pickup = LatLng(widget.trip.pickupLat, widget.trip.pickupLng);
    final dest = LatLng(widget.trip.destLat, widget.trip.destLng);
    final driver = widget.trip.driver;

    return FlutterMap(
      mapController: _mapController,
      options: MapOptions(
        initialCenter: pickup,
        initialZoom: 14,
        interactionOptions: const InteractionOptions(flags: InteractiveFlag.all),
        onMapReady: _fitBounds,
      ),
      children: [
        if (token.isNotEmpty)
          TileLayer(
            urlTemplate:
                'https://api.mapbox.com/styles/v1/mapbox/streets-v12/tiles/256/{z}/{x}/{y}@2x?access_token=$token',
            userAgentPackageName: 'com.rozy.passenger',
          )
        else
          TileLayer(
            urlTemplate: 'https://tile.openstreetmap.org/{z}/{x}/{y}.png',
            userAgentPackageName: 'com.rozy.passenger',
          ),
        if (driver != null &&
            (widget.trip.status == 'driver_assigned' ||
                widget.trip.status == 'driver_arriving' ||
                widget.trip.status == 'in_progress'))
          PolylineLayer(
            polylines: [
              Polyline(
                points: widget.trip.status == 'in_progress'
                    ? [LatLng(driver.lat, driver.lng), dest]
                    : [LatLng(driver.lat, driver.lng), pickup],
                color: RozyColors.gold.withValues(alpha: 0.85),
                strokeWidth: 4,
              ),
            ],
          ),
        MarkerLayer(
          markers: [
            Marker(
              point: pickup,
              width: 44,
              height: 44,
              child: const Icon(Icons.location_on, color: RozyColors.gold, size: 40),
            ),
            Marker(
              point: dest,
              width: 40,
              height: 40,
              child: const Icon(Icons.flag, color: RozyColors.darkGold, size: 34),
            ),
            if (driver != null)
              Marker(
                point: LatLng(driver.lat, driver.lng),
                width: 52,
                height: 52,
                child: _VehicleMarker(rideType: driver.rideType),
              ),
          ],
        ),
      ],
    );
  }
}

class _VehicleMarker extends StatelessWidget {
  const _VehicleMarker({required this.rideType});

  final String rideType;

  @override
  Widget build(BuildContext context) {
    final isBoda = rideType == 'boda';
    return Container(
      decoration: BoxDecoration(
        color: RozyColors.charcoal,
        shape: BoxShape.circle,
        border: Border.all(color: RozyColors.gold, width: 2),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.25),
            blurRadius: 6,
            offset: const Offset(0, 2),
          ),
        ],
      ),
      child: Icon(
        isBoda ? Icons.two_wheeler : Icons.directions_car_filled,
        color: RozyColors.gold,
        size: isBoda ? 26 : 24,
      ),
    );
  }
}
