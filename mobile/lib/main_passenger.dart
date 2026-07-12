import 'package:flutter/material.dart';

import 'core/services/app_services.dart';
import 'core/theme/rozy_theme.dart';
import 'features/auth/auth_gate.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await AppServices.bootstrap();
  runApp(const RozyPassengerApp());
}

class RozyPassengerApp extends StatelessWidget {
  const RozyPassengerApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Rozy',
      debugShowCheckedModeBanner: false,
      theme: RozyTheme.forVariant(RozyAppVariant.passenger),
      home: const AuthGate(role: 'passenger'),
    );
  }
}
