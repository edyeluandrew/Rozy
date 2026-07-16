import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';

import '../../core/api/api_client.dart';
import '../../core/api/verification_api.dart';
import '../../core/models/operator_profile.dart';
import '../../core/services/app_services.dart';
import '../../core/theme/rozy_colors.dart';
import '../driver/driver_shell.dart';

class _DocSlot {
  _DocSlot({required this.type, required this.label});

  final String type;
  final String label;
  Uint8List? bytes;
  String? filename;
  UploadedDoc? uploaded;
  bool uploading = false;
  String? error;
}

class VerificationWizard extends StatefulWidget {
  const VerificationWizard({super.key, required this.profile});

  final OperatorProfile profile;

  @override
  State<VerificationWizard> createState() => _VerificationWizardState();
}

class _VerificationWizardState extends State<VerificationWizard> {
  final _formKey = GlobalKey<FormState>();
  final _picker = ImagePicker();
  final _name = TextEditingController();
  final _nin = TextEditingController();
  final _permit = TextEditingController();
  final _plate = TextEditingController();
  final _bikeColor = TextEditingController();
  final _carMake = TextEditingController();
  bool _loading = false;
  String? _error;

  late final List<_DocSlot> _docs;

  bool get _isBoda => widget.profile.rideType == 'boda';

  bool get _docsReady => _docs.every((d) => d.uploaded != null);

  @override
  void initState() {
    super.initState();
    _docs = [
      _DocSlot(type: 'nin_front', label: 'National ID (front)'),
      _DocSlot(type: 'selfie', label: 'Selfie with ID'),
      _DocSlot(type: 'permit', label: 'Driving permit'),
    ];
  }

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

  Future<void> _pickDoc(_DocSlot slot, ImageSource source) async {
    try {
      final picked = await _picker.pickImage(
        source: source,
        maxWidth: 2000,
        imageQuality: 85,
      );
      if (picked == null) return;

      final bytes = await picked.readAsBytes();
      setState(() {
        slot.bytes = bytes;
        slot.filename = picked.name;
        slot.uploaded = null;
        slot.error = null;
        slot.uploading = true;
      });

      final uploaded = await AppServices.live.verification.upload(
        docType: slot.type,
        bytes: bytes,
        filename: picked.name.isNotEmpty ? picked.name : '${slot.type}.jpg',
      );

      if (!mounted) return;
      setState(() {
        slot.uploaded = uploaded;
        slot.uploading = false;
      });
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() {
        slot.uploading = false;
        slot.error = e.message;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        slot.uploading = false;
        slot.error = 'Upload failed';
      });
    }
  }

  Future<void> _showPickOptions(_DocSlot slot) async {
    await showModalBottomSheet<void>(
      context: context,
      builder: (ctx) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: const Icon(Icons.photo_camera_outlined),
              title: const Text('Take photo'),
              onTap: () {
                Navigator.pop(ctx);
                _pickDoc(slot, ImageSource.camera);
              },
            ),
            ListTile(
              leading: const Icon(Icons.photo_library_outlined),
              title: const Text('Choose from gallery'),
              onTap: () {
                Navigator.pop(ctx);
                _pickDoc(slot, ImageSource.gallery);
              },
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;
    if (!_docsReady) {
      setState(() => _error = 'Upload all required documents first');
      return;
    }

    setState(() {
      _loading = true;
      _error = null;
    });

    try {
      final body = {
        'legal_name': _name.text.trim(),
        'nin': _nin.text.trim(),
        'permit_number': _permit.text.trim(),
        'permit_expiry': '2027-01-01',
        'insurance_expiry': '2026-12-01',
        'plate': _plate.text.trim(),
        if (_isBoda) 'bike_color': _bikeColor.text.trim(),
        if (!_isBoda) 'car_make': _carMake.text.trim(),
        'documents': _docs.map((d) => d.uploaded!.toJson()).toList(),
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

  Widget _docTile(_DocSlot slot) {
    final ready = slot.uploaded != null;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Row(
          children: [
            ClipRRect(
              borderRadius: BorderRadius.circular(8),
              child: SizedBox(
                width: 56,
                height: 56,
                child: slot.bytes != null
                    ? Image.memory(slot.bytes!, fit: BoxFit.cover)
                    : ColoredBox(
                        color: RozyColors.beige,
                        child: Icon(
                          ready ? Icons.check_circle : Icons.image_outlined,
                          color: ready ? Colors.green : RozyColors.grey,
                        ),
                      ),
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(slot.label, style: Theme.of(context).textTheme.titleSmall),
                  if (slot.uploading)
                    const Text('Uploading…', style: TextStyle(color: RozyColors.grey, fontSize: 12))
                  else if (ready)
                    const Text('Uploaded', style: TextStyle(color: Colors.green, fontSize: 12))
                  else if (slot.error != null)
                    Text(slot.error!, style: const TextStyle(color: Colors.red, fontSize: 12))
                  else
                    const Text('Required', style: TextStyle(color: RozyColors.grey, fontSize: 12)),
                ],
              ),
            ),
            slot.uploading
                ? const SizedBox(
                    width: 24,
                    height: 24,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : TextButton(
                    onPressed: _loading ? null : () => _showPickOptions(slot),
                    child: Text(ready ? 'Replace' : 'Add'),
                  ),
          ],
        ),
      ),
    );
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
            Text('Documents', style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 8),
            ..._docs.map(_docTile),
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
              onPressed: (_loading || !_docsReady) ? null : _submit,
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
