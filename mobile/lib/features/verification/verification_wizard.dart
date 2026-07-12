import 'package:flutter/material.dart';

import '../../core/api/api_client.dart';
import '../../core/models/operator_profile.dart';
import '../../core/services/app_services.dart';
import '../../core/theme/rozy_colors.dart';
import '../driver/driver_shell.dart';

class VerificationWizard extends StatefulWidget {
  const VerificationWizard({super.key, required this.profile});

  final OperatorProfile profile;

  @override
  State<VerificationWizard> createState() => _VerificationWizardState();
}

class _VerificationWizardState extends State<VerificationWizard> {
  final _formKey = GlobalKey<FormState>();
  final _name = TextEditingController();
  final _nin = TextEditingController();
  final _permit = TextEditingController();
  final _plate = TextEditingController();
  final _bikeColor = TextEditingController();
  final _carMake = TextEditingController();
  bool _loading = false;
  String? _error;

  bool get _isBoda => widget.profile.rideType == 'boda';

  @override
  void dispose() {
    _name.dispose();
    _nin.dispose();
    _permit.dispose();
    _plate.dispose();
    _bikeColor.dispose();
    _carMake.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() { _loading = true; _error = null; });

    try {
      // MVP: document uploads come next; submit metadata with placeholder docs for dev
      final body = {
        'legal_name': _name.text.trim(),
        'nin': _nin.text.trim(),
        'permit_number': _permit.text.trim(),
        'permit_expiry': '2027-01-01',
        'insurance_expiry': '2026-12-01',
        'plate': _plate.text.trim(),
        if (_isBoda) 'bike_color': _bikeColor.text.trim(),
        if (!_isBoda) 'car_make': _carMake.text.trim(),
        'documents': [
          {
            'doc_type': 'nin_front',
            'storage_key': 'pending/nin_front',
            'sha256_hash': 'pending',
            'mime_type': 'image/jpeg',
          },
          {
            'doc_type': 'selfie',
            'storage_key': 'pending/selfie',
            'sha256_hash': 'pending',
            'mime_type': 'image/jpeg',
          },
          {
            'doc_type': 'permit',
            'storage_key': 'pending/permit',
            'sha256_hash': 'pending',
            'mime_type': 'image/jpeg',
          },
        ],
      };
      await AppServices.live.verification.submit(body);
      if (!mounted) return;
      Navigator.of(context).pushReplacement(
        MaterialPageRoute(builder: (_) => DriverShell(profile: widget.profile)),
      );
    } on ApiException catch (e) {
      setState(() => _error = e.message);
    } catch (_) {
      setState(() => _error = 'Submission failed');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Verification')),
      body: Form(
        key: _formKey,
        child: ListView(
          padding: const EdgeInsets.all(24),
          children: [
            Text(
              'Submit your details for ${widget.profile.rideTypeLabel}. Admin will review before you can go online.',
              style: Theme.of(context).textTheme.bodyMedium?.copyWith(color: RozyColors.grey),
            ),
            const SizedBox(height: 16),
            TextFormField(
              controller: _name,
              decoration: const InputDecoration(labelText: 'Full name (as on ID)'),
              validator: (v) => v == null || v.isEmpty ? 'Required' : null,
            ),
            TextFormField(
              controller: _nin,
              decoration: const InputDecoration(labelText: 'National ID (NIN)'),
              validator: (v) => v == null || v.length < 5 ? 'Required' : null,
            ),
            TextFormField(
              controller: _permit,
              decoration: const InputDecoration(labelText: 'Driving permit number'),
              validator: (v) => v == null || v.isEmpty ? 'Required' : null,
            ),
            TextFormField(
              controller: _plate,
              decoration: const InputDecoration(labelText: 'Vehicle plate number'),
              validator: (v) => v == null || v.isEmpty ? 'Required' : null,
            ),
            if (_isBoda)
              TextFormField(
                controller: _bikeColor,
                decoration: const InputDecoration(labelText: 'Bike colour'),
                validator: (v) => v == null || v.isEmpty ? 'Required' : null,
              )
            else
              TextFormField(
                controller: _carMake,
                decoration: const InputDecoration(labelText: 'Car make / model'),
                validator: (v) => v == null || v.isEmpty ? 'Required' : null,
              ),
            if (_error != null) ...[
              const SizedBox(height: 12),
              Text(_error!, style: const TextStyle(color: Colors.red)),
            ],
            const SizedBox(height: 24),
            ElevatedButton(
              onPressed: _loading ? null : _submit,
              child: _loading
                  ? const SizedBox(height: 22, width: 22, child: CircularProgressIndicator(strokeWidth: 2))
                  : const Text('Submit for review'),
            ),
          ],
        ),
      ),
    );
  }
}
