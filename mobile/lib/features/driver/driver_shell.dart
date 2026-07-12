import 'dart:async';

import 'package:flutter/material.dart';

import 'package:geolocator/geolocator.dart';

import '../../core/models/operator_profile.dart';
import '../../core/services/app_services.dart';
import '../../core/services/driver_location_tracker.dart';
import '../../core/services/rozy_realtime.dart';
import '../../core/theme/rozy_colors.dart';
import '../../core/theme/rozy_theme.dart';
import '../auth/login_screen.dart';
import '../verification/verification_wizard.dart';
import 'driver_active_trip_screen.dart';
import 'driver_wallet_screen.dart';

class DriverShell extends StatefulWidget {
  const DriverShell({super.key, required this.profile});

  final OperatorProfile profile;

  @override
  State<DriverShell> createState() => _DriverShellState();
}

class _DriverShellState extends State<DriverShell> {
  late OperatorProfile _profile;
  Map<String, dynamic>? _incomingTrip;
  bool _busy = false;
  Timer? _pollTimer;
  String? _error;
  RozyRealtime? _realtime;

  @override
  void initState() {
    super.initState();
    _profile = widget.profile;
    _connectRealtime();
    if (_isOnline) {
      _startPolling();
    }
  }

  Future<void> _connectRealtime() async {
    final token = await AppServices.live.session.getToken();
    if (token == null || !mounted) return;
    _realtime?.disconnect();
    _realtime = RozyRealtime(token: token);
    _realtime!.connect((event, payload) {
      if (!mounted) return;
      switch (event) {
        case 'operator:ride_request':
          setState(() => _incomingTrip = Map<String, dynamic>.from(payload));
        case 'wallet:updated':
          final balance = payload['balance'];
          if (balance is num) {
            setState(() {
              _profile = OperatorProfile(
                id: _profile.id,
                rideType: _profile.rideType,
                operatorType: _profile.operatorType,
                status: _profile.status == 'wallet_blocked' ? 'offline' : _profile.status,
                walletBalance: balance.toInt(),
                walletMinBalance: _profile.walletMinBalance,
              );
            });
          }
      }
    });
  }

  @override
  void dispose() {
    _pollTimer?.cancel();
    _realtime?.disconnect();
    super.dispose();
  }

  bool get _isOnline =>
      _profile.status == 'available' || _profile.status == 'busy';

  bool get _pending => _profile.status == 'pending_verification';

  bool get _walletBlocked => _profile.status == 'wallet_blocked';

  void _startPolling() {
    _pollTimer?.cancel();
    _pollTimer = Timer.periodic(const Duration(seconds: 4), (_) => _pollIncoming());
    _pollIncoming();
  }

  void _stopPolling() {
    _pollTimer?.cancel();
    _pollTimer = null;
  }

  Future<void> _pollIncoming() async {
    if (!_isOnline || _busy) return;
    try {
      final trip = await AppServices.live.operator.incomingTrip();
      if (!mounted) return;
      setState(() => _incomingTrip = trip);
    } catch (_) {}
  }

  Future<void> _logout(BuildContext context) async {
    _stopPolling();
    _realtime?.disconnect();
    driverLocationTracker.stop();
    await AppServices.live.logout();
    if (!context.mounted) return;
    Navigator.of(context).pushAndRemoveUntil(
      MaterialPageRoute(builder: (_) => const LoginScreen(role: 'driver')),
      (_) => false,
    );
  }

  Future<void> _toggleOnline() async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      double lat = -0.6072;
      double lng = 30.6586;
      try {
        final pos = await Geolocator.getCurrentPosition(
          locationSettings: const LocationSettings(accuracy: LocationAccuracy.high),
        );
        lat = pos.latitude;
        lng = pos.longitude;
      } catch (_) {}

