import 'package:flutter/material.dart';
import 'package:latlong2/latlong.dart';

import '../../core/api/api_client.dart';
import '../../core/config/app_config.dart';
import '../../core/models/place.dart';
import '../../core/models/trip.dart';
import '../../core/services/app_services.dart';
import '../../core/theme/rozy_colors.dart';
import 'passenger_active_trip_screen.dart';
import 'rozy_map.dart';

class RideRequestScreen extends StatefulWidget {
  const RideRequestScreen({super.key});

  @override
  State<RideRequestScreen> createState() => _RideRequestScreenState();
}

class _RideRequestScreenState extends State<RideRequestScreen> {
  String _rideType = 'boda';
  MapPinMode _pinMode = MapPinMode.pickup;
  FareEstimate? _fareEstimate;
  Trip? _activeTrip;
  List<Place> _places = [];
  bool _loading = false;
  String? _error;

  LatLng _pickup = LatLng(AppConfig.defaultLat, AppConfig.defaultLng);
  LatLng _destination = const LatLng(-0.6010, 30.6490);

  void _onMapTap(LatLng point) {
    setState(() {
      if (_pinMode == MapPinMode.pickup) {
        _pickup = point;
      } else {
        _destination = point;
      }
      _fareEstimate = null;
    });
  }

  Future<void> _searchPlaces(String q) async {
    if (q.length < 2) return;
    final results = await AppServices.live.trip.searchPlaces(q);
    setState(() => _places = results);
  }

  Future<void> _fetchEstimate() async {
    setState(() { _loading = true; _error = null; });
    try {
      final est = await AppServices.live.trip.estimate(
        pickupLat: _pickup.latitude,
        pickupLng: _pickup.longitude,
        destLat: _destination.latitude,
        destLng: _destination.longitude,
        rideType: _rideType,
      );
      setState(() => _fareEstimate = est);
    } on ApiException catch (e) {
      setState(() => _error = e.message);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _requestRide() async {
    setState(() { _loading = true; _error = null; });
    try {
      final trip = await AppServices.live.trip.requestTrip(
        pickupLat: _pickup.latitude,
        pickupLng: _pickup.longitude,
        destLat: _destination.latitude,
        destLng: _destination.longitude,
        rideType: _rideType,
        pickupLandmark: 'Map pin',
        destLandmark: 'Map pin',
      );
      if (trip.tripPin != null) {
        await AppServices.live.session.saveTripPin(trip.tripPin!);
      }
      if (!mounted) return;
      Navigator.of(context).pushReplacement(
        MaterialPageRoute(
          builder: (_) => PassengerActiveTripScreen(
            initialTrip: trip,
            tripPin: trip.tripPin,
          ),
        ),
      );
    } on ApiException catch (e) {
      setState(() => _error = e.message);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final usingMapbox = AppConfig.mapboxToken.isNotEmpty;

    return Scaffold(
      appBar: AppBar(title: const Text('Request a ride')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          SizedBox(
            height: 280,
            child: RozyMap(
              pickup: _pickup,
              destination: _destination,
              pinMode: _pinMode,
              onTap: _onMapTap,
            ),
          ),
          const SizedBox(height: 8),
          Row(
            children: [
              ChoiceChip(
                label: const Text('Set pickup'),
                selected: _pinMode == MapPinMode.pickup,
                onSelected: (_) => setState(() => _pinMode = MapPinMode.pickup),
                selectedColor: RozyColors.gold,
              ),
              const SizedBox(width: 8),
              ChoiceChip(
                label: const Text('Set destination'),
                selected: _pinMode == MapPinMode.destination,
                onSelected: (_) => setState(() => _pinMode = MapPinMode.destination),
                selectedColor: RozyColors.beige,
              ),
            ],
          ),
          Text(
            usingMapbox ? 'Mapbox · tap map to move pins' : 'OSM fallback · add MAPBOX_ACCESS_TOKEN',
            style: Theme.of(context).textTheme.bodySmall?.copyWith(color: RozyColors.grey),
          ),
          const SizedBox(height: 12),
          TextField(
            decoration: const InputDecoration(labelText: 'Search Mbarara places'),
            onChanged: _searchPlaces,
          ),
          ..._places.take(3).map((p) => ListTile(
                dense: true,
                title: Text(p.name),
                subtitle: Text(p.landmarkNote ?? ''),
                onTap: () => setState(() {
                  _destination = LatLng(p.lat, p.lng);
                  _places = [];
                  _fareEstimate = null;
                  _pinMode = MapPinMode.destination;
                }),
              )),
          DropdownButtonFormField<String>(
            value: _rideType,
            decoration: const InputDecoration(labelText: 'Ride type'),
            items: const [
              DropdownMenuItem(value: 'boda', child: Text('Rozy Boda')),
              DropdownMenuItem(value: 'car_basic', child: Text('Rozy Car Basic')),
              DropdownMenuItem(value: 'car_xl', child: Text('Rozy Car XL')),
            ],
            onChanged: (v) => setState(() { _rideType = v ?? 'boda'; _fareEstimate = null; }),
          ),
          if (_fareEstimate != null)
            Card(
              child: ListTile(
                title: Text('Est. UGX ${_fareEstimate!.estimatedFare}'),
                subtitle: Text('~${_fareEstimate!.distanceKm.toStringAsFixed(1)} km'),
              ),
            ),
          if (_activeTrip != null)
            Card(
              color: RozyColors.beige,
              child: ListTile(
                title: Text('Trip ${_activeTrip!.status}'),
                subtitle: Text(
                  'UGX ${_activeTrip!.estimatedFare ?? 0} · PIN ${_activeTrip!.tripPin ?? "----"}',
                ),
              ),
            ),
          if (_error != null) Text(_error!, style: const TextStyle(color: Colors.red)),
          const SizedBox(height: 12),
          ElevatedButton(onPressed: _loading ? null : _fetchEstimate, child: const Text('Get fare estimate')),
          const SizedBox(height: 8),
          ElevatedButton(onPressed: _loading ? null : _requestRide, child: const Text('Request ride')),
        ],
      ),
    );
  }
}
