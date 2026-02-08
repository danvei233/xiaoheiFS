// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'vps_instance.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$VpsInstanceImpl _$$VpsInstanceImplFromJson(Map<String, dynamic> json) =>
    _$VpsInstanceImpl(
      id: (json['id'] as num?)?.toInt(),
      name: json['name'] as String?,
      status: json['status'] as String?,
      ip: json['ip'] as String?,
      ipv6: json['ipv6'] as String?,
      regionName: json['region_name'] as String?,
      regionLine: json['region_line'] as String?,
      packageName: json['package_name'] as String?,
      specText: json['spec_text'] as String?,
      cpuCores: (json['cpu_cores'] as num?)?.toInt(),
      memoryMb: (json['memory_mb'] as num?)?.toInt(),
      diskGb: (json['disk_gb'] as num?)?.toInt(),
      bandwidthMb: (json['bandwidth_mb'] as num?)?.toInt(),
      expireAt: json['expire_at'] as String?,
      createdAt: json['created_at'] as String?,
      osName: json['os_name'] as String?,
      osType: json['os_type'] as String?,
      panelUrl: json['panel_url'] as String?,
      vncUrl: json['vnc_url'] as String?,
      metrics: json['metrics'] as Map<String, dynamic>?,
    );

Map<String, dynamic> _$$VpsInstanceImplToJson(_$VpsInstanceImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'name': instance.name,
      'status': instance.status,
      'ip': instance.ip,
      'ipv6': instance.ipv6,
      'region_name': instance.regionName,
      'region_line': instance.regionLine,
      'package_name': instance.packageName,
      'spec_text': instance.specText,
      'cpu_cores': instance.cpuCores,
      'memory_mb': instance.memoryMb,
      'disk_gb': instance.diskGb,
      'bandwidth_mb': instance.bandwidthMb,
      'expire_at': instance.expireAt,
      'created_at': instance.createdAt,
      'os_name': instance.osName,
      'os_type': instance.osType,
      'panel_url': instance.panelUrl,
      'vnc_url': instance.vncUrl,
      'metrics': instance.metrics,
    };

_$VpsMetricsImpl _$$VpsMetricsImplFromJson(Map<String, dynamic> json) =>
    _$VpsMetricsImpl(
      cpuUsage: (json['cpu_usage'] as num?)?.toDouble(),
      memoryUsage: (json['memory_usage'] as num?)?.toDouble(),
      diskUsage: (json['disk_usage'] as num?)?.toDouble(),
      networkIn: (json['network_in'] as num?)?.toInt(),
      networkOut: (json['network_out'] as num?)?.toInt(),
    );

Map<String, dynamic> _$$VpsMetricsImplToJson(_$VpsMetricsImpl instance) =>
    <String, dynamic>{
      'cpu_usage': instance.cpuUsage,
      'memory_usage': instance.memoryUsage,
      'disk_usage': instance.diskUsage,
      'network_in': instance.networkIn,
      'network_out': instance.networkOut,
    };

_$CatalogImpl _$$CatalogImplFromJson(Map<String, dynamic> json) =>
    _$CatalogImpl(
      regions: (json['regions'] as List<dynamic>?)
          ?.map((e) => Region.fromJson(e as Map<String, dynamic>))
          .toList(),
      packages: (json['packages'] as List<dynamic>?)
          ?.map((e) => PackageModel.fromJson(e as Map<String, dynamic>))
          .toList(),
      systemImages: (json['system_images'] as List<dynamic>?)
          ?.map((e) => SystemImage.fromJson(e as Map<String, dynamic>))
          .toList(),
      billingCycles: (json['billing_cycles'] as List<dynamic>?)
          ?.map((e) => BillingCycle.fromJson(e as Map<String, dynamic>))
          .toList(),
    );

Map<String, dynamic> _$$CatalogImplToJson(_$CatalogImpl instance) =>
    <String, dynamic>{
      'regions': instance.regions,
      'packages': instance.packages,
      'system_images': instance.systemImages,
      'billing_cycles': instance.billingCycles,
    };

_$RegionImpl _$$RegionImplFromJson(Map<String, dynamic> json) => _$RegionImpl(
      id: (json['id'] as num?)?.toInt(),
      name: json['name'] as String?,
      lines: (json['lines'] as List<dynamic>?)
          ?.map((e) => RegionLine.fromJson(e as Map<String, dynamic>))
          .toList(),
    );

Map<String, dynamic> _$$RegionImplToJson(_$RegionImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'name': instance.name,
      'lines': instance.lines,
    };

_$RegionLineImpl _$$RegionLineImplFromJson(Map<String, dynamic> json) =>
    _$RegionLineImpl(
      id: (json['id'] as num?)?.toInt(),
      name: json['name'] as String?,
    );

Map<String, dynamic> _$$RegionLineImplToJson(_$RegionLineImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'name': instance.name,
    };

_$PackageModelImpl _$$PackageModelImplFromJson(Map<String, dynamic> json) =>
    _$PackageModelImpl(
      id: (json['id'] as num?)?.toInt(),
      name: json['name'] as String?,
      cpuCores: (json['cpu_cores'] as num?)?.toInt(),
      memoryMb: (json['memory_mb'] as num?)?.toInt(),
      diskGb: (json['disk_gb'] as num?)?.toInt(),
      bandwidthMb: (json['bandwidth_mb'] as num?)?.toInt(),
      priceMonthly: (json['price_monthly'] as num?)?.toDouble(),
      priceQuarterly: (json['price_quarterly'] as num?)?.toDouble(),
      priceHalfYearly: (json['price_half_yearly'] as num?)?.toDouble(),
      priceYearly: (json['price_yearly'] as num?)?.toDouble(),
    );

Map<String, dynamic> _$$PackageModelImplToJson(_$PackageModelImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'name': instance.name,
      'cpu_cores': instance.cpuCores,
      'memory_mb': instance.memoryMb,
      'disk_gb': instance.diskGb,
      'bandwidth_mb': instance.bandwidthMb,
      'price_monthly': instance.priceMonthly,
      'price_quarterly': instance.priceQuarterly,
      'price_half_yearly': instance.priceHalfYearly,
      'price_yearly': instance.priceYearly,
    };

_$SystemImageImpl _$$SystemImageImplFromJson(Map<String, dynamic> json) =>
    _$SystemImageImpl(
      id: (json['id'] as num?)?.toInt(),
      name: json['name'] as String?,
      type: json['type'] as String?,
      version: json['version'] as String?,
      imageUrl: json['image_url'] as String?,
    );

Map<String, dynamic> _$$SystemImageImplToJson(_$SystemImageImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'name': instance.name,
      'type': instance.type,
      'version': instance.version,
      'image_url': instance.imageUrl,
    };

_$BillingCycleImpl _$$BillingCycleImplFromJson(Map<String, dynamic> json) =>
    _$BillingCycleImpl(
      id: (json['id'] as num?)?.toInt(),
      name: json['name'] as String?,
      displayName: json['display_name'] as String?,
      months: (json['months'] as num?)?.toInt(),
    );

Map<String, dynamic> _$$BillingCycleImplToJson(_$BillingCycleImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'name': instance.name,
      'display_name': instance.displayName,
      'months': instance.months,
    };
