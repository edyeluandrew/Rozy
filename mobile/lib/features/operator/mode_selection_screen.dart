import 'package:flutter/material.dart';

import '../../core/api/api_client.dart';
import '../../core/services/app_services.dart';
import '../../core/theme/rozy_colors.dart';
import '../driver/driver_shell.dart';

class ModeSelectionScreen extends StatefulWidget {
  const ModeSelectionScreen({super.key});

  @override
  State<ModeSelectionScreen> createState() => _ModeSelectionScreenState();
}

class _ModeSelectionScreenState extends State<ModeSelectionScreen> {
  String? _selected;
  bool _loading = false;
  String? _error;

  Future<void> _register() async {
    if (_selected == null) return;

    setState(() {
      _loading = true;
      _error = null;
    });

    try {
      final profile = await AppServices.live.operator.register(_selected!);
      if (!mounted) return;
      Navigator.of(context).pushReplacement(
        MaterialPageRoute(builder: (_) => DriverShell(profile: profile)),
      );
    } on ApiException catch (e) {
      setState(() => _error = e.message);
    } catch (_) {
      setState(() => _error = 'Registration failed. Try again.');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Choose your vehicle')),
      body: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              'One account, one mode — choose carefully. This cannot be changed later.',
              style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                    color: RozyColors.grey,
                  ),
            ),
            const SizedBox(height: 24),
            _ModeCard(
              title: 'Rozy Boda',
              subtitle: 'Motorcycle',
              value: 'boda',
              groupValue: _selected,
              onSelect: (v) => setState(() => _selected = v),
            ),
            const SizedBox(height: 12),
            _ModeCard(
              title: 'Rozy Car Basic',
              subtitle: 'Standard car · 1–4 passengers',
              value: 'car_basic',
              groupValue: _selected,
              onSelect: (v) => setState(() => _selected = v),
            ),
            const SizedBox(height: 12),
            _ModeCard(
              title: 'Rozy Car XL',
              subtitle: 'Larger car · 5–7 passengers',
              value: 'car_xl',
              groupValue: _selected,
              onSelect: (v) => setState(() => _selected = v),
            ),
            if (_error != null) ...[
              const SizedBox(height: 16),
              Text(_error!, style: const TextStyle(color: Colors.red)),
            ],
            const Spacer(),
            ElevatedButton(
              onPressed: _loading || _selected == null ? null : _register,
              child: _loading
                  ? const SizedBox(
                      height: 22,
                      width: 22,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Text('Continue to verification'),
            ),
          ],
        ),
      ),
    );
  }
}

class _ModeCard extends StatelessWidget {
  const _ModeCard({
    required this.title,
    required this.subtitle,
    required this.value,
    required this.groupValue,
    required this.onSelect,
  });

  final String title;
  final String subtitle;
  final String value;
  final String? groupValue;
  final ValueChanged<String> onSelect;

  @override
  Widget build(BuildContext context) {
    final selected = groupValue == value;
    return InkWell(
      onTap: () => onSelect(value),
      borderRadius: BorderRadius.circular(16),
      child: Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: selected ? RozyColors.beige : RozyColors.white,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(
            color: selected ? RozyColors.gold : RozyColors.border,
            width: selected ? 2 : 1,
          ),
        ),
        child: Row(
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(title, style: Theme.of(context).textTheme.titleMedium),
                  Text(subtitle, style: Theme.of(context).textTheme.bodySmall),
                ],
              ),
            ),
            Icon(
              selected ? Icons.radio_button_checked : Icons.radio_button_off,
              color: selected ? RozyColors.darkGold : RozyColors.grey,
            ),
          ],
        ),
      ),
    );
  }
}
