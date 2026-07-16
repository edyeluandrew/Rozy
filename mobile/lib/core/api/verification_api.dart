import 'api_client.dart';

class UploadedDoc {
  UploadedDoc({
    required this.docType,
    required this.storageKey,
    required this.sha256Hash,
    required this.mimeType,
  });

  final String docType;
  final String storageKey;
  final String sha256Hash;
  final String mimeType;

  Map<String, dynamic> toJson() => {
        'doc_type': docType,
        'storage_key': storageKey,
        'sha256_hash': sha256Hash,
        'mime_type': mimeType,
      };
}

class VerificationApi {
  VerificationApi(this._client);

  final ApiClient _client;

  Future<Map<String, dynamic>> status() async {
    return _client.get('/operator/verification/status', auth: true);
  }

  Future<Map<String, dynamic>> submit(Map<String, dynamic> body) async {
    return _client.post('/operator/verification/submit', auth: true, body: body);
  }

  Future<UploadedDoc> upload({
    required String docType,
    required List<int> bytes,
    required String filename,
    String mimeType = 'image/jpeg',
  }) async {
    final data = await _client.uploadMultipart(
      '/operator/verification/upload',
      fileField: 'file',
      bytes: bytes,
      filename: filename,
      fields: {'doc_type': docType},
      auth: true,
    );
    return UploadedDoc(
      docType: docType,
      storageKey: data['storage_key'] as String,
      sha256Hash: data['sha256_hash'] as String,
      mimeType: data['mime_type'] as String? ?? mimeType,
    );
  }
}
