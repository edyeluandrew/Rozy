import 'package:flutter/material.dart';

import '../../core/services/app_services.dart';
import '../../core/theme/rozy_colors.dart';
import '../auth/login_screen.dart';

class PassengerHome extends StatelessWidget {
  const PassengerHome({super.key});

  Future<void> _logout(BuildContext context) async {
    await AppServices.live.logout();
    if (!context.mounted) return;
    Navigator.of(context).pushAndRemoveUntil(
      MaterialPageRoute(builder: (_) => const LoginScreen(role: 'passenger')),
      (_) => false,
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Rozy'),
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
            Card(
              child: Padding(
                padding: const EdgeInsets.all(20),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('Where to?', style: Theme.of(context).textTheme.titleLarge),
                    const SizedBox(height: 8),
                    Text(
                      'Map & ride request coming next',
                      style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                            color: RozyColors.grey,
                          ),
                    ),
                  ],
                ),
              ),
            ),
            const Spacer(),
            ElevatedButton(onPressed: () {}, child: const Text('Request Rozy Boda')),
          ],
        ),
      ),
    );
  }
}
