import 'api_client.dart';
import '../models/auth_user.dart';

class AuthApi {
  AuthApi(this._client);

  final ApiClient _client;

  Future<void> requestOtp(String phone) async {
    await _client.post('/auth/otp/request', body: {'phone': phone});
  }

  Future<AuthSession> verifyOtp({
    required String phone,
    required String code,
    required String role,
  }) async {
    final data = await _client.post('/auth/otp/verify', body: {
      'phone': phone,
      'code': code,
      'role': role,
    });

    final userJson = data['user'] as Map<String, dynamic>;
    return AuthSession(
      token: data['token'] as String,
      user: AuthUser.fromJson(userJson),
    );
  }

  Future<AuthUser> me() async {
    final data = await _client.get('/auth/me', auth: true);
    return AuthUser.fromJson(data);
  }
}

class AuthSession {
  const AuthSession({required this.token, required this.user});

  final String token;
  final AuthUser user;
}
