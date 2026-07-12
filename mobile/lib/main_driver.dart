import 'package:flutter/material.dart';

import 'core/services/app_services.dart';
import 'core/theme/rozy_theme.dart';
import 'features/auth/auth_gate.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await AppServices.bootstrap();
  runApp(const RozyDriverApp());
}

class RozyDriverApp extends StatelessWidget {
  const RozyDriverApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Rozy Driver',
      debugShowCheckedModeBanner: false,
      theme: RozyTheme.forVariant(RozyAppVariant.driver),
      home: const AuthGate(role: 'driver'),
    );
  }
}
