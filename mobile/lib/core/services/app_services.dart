import '../api/api_client.dart';
import '../api/auth_api.dart';
import '../api/operator_api.dart';
import '../api/trip_api.dart';
import '../api/verification_api.dart';
import '../api/wallet_api.dart';
import '../storage/session_storage.dart';

class AppServices {
  AppServices._({
    required this.api,
    required this.auth,
    required this.operator,
    required this.trip,
    required this.verification,
    required this.wallet,
    required this.session,
  });

  final ApiClient api;
  final AuthApi auth;
  final OperatorApi operator;
  final TripApi trip;
  final VerificationApi verification;
  final WalletApi wallet;
  final SessionStorage session;

  static late AppServices live;

  static Future<AppServices> bootstrap() async {
    final api = ApiClient();
    final session = SessionStorage();
    final token = await session.getToken();
    if (token != null) api.setToken(token);

    live = AppServices._(
      api: api,
      auth: AuthApi(api),
      operator: OperatorApi(api),
      trip: TripApi(api),
      verification: VerificationApi(api),
      wallet: WalletApi(api),
      session: session,
    );
    return live;
  }

  Future<void> persistLogin(AuthSession session) async {
    api.setToken(session.token);
    await this.session.saveSession(
      token: session.token,
      phone: session.user.phone,
      role: session.user.role,
    );
  }

  Future<void> logout() async {
    api.setToken(null);
    await session.clear();
  }
}
