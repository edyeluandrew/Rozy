class AuthUser {
  const AuthUser({
    required this.id,
    required this.phone,
    required this.role,
  });

  final String id;
  final String phone;
  final String role;

  factory AuthUser.fromJson(Map<String, dynamic> json) {
    return AuthUser(
      id: json['id'] as String,
      phone: json['phone'] as String,
      role: json['role'] as String,
    );
  }

  bool get isDriver => role == 'driver';
  bool get isPassenger => role == 'passenger';
}
