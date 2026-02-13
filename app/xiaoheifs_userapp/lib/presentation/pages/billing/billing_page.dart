import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/constants/app_colors.dart';
import '../../../core/constants/app_strings.dart';
import '../../../core/constants/input_limits.dart';
import '../../../core/utils/money_formatter.dart';
import '../../providers/wallet_provider.dart';

/// 钱包页面
class BillingPage extends ConsumerWidget {
  const BillingPage({super.key});

  static final RegExp _moneyPattern = RegExp(r'^\d+(\.\d{1,2})?$');

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final walletState = ref.watch(walletProvider);
    return Scaffold(
      body: walletState.loading
          ? const Center(child: CircularProgressIndicator())
          : walletState.error != null
          ? Center(
              child: Padding(
                padding: const EdgeInsets.all(24),
                child: Text(
                  '加载钱包失败：${walletState.error}',
                  style: const TextStyle(color: AppColors.danger),
                ),
              ),
            )
          : RefreshIndicator(
              onRefresh: () => ref.read(walletProvider.notifier).refresh(),
              child: SingleChildScrollView(
                padding: const EdgeInsets.all(24),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    _buildBalanceCard(walletState.wallet),
                    const SizedBox(height: 24),
                    _buildActionButtons(context, ref, walletState.wallet),
                    const SizedBox(height: 24),
                    _buildTransactionList(walletState.transactions),
                  ],
                ),
              ),
            ),
    );
  }

  Widget _buildBalanceCard(Map<String, dynamic>? wallet) {
    final rawWallet = wallet?['wallet'];
    final data = rawWallet is Map ? rawWallet.cast<String, dynamic>() : wallet;
    final balance = double.tryParse('${data?['balance'] ?? 0}') ?? 0;
    final currency = data?['currency'] ?? 'CNY';
    return Card(
      child: Container(
        width: double.infinity,
        padding: const EdgeInsets.all(24),
        decoration: BoxDecoration(
          gradient: LinearGradient(
            colors: [AppColors.primary, AppColors.primaryDark],
          ),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              AppStrings.walletBalance,
              style: TextStyle(
                fontSize: 14,
                color: Colors.white.withOpacity(0.8),
              ),
            ),
            const SizedBox(height: 8),
            Text(
              MoneyFormatter.format(
                balance,
                currency: currency == 'CNY' ? '¥' : currency,
              ),
              style: const TextStyle(
                fontSize: 36,
                fontWeight: FontWeight.bold,
                color: Colors.white,
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildActionButtons(
    BuildContext context,
    WidgetRef ref,
    Map<String, dynamic>? wallet,
  ) {
    return Row(
      children: [
        Expanded(
          child: ElevatedButton.icon(
            onPressed: () => _showRechargeDialog(context, ref),
            icon: const Icon(Icons.add),
            label: const Text(AppStrings.recharge),
            style: ElevatedButton.styleFrom(
              padding: const EdgeInsets.symmetric(vertical: 16),
            ),
          ),
        ),
        const SizedBox(width: 16),
        Expanded(
          child: OutlinedButton.icon(
            onPressed: () => _showWithdrawDialog(context, ref, wallet),
            icon: const Icon(Icons.remove),
            label: const Text(AppStrings.withdraw),
            style: OutlinedButton.styleFrom(
              padding: const EdgeInsets.symmetric(vertical: 16),
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildTransactionList(List<Map<String, dynamic>> transactions) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              AppStrings.transactionHistory,
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 16),
            if (transactions.isEmpty)
              const Center(
                child: Padding(
                  padding: EdgeInsets.all(32),
                  child: Text(AppStrings.noTransactions),
                ),
              )
            else
              ...transactions.map((tx) {
                final type = tx['type'] ?? '';
                final amount = tx['amount'] ?? 0;
                final createdAt = tx['created_at'] ?? '';
                final amountValue = amount is num
                    ? amount.toDouble()
                    : double.tryParse(amount.toString()) ?? 0;
                final typeLabel = _mapTxType(type.toString());
                final amountColor = amountValue < 0
                    ? AppColors.danger
                    : AppColors.success;
                return ListTile(
                  title: Text(typeLabel),
                  subtitle: Text('$createdAt'),
                  trailing: Text(
                    MoneyFormatter.format(amountValue),
                    style: TextStyle(
                      color: amountColor,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                );
              }),
          ],
        ),
      ),
    );
  }

  Future<void> _showRechargeDialog(BuildContext context, WidgetRef ref) async {
    final amountController = TextEditingController();
    final noteController = TextEditingController();

    await showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text(AppStrings.recharge),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: amountController,
              keyboardType: TextInputType.number,
              decoration: const InputDecoration(labelText: '充值金额'),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: noteController,
              maxLength: InputLimits.paymentNote,
              decoration: const InputDecoration(labelText: '备注'),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text(AppStrings.cancel),
          ),
          TextButton(
            onPressed: () async {
              final amountText = amountController.text.trim();
              if (!_moneyPattern.hasMatch(amountText)) {
                ScaffoldMessenger.of(
                  context,
                ).showSnackBar(const SnackBar(content: Text('金额格式不正确，最多保留2位小数')));
                return;
              }
              final amount = double.parse(amountText);
              if (amount <= 0) {
                ScaffoldMessenger.of(
                  context,
                ).showSnackBar(const SnackBar(content: Text('请输入有效金额')));
                return;
              }
              final note = noteController.text.trim();
              if (runeLength(note) > InputLimits.paymentNote) {
                ScaffoldMessenger.of(
                  context,
                ).showSnackBar(const SnackBar(content: Text('备注长度不能超过 500 个字符')));
                return;
              }
              try {
                await ref.read(walletProvider.notifier).recharge({
                  'amount': amount,
                  'note': note,
                });
                if (context.mounted) {
                  Navigator.pop(context);
                  ScaffoldMessenger.of(
                    context,
                  ).showSnackBar(const SnackBar(content: Text('充值提交成功')));
                  await ref.read(walletProvider.notifier).refresh();
                }
              } catch (e) {
                if (context.mounted) {
                  ScaffoldMessenger.of(
                    context,
                  ).showSnackBar(SnackBar(content: Text(e.toString())));
                }
              }
            },
            child: const Text(AppStrings.confirm),
          ),
        ],
      ),
    );
  }

  Future<void> _showWithdrawDialog(
    BuildContext context,
    WidgetRef ref,
    Map<String, dynamic>? wallet,
  ) async {
    final amountController = TextEditingController();
    final noteController = TextEditingController();
    final rawWallet = wallet?['wallet'];
    final data = rawWallet is Map ? rawWallet.cast<String, dynamic>() : wallet;
    final balance = double.tryParse('${data?['balance'] ?? 0}') ?? 0;

    await showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text(AppStrings.withdraw),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text('可用余额：${MoneyFormatter.format(balance)}'),
            const SizedBox(height: 12),
            TextField(
              controller: amountController,
              keyboardType: TextInputType.number,
              decoration: const InputDecoration(labelText: '提现金额'),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: noteController,
              maxLength: InputLimits.paymentNote,
              decoration: const InputDecoration(labelText: '备注'),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text(AppStrings.cancel),
          ),
          TextButton(
            onPressed: () async {
              final amountText = amountController.text.trim();
              if (!_moneyPattern.hasMatch(amountText)) {
                ScaffoldMessenger.of(
                  context,
                ).showSnackBar(const SnackBar(content: Text('金额格式不正确，最多保留2位小数')));
                return;
              }
              final amount = double.parse(amountText);
              if (amount <= 0) {
                ScaffoldMessenger.of(
                  context,
                ).showSnackBar(const SnackBar(content: Text('请输入有效金额')));
                return;
              }
              if (amount > balance) {
                ScaffoldMessenger.of(
                  context,
                ).showSnackBar(const SnackBar(content: Text('提现金额不能大于余额')));
                return;
              }
              final note = noteController.text.trim();
              if (runeLength(note) > InputLimits.paymentNote) {
                ScaffoldMessenger.of(
                  context,
                ).showSnackBar(const SnackBar(content: Text('备注长度不能超过 500 个字符')));
                return;
              }
              try {
                await ref.read(walletProvider.notifier).withdraw({
                  'amount': amount,
                  'note': note,
                  'meta': {'channel': 'manual'},
                });
                if (context.mounted) {
                  Navigator.pop(context);
                  ScaffoldMessenger.of(
                    context,
                  ).showSnackBar(const SnackBar(content: Text('提现提交成功')));
                  await ref.read(walletProvider.notifier).refresh();
                }
              } catch (e) {
                if (context.mounted) {
                  ScaffoldMessenger.of(
                    context,
                  ).showSnackBar(SnackBar(content: Text(e.toString())));
                }
              }
            },
            child: const Text(AppStrings.confirm),
          ),
        ],
      ),
    );
  }

  String _mapTxType(String type) {
    switch (type.toLowerCase()) {
      case 'debit':
        return '支出';
      case 'credit':
        return '收入';
      case 'recharge':
        return '充值';
      case 'withdraw':
        return '提现';
      case 'refund':
        return '退款';
      default:
        return type;
    }
  }
}
