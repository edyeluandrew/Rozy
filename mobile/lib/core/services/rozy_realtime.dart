import 'dart:async';
import 'dart:convert';

import 'package:web_socket_channel/web_socket_channel.dart';

import '../config/app_config.dart';

typedef RealtimeHandler = void Function(String event, Map<String, dynamic> payload);

class RozyRealtime {
  RozyRealtime({required this.token});

  final String token;
  WebSocketChannel? _channel;
  StreamSubscription? _sub;

  String get _wsUrl {
    final base = AppConfig.apiBaseUrl.replaceFirst('http://', 'ws://').replaceFirst('https://', 'wss://');
    return '$base/ws?token=$token';
  }

  void connect(RealtimeHandler onEvent) {
    disconnect();
    _channel = WebSocketChannel.connect(Uri.parse(_wsUrl));
    _sub = _channel!.stream.listen((raw) {
      try {
        final data = jsonDecode(raw as String) as Map<String, dynamic>;
        final event = data['event'] as String? ?? '';
        final payload = data['payload'] as Map<String, dynamic>? ?? {};
        onEvent(event, payload);
      } catch (_) {}
    });
  }

  void disconnect() {
    _sub?.cancel();
    _sub = null;
    _channel?.sink.close();
    _channel = null;
  }
}
