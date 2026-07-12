import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../core/api/api_client.dart';
import '../../core/services/app_services.dart';
import '../../core/theme/rozy_colors.dart';
import 'otp_screen.dart';

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key, required this.role});

  /// `passenger` or `driver`
  final String role;

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _controller = TextEditingController();
  bool _loading = false;
  String? _error;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  String get _phone {
    var digits = _controller.text.replaceAll(RegExp(r'\D'), '');
    if (digits.startsWith('256')) digits = digits.substring(3);
    if (digits.startsWith('0')) digits = digits.substring(1);
    return '+256$digits';
  }

  Future<void> _submit() async {
    setState(() {
      _loading = true;
      _error = null;
    });

    try {
      await AppServices.live.auth.requestOtp(_phone);
      if (!mounted) return;
      Navigator.of(context).push(
        MaterialPageRoute(
          builder: (_) => OtpScreen(phone: _phone, role: widget.role),
        ),
      );
    } on ApiException catch (e) {
      setState(() => _error = e.message);
    } catch (_) {
      setState(() => _error = 'Could not send OTP. Check your connection.');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final isDriver = widget.role == 'driver';

    return Scaffold(
      appBar: isDriver ? AppBar(title: const Text('Rozy Driver')) : null,
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              if (!isDriver) ...[
                const SizedBox(height: 32),
                Text(
                  'Rozy',
                  style: Theme.of(context).textTheme.headlineLarge?.copyWith(
                        fontWeight: FontWeight.bold,
                        color: RozyColors.charcoal,
                      ),
                ),
                const SizedBox(height: 8),
              ],
              Text(
                isDriver ? 'Sign in to drive' : 'Get a ride in Mbarara',
                style: Theme.of(context).textTheme.titleMedium?.copyWith(
                      color: RozyColors.grey,
                    ),
              ),
              const SizedBox(height: 32),
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(20),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text('Phone number', style: Theme.of(context).textTheme.labelLarge),
                      const SizedBox(height: 12),
                      TextField(
                        controller: _controller,
                        keyboardType: TextInputType.phone,
                        inputFormatters: [FilteringTextInputFormatter.digitsOnly],
                        decoration: const InputDecoration(
                          prefixText: '+256 ',
                          hintText: '700000001',
                        ),
                        onSubmitted: (_) => _submit(),
                      ),
                      if (_error != null) ...[
                        const SizedBox(height: 12),
                        Text(_error!, style: const TextStyle(color: Colors.red)),
                      ],
                    ],
                  ),
                ),
              ),
              const Spacer(),
              ElevatedButton(
                onPressed: _loading ? null : _submit,
                child: _loading
                    ? const SizedBox(
                        height: 22,
                        width: 22,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Text('Send OTP'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
