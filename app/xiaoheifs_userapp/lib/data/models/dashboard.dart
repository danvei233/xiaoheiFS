import 'package:freezed_annotation/freezed_annotation.dart';
import 'vps_instance.dart';

part 'dashboard.freezed.dart';
part 'dashboard.g.dart';

/// Dashboard数据模型
@freezed
class DashboardData with _$DashboardData {
  const factory DashboardData({
    DashboardMetrics? metrics,
    List<ChartPoint>? spendTrend,
    List<OrderDistribution>? orderDistribution,
    List<VpsInstance>? expiringInstances,
  }) = _DashboardData;

  factory DashboardData.fromJson(Map<String, dynamic> json) =>
      _$DashboardDataFromJson(json);
}

/// Dashboard指标模型
@freezed
class DashboardMetrics with _$DashboardMetrics {
  const factory DashboardMetrics({
    double? balance,
    String? currency,
    @JsonKey(name: 'vps_total') int? vpsTotal,
    @JsonKey(name: 'orders_total') int? ordersTotal,
    @JsonKey(name: 'spend_30d') double? spend30d,
    @JsonKey(name: 'realname_status') String? realnameStatus,
    int? expiring,
    @JsonKey(name: 'cart_items') int? cartItems,
    @JsonKey(name: 'pending_orders') int? pendingOrders,
  }) = _DashboardMetrics;

  factory DashboardMetrics.fromJson(Map<String, dynamic> json) =>
      _$DashboardMetricsFromJson(json);
}

/// 图表数据点模型
@freezed
class ChartPoint with _$ChartPoint {
  const factory ChartPoint({
    String? date,
    double? value,
  }) = _ChartPoint;

  factory ChartPoint.fromJson(Map<String, dynamic> json) =>
      _$ChartPointFromJson(json);
}

/// 订单分布模型
@freezed
class OrderDistribution with _$OrderDistribution {
  const factory OrderDistribution({
    String? status,
    int? count,
  }) = _OrderDistribution;

  factory OrderDistribution.fromJson(Map<String, dynamic> json) =>
      _$OrderDistributionFromJson(json);
}