      final updated = _isOnline
          ? await AppServices.live.operator.goOffline()
          : await AppServices.live.operator.goOnline(lat: lat, lng: lng);
      if (!mounted) return;
      setState(() => _profile = updated);
      if (_isOnline) {
        _startPolling();
        driverLocationTracker.start();
      } else {
        _stopPolling();
        driverLocationTracker.stop();
        _incomingTrip = null;
      }
    } catch (e) {
      if (mounted) setState(() => _error = e.toString());
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _acceptTrip() async {
    final tripId = _incomingTrip?['id'] as String? ?? _incomingTrip?['trip_id'] as String?;
    if (tripId == null) return;
    setState(() => _busy = true);
    try {
      await AppServices.live.operator.acceptTrip(tripId);
      if (!mounted) return;
      Navigator.of(context).pushReplacement(
        MaterialPageRoute(
          builder: (_) => DriverActiveTripScreen(
            tripId: tripId,
            profile: _profile,
          ),
        ),
      );
    } catch (e) {
      if (mounted) setState(() => _error = e.toString());
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _showRechargeSheet() async {
    final amountController = TextEditingController(text: '10000');
    var provider = 'mtn';
    String? sheetError;

    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      builder: (ctx) {
        return StatefulBuilder(
          builder: (context, setSheetState) {
            return Padding(
              padding: EdgeInsets.only(
                left: 24,
                right: 24,
                top: 24,
                bottom: MediaQuery.of(ctx).viewInsets.bottom + 24,
              ),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text('Top up wallet', style: Theme.of(context).textTheme.titleLarge),
                  const SizedBox(height: 16),
                  TextField(
                    controller: amountController,
                    keyboardType: TextInputType.number,
                    decoration: const InputDecoration(
                      labelText: 'Amount (UGX)',
                      hintText: '10000',
                    ),
                  ),
                  const SizedBox(height: 12),
                  SegmentedButton<String>(
                    segments: const [
                      ButtonSegment(value: 'mtn', label: Text('MTN MoMo')),
                      ButtonSegment(value: 'airtel', label: Text('Airtel')),
                    ],
                    selected: {provider},
                    onSelectionChanged: (v) => setSheetState(() => provider = v.first),
                  ),
                  if (sheetError != null) ...[
                    const SizedBox(height: 12),
                    Text(sheetError!, style: const TextStyle(color: Colors.red)),
                  ],
                  const SizedBox(height: 16),
                  ElevatedButton(
                    onPressed: () async {
                      final amount = int.tryParse(amountController.text.trim()) ?? 0;
                      if (amount < 1000) {
                        setSheetState(() => sheetError = 'Minimum top-up is UGX 1,000');
                        return;
                      }
                      try {
                        final result = await AppServices.live.wallet.initiateRecharge(
                          amount: amount,
                          provider: provider,
                        );
                        if (!context.mounted) return;
                        Navigator.of(context).pop();
                        ScaffoldMessenger.of(this.context).showSnackBar(
                          SnackBar(content: Text(result.instructions)),
                        );
                      } catch (e) {
                        setSheetState(() => sheetError = e.toString());
                      }
                    },
                    child: const Text('Request payment'),
                  ),
                ],
              ),
            );
          },
        );
      },
    );
  }

  Future<void> _rejectTrip() async {
    final tripId = _incomingTrip?['id'] as String? ?? _incomingTrip?['trip_id'] as String?;
    if (tripId == null) return;
    setState(() => _busy = true);
    try {
      await AppServices.live.operator.rejectTrip(tripId);
      if (!mounted) return;
      setState(() => _incomingTrip = null);
    } catch (e) {
      if (mounted) setState(() => _error = e.toString());
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Rozy Driver'),
        actions: [
          IconButton(
            icon: const Icon(Icons.logout),
            onPressed: () => _logout(context),
          ),
        ],
      ),
      body: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Container(
              padding: const EdgeInsets.all(20),
              decoration: RozyTheme.premiumCardDecoration,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    _profile.rideTypeLabel,
                    style: Theme.of(context).textTheme.titleMedium?.copyWith(
                          color: RozyColors.cream,
                        ),
                  ),
                  const SizedBox(height: 8),
                  Text(
                    'Status: ${_profile.status}',
                    style: Theme.of(context).textTheme.bodySmall?.copyWith(
                          color: RozyColors.grey,
                        ),
                  ),
                  const SizedBox(height: 12),
                  Text(
                    'Wallet balance',
                    style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                          color: RozyColors.grey,
                        ),
                  ),
                  Text(
                    'UGX ${_profile.walletBalance}',
                    style: Theme.of(context).textTheme.headlineMedium?.copyWith(
                          color: RozyColors.gold,
                          fontWeight: FontWeight.bold,
                        ),
                  ),
                  if (_profile.walletBalance < _profile.walletMinBalance)
                    Padding(
                      padding: const EdgeInsets.only(top: 8),
                      child: Text(
                        'Minimum UGX ${_profile.walletMinBalance} required to go online',
                        style: Theme.of(context).textTheme.bodySmall?.copyWith(
                              color: RozyColors.gold,
                            ),
                      ),
                    ),
                  const SizedBox(height: 12),
                  Row(
                    children: [
                      Expanded(
                        child: OutlinedButton(
                          onPressed: _busy
                              ? null
                              : () async {
                                  final updated = await Navigator.of(context).push<OperatorProfile>(
                                    MaterialPageRoute(
                                      builder: (_) => DriverWalletScreen(profile: _profile),
                                    ),
                                  );
                                  if (updated != null && mounted) {
                                    setState(() => _profile = updated);
                                  } else if (mounted) {
                                    try {
                                      final status = await AppServices.live.operator.profileStatus();
                                      if (status.profile != null) {
                                        setState(() => _profile = status.profile!);
                                      }
                                    } catch (_) {}
                                  }
                                },
                          style: OutlinedButton.styleFrom(
                            foregroundColor: RozyColors.cream,
                            side: const BorderSide(color: RozyColors.gold),
                          ),
                          child: const Text('Wallet'),
                        ),
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: OutlinedButton(
                          onPressed: _busy ? null : _showRechargeSheet,
                          style: OutlinedButton.styleFrom(
                            foregroundColor: RozyColors.cream,
                            side: const BorderSide(color: RozyColors.gold),
                          ),
                          child: const Text('Quick top-up'),
                        ),
                      ),
                    ],
                  ),
                ],
              ),
            ),
            const SizedBox(height: 16),
            if (_pending)
              Card(
                color: RozyColors.beige,
                child: Padding(
                  padding: const EdgeInsets.all(16),
                  child: Row(
                    children: [
                      const Icon(Icons.info_outline, color: RozyColors.darkGold),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Text(
                          'Verification pending. Upload documents to go online.',
                          style: Theme.of(context).textTheme.bodyMedium,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            if (_walletBlocked)
              Card(
                color: RozyColors.beige,
                child: Padding(
                  padding: const EdgeInsets.all(16),
                  child: Text(
                    'Wallet too low. Top up via MTN or Airtel Money — your balance updates automatically when payment succeeds.',
                    style: Theme.of(context).textTheme.bodyMedium,
                  ),
                ),
              ),
            if (_incomingTrip != null) ...[
              const SizedBox(height: 16),
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(16),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Incoming trip',
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                      const SizedBox(height: 8),
                      Text('Fare: UGX ${_incomingTrip!['estimated_fare'] ?? '—'}'),
                      Text('Pickup: ${_incomingTrip!['pickup_landmark'] ?? 'See map'}'),
                      const SizedBox(height: 12),
                      Row(
                        children: [
                          Expanded(
                            child: OutlinedButton(
                              onPressed: _busy ? null : _rejectTrip,
                              child: const Text('Reject'),
                            ),
                          ),
                          const SizedBox(width: 12),
                          Expanded(
                            child: ElevatedButton(
                              onPressed: _busy ? null : _acceptTrip,
                              child: const Text('Accept'),
                            ),
                          ),
                        ],
                      ),
                    ],
                  ),
                ),
              ),
            ],
            if (_error != null) ...[
              const SizedBox(height: 12),
              Text(_error!, style: const TextStyle(color: Colors.red)),
            ],
            const Spacer(),
            if (_pending)
              ElevatedButton(
                onPressed: () {
                  Navigator.of(context).push(
                    MaterialPageRoute(
                      builder: (_) => VerificationWizard(profile: _profile),
                    ),
                  );
                },
                child: const Text('Complete verification'),
              )
            else
              ElevatedButton(
                onPressed: (_busy || _walletBlocked) ? null : _toggleOnline,
                child: _busy
                    ? const SizedBox(
                        height: 20,
                        width: 20,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : Text(_isOnline ? 'Go offline' : 'Go online'),
              ),
          ],
        ),
      ),
    );
  }
}
