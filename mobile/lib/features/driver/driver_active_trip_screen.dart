import 'dart:async';

import 'package:flutter/material.dart';

import '../../core/api/api_client.dart';
import '../../core/models/trip.dart';
import '../../core/services/app_services.dart';
import '../../core/services/driver_location_tracker.dart';
import '../../core/theme/rozy_colors.dart';
import '../../core/theme/rozy_theme.dart';
import 'driver_shell.dart';
import '../../core/models/operator_profile.dart';

class DriverActiveTripScreen extends StatefulWidget {
  const DriverActiveTripScreen({
    super.key,
    required this.tripId,
    required this.profile,
  });

  final String tripId;
  final OperatorProfile profile;

  @override
  State<DriverActiveTripScreen> createState() => _DriverActiveTripScreenState();
}

class _DriverActiveTripScreenState extends State<DriverActiveTripScreen> {
  Trip? _trip;
  TripCompleteResult? _result;
  final _pinController = TextEditingController();
  bool _busy = false;
  String? _error;
  Timer? _pollTimer;

  @override
  void initState() {
    super.initState();
    _refresh();
    driverLocationTracker.start();
    _pollTimer = Timer.periodic(const Duration(seconds: 4), (_) => _refresh());
  }

  @override
  void dispose() {
    driverLocationTracker.stop();
    _pollTimer?.cancel();
    _pinController.dispose();
    super.dispose();
  }

  Future<void> _refresh() async {
    if (_busy || _result != null) return;
    try {
      final active = await AppServices.live.operator.activeTrip();
      if (!mounted) return;
      if (active != null) {
        setState(() => _trip = active);
      }
    } catch (_) {}
  }

  Future<void> _arrived() async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await AppServices.live.operator.markArrived(widget.tripId);
      await _refresh();
    } on ApiException catch (e) {
      if (mounted) setState(() => _error = e.message);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _start() async {
    final pin = _pinController.text.trim();
    if (pin.length != 4) {
      setState(() => _error = 'Enter the 4-digit passenger PIN');
      return;
    }
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await AppServices.live.operator.startTrip(widget.tripId, pin);
      await _refresh();
    } on ApiException catch (e) {
      if (mounted) setState(() => _error = e.message);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _complete() async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final result = await AppServices.live.operator.completeTrip(widget.tripId);
      if (!mounted) return;
      setState(() => _result = result);
    } on ApiException catch (e) {
      if (mounted) setState(() => _error = e.message);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  void _backToShell() {
    Navigator.of(context).pushReplacement(
      MaterialPageRoute(
        builder: (_) => DriverShell(profile: widget.profile),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final trip = _trip;
    final completed = _result != null;

    return Scaffold(
      appBar: AppBar(
        title: const Text('Active trip'),
        leading: completed
            ? null
            : IconButton(
                icon: const Icon(Icons.close),
                onPressed: _backToShell,
              ),
      ),
      body: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (trip != null)
              Container(
                padding: const EdgeInsets.all(20),
                decoration: RozyTheme.premiumCardDecoration,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      trip.rideTypeLabel,
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(color: RozyColors.cream),
                    ),
                    const SizedBox(height: 8),
                    Text(
                      trip.statusLabel,
                      style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                            color: RozyColors.gold,
                            fontWeight: FontWeight.bold,
                          ),
                    ),
                    const SizedBox(height: 12),
                    Text(
                      'Collect UGX ${trip.estimatedFare ?? 0} cash from passenger',
                      style: Theme.of(context).textTheme.bodyLarge?.copyWith(color: RozyColors.cream),
                    ),
                    if (trip.pickupLandmark != null)
                      Padding(
                        padding: const EdgeInsets.only(top: 8),
                        child: Text(
                          'Pickup: ${trip.pickupLandmark}',
                          style: TextStyle(color: RozyColors.grey.withValues(alpha: 0.9)),
                        ),
                      ),
                  ],
                ),
              ),
            const SizedBox(height: 20),
            if (completed) ...[
              Card(
                color: RozyColors.beige,
                child: Padding(
                  padding: const EdgeInsets.all(20),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text('Trip completed', style: Theme.of(context).textTheme.titleLarge),
                      const SizedBox(height: 12),
                      Text('Cash collected: UGX ${_result!.finalFare}'),
                      Text('Rozy fee deducted: UGX ${_result!.rozyFee}'),
                      Text('Wallet balance: UGX ${_result!.walletBalance}'),
                    ],
                  ),
                ),
              ),
              const Spacer(),
              ElevatedButton(onPressed: _backToShell, child: const Text('Back to home')),
            ] else if (trip?.status == 'driver_arriving' && trip?.arrivedAt == null) ...[
              const Text(
                'Tap when you reach the pickup point.',
                style: TextStyle(fontSize: 16),
              ),
              const Spacer(),
              ElevatedButton(
                onPressed: _busy ? null : _arrived,
                child: _busy
                    ? const SizedBox(height: 20, width: 20, child: CircularProgressIndicator(strokeWidth: 2))
                    : const Text("I've arrived"),
              ),
            ] else if (trip?.status == 'driver_arriving') ...[
              TextField(
                controller: _pinController,
                keyboardType: TextInputType.number,
                maxLength: 4,
                decoration: const InputDecoration(
                  labelText: 'Passenger PIN',
                  hintText: '4 digits',
                  counterText: '',
                ),
              ),
              const SizedBox(height: 12),
              const Text('Ask the passenger for their trip PIN to start.'),
              const Spacer(),
              ElevatedButton(
                onPressed: _busy ? null : _start,
                child: _busy
                    ? const SizedBox(height: 20, width: 20, child: CircularProgressIndicator(strokeWidth: 2))
                    : const Text('Start trip'),
              ),
            ] else if (trip?.status == 'in_progress') ...[
              const Text(
                'Drive to the destination. Collect cash from the passenger, then complete the trip.',
                style: TextStyle(fontSize: 16),
              ),
              const Spacer(),
              ElevatedButton(
                onPressed: _busy ? null : _complete,
                child: _busy
                    ? const SizedBox(height: 20, width: 20, child: CircularProgressIndicator(strokeWidth: 2))
                    : const Text('Complete trip'),
              ),
            ] else ...[
              const Spacer(),
              const Center(child: CircularProgressIndicator()),
            ],
            if (_error != null) ...[
              const SizedBox(height: 12),
              Text(_error!, style: const TextStyle(color: Colors.red)),
            ],
          ],
        ),
      ),
    );
  }
}
