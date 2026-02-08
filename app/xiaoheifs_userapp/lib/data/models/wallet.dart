import 'package:freezed_annotation/freezed_annotation.dart';

part 'wallet.freezed.dart';
part 'wallet.g.dart';

/// 钱包模型
@freezed
class Wallet with _$Wallet {
  const factory Wallet({
    double? balance,
    String? currency,
    @JsonKey(name: 'frozen_balance') double? frozenBalance,
  }) = _Wallet;

  factory Wallet.fromJson(Map<String, dynamic> json) => _$WalletFromJson(json);
}

/// 交易记录模型
@freezed
class Transaction with _$Transaction {
  const factory Transaction({
    int? id,
    String? type,
    @JsonKey(name: 'transaction_type') String? transactionType,
    double? amount,
    String? currency,
    @JsonKey(name: 'balance_before') double? balanceBefore,
    @JsonKey(name: 'balance_after') double? balanceAfter,
    String? status,
    String? description,
    @JsonKey(name: 'created_at') String? createdAt,
  }) = _Transaction;

  factory Transaction.fromJson(Map<String, dynamic> json) =>
      _$TransactionFromJson(json);
}

/// 钱包订单模型（充值/提现）
@freezed
class WalletOrder with _$WalletOrder {
  const factory WalletOrder({
    int? id,
    String? type,
    double? amount,
    String? currency,
    String? status,
    @JsonKey(name: 'payment_method') String? paymentMethod,
    @JsonKey(name: 'payment_url') String? paymentUrl,
    String? description,
    @JsonKey(name: 'created_at') String? createdAt,
    @JsonKey(name: 'updated_at') String? updatedAt,
  }) = _WalletOrder;

  factory WalletOrder.fromJson(Map<String, dynamic> json) =>
      _$WalletOrderFromJson(json);
}

/// 支付方式模型
@freezed
class PaymentProvider with _$PaymentProvider {
  const factory PaymentProvider({
    String? name,
    String? displayName,
    bool? enabled,
  }) = _PaymentProvider;

  factory PaymentProvider.fromJson(Map<String, dynamic> json) =>
      _$PaymentProviderFromJson(json);
}
