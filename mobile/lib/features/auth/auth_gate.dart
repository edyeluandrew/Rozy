import 'package:flutter/material.dart';

import '../../core/api/api_client.dart';
import '../../core/services/app_services.dart';
import '../../core/theme/rozy_colors.dart';
import '../auth/login_screen.dart';
import '../driver/driver_shell.dart';
import '../driver/driver_active_trip_screen.dart';
import '../passenger/ride_request_screen.dart';
import '../passenger/passenger_active_trip_screen.dart';
import '../operator/mode_selection_screen.dart';

class AuthGate extends StatefulWidget {
  const AuthGate({super.key, required this.role});

  final String role;

  @override
  State<AuthGate> createState() => _AuthGateState();
}

class _AuthGateState extends State<AuthGate> {
  bool _loading = true;
  Widget? _screen;

  @override
  void initState() {
    super.initState();
    _resolve();
  }

  Future<void> _resolve() async {
    try {
      final token = await AppServices.live.session.getToken();
      if (token == null) {
        setState(() {
          _screen = LoginScreen(role: widget.role);
          _loading = false;
        });
        return;
      }

      AppServices.live.api.setToken(token);

      if (widget.role == 'driver') {
        final status = await AppServices.live.operator.profileStatus();
        if (!status.registered) {
          setState(() {
            _screen = const ModeSelectionScreen();
            _loading = false;
          });
          return;
        }
        final activeTrip = await AppServices.live.operator.activeTrip();
        setState(() {
          _screen = activeTrip != null
              ? DriverActiveTripScreen(tripId: activeTrip.id, profile: status.profile!)
              : DriverShell(profile: status.profile!);
          _loading = false;
        });
      } else {
        await AppServices.live.auth.me();
        final activeTrip = await AppServices.live.trip.activeTrip();
        final pin = await AppServices.live.session.getTripPin();
        setState(() {
          _screen = activeTrip != null
              ? PassengerActiveTripScreen(initialTrip: activeTrip, tripPin: pin)
              : const RideRequestScreen();
          _loading = false;
        });
      }
    } catch (_) {
      await AppServices.live.logout();
      setState(() {
        _screen = LoginScreen(role: widget.role);
        _loading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_loading) {
      return Scaffold(
        backgroundColor: RozyColors.cream,
        body: Center(
          child: CircularProgressIndicator(color: RozyColors.gold),
        ),
      );
    }
    return _screen ?? LoginScreen(role: widget.role);
  }
}
