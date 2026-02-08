// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'wallet.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

Wallet _$WalletFromJson(Map<String, dynamic> json) {
  return _Wallet.fromJson(json);
}

/// @nodoc
mixin _$Wallet {
  double? get balance => throw _privateConstructorUsedError;
  String? get currency => throw _privateConstructorUsedError;
  @JsonKey(name: 'frozen_balance')
  double? get frozenBalance => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $WalletCopyWith<Wallet> get copyWith => throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $WalletCopyWith<$Res> {
  factory $WalletCopyWith(Wallet value, $Res Function(Wallet) then) =
      _$WalletCopyWithImpl<$Res, Wallet>;
  @useResult
  $Res call(
      {double? balance,
      String? currency,
      @JsonKey(name: 'frozen_balance') double? frozenBalance});
}

/// @nodoc
class _$WalletCopyWithImpl<$Res, $Val extends Wallet>
    implements $WalletCopyWith<$Res> {
  _$WalletCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? balance = freezed,
    Object? currency = freezed,
    Object? frozenBalance = freezed,
  }) {
    return _then(_value.copyWith(
      balance: freezed == balance
          ? _value.balance
          : balance // ignore: cast_nullable_to_non_nullable
              as double?,
      currency: freezed == currency
          ? _value.currency
          : currency // ignore: cast_nullable_to_non_nullable
              as String?,
      frozenBalance: freezed == frozenBalance
          ? _value.frozenBalance
          : frozenBalance // ignore: cast_nullable_to_non_nullable
              as double?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$WalletImplCopyWith<$Res> implements $WalletCopyWith<$Res> {
  factory _$$WalletImplCopyWith(
          _$WalletImpl value, $Res Function(_$WalletImpl) then) =
      __$$WalletImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {double? balance,
      String? currency,
      @JsonKey(name: 'frozen_balance') double? frozenBalance});
}

/// @nodoc
class __$$WalletImplCopyWithImpl<$Res>
    extends _$WalletCopyWithImpl<$Res, _$WalletImpl>
    implements _$$WalletImplCopyWith<$Res> {
  __$$WalletImplCopyWithImpl(
      _$WalletImpl _value, $Res Function(_$WalletImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? balance = freezed,
    Object? currency = freezed,
    Object? frozenBalance = freezed,
  }) {
    return _then(_$WalletImpl(
      balance: freezed == balance
          ? _value.balance
          : balance // ignore: cast_nullable_to_non_nullable
              as double?,
      currency: freezed == currency
          ? _value.currency
          : currency // ignore: cast_nullable_to_non_nullable
              as String?,
      frozenBalance: freezed == frozenBalance
          ? _value.frozenBalance
          : frozenBalance // ignore: cast_nullable_to_non_nullable
              as double?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$WalletImpl implements _Wallet {
  const _$WalletImpl(
      {this.balance,
      this.currency,
      @JsonKey(name: 'frozen_balance') this.frozenBalance});

  factory _$WalletImpl.fromJson(Map<String, dynamic> json) =>
      _$$WalletImplFromJson(json);

  @override
  final double? balance;
  @override
  final String? currency;
  @override
  @JsonKey(name: 'frozen_balance')
  final double? frozenBalance;

  @override
  String toString() {
    return 'Wallet(balance: $balance, currency: $currency, frozenBalance: $frozenBalance)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$WalletImpl &&
            (identical(other.balance, balance) || other.balance == balance) &&
            (identical(other.currency, currency) ||
                other.currency == currency) &&
            (identical(other.frozenBalance, frozenBalance) ||
                other.frozenBalance == frozenBalance));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode =>
      Object.hash(runtimeType, balance, currency, frozenBalance);

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$WalletImplCopyWith<_$WalletImpl> get copyWith =>
      __$$WalletImplCopyWithImpl<_$WalletImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$WalletImplToJson(
      this,
    );
  }
}

abstract class _Wallet implements Wallet {
  const factory _Wallet(
          {final double? balance,
          final String? currency,
          @JsonKey(name: 'frozen_balance') final double? frozenBalance}) =
      _$WalletImpl;

  factory _Wallet.fromJson(Map<String, dynamic> json) = _$WalletImpl.fromJson;

  @override
  double? get balance;
  @override
  String? get currency;
  @override
  @JsonKey(name: 'frozen_balance')
  double? get frozenBalance;
  @override
  @JsonKey(ignore: true)
  _$$WalletImplCopyWith<_$WalletImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

Transaction _$TransactionFromJson(Map<String, dynamic> json) {
  return _Transaction.fromJson(json);
}

/// @nodoc
mixin _$Transaction {
  int? get id => throw _privateConstructorUsedError;
  String? get type => throw _privateConstructorUsedError;
  @JsonKey(name: 'transaction_type')
  String? get transactionType => throw _privateConstructorUsedError;
  double? get amount => throw _privateConstructorUsedError;
  String? get currency => throw _privateConstructorUsedError;
  @JsonKey(name: 'balance_before')
  double? get balanceBefore => throw _privateConstructorUsedError;
  @JsonKey(name: 'balance_after')
  double? get balanceAfter => throw _privateConstructorUsedError;
  String? get status => throw _privateConstructorUsedError;
  String? get description => throw _privateConstructorUsedError;
  @JsonKey(name: 'created_at')
  String? get createdAt => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $TransactionCopyWith<Transaction> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $TransactionCopyWith<$Res> {
  factory $TransactionCopyWith(
          Transaction value, $Res Function(Transaction) then) =
      _$TransactionCopyWithImpl<$Res, Transaction>;
  @useResult
  $Res call(
      {int? id,
      String? type,
      @JsonKey(name: 'transaction_type') String? transactionType,
      double? amount,
      String? currency,
      @JsonKey(name: 'balance_before') double? balanceBefore,
      @JsonKey(name: 'balance_after') double? balanceAfter,
      String? status,
      String? description,
      @JsonKey(name: 'created_at') String? createdAt});
}

/// @nodoc
class _$TransactionCopyWithImpl<$Res, $Val extends Transaction>
    implements $TransactionCopyWith<$Res> {
  _$TransactionCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? type = freezed,
    Object? transactionType = freezed,
    Object? amount = freezed,
    Object? currency = freezed,
    Object? balanceBefore = freezed,
    Object? balanceAfter = freezed,
    Object? status = freezed,
    Object? description = freezed,
    Object? createdAt = freezed,
  }) {
    return _then(_value.copyWith(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      type: freezed == type
          ? _value.type
          : type // ignore: cast_nullable_to_non_nullable
              as String?,
      transactionType: freezed == transactionType
          ? _value.transactionType
          : transactionType // ignore: cast_nullable_to_non_nullable
              as String?,
      amount: freezed == amount
          ? _value.amount
          : amount // ignore: cast_nullable_to_non_nullable
              as double?,
      currency: freezed == currency
          ? _value.currency
          : currency // ignore: cast_nullable_to_non_nullable
              as String?,
      balanceBefore: freezed == balanceBefore
          ? _value.balanceBefore
          : balanceBefore // ignore: cast_nullable_to_non_nullable
              as double?,
      balanceAfter: freezed == balanceAfter
          ? _value.balanceAfter
          : balanceAfter // ignore: cast_nullable_to_non_nullable
              as double?,
      status: freezed == status
          ? _value.status
          : status // ignore: cast_nullable_to_non_nullable
              as String?,
      description: freezed == description
          ? _value.description
          : description // ignore: cast_nullable_to_non_nullable
              as String?,
      createdAt: freezed == createdAt
          ? _value.createdAt
          : createdAt // ignore: cast_nullable_to_non_nullable
              as String?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$TransactionImplCopyWith<$Res>
    implements $TransactionCopyWith<$Res> {
  factory _$$TransactionImplCopyWith(
          _$TransactionImpl value, $Res Function(_$TransactionImpl) then) =
      __$$TransactionImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {int? id,
      String? type,
      @JsonKey(name: 'transaction_type') String? transactionType,
      double? amount,
      String? currency,
      @JsonKey(name: 'balance_before') double? balanceBefore,
      @JsonKey(name: 'balance_after') double? balanceAfter,
      String? status,
      String? description,
      @JsonKey(name: 'created_at') String? createdAt});
}

/// @nodoc
class __$$TransactionImplCopyWithImpl<$Res>
    extends _$TransactionCopyWithImpl<$Res, _$TransactionImpl>
    implements _$$TransactionImplCopyWith<$Res> {
  __$$TransactionImplCopyWithImpl(
      _$TransactionImpl _value, $Res Function(_$TransactionImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? type = freezed,
    Object? transactionType = freezed,
    Object? amount = freezed,
    Object? currency = freezed,
    Object? balanceBefore = freezed,
    Object? balanceAfter = freezed,
    Object? status = freezed,
    Object? description = freezed,
    Object? createdAt = freezed,
  }) {
    return _then(_$TransactionImpl(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      type: freezed == type
          ? _value.type
          : type // ignore: cast_nullable_to_non_nullable
              as String?,
      transactionType: freezed == transactionType
          ? _value.transactionType
          : transactionType // ignore: cast_nullable_to_non_nullable
              as String?,
      amount: freezed == amount
          ? _value.amount
          : amount // ignore: cast_nullable_to_non_nullable
              as double?,
      currency: freezed == currency
          ? _value.currency
          : currency // ignore: cast_nullable_to_non_nullable
              as String?,
      balanceBefore: freezed == balanceBefore
          ? _value.balanceBefore
          : balanceBefore // ignore: cast_nullable_to_non_nullable
              as double?,
      balanceAfter: freezed == balanceAfter
          ? _value.balanceAfter
          : balanceAfter // ignore: cast_nullable_to_non_nullable
              as double?,
      status: freezed == status
          ? _value.status
          : status // ignore: cast_nullable_to_non_nullable
              as String?,
      description: freezed == description
          ? _value.description
          : description // ignore: cast_nullable_to_non_nullable
              as String?,
      createdAt: freezed == createdAt
          ? _value.createdAt
          : createdAt // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$TransactionImpl implements _Transaction {
  const _$TransactionImpl(
      {this.id,
      this.type,
      @JsonKey(name: 'transaction_type') this.transactionType,
      this.amount,
      this.currency,
      @JsonKey(name: 'balance_before') this.balanceBefore,
      @JsonKey(name: 'balance_after') this.balanceAfter,
      this.status,
      this.description,
      @JsonKey(name: 'created_at') this.createdAt});

  factory _$TransactionImpl.fromJson(Map<String, dynamic> json) =>
      _$$TransactionImplFromJson(json);

  @override
  final int? id;
  @override
  final String? type;
  @override
  @JsonKey(name: 'transaction_type')
  final String? transactionType;
  @override
  final double? amount;
  @override
  final String? currency;
  @override
  @JsonKey(name: 'balance_before')
  final double? balanceBefore;
  @override
  @JsonKey(name: 'balance_after')
  final double? balanceAfter;
  @override
  final String? status;
  @override
  final String? description;
  @override
  @JsonKey(name: 'created_at')
  final String? createdAt;

  @override
  String toString() {
    return 'Transaction(id: $id, type: $type, transactionType: $transactionType, amount: $amount, currency: $currency, balanceBefore: $balanceBefore, balanceAfter: $balanceAfter, status: $status, description: $description, createdAt: $createdAt)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$TransactionImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.type, type) || other.type == type) &&
            (identical(other.transactionType, transactionType) ||
                other.transactionType == transactionType) &&
            (identical(other.amount, amount) || other.amount == amount) &&
            (identical(other.currency, currency) ||
                other.currency == currency) &&
            (identical(other.balanceBefore, balanceBefore) ||
                other.balanceBefore == balanceBefore) &&
            (identical(other.balanceAfter, balanceAfter) ||
                other.balanceAfter == balanceAfter) &&
            (identical(other.status, status) || other.status == status) &&
            (identical(other.description, description) ||
                other.description == description) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode => Object.hash(
      runtimeType,
      id,
      type,
      transactionType,
      amount,
      currency,
      balanceBefore,
      balanceAfter,
      status,
      description,
      createdAt);

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$TransactionImplCopyWith<_$TransactionImpl> get copyWith =>
      __$$TransactionImplCopyWithImpl<_$TransactionImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$TransactionImplToJson(
      this,
    );
  }
}

abstract class _Transaction implements Transaction {
  const factory _Transaction(
          {final int? id,
          final String? type,
          @JsonKey(name: 'transaction_type') final String? transactionType,
          final double? amount,
          final String? currency,
          @JsonKey(name: 'balance_before') final double? balanceBefore,
          @JsonKey(name: 'balance_after') final double? balanceAfter,
          final String? status,
          final String? description,
          @JsonKey(name: 'created_at') final String? createdAt}) =
      _$TransactionImpl;

  factory _Transaction.fromJson(Map<String, dynamic> json) =
      _$TransactionImpl.fromJson;

  @override
  int? get id;
  @override
  String? get type;
  @override
  @JsonKey(name: 'transaction_type')
  String? get transactionType;
  @override
  double? get amount;
  @override
  String? get currency;
  @override
  @JsonKey(name: 'balance_before')
  double? get balanceBefore;
  @override
  @JsonKey(name: 'balance_after')
  double? get balanceAfter;
  @override
  String? get status;
  @override
  String? get description;
  @override
  @JsonKey(name: 'created_at')
  String? get createdAt;
  @override
  @JsonKey(ignore: true)
  _$$TransactionImplCopyWith<_$TransactionImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

WalletOrder _$WalletOrderFromJson(Map<String, dynamic> json) {
  return _WalletOrder.fromJson(json);
}

/// @nodoc
mixin _$WalletOrder {
  int? get id => throw _privateConstructorUsedError;
  String? get type => throw _privateConstructorUsedError;
  double? get amount => throw _privateConstructorUsedError;
  String? get currency => throw _privateConstructorUsedError;
  String? get status => throw _privateConstructorUsedError;
  @JsonKey(name: 'payment_method')
  String? get paymentMethod => throw _privateConstructorUsedError;
  @JsonKey(name: 'payment_url')
  String? get paymentUrl => throw _privateConstructorUsedError;
  String? get description => throw _privateConstructorUsedError;
  @JsonKey(name: 'created_at')
  String? get createdAt => throw _privateConstructorUsedError;
  @JsonKey(name: 'updated_at')
  String? get updatedAt => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $WalletOrderCopyWith<WalletOrder> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $WalletOrderCopyWith<$Res> {
  factory $WalletOrderCopyWith(
          WalletOrder value, $Res Function(WalletOrder) then) =
      _$WalletOrderCopyWithImpl<$Res, WalletOrder>;
  @useResult
  $Res call(
      {int? id,
      String? type,
      double? amount,
      String? currency,
      String? status,
      @JsonKey(name: 'payment_method') String? paymentMethod,
      @JsonKey(name: 'payment_url') String? paymentUrl,
      String? description,
      @JsonKey(name: 'created_at') String? createdAt,
      @JsonKey(name: 'updated_at') String? updatedAt});
}

/// @nodoc
class _$WalletOrderCopyWithImpl<$Res, $Val extends WalletOrder>
    implements $WalletOrderCopyWith<$Res> {
  _$WalletOrderCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? type = freezed,
    Object? amount = freezed,
    Object? currency = freezed,
    Object? status = freezed,
    Object? paymentMethod = freezed,
    Object? paymentUrl = freezed,
    Object? description = freezed,
    Object? createdAt = freezed,
    Object? updatedAt = freezed,
  }) {
    return _then(_value.copyWith(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      type: freezed == type
          ? _value.type
          : type // ignore: cast_nullable_to_non_nullable
              as String?,
      amount: freezed == amount
          ? _value.amount
          : amount // ignore: cast_nullable_to_non_nullable
              as double?,
      currency: freezed == currency
          ? _value.currency
          : currency // ignore: cast_nullable_to_non_nullable
              as String?,
      status: freezed == status
          ? _value.status
          : status // ignore: cast_nullable_to_non_nullable
              as String?,
      paymentMethod: freezed == paymentMethod
          ? _value.paymentMethod
          : paymentMethod // ignore: cast_nullable_to_non_nullable
              as String?,
      paymentUrl: freezed == paymentUrl
          ? _value.paymentUrl
          : paymentUrl // ignore: cast_nullable_to_non_nullable
              as String?,
      description: freezed == description
          ? _value.description
          : description // ignore: cast_nullable_to_non_nullable
              as String?,
      createdAt: freezed == createdAt
          ? _value.createdAt
          : createdAt // ignore: cast_nullable_to_non_nullable
              as String?,
      updatedAt: freezed == updatedAt
          ? _value.updatedAt
          : updatedAt // ignore: cast_nullable_to_non_nullable
              as String?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$WalletOrderImplCopyWith<$Res>
    implements $WalletOrderCopyWith<$Res> {
  factory _$$WalletOrderImplCopyWith(
          _$WalletOrderImpl value, $Res Function(_$WalletOrderImpl) then) =
      __$$WalletOrderImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {int? id,
      String? type,
      double? amount,
      String? currency,
      String? status,
      @JsonKey(name: 'payment_method') String? paymentMethod,
      @JsonKey(name: 'payment_url') String? paymentUrl,
      String? description,
      @JsonKey(name: 'created_at') String? createdAt,
      @JsonKey(name: 'updated_at') String? updatedAt});
}

/// @nodoc
class __$$WalletOrderImplCopyWithImpl<$Res>
    extends _$WalletOrderCopyWithImpl<$Res, _$WalletOrderImpl>
    implements _$$WalletOrderImplCopyWith<$Res> {
  __$$WalletOrderImplCopyWithImpl(
      _$WalletOrderImpl _value, $Res Function(_$WalletOrderImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? type = freezed,
    Object? amount = freezed,
    Object? currency = freezed,
    Object? status = freezed,
    Object? paymentMethod = freezed,
    Object? paymentUrl = freezed,
    Object? description = freezed,
    Object? createdAt = freezed,
    Object? updatedAt = freezed,
  }) {
    return _then(_$WalletOrderImpl(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      type: freezed == type
          ? _value.type
          : type // ignore: cast_nullable_to_non_nullable
              as String?,
      amount: freezed == amount
          ? _value.amount
          : amount // ignore: cast_nullable_to_non_nullable
              as double?,
      currency: freezed == currency
          ? _value.currency
          : currency // ignore: cast_nullable_to_non_nullable
              as String?,
      status: freezed == status
          ? _value.status
          : status // ignore: cast_nullable_to_non_nullable
              as String?,
      paymentMethod: freezed == paymentMethod
          ? _value.paymentMethod
          : paymentMethod // ignore: cast_nullable_to_non_nullable
              as String?,
      paymentUrl: freezed == paymentUrl
          ? _value.paymentUrl
          : paymentUrl // ignore: cast_nullable_to_non_nullable
              as String?,
      description: freezed == description
          ? _value.description
          : description // ignore: cast_nullable_to_non_nullable
              as String?,
      createdAt: freezed == createdAt
          ? _value.createdAt
          : createdAt // ignore: cast_nullable_to_non_nullable
              as String?,
      updatedAt: freezed == updatedAt
          ? _value.updatedAt
          : updatedAt // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$WalletOrderImpl implements _WalletOrder {
  const _$WalletOrderImpl(
      {this.id,
      this.type,
      this.amount,
      this.currency,
      this.status,
      @JsonKey(name: 'payment_method') this.paymentMethod,
      @JsonKey(name: 'payment_url') this.paymentUrl,
      this.description,
      @JsonKey(name: 'created_at') this.createdAt,
      @JsonKey(name: 'updated_at') this.updatedAt});

  factory _$WalletOrderImpl.fromJson(Map<String, dynamic> json) =>
      _$$WalletOrderImplFromJson(json);

  @override
  final int? id;
  @override
  final String? type;
  @override
  final double? amount;
  @override
  final String? currency;
  @override
  final String? status;
  @override
  @JsonKey(name: 'payment_method')
  final String? paymentMethod;
  @override
  @JsonKey(name: 'payment_url')
  final String? paymentUrl;
  @override
  final String? description;
  @override
  @JsonKey(name: 'created_at')
  final String? createdAt;
  @override
  @JsonKey(name: 'updated_at')
  final String? updatedAt;

  @override
  String toString() {
    return 'WalletOrder(id: $id, type: $type, amount: $amount, currency: $currency, status: $status, paymentMethod: $paymentMethod, paymentUrl: $paymentUrl, description: $description, createdAt: $createdAt, updatedAt: $updatedAt)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$WalletOrderImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.type, type) || other.type == type) &&
            (identical(other.amount, amount) || other.amount == amount) &&
            (identical(other.currency, currency) ||
                other.currency == currency) &&
            (identical(other.status, status) || other.status == status) &&
            (identical(other.paymentMethod, paymentMethod) ||
                other.paymentMethod == paymentMethod) &&
            (identical(other.paymentUrl, paymentUrl) ||
                other.paymentUrl == paymentUrl) &&
            (identical(other.description, description) ||
                other.description == description) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt) &&
            (identical(other.updatedAt, updatedAt) ||
                other.updatedAt == updatedAt));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode => Object.hash(runtimeType, id, type, amount, currency,
      status, paymentMethod, paymentUrl, description, createdAt, updatedAt);

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$WalletOrderImplCopyWith<_$WalletOrderImpl> get copyWith =>
      __$$WalletOrderImplCopyWithImpl<_$WalletOrderImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$WalletOrderImplToJson(
      this,
    );
  }
}

abstract class _WalletOrder implements WalletOrder {
  const factory _WalletOrder(
          {final int? id,
          final String? type,
          final double? amount,
          final String? currency,
          final String? status,
          @JsonKey(name: 'payment_method') final String? paymentMethod,
          @JsonKey(name: 'payment_url') final String? paymentUrl,
          final String? description,
          @JsonKey(name: 'created_at') final String? createdAt,
          @JsonKey(name: 'updated_at') final String? updatedAt}) =
      _$WalletOrderImpl;

  factory _WalletOrder.fromJson(Map<String, dynamic> json) =
      _$WalletOrderImpl.fromJson;

  @override
  int? get id;
  @override
  String? get type;
  @override
  double? get amount;
  @override
  String? get currency;
  @override
  String? get status;
  @override
  @JsonKey(name: 'payment_method')
  String? get paymentMethod;
  @override
  @JsonKey(name: 'payment_url')
  String? get paymentUrl;
  @override
  String? get description;
  @override
  @JsonKey(name: 'created_at')
  String? get createdAt;
  @override
  @JsonKey(name: 'updated_at')
  String? get updatedAt;
  @override
  @JsonKey(ignore: true)
  _$$WalletOrderImplCopyWith<_$WalletOrderImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

PaymentProvider _$PaymentProviderFromJson(Map<String, dynamic> json) {
  return _PaymentProvider.fromJson(json);
}

/// @nodoc
mixin _$PaymentProvider {
  String? get name => throw _privateConstructorUsedError;
  String? get displayName => throw _privateConstructorUsedError;
  bool? get enabled => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $PaymentProviderCopyWith<PaymentProvider> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $PaymentProviderCopyWith<$Res> {
  factory $PaymentProviderCopyWith(
          PaymentProvider value, $Res Function(PaymentProvider) then) =
      _$PaymentProviderCopyWithImpl<$Res, PaymentProvider>;
  @useResult
  $Res call({String? name, String? displayName, bool? enabled});
}

/// @nodoc
class _$PaymentProviderCopyWithImpl<$Res, $Val extends PaymentProvider>
    implements $PaymentProviderCopyWith<$Res> {
  _$PaymentProviderCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? name = freezed,
    Object? displayName = freezed,
    Object? enabled = freezed,
  }) {
    return _then(_value.copyWith(
      name: freezed == name
          ? _value.name
          : name // ignore: cast_nullable_to_non_nullable
              as String?,
      displayName: freezed == displayName
          ? _value.displayName
          : displayName // ignore: cast_nullable_to_non_nullable
              as String?,
      enabled: freezed == enabled
          ? _value.enabled
          : enabled // ignore: cast_nullable_to_non_nullable
              as bool?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$PaymentProviderImplCopyWith<$Res>
    implements $PaymentProviderCopyWith<$Res> {
  factory _$$PaymentProviderImplCopyWith(_$PaymentProviderImpl value,
          $Res Function(_$PaymentProviderImpl) then) =
      __$$PaymentProviderImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String? name, String? displayName, bool? enabled});
}

/// @nodoc
class __$$PaymentProviderImplCopyWithImpl<$Res>
    extends _$PaymentProviderCopyWithImpl<$Res, _$PaymentProviderImpl>
    implements _$$PaymentProviderImplCopyWith<$Res> {
  __$$PaymentProviderImplCopyWithImpl(
      _$PaymentProviderImpl _value, $Res Function(_$PaymentProviderImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? name = freezed,
    Object? displayName = freezed,
    Object? enabled = freezed,
  }) {
    return _then(_$PaymentProviderImpl(
      name: freezed == name
          ? _value.name
          : name // ignore: cast_nullable_to_non_nullable
              as String?,
      displayName: freezed == displayName
          ? _value.displayName
          : displayName // ignore: cast_nullable_to_non_nullable
              as String?,
      enabled: freezed == enabled
          ? _value.enabled
          : enabled // ignore: cast_nullable_to_non_nullable
              as bool?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$PaymentProviderImpl implements _PaymentProvider {
  const _$PaymentProviderImpl({this.name, this.displayName, this.enabled});

  factory _$PaymentProviderImpl.fromJson(Map<String, dynamic> json) =>
      _$$PaymentProviderImplFromJson(json);

  @override
  final String? name;
  @override
  final String? displayName;
  @override
  final bool? enabled;

  @override
  String toString() {
    return 'PaymentProvider(name: $name, displayName: $displayName, enabled: $enabled)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$PaymentProviderImpl &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.displayName, displayName) ||
                other.displayName == displayName) &&
            (identical(other.enabled, enabled) || other.enabled == enabled));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode => Object.hash(runtimeType, name, displayName, enabled);

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$PaymentProviderImplCopyWith<_$PaymentProviderImpl> get copyWith =>
      __$$PaymentProviderImplCopyWithImpl<_$PaymentProviderImpl>(
          this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$PaymentProviderImplToJson(
      this,
    );
  }
}

abstract class _PaymentProvider implements PaymentProvider {
  const factory _PaymentProvider(
      {final String? name,
      final String? displayName,
      final bool? enabled}) = _$PaymentProviderImpl;

  factory _PaymentProvider.fromJson(Map<String, dynamic> json) =
      _$PaymentProviderImpl.fromJson;

  @override
  String? get name;
  @override
  String? get displayName;
  @override
  bool? get enabled;
  @override
  @JsonKey(ignore: true)
  _$$PaymentProviderImplCopyWith<_$PaymentProviderImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
