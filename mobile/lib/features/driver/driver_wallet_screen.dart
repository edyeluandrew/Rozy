import 'package:flutter/material.dart';

import '../../core/api/wallet_api.dart';
import '../../core/models/operator_profile.dart';
import '../../core/services/app_services.dart';
import '../../core/theme/rozy_colors.dart';
import '../../core/theme/rozy_theme.dart';

class DriverWalletScreen extends StatefulWidget {
  const DriverWalletScreen({super.key, required this.profile});

  final OperatorProfile profile;

  @override
  State<DriverWalletScreen> createState() => _DriverWalletScreenState();
}

class _DriverWalletScreenState extends State<DriverWalletScreen> {
  late OperatorProfile _profile;
  List<WalletTransaction> _transactions = [];
  bool _loading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _profile = widget.profile;
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final data = await AppServices.live.wallet.getWallet();
      if (!mounted) return;
      setState(() {
        _profile = _profile.copyWith(
          walletBalance: data.wallet.balance,
          walletMinBalance: data.wallet.minBalance,
        );
        _transactions = data.transactions;
      });
    } catch (e) {
      if (mounted) setState(() => _error = e.toString());
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _showRechargeSheet() async {
    final amountController = TextEditingController(text: '10000');
    var provider = 'mtn';
    String? sheetError;

    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      builder: (ctx) {
        return StatefulBuilder(
          builder: (context, setSheetState) {
            return Padding(
              padding: EdgeInsets.only(
                left: 24,
                right: 24,
                top: 24,
                bottom: MediaQuery.of(ctx).viewInsets.bottom + 24,
              ),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text('Top up wallet', style: Theme.of(context).textTheme.titleLarge),
                  const SizedBox(height: 16),
                  TextField(
                    controller: amountController,
                    keyboardType: TextInputType.number,
                    decoration: const InputDecoration(
                      labelText: 'Amount (UGX)',
                      hintText: '10000',
                    ),
                  ),
                  const SizedBox(height: 12),
                  SegmentedButton<String>(
                    segments: const [
                      ButtonSegment(value: 'mtn', label: Text('MTN MoMo')),
                      ButtonSegment(value: 'airtel', label: Text('Airtel')),
                    ],
                    selected: {provider},
                    onSelectionChanged: (v) => setSheetState(() => provider = v.first),
                  ),
                  if (sheetError != null) ...[
                    const SizedBox(height: 12),
                    Text(sheetError!, style: const TextStyle(color: Colors.red)),
                  ],
                  const SizedBox(height: 16),
                  ElevatedButton(
                    onPressed: () async {
                      final amount = int.tryParse(amountController.text.trim()) ?? 0;
                      if (amount < 1000) {
                        setSheetState(() => sheetError = 'Minimum top-up is UGX 1,000');
                        return;
                      }
                      try {
                        final result = await AppServices.live.wallet.initiateRecharge(
                          amount: amount,
                          provider: provider,
                        );
                        if (!context.mounted) return;
                        Navigator.of(context).pop();
                        ScaffoldMessenger.of(this.context).showSnackBar(
                          SnackBar(content: Text(result.instructions)),
                        );
                        await _load();
                      } catch (e) {
                        setSheetState(() => sheetError = e.toString());
                      }
                    },
                    child: const Text('Request payment'),
                  ),
                ],
              ),
            );
          },
        );
      },
    );
  }

  String _txLabel(WalletTransaction tx) {
    switch (tx.type) {
      case 'recharge':
        return 'Wallet top-up';
      case 'trip_fee':
        return 'Rozy trip fee';
      case 'admin_credit':
        return 'Admin credit';
      default:
        return tx.type;
    }
  }

  @override
  Widget build(BuildContext context) {
    final low = _profile.walletBalance < _profile.walletMinBalance;

    return Scaffold(
      appBar: AppBar(
        title: const Text('Wallet'),
        actions: [
          IconButton(icon: const Icon(Icons.refresh), onPressed: _loading ? null : _load),
        ],
      ),
      body: _loading && _transactions.isEmpty
          ? const Center(child: CircularProgressIndicator())
          : RefreshIndicator(
              onRefresh: _load,
              child: ListView(
                padding: const EdgeInsets.all(20),
                children: [
                  Container(
                    padding: const EdgeInsets.all(24),
                    decoration: RozyTheme.premiumCardDecoration,
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          'Available balance',
                          style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                                color: RozyColors.grey,
                              ),
                        ),
                        const SizedBox(height: 8),
                        Text(
                          'UGX ${_profile.walletBalance}',
                          style: Theme.of(context).textTheme.displaySmall?.copyWith(
                                color: RozyColors.gold,
                                fontWeight: FontWeight.bold,
                              ),
                        ),
                        const SizedBox(height: 8),
                        Text(
                          'Minimum UGX ${_profile.walletMinBalance} to go online',
                          style: Theme.of(context).textTheme.bodySmall?.copyWith(
                                color: low ? RozyColors.gold : RozyColors.grey,
                              ),
                        ),
                        const SizedBox(height: 16),
                        SizedBox(
                          width: double.infinity,
                          child: ElevatedButton(
                            onPressed: _showRechargeSheet,
                            child: const Text('Top up via MTN / Airtel'),
                          ),
                        ),
                      ],
                    ),
                  ),
                  if (_error != null) ...[
                    const SizedBox(height: 12),
                    Text(_error!, style: const TextStyle(color: Colors.red)),
                  ],
                  const SizedBox(height: 24),
                  Text(
                    'Recent transactions',
                    style: Theme.of(context).textTheme.titleMedium?.copyWith(
                          fontWeight: FontWeight.bold,
                        ),
                  ),
                  const SizedBox(height: 12),
                  if (_transactions.isEmpty)
                    const Card(
                      child: Padding(
                        padding: EdgeInsets.all(20),
                        child: Text('No transactions yet.'),
                      ),
                    )
                  else
                    ..._transactions.map(
                      (tx) => Card(
                        child: ListTile(
                          title: Text(_txLabel(tx)),
                          subtitle: Text(tx.createdAt.isNotEmpty ? tx.createdAt : tx.reference),
                          trailing: Text(
                            '${tx.amount >= 0 ? '+' : ''}${tx.amount}',
                            style: TextStyle(
                              fontWeight: FontWeight.bold,
                              color: tx.amount >= 0 ? Colors.green.shade700 : RozyColors.charcoal,
                            ),
                          ),
                        ),
                      ),
                    ),
                ],
              ),
            ),
    );
  }
}

extension on OperatorProfile {
  OperatorProfile copyWith({
    String? id,
    String? rideType,
    String? operatorType,
    String? status,
    int? walletBalance,
    int? walletMinBalance,
  }) {
    return OperatorProfile(
      id: id ?? this.id,
      rideType: rideType ?? this.rideType,
      operatorType: operatorType ?? this.operatorType,
      status: status ?? this.status,
      walletBalance: walletBalance ?? this.walletBalance,
      walletMinBalance: walletMinBalance ?? this.walletMinBalance,
    );
  }
}
