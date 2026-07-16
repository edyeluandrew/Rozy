import 'dart:convert';

import 'package:http/http.dart' as http;

import '../config/app_config.dart';

class ApiException implements Exception {
  ApiException(this.message, {this.statusCode});

  final String message;
  final int? statusCode;

  @override
  String toString() => message;
}

class ApiClient {
  ApiClient({http.Client? client, String? baseUrl})
      : _client = client ?? http.Client(),
        _baseUrl = baseUrl ?? AppConfig.apiBaseUrl;

  final http.Client _client;
  final String _baseUrl;

  String? _token;

  void setToken(String? token) => _token = token;

  Future<Map<String, dynamic>> post(
    String path, {
    Map<String, dynamic>? body,
    bool auth = false,
  }) async {
    final response = await _client.post(
      Uri.parse('$_baseUrl$path'),
      headers: _headers(auth),
      body: body == null ? null : jsonEncode(body),
    );
    return _decode(response);
  }

  Future<Map<String, dynamic>> get(String path, {bool auth = false}) async {
    final response = await _client.get(
      Uri.parse('$_baseUrl$path'),
      headers: _headers(auth),
    );
    return _decode(response);
  }

  Future<Map<String, dynamic>> uploadMultipart(
    String path, {
    required String fileField,
    required List<int> bytes,
    required String filename,
    Map<String, String>? fields,
    bool auth = false,
  }) async {
    final request = http.MultipartRequest('POST', Uri.parse('$_baseUrl$path'));
    if (auth && _token != null) {
      request.headers['Authorization'] = 'Bearer $_token';
    }
    if (fields != null) {
      request.fields.addAll(fields);
    }
    request.files.add(http.MultipartFile.fromBytes(
      fileField,
      bytes,
      filename: filename,
    ));
    final streamed = await _client.send(request);
    final response = await http.Response.fromStream(streamed);
    return _decode(response);
  }

  Map<String, String> _headers(bool auth) {
    final headers = {'Content-Type': 'application/json'};
    if (auth && _token != null) {
      headers['Authorization'] = 'Bearer $_token';
    }
    return headers;
  }

  Map<String, dynamic> _decode(http.Response response) {
    Map<String, dynamic> data = {};
    if (response.body.isNotEmpty) {
      final decoded = jsonDecode(response.body);
      if (decoded is Map<String, dynamic>) data = decoded;
    }

    if (response.statusCode >= 400) {
      throw ApiException(
        data['error']?.toString() ?? 'Request failed',
        statusCode: response.statusCode,
      );
    }
    return data;
  }
}
