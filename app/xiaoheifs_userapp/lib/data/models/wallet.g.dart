// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'wallet.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$WalletImpl _$$WalletImplFromJson(Map<String, dynamic> json) => _$WalletImpl(
      balance: (json['balance'] as num?)?.toDouble(),
      currency: json['currency'] as String?,
      frozenBalance: (json['frozen_balance'] as num?)?.toDouble(),
    );

Map<String, dynamic> _$$WalletImplToJson(_$WalletImpl instance) =>
    <String, dynamic>{
      'balance': instance.balance,
      'currency': instance.currency,
      'frozen_balance': instance.frozenBalance,
    };

_$TransactionImpl _$$TransactionImplFromJson(Map<String, dynamic> json) =>
    _$TransactionImpl(
      id: (json['id'] as num?)?.toInt(),
      type: json['type'] as String?,
      transactionType: json['transaction_type'] as String?,
      amount: (json['amount'] as num?)?.toDouble(),
      currency: json['currency'] as String?,
      balanceBefore: (json['balance_before'] as num?)?.toDouble(),
      balanceAfter: (json['balance_after'] as num?)?.toDouble(),
      status: json['status'] as String?,
      description: json['description'] as String?,
      createdAt: json['created_at'] as String?,
    );

Map<String, dynamic> _$$TransactionImplToJson(_$TransactionImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'type': instance.type,
      'transaction_type': instance.transactionType,
      'amount': instance.amount,
      'currency': instance.currency,
      'balance_before': instance.balanceBefore,
      'balance_after': instance.balanceAfter,
      'status': instance.status,
      'description': instance.description,
      'created_at': instance.createdAt,
    };

_$WalletOrderImpl _$$WalletOrderImplFromJson(Map<String, dynamic> json) =>
    _$WalletOrderImpl(
      id: (json['id'] as num?)?.toInt(),
      type: json['type'] as String?,
      amount: (json['amount'] as num?)?.toDouble(),
      currency: json['currency'] as String?,
      status: json['status'] as String?,
      paymentMethod: json['payment_method'] as String?,
      paymentUrl: json['payment_url'] as String?,
      description: json['description'] as String?,
      createdAt: json['created_at'] as String?,
      updatedAt: json['updated_at'] as String?,
    );

Map<String, dynamic> _$$WalletOrderImplToJson(_$WalletOrderImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'type': instance.type,
      'amount': instance.amount,
      'currency': instance.currency,
      'status': instance.status,
      'payment_method': instance.paymentMethod,
      'payment_url': instance.paymentUrl,
      'description': instance.description,
      'created_at': instance.createdAt,
      'updated_at': instance.updatedAt,
    };

_$PaymentProviderImpl _$$PaymentProviderImplFromJson(
        Map<String, dynamic> json) =>
    _$PaymentProviderImpl(
      name: json['name'] as String?,
      displayName: json['displayName'] as String?,
      enabled: json['enabled'] as bool?,
    );

Map<String, dynamic> _$$PaymentProviderImplToJson(
        _$PaymentProviderImpl instance) =>
    <String, dynamic>{
      'name': instance.name,
      'displayName': instance.displayName,
      'enabled': instance.enabled,
    };
