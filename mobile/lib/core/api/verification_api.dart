import 'api_client.dart';

class VerificationApi {
  VerificationApi(this._client);

  final ApiClient _client;

  Future<Map<String, dynamic>> status() async {
    return _client.get('/operator/verification/status', auth: true);
  }

  Future<Map<String, dynamic>> submit(Map<String, dynamic> body) async {
    return _client.post('/operator/verification/submit', auth: true, body: body);
  }
}
