class OperatorProfile {
  const OperatorProfile({
    required this.id,
    required this.rideType,
    required this.operatorType,
    required this.status,
    required this.walletBalance,
    required this.walletMinBalance,
  });

  final String id;
  final String rideType;
  final String operatorType;
  final String status;
  final int walletBalance;
  final int walletMinBalance;

  factory OperatorProfile.fromJson(Map<String, dynamic> json) {
    return OperatorProfile(
      id: json['id'] as String,
      rideType: json['ride_type'] as String,
      operatorType: json['operator_type'] as String,
      status: json['status'] as String,
      walletBalance: (json['wallet_balance'] as num).toInt(),
      walletMinBalance: (json['wallet_min_balance'] as num).toInt(),
    );
  }

  String get rideTypeLabel {
    switch (rideType) {
      case 'boda':
        return 'Rozy Boda';
      case 'car_basic':
        return 'Rozy Car Basic';
      case 'car_xl':
        return 'Rozy Car XL';
      default:
        return rideType;
    }
  }
}
