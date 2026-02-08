// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'dashboard.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

DashboardData _$DashboardDataFromJson(Map<String, dynamic> json) {
  return _DashboardData.fromJson(json);
}

/// @nodoc
mixin _$DashboardData {
  DashboardMetrics? get metrics => throw _privateConstructorUsedError;
  List<ChartPoint>? get spendTrend => throw _privateConstructorUsedError;
  List<OrderDistribution>? get orderDistribution =>
      throw _privateConstructorUsedError;
  List<VpsInstance>? get expiringInstances =>
      throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $DashboardDataCopyWith<DashboardData> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $DashboardDataCopyWith<$Res> {
  factory $DashboardDataCopyWith(
          DashboardData value, $Res Function(DashboardData) then) =
      _$DashboardDataCopyWithImpl<$Res, DashboardData>;
  @useResult
  $Res call(
      {DashboardMetrics? metrics,
      List<ChartPoint>? spendTrend,
      List<OrderDistribution>? orderDistribution,
      List<VpsInstance>? expiringInstances});

  $DashboardMetricsCopyWith<$Res>? get metrics;
}

/// @nodoc
class _$DashboardDataCopyWithImpl<$Res, $Val extends DashboardData>
    implements $DashboardDataCopyWith<$Res> {
  _$DashboardDataCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? metrics = freezed,
    Object? spendTrend = freezed,
    Object? orderDistribution = freezed,
    Object? expiringInstances = freezed,
  }) {
    return _then(_value.copyWith(
      metrics: freezed == metrics
          ? _value.metrics
          : metrics // ignore: cast_nullable_to_non_nullable
              as DashboardMetrics?,
      spendTrend: freezed == spendTrend
          ? _value.spendTrend
          : spendTrend // ignore: cast_nullable_to_non_nullable
              as List<ChartPoint>?,
      orderDistribution: freezed == orderDistribution
          ? _value.orderDistribution
          : orderDistribution // ignore: cast_nullable_to_non_nullable
              as List<OrderDistribution>?,
      expiringInstances: freezed == expiringInstances
          ? _value.expiringInstances
          : expiringInstances // ignore: cast_nullable_to_non_nullable
              as List<VpsInstance>?,
    ) as $Val);
  }

  @override
  @pragma('vm:prefer-inline')
  $DashboardMetricsCopyWith<$Res>? get metrics {
    if (_value.metrics == null) {
      return null;
    }

    return $DashboardMetricsCopyWith<$Res>(_value.metrics!, (value) {
      return _then(_value.copyWith(metrics: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$DashboardDataImplCopyWith<$Res>
    implements $DashboardDataCopyWith<$Res> {
  factory _$$DashboardDataImplCopyWith(
          _$DashboardDataImpl value, $Res Function(_$DashboardDataImpl) then) =
      __$$DashboardDataImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {DashboardMetrics? metrics,
      List<ChartPoint>? spendTrend,
      List<OrderDistribution>? orderDistribution,
      List<VpsInstance>? expiringInstances});

  @override
  $DashboardMetricsCopyWith<$Res>? get metrics;
}

/// @nodoc
class __$$DashboardDataImplCopyWithImpl<$Res>
    extends _$DashboardDataCopyWithImpl<$Res, _$DashboardDataImpl>
    implements _$$DashboardDataImplCopyWith<$Res> {
  __$$DashboardDataImplCopyWithImpl(
      _$DashboardDataImpl _value, $Res Function(_$DashboardDataImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? metrics = freezed,
    Object? spendTrend = freezed,
    Object? orderDistribution = freezed,
    Object? expiringInstances = freezed,
  }) {
    return _then(_$DashboardDataImpl(
      metrics: freezed == metrics
          ? _value.metrics
          : metrics // ignore: cast_nullable_to_non_nullable
              as DashboardMetrics?,
      spendTrend: freezed == spendTrend
          ? _value._spendTrend
          : spendTrend // ignore: cast_nullable_to_non_nullable
              as List<ChartPoint>?,
      orderDistribution: freezed == orderDistribution
          ? _value._orderDistribution
          : orderDistribution // ignore: cast_nullable_to_non_nullable
              as List<OrderDistribution>?,
      expiringInstances: freezed == expiringInstances
          ? _value._expiringInstances
          : expiringInstances // ignore: cast_nullable_to_non_nullable
              as List<VpsInstance>?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$DashboardDataImpl implements _DashboardData {
  const _$DashboardDataImpl(
      {this.metrics,
      final List<ChartPoint>? spendTrend,
      final List<OrderDistribution>? orderDistribution,
      final List<VpsInstance>? expiringInstances})
      : _spendTrend = spendTrend,
        _orderDistribution = orderDistribution,
        _expiringInstances = expiringInstances;

  factory _$DashboardDataImpl.fromJson(Map<String, dynamic> json) =>
      _$$DashboardDataImplFromJson(json);

  @override
  final DashboardMetrics? metrics;
  final List<ChartPoint>? _spendTrend;
  @override
  List<ChartPoint>? get spendTrend {
    final value = _spendTrend;
    if (value == null) return null;
    if (_spendTrend is EqualUnmodifiableListView) return _spendTrend;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  final List<OrderDistribution>? _orderDistribution;
  @override
  List<OrderDistribution>? get orderDistribution {
    final value = _orderDistribution;
    if (value == null) return null;
    if (_orderDistribution is EqualUnmodifiableListView)
      return _orderDistribution;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  final List<VpsInstance>? _expiringInstances;
  @override
  List<VpsInstance>? get expiringInstances {
    final value = _expiringInstances;
    if (value == null) return null;
    if (_expiringInstances is EqualUnmodifiableListView)
      return _expiringInstances;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  @override
  String toString() {
    return 'DashboardData(metrics: $metrics, spendTrend: $spendTrend, orderDistribution: $orderDistribution, expiringInstances: $expiringInstances)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$DashboardDataImpl &&
            (identical(other.metrics, metrics) || other.metrics == metrics) &&
            const DeepCollectionEquality()
                .equals(other._spendTrend, _spendTrend) &&
            const DeepCollectionEquality()
                .equals(other._orderDistribution, _orderDistribution) &&
            const DeepCollectionEquality()
                .equals(other._expiringInstances, _expiringInstances));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode => Object.hash(
      runtimeType,
      metrics,
      const DeepCollectionEquality().hash(_spendTrend),
      const DeepCollectionEquality().hash(_orderDistribution),
      const DeepCollectionEquality().hash(_expiringInstances));

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$DashboardDataImplCopyWith<_$DashboardDataImpl> get copyWith =>
      __$$DashboardDataImplCopyWithImpl<_$DashboardDataImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$DashboardDataImplToJson(
      this,
    );
  }
}

abstract class _DashboardData implements DashboardData {
  const factory _DashboardData(
      {final DashboardMetrics? metrics,
      final List<ChartPoint>? spendTrend,
      final List<OrderDistribution>? orderDistribution,
      final List<VpsInstance>? expiringInstances}) = _$DashboardDataImpl;

  factory _DashboardData.fromJson(Map<String, dynamic> json) =
      _$DashboardDataImpl.fromJson;

  @override
  DashboardMetrics? get metrics;
  @override
  List<ChartPoint>? get spendTrend;
  @override
  List<OrderDistribution>? get orderDistribution;
  @override
  List<VpsInstance>? get expiringInstances;
  @override
  @JsonKey(ignore: true)
  _$$DashboardDataImplCopyWith<_$DashboardDataImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

DashboardMetrics _$DashboardMetricsFromJson(Map<String, dynamic> json) {
  return _DashboardMetrics.fromJson(json);
}

/// @nodoc
mixin _$DashboardMetrics {
  double? get balance => throw _privateConstructorUsedError;
  String? get currency => throw _privateConstructorUsedError;
  @JsonKey(name: 'vps_total')
  int? get vpsTotal => throw _privateConstructorUsedError;
  @JsonKey(name: 'orders_total')
  int? get ordersTotal => throw _privateConstructorUsedError;
  @JsonKey(name: 'spend_30d')
  double? get spend30d => throw _privateConstructorUsedError;
  @JsonKey(name: 'realname_status')
  String? get realnameStatus => throw _privateConstructorUsedError;
  int? get expiring => throw _privateConstructorUsedError;
  @JsonKey(name: 'cart_items')
  int? get cartItems => throw _privateConstructorUsedError;
  @JsonKey(name: 'pending_orders')
  int? get pendingOrders => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $DashboardMetricsCopyWith<DashboardMetrics> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $DashboardMetricsCopyWith<$Res> {
  factory $DashboardMetricsCopyWith(
          DashboardMetrics value, $Res Function(DashboardMetrics) then) =
      _$DashboardMetricsCopyWithImpl<$Res, DashboardMetrics>;
  @useResult
  $Res call(
      {double? balance,
      String? currency,
      @JsonKey(name: 'vps_total') int? vpsTotal,
      @JsonKey(name: 'orders_total') int? ordersTotal,
      @JsonKey(name: 'spend_30d') double? spend30d,
      @JsonKey(name: 'realname_status') String? realnameStatus,
      int? expiring,
      @JsonKey(name: 'cart_items') int? cartItems,
      @JsonKey(name: 'pending_orders') int? pendingOrders});
}

/// @nodoc
class _$DashboardMetricsCopyWithImpl<$Res, $Val extends DashboardMetrics>
    implements $DashboardMetricsCopyWith<$Res> {
  _$DashboardMetricsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? balance = freezed,
    Object? currency = freezed,
    Object? vpsTotal = freezed,
    Object? ordersTotal = freezed,
    Object? spend30d = freezed,
    Object? realnameStatus = freezed,
    Object? expiring = freezed,
    Object? cartItems = freezed,
    Object? pendingOrders = freezed,
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
      vpsTotal: freezed == vpsTotal
          ? _value.vpsTotal
          : vpsTotal // ignore: cast_nullable_to_non_nullable
              as int?,
      ordersTotal: freezed == ordersTotal
          ? _value.ordersTotal
          : ordersTotal // ignore: cast_nullable_to_non_nullable
              as int?,
      spend30d: freezed == spend30d
          ? _value.spend30d
          : spend30d // ignore: cast_nullable_to_non_nullable
              as double?,
      realnameStatus: freezed == realnameStatus
          ? _value.realnameStatus
          : realnameStatus // ignore: cast_nullable_to_non_nullable
              as String?,
      expiring: freezed == expiring
          ? _value.expiring
          : expiring // ignore: cast_nullable_to_non_nullable
              as int?,
      cartItems: freezed == cartItems
          ? _value.cartItems
          : cartItems // ignore: cast_nullable_to_non_nullable
              as int?,
      pendingOrders: freezed == pendingOrders
          ? _value.pendingOrders
          : pendingOrders // ignore: cast_nullable_to_non_nullable
              as int?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$DashboardMetricsImplCopyWith<$Res>
    implements $DashboardMetricsCopyWith<$Res> {
  factory _$$DashboardMetricsImplCopyWith(_$DashboardMetricsImpl value,
          $Res Function(_$DashboardMetricsImpl) then) =
      __$$DashboardMetricsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {double? balance,
      String? currency,
      @JsonKey(name: 'vps_total') int? vpsTotal,
      @JsonKey(name: 'orders_total') int? ordersTotal,
      @JsonKey(name: 'spend_30d') double? spend30d,
      @JsonKey(name: 'realname_status') String? realnameStatus,
      int? expiring,
      @JsonKey(name: 'cart_items') int? cartItems,
      @JsonKey(name: 'pending_orders') int? pendingOrders});
}

/// @nodoc
class __$$DashboardMetricsImplCopyWithImpl<$Res>
    extends _$DashboardMetricsCopyWithImpl<$Res, _$DashboardMetricsImpl>
    implements _$$DashboardMetricsImplCopyWith<$Res> {
  __$$DashboardMetricsImplCopyWithImpl(_$DashboardMetricsImpl _value,
      $Res Function(_$DashboardMetricsImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? balance = freezed,
    Object? currency = freezed,
    Object? vpsTotal = freezed,
    Object? ordersTotal = freezed,
    Object? spend30d = freezed,
    Object? realnameStatus = freezed,
    Object? expiring = freezed,
    Object? cartItems = freezed,
    Object? pendingOrders = freezed,
  }) {
    return _then(_$DashboardMetricsImpl(
      balance: freezed == balance
          ? _value.balance
          : balance // ignore: cast_nullable_to_non_nullable
              as double?,
      currency: freezed == currency
          ? _value.currency
          : currency // ignore: cast_nullable_to_non_nullable
              as String?,
      vpsTotal: freezed == vpsTotal
          ? _value.vpsTotal
          : vpsTotal // ignore: cast_nullable_to_non_nullable
              as int?,
      ordersTotal: freezed == ordersTotal
          ? _value.ordersTotal
          : ordersTotal // ignore: cast_nullable_to_non_nullable
              as int?,
      spend30d: freezed == spend30d
          ? _value.spend30d
          : spend30d // ignore: cast_nullable_to_non_nullable
              as double?,
      realnameStatus: freezed == realnameStatus
          ? _value.realnameStatus
          : realnameStatus // ignore: cast_nullable_to_non_nullable
              as String?,
      expiring: freezed == expiring
          ? _value.expiring
          : expiring // ignore: cast_nullable_to_non_nullable
              as int?,
      cartItems: freezed == cartItems
          ? _value.cartItems
          : cartItems // ignore: cast_nullable_to_non_nullable
              as int?,
      pendingOrders: freezed == pendingOrders
          ? _value.pendingOrders
          : pendingOrders // ignore: cast_nullable_to_non_nullable
              as int?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$DashboardMetricsImpl implements _DashboardMetrics {
  const _$DashboardMetricsImpl(
      {this.balance,
      this.currency,
      @JsonKey(name: 'vps_total') this.vpsTotal,
      @JsonKey(name: 'orders_total') this.ordersTotal,
      @JsonKey(name: 'spend_30d') this.spend30d,
      @JsonKey(name: 'realname_status') this.realnameStatus,
      this.expiring,
      @JsonKey(name: 'cart_items') this.cartItems,
      @JsonKey(name: 'pending_orders') this.pendingOrders});

  factory _$DashboardMetricsImpl.fromJson(Map<String, dynamic> json) =>
      _$$DashboardMetricsImplFromJson(json);

  @override
  final double? balance;
  @override
  final String? currency;
  @override
  @JsonKey(name: 'vps_total')
  final int? vpsTotal;
  @override
  @JsonKey(name: 'orders_total')
  final int? ordersTotal;
  @override
  @JsonKey(name: 'spend_30d')
  final double? spend30d;
  @override
  @JsonKey(name: 'realname_status')
  final String? realnameStatus;
  @override
  final int? expiring;
  @override
  @JsonKey(name: 'cart_items')
  final int? cartItems;
  @override
  @JsonKey(name: 'pending_orders')
  final int? pendingOrders;

  @override
  String toString() {
    return 'DashboardMetrics(balance: $balance, currency: $currency, vpsTotal: $vpsTotal, ordersTotal: $ordersTotal, spend30d: $spend30d, realnameStatus: $realnameStatus, expiring: $expiring, cartItems: $cartItems, pendingOrders: $pendingOrders)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$DashboardMetricsImpl &&
            (identical(other.balance, balance) || other.balance == balance) &&
            (identical(other.currency, currency) ||
                other.currency == currency) &&
            (identical(other.vpsTotal, vpsTotal) ||
                other.vpsTotal == vpsTotal) &&
            (identical(other.ordersTotal, ordersTotal) ||
                other.ordersTotal == ordersTotal) &&
            (identical(other.spend30d, spend30d) ||
                other.spend30d == spend30d) &&
            (identical(other.realnameStatus, realnameStatus) ||
                other.realnameStatus == realnameStatus) &&
            (identical(other.expiring, expiring) ||
                other.expiring == expiring) &&
            (identical(other.cartItems, cartItems) ||
                other.cartItems == cartItems) &&
            (identical(other.pendingOrders, pendingOrders) ||
                other.pendingOrders == pendingOrders));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode => Object.hash(
      runtimeType,
      balance,
      currency,
      vpsTotal,
      ordersTotal,
      spend30d,
      realnameStatus,
      expiring,
      cartItems,
      pendingOrders);

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$DashboardMetricsImplCopyWith<_$DashboardMetricsImpl> get copyWith =>
      __$$DashboardMetricsImplCopyWithImpl<_$DashboardMetricsImpl>(
          this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$DashboardMetricsImplToJson(
      this,
    );
  }
}

abstract class _DashboardMetrics implements DashboardMetrics {
  const factory _DashboardMetrics(
          {final double? balance,
          final String? currency,
          @JsonKey(name: 'vps_total') final int? vpsTotal,
          @JsonKey(name: 'orders_total') final int? ordersTotal,
          @JsonKey(name: 'spend_30d') final double? spend30d,
          @JsonKey(name: 'realname_status') final String? realnameStatus,
          final int? expiring,
          @JsonKey(name: 'cart_items') final int? cartItems,
          @JsonKey(name: 'pending_orders') final int? pendingOrders}) =
      _$DashboardMetricsImpl;

  factory _DashboardMetrics.fromJson(Map<String, dynamic> json) =
      _$DashboardMetricsImpl.fromJson;

  @override
  double? get balance;
  @override
  String? get currency;
  @override
  @JsonKey(name: 'vps_total')
  int? get vpsTotal;
  @override
  @JsonKey(name: 'orders_total')
  int? get ordersTotal;
  @override
  @JsonKey(name: 'spend_30d')
  double? get spend30d;
  @override
  @JsonKey(name: 'realname_status')
  String? get realnameStatus;
  @override
  int? get expiring;
  @override
  @JsonKey(name: 'cart_items')
  int? get cartItems;
  @override
  @JsonKey(name: 'pending_orders')
  int? get pendingOrders;
  @override
  @JsonKey(ignore: true)
  _$$DashboardMetricsImplCopyWith<_$DashboardMetricsImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

ChartPoint _$ChartPointFromJson(Map<String, dynamic> json) {
  return _ChartPoint.fromJson(json);
}

/// @nodoc
mixin _$ChartPoint {
  String? get date => throw _privateConstructorUsedError;
  double? get value => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $ChartPointCopyWith<ChartPoint> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ChartPointCopyWith<$Res> {
  factory $ChartPointCopyWith(
          ChartPoint value, $Res Function(ChartPoint) then) =
      _$ChartPointCopyWithImpl<$Res, ChartPoint>;
  @useResult
  $Res call({String? date, double? value});
}

/// @nodoc
class _$ChartPointCopyWithImpl<$Res, $Val extends ChartPoint>
    implements $ChartPointCopyWith<$Res> {
  _$ChartPointCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? date = freezed,
    Object? value = freezed,
  }) {
    return _then(_value.copyWith(
      date: freezed == date
          ? _value.date
          : date // ignore: cast_nullable_to_non_nullable
              as String?,
      value: freezed == value
          ? _value.value
          : value // ignore: cast_nullable_to_non_nullable
              as double?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$ChartPointImplCopyWith<$Res>
    implements $ChartPointCopyWith<$Res> {
  factory _$$ChartPointImplCopyWith(
          _$ChartPointImpl value, $Res Function(_$ChartPointImpl) then) =
      __$$ChartPointImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String? date, double? value});
}

/// @nodoc
class __$$ChartPointImplCopyWithImpl<$Res>
    extends _$ChartPointCopyWithImpl<$Res, _$ChartPointImpl>
    implements _$$ChartPointImplCopyWith<$Res> {
  __$$ChartPointImplCopyWithImpl(
      _$ChartPointImpl _value, $Res Function(_$ChartPointImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? date = freezed,
    Object? value = freezed,
  }) {
    return _then(_$ChartPointImpl(
      date: freezed == date
          ? _value.date
          : date // ignore: cast_nullable_to_non_nullable
              as String?,
      value: freezed == value
          ? _value.value
          : value // ignore: cast_nullable_to_non_nullable
              as double?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$ChartPointImpl implements _ChartPoint {
  const _$ChartPointImpl({this.date, this.value});

  factory _$ChartPointImpl.fromJson(Map<String, dynamic> json) =>
      _$$ChartPointImplFromJson(json);

  @override
  final String? date;
  @override
  final double? value;

  @override
  String toString() {
    return 'ChartPoint(date: $date, value: $value)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ChartPointImpl &&
            (identical(other.date, date) || other.date == date) &&
            (identical(other.value, value) || other.value == value));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode => Object.hash(runtimeType, date, value);

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$ChartPointImplCopyWith<_$ChartPointImpl> get copyWith =>
      __$$ChartPointImplCopyWithImpl<_$ChartPointImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$ChartPointImplToJson(
      this,
    );
  }
}

abstract class _ChartPoint implements ChartPoint {
  const factory _ChartPoint({final String? date, final double? value}) =
      _$ChartPointImpl;

  factory _ChartPoint.fromJson(Map<String, dynamic> json) =
      _$ChartPointImpl.fromJson;

  @override
  String? get date;
  @override
  double? get value;
  @override
  @JsonKey(ignore: true)
  _$$ChartPointImplCopyWith<_$ChartPointImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

OrderDistribution _$OrderDistributionFromJson(Map<String, dynamic> json) {
  return _OrderDistribution.fromJson(json);
}

/// @nodoc
mixin _$OrderDistribution {
  String? get status => throw _privateConstructorUsedError;
  int? get count => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $OrderDistributionCopyWith<OrderDistribution> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $OrderDistributionCopyWith<$Res> {
  factory $OrderDistributionCopyWith(
          OrderDistribution value, $Res Function(OrderDistribution) then) =
      _$OrderDistributionCopyWithImpl<$Res, OrderDistribution>;
  @useResult
  $Res call({String? status, int? count});
}

/// @nodoc
class _$OrderDistributionCopyWithImpl<$Res, $Val extends OrderDistribution>
    implements $OrderDistributionCopyWith<$Res> {
  _$OrderDistributionCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? status = freezed,
    Object? count = freezed,
  }) {
    return _then(_value.copyWith(
      status: freezed == status
          ? _value.status
          : status // ignore: cast_nullable_to_non_nullable
              as String?,
      count: freezed == count
          ? _value.count
          : count // ignore: cast_nullable_to_non_nullable
              as int?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$OrderDistributionImplCopyWith<$Res>
    implements $OrderDistributionCopyWith<$Res> {
  factory _$$OrderDistributionImplCopyWith(_$OrderDistributionImpl value,
          $Res Function(_$OrderDistributionImpl) then) =
      __$$OrderDistributionImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String? status, int? count});
}

/// @nodoc
class __$$OrderDistributionImplCopyWithImpl<$Res>
    extends _$OrderDistributionCopyWithImpl<$Res, _$OrderDistributionImpl>
    implements _$$OrderDistributionImplCopyWith<$Res> {
  __$$OrderDistributionImplCopyWithImpl(_$OrderDistributionImpl _value,
      $Res Function(_$OrderDistributionImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? status = freezed,
    Object? count = freezed,
  }) {
    return _then(_$OrderDistributionImpl(
      status: freezed == status
          ? _value.status
          : status // ignore: cast_nullable_to_non_nullable
              as String?,
      count: freezed == count
          ? _value.count
          : count // ignore: cast_nullable_to_non_nullable
              as int?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$OrderDistributionImpl implements _OrderDistribution {
  const _$OrderDistributionImpl({this.status, this.count});

  factory _$OrderDistributionImpl.fromJson(Map<String, dynamic> json) =>
      _$$OrderDistributionImplFromJson(json);

  @override
  final String? status;
  @override
  final int? count;

  @override
  String toString() {
    return 'OrderDistribution(status: $status, count: $count)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$OrderDistributionImpl &&
            (identical(other.status, status) || other.status == status) &&
            (identical(other.count, count) || other.count == count));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode => Object.hash(runtimeType, status, count);

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$OrderDistributionImplCopyWith<_$OrderDistributionImpl> get copyWith =>
      __$$OrderDistributionImplCopyWithImpl<_$OrderDistributionImpl>(
          this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$OrderDistributionImplToJson(
      this,
    );
  }
}

abstract class _OrderDistribution implements OrderDistribution {
  const factory _OrderDistribution({final String? status, final int? count}) =
      _$OrderDistributionImpl;

  factory _OrderDistribution.fromJson(Map<String, dynamic> json) =
      _$OrderDistributionImpl.fromJson;

  @override
  String? get status;
  @override
  int? get count;
  @override
  @JsonKey(ignore: true)
  _$$OrderDistributionImplCopyWith<_$OrderDistributionImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
