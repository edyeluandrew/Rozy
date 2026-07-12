import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:latlong2/latlong.dart';

import '../../core/config/app_config.dart';
import '../../core/theme/rozy_colors.dart';

enum MapPinMode { pickup, destination }

class RozyMap extends StatelessWidget {
  const RozyMap({
    super.key,
    required this.pickup,
    required this.destination,
    required this.pinMode,
    required this.onTap,
  });

  final LatLng pickup;
  final LatLng destination;
  final MapPinMode pinMode;
  final ValueChanged<LatLng> onTap;

  @override
  Widget build(BuildContext context) {
    final token = AppConfig.mapboxToken;
    final hasMapbox = token.isNotEmpty;

    return ClipRRect(
      borderRadius: BorderRadius.circular(16),
      child: FlutterMap(
        options: MapOptions(
          initialCenter: pickup,
          initialZoom: 14,
          onTap: (_, point) => onTap(point),
        ),
        children: [
          if (hasMapbox)
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
          MarkerLayer(
            markers: [
              Marker(
                point: pickup,
                width: 40,
                height: 40,
                child: Icon(Icons.location_on, color: RozyColors.gold, size: 36),
              ),
              Marker(
                point: destination,
                width: 40,
                height: 40,
                child: Icon(Icons.flag, color: RozyColors.darkGold, size: 32),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
