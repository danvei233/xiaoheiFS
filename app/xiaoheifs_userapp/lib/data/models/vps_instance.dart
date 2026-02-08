import 'package:freezed_annotation/freezed_annotation.dart';

part 'vps_instance.freezed.dart';
part 'vps_instance.g.dart';

/// VPS实例模型
@freezed
class VpsInstance with _$VpsInstance {
  const factory VpsInstance({
    int? id,
    String? name,
    String? status,
    String? ip,
    @JsonKey(name: 'ipv6') String? ipv6,
    @JsonKey(name: 'region_name') String? regionName,
    @JsonKey(name: 'region_line') String? regionLine,
    @JsonKey(name: 'package_name') String? packageName,
    @JsonKey(name: 'spec_text') String? specText,
    @JsonKey(name: 'cpu_cores') int? cpuCores,
    @JsonKey(name: 'memory_mb') int? memoryMb,
    @JsonKey(name: 'disk_gb') int? diskGb,
    @JsonKey(name: 'bandwidth_mb') int? bandwidthMb,
    @JsonKey(name: 'expire_at') String? expireAt,
    @JsonKey(name: 'created_at') String? createdAt,
    @JsonKey(name: 'os_name') String? osName,
    @JsonKey(name: 'os_type') String? osType,
    @JsonKey(name: 'panel_url') String? panelUrl,
    @JsonKey(name: 'vnc_url') String? vncUrl,
    Map<String, dynamic>? metrics,
  }) = _VpsInstance;

  factory VpsInstance.fromJson(Map<String, dynamic> json) =>
      _$VpsInstanceFromJson(json);
}

/// VPS监控数据模型
@freezed
class VpsMetrics with _$VpsMetrics {
  const factory VpsMetrics({
    @JsonKey(name: 'cpu_usage') double? cpuUsage,
    @JsonKey(name: 'memory_usage') double? memoryUsage,
    @JsonKey(name: 'disk_usage') double? diskUsage,
    @JsonKey(name: 'network_in') int? networkIn,
    @JsonKey(name: 'network_out') int? networkOut,
  }) = _VpsMetrics;

  factory VpsMetrics.fromJson(Map<String, dynamic> json) =>
      _$VpsMetricsFromJson(json);
}

/// 商品目录模型
@freezed
class Catalog with _$Catalog {
  const factory Catalog({
    List<Region>? regions,
    List<PackageModel>? packages,
    @JsonKey(name: 'system_images') List<SystemImage>? systemImages,
    @JsonKey(name: 'billing_cycles') List<BillingCycle>? billingCycles,
  }) = _Catalog;

  factory Catalog.fromJson(Map<String, dynamic> json) =>
      _$CatalogFromJson(json);
}

/// 地区模型
@freezed
class Region with _$Region {
  const factory Region({
    int? id,
    String? name,
    List<RegionLine>? lines,
  }) = _Region;

  factory Region.fromJson(Map<String, dynamic> json) =>
      _$RegionFromJson(json);
}

/// 地区线路模型
@freezed
class RegionLine with _$RegionLine {
  const factory RegionLine({
    int? id,
    String? name,
  }) = _RegionLine;

  factory RegionLine.fromJson(Map<String, dynamic> json) =>
      _$RegionLineFromJson(json);
}

/// 套餐模型
@freezed
class PackageModel with _$PackageModel {
  const factory PackageModel({
    int? id,
    String? name,
    @JsonKey(name: 'cpu_cores') int? cpuCores,
    @JsonKey(name: 'memory_mb') int? memoryMb,
    @JsonKey(name: 'disk_gb') int? diskGb,
    @JsonKey(name: 'bandwidth_mb') int? bandwidthMb,
    @JsonKey(name: 'price_monthly') double? priceMonthly,
    @JsonKey(name: 'price_quarterly') double? priceQuarterly,
    @JsonKey(name: 'price_half_yearly') double? priceHalfYearly,
    @JsonKey(name: 'price_yearly') double? priceYearly,
  }) = _PackageModel;

  factory PackageModel.fromJson(Map<String, dynamic> json) =>
      _$PackageModelFromJson(json);
}

/// 系统镜像模型
@freezed
class SystemImage with _$SystemImage {
  const factory SystemImage({
    int? id,
    String? name,
    String? type,
    String? version,
    @JsonKey(name: 'image_url') String? imageUrl,
  }) = _SystemImage;

  factory SystemImage.fromJson(Map<String, dynamic> json) =>
      _$SystemImageFromJson(json);
}

/// 计费周期模型
@freezed
class BillingCycle with _$BillingCycle {
  const factory BillingCycle({
    int? id,
    String? name,
    @JsonKey(name: 'display_name') String? displayName,
    @JsonKey(name: 'months') int? months,
  }) = _BillingCycle;

  factory BillingCycle.fromJson(Map<String, dynamic> json) =>
      _$BillingCycleFromJson(json);
}
