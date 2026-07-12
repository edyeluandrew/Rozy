import 'api_client.dart';

class WalletView {
  const WalletView({
    required this.balance,
    required this.minBalance,
  });

  final int balance;
  final int minBalance;

  factory WalletView.fromJson(Map<String, dynamic> json) => WalletView(
        balance: (json['balance'] as num).toInt(),
        minBalance: (json['min_balance'] as num).toInt(),
      );
}

class WalletTransaction {
  const WalletTransaction({
    required this.id,
    required this.type,
    required this.amount,
    required this.balanceAfter,
    required this.reference,
    required this.createdAt,
  });

  final String id;
  final String type;
  final int amount;
  final int balanceAfter;
  final String reference;
  final String createdAt;

  factory WalletTransaction.fromJson(Map<String, dynamic> json) => WalletTransaction(
        id: json['id'] as String,
        type: json['type'] as String,
        amount: (json['amount'] as num).toInt(),
        balanceAfter: (json['balance_after'] as num).toInt(),
        reference: json['reference'] as String? ?? '',
        createdAt: json['created_at'] as String? ?? '',
      );
}

class WalletRecharge {
  const WalletRecharge({
    required this.id,
    required this.amount,
    required this.provider,
    required this.status,
    required this.idempotencyKey,
  });

  final String id;
  final int amount;
  final String provider;
  final String status;
  final String idempotencyKey;

  factory WalletRecharge.fromJson(Map<String, dynamic> json) => WalletRecharge(
        id: json['id'] as String,
        amount: (json['amount'] as num).toInt(),
        provider: json['provider'] as String,
        status: json['status'] as String,
        idempotencyKey: json['idempotency_key'] as String,
      );
}

class WalletApi {
  WalletApi(this._client);

  final ApiClient _client;

  Future<({WalletView wallet, List<WalletTransaction> transactions})> getWallet() async {
    final data = await _client.get('/operator/wallet', auth: true);
    final wallet = WalletView.fromJson(data['wallet'] as Map<String, dynamic>);
    final raw = data['transactions'] as List<dynamic>? ?? [];
    final txs = raw
        .map((e) => WalletTransaction.fromJson(e as Map<String, dynamic>))
        .toList();
    return (wallet: wallet, transactions: txs);
  }

  Future<({WalletRecharge recharge, String instructions})> initiateRecharge({
    required int amount,
    required String provider,
  }) async {
    final data = await _client.post(
      '/operator/wallet/recharge',
      auth: true,
      body: {'amount': amount, 'provider': provider},
    );
    return (
      recharge: WalletRecharge.fromJson(data['recharge'] as Map<String, dynamic>),
      instructions: data['instructions'] as String? ?? '',
    );
  }
}
