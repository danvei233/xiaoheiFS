// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'dashboard.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$DashboardDataImpl _$$DashboardDataImplFromJson(Map<String, dynamic> json) =>
    _$DashboardDataImpl(
      metrics: json['metrics'] == null
          ? null
          : DashboardMetrics.fromJson(json['metrics'] as Map<String, dynamic>),
      spendTrend: (json['spendTrend'] as List<dynamic>?)
          ?.map((e) => ChartPoint.fromJson(e as Map<String, dynamic>))
          .toList(),
      orderDistribution: (json['orderDistribution'] as List<dynamic>?)
          ?.map((e) => OrderDistribution.fromJson(e as Map<String, dynamic>))
          .toList(),
      expiringInstances: (json['expiringInstances'] as List<dynamic>?)
          ?.map((e) => VpsInstance.fromJson(e as Map<String, dynamic>))
          .toList(),
    );

Map<String, dynamic> _$$DashboardDataImplToJson(_$DashboardDataImpl instance) =>
    <String, dynamic>{
      'metrics': instance.metrics,
      'spendTrend': instance.spendTrend,
      'orderDistribution': instance.orderDistribution,
      'expiringInstances': instance.expiringInstances,
    };

_$DashboardMetricsImpl _$$DashboardMetricsImplFromJson(
        Map<String, dynamic> json) =>
    _$DashboardMetricsImpl(
      balance: (json['balance'] as num?)?.toDouble(),
      currency: json['currency'] as String?,
      vpsTotal: (json['vps_total'] as num?)?.toInt(),
      ordersTotal: (json['orders_total'] as num?)?.toInt(),
      spend30d: (json['spend_30d'] as num?)?.toDouble(),
      realnameStatus: json['realname_status'] as String?,
      expiring: (json['expiring'] as num?)?.toInt(),
      cartItems: (json['cart_items'] as num?)?.toInt(),
      pendingOrders: (json['pending_orders'] as num?)?.toInt(),
    );

Map<String, dynamic> _$$DashboardMetricsImplToJson(
        _$DashboardMetricsImpl instance) =>
    <String, dynamic>{
      'balance': instance.balance,
      'currency': instance.currency,
      'vps_total': instance.vpsTotal,
      'orders_total': instance.ordersTotal,
      'spend_30d': instance.spend30d,
      'realname_status': instance.realnameStatus,
      'expiring': instance.expiring,
      'cart_items': instance.cartItems,
      'pending_orders': instance.pendingOrders,
    };

_$ChartPointImpl _$$ChartPointImplFromJson(Map<String, dynamic> json) =>
    _$ChartPointImpl(
      date: json['date'] as String?,
      value: (json['value'] as num?)?.toDouble(),
    );

Map<String, dynamic> _$$ChartPointImplToJson(_$ChartPointImpl instance) =>
    <String, dynamic>{
      'date': instance.date,
      'value': instance.value,
    };

_$OrderDistributionImpl _$$OrderDistributionImplFromJson(
        Map<String, dynamic> json) =>
    _$OrderDistributionImpl(
      status: json['status'] as String?,
      count: (json['count'] as num?)?.toInt(),
    );

Map<String, dynamic> _$$OrderDistributionImplToJson(
        _$OrderDistributionImpl instance) =>
    <String, dynamic>{
      'status': instance.status,
      'count': instance.count,
    };
