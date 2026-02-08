// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'vps_instance.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

VpsInstance _$VpsInstanceFromJson(Map<String, dynamic> json) {
  return _VpsInstance.fromJson(json);
}

/// @nodoc
mixin _$VpsInstance {
  int? get id => throw _privateConstructorUsedError;
  String? get name => throw _privateConstructorUsedError;
  String? get status => throw _privateConstructorUsedError;
  String? get ip => throw _privateConstructorUsedError;
  @JsonKey(name: 'ipv6')
  String? get ipv6 => throw _privateConstructorUsedError;
  @JsonKey(name: 'region_name')
  String? get regionName => throw _privateConstructorUsedError;
  @JsonKey(name: 'region_line')
  String? get regionLine => throw _privateConstructorUsedError;
  @JsonKey(name: 'package_name')
  String? get packageName => throw _privateConstructorUsedError;
  @JsonKey(name: 'spec_text')
  String? get specText => throw _privateConstructorUsedError;
  @JsonKey(name: 'cpu_cores')
  int? get cpuCores => throw _privateConstructorUsedError;
  @JsonKey(name: 'memory_mb')
  int? get memoryMb => throw _privateConstructorUsedError;
  @JsonKey(name: 'disk_gb')
  int? get diskGb => throw _privateConstructorUsedError;
  @JsonKey(name: 'bandwidth_mb')
  int? get bandwidthMb => throw _privateConstructorUsedError;
  @JsonKey(name: 'expire_at')
  String? get expireAt => throw _privateConstructorUsedError;
  @JsonKey(name: 'created_at')
  String? get createdAt => throw _privateConstructorUsedError;
  @JsonKey(name: 'os_name')
  String? get osName => throw _privateConstructorUsedError;
  @JsonKey(name: 'os_type')
  String? get osType => throw _privateConstructorUsedError;
  @JsonKey(name: 'panel_url')
  String? get panelUrl => throw _privateConstructorUsedError;
  @JsonKey(name: 'vnc_url')
  String? get vncUrl => throw _privateConstructorUsedError;
  Map<String, dynamic>? get metrics => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $VpsInstanceCopyWith<VpsInstance> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $VpsInstanceCopyWith<$Res> {
  factory $VpsInstanceCopyWith(
          VpsInstance value, $Res Function(VpsInstance) then) =
      _$VpsInstanceCopyWithImpl<$Res, VpsInstance>;
  @useResult
  $Res call(
      {int? id,
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
      Map<String, dynamic>? metrics});
}

/// @nodoc
class _$VpsInstanceCopyWithImpl<$Res, $Val extends VpsInstance>
    implements $VpsInstanceCopyWith<$Res> {
  _$VpsInstanceCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? name = freezed,
    Object? status = freezed,
    Object? ip = freezed,
    Object? ipv6 = freezed,
    Object? regionName = freezed,
    Object? regionLine = freezed,
    Object? packageName = freezed,
    Object? specText = freezed,
    Object? cpuCores = freezed,
    Object? memoryMb = freezed,
    Object? diskGb = freezed,
    Object? bandwidthMb = freezed,
    Object? expireAt = freezed,
    Object? createdAt = freezed,
    Object? osName = freezed,
    Object? osType = freezed,
    Object? panelUrl = freezed,
    Object? vncUrl = freezed,
    Object? metrics = freezed,
  }) {
    return _then(_value.copyWith(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      name: freezed == name
          ? _value.name
          : name // ignore: cast_nullable_to_non_nullable
              as String?,
      status: freezed == status
          ? _value.status
          : status // ignore: cast_nullable_to_non_nullable
              as String?,
      ip: freezed == ip
          ? _value.ip
          : ip // ignore: cast_nullable_to_non_nullable
              as String?,
      ipv6: freezed == ipv6
          ? _value.ipv6
          : ipv6 // ignore: cast_nullable_to_non_nullable
              as String?,
      regionName: freezed == regionName
          ? _value.regionName
          : regionName // ignore: cast_nullable_to_non_nullable
              as String?,
      regionLine: freezed == regionLine
          ? _value.regionLine
          : regionLine // ignore: cast_nullable_to_non_nullable
              as String?,
      packageName: freezed == packageName
          ? _value.packageName
          : packageName // ignore: cast_nullable_to_non_nullable
              as String?,
      specText: freezed == specText
          ? _value.specText
          : specText // ignore: cast_nullable_to_non_nullable
              as String?,
      cpuCores: freezed == cpuCores
          ? _value.cpuCores
          : cpuCores // ignore: cast_nullable_to_non_nullable
              as int?,
      memoryMb: freezed == memoryMb
          ? _value.memoryMb
          : memoryMb // ignore: cast_nullable_to_non_nullable
              as int?,
      diskGb: freezed == diskGb
          ? _value.diskGb
          : diskGb // ignore: cast_nullable_to_non_nullable
              as int?,
      bandwidthMb: freezed == bandwidthMb
          ? _value.bandwidthMb
          : bandwidthMb // ignore: cast_nullable_to_non_nullable
              as int?,
      expireAt: freezed == expireAt
          ? _value.expireAt
          : expireAt // ignore: cast_nullable_to_non_nullable
              as String?,
      createdAt: freezed == createdAt
          ? _value.createdAt
          : createdAt // ignore: cast_nullable_to_non_nullable
              as String?,
      osName: freezed == osName
          ? _value.osName
          : osName // ignore: cast_nullable_to_non_nullable
              as String?,
      osType: freezed == osType
          ? _value.osType
          : osType // ignore: cast_nullable_to_non_nullable
              as String?,
      panelUrl: freezed == panelUrl
          ? _value.panelUrl
          : panelUrl // ignore: cast_nullable_to_non_nullable
              as String?,
      vncUrl: freezed == vncUrl
          ? _value.vncUrl
          : vncUrl // ignore: cast_nullable_to_non_nullable
              as String?,
      metrics: freezed == metrics
          ? _value.metrics
          : metrics // ignore: cast_nullable_to_non_nullable
              as Map<String, dynamic>?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$VpsInstanceImplCopyWith<$Res>
    implements $VpsInstanceCopyWith<$Res> {
  factory _$$VpsInstanceImplCopyWith(
          _$VpsInstanceImpl value, $Res Function(_$VpsInstanceImpl) then) =
      __$$VpsInstanceImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {int? id,
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
      Map<String, dynamic>? metrics});
}

/// @nodoc
class __$$VpsInstanceImplCopyWithImpl<$Res>
    extends _$VpsInstanceCopyWithImpl<$Res, _$VpsInstanceImpl>
    implements _$$VpsInstanceImplCopyWith<$Res> {
  __$$VpsInstanceImplCopyWithImpl(
      _$VpsInstanceImpl _value, $Res Function(_$VpsInstanceImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? name = freezed,
    Object? status = freezed,
    Object? ip = freezed,
    Object? ipv6 = freezed,
    Object? regionName = freezed,
    Object? regionLine = freezed,
    Object? packageName = freezed,
    Object? specText = freezed,
    Object? cpuCores = freezed,
    Object? memoryMb = freezed,
    Object? diskGb = freezed,
    Object? bandwidthMb = freezed,
    Object? expireAt = freezed,
    Object? createdAt = freezed,
    Object? osName = freezed,
    Object? osType = freezed,
    Object? panelUrl = freezed,
    Object? vncUrl = freezed,
    Object? metrics = freezed,
  }) {
    return _then(_$VpsInstanceImpl(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      name: freezed == name
          ? _value.name
          : name // ignore: cast_nullable_to_non_nullable
              as String?,
      status: freezed == status
          ? _value.status
          : status // ignore: cast_nullable_to_non_nullable
              as String?,
      ip: freezed == ip
          ? _value.ip
          : ip // ignore: cast_nullable_to_non_nullable
              as String?,
      ipv6: freezed == ipv6
          ? _value.ipv6
          : ipv6 // ignore: cast_nullable_to_non_nullable
              as String?,
      regionName: freezed == regionName
          ? _value.regionName
          : regionName // ignore: cast_nullable_to_non_nullable
              as String?,
      regionLine: freezed == regionLine
          ? _value.regionLine
          : regionLine // ignore: cast_nullable_to_non_nullable
              as String?,
      packageName: freezed == packageName
          ? _value.packageName
          : packageName // ignore: cast_nullable_to_non_nullable
              as String?,
      specText: freezed == specText
          ? _value.specText
          : specText // ignore: cast_nullable_to_non_nullable
              as String?,
      cpuCores: freezed == cpuCores
          ? _value.cpuCores
          : cpuCores // ignore: cast_nullable_to_non_nullable
              as int?,
      memoryMb: freezed == memoryMb
          ? _value.memoryMb
          : memoryMb // ignore: cast_nullable_to_non_nullable
              as int?,
      diskGb: freezed == diskGb
          ? _value.diskGb
          : diskGb // ignore: cast_nullable_to_non_nullable
              as int?,
      bandwidthMb: freezed == bandwidthMb
          ? _value.bandwidthMb
          : bandwidthMb // ignore: cast_nullable_to_non_nullable
              as int?,
      expireAt: freezed == expireAt
          ? _value.expireAt
          : expireAt // ignore: cast_nullable_to_non_nullable
              as String?,
      createdAt: freezed == createdAt
          ? _value.createdAt
          : createdAt // ignore: cast_nullable_to_non_nullable
              as String?,
      osName: freezed == osName
          ? _value.osName
          : osName // ignore: cast_nullable_to_non_nullable
              as String?,
      osType: freezed == osType
          ? _value.osType
          : osType // ignore: cast_nullable_to_non_nullable
              as String?,
      panelUrl: freezed == panelUrl
          ? _value.panelUrl
          : panelUrl // ignore: cast_nullable_to_non_nullable
              as String?,
      vncUrl: freezed == vncUrl
          ? _value.vncUrl
          : vncUrl // ignore: cast_nullable_to_non_nullable
              as String?,
      metrics: freezed == metrics
          ? _value._metrics
          : metrics // ignore: cast_nullable_to_non_nullable
              as Map<String, dynamic>?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$VpsInstanceImpl implements _VpsInstance {
  const _$VpsInstanceImpl(
      {this.id,
      this.name,
      this.status,
      this.ip,
      @JsonKey(name: 'ipv6') this.ipv6,
      @JsonKey(name: 'region_name') this.regionName,
      @JsonKey(name: 'region_line') this.regionLine,
      @JsonKey(name: 'package_name') this.packageName,
      @JsonKey(name: 'spec_text') this.specText,
      @JsonKey(name: 'cpu_cores') this.cpuCores,
      @JsonKey(name: 'memory_mb') this.memoryMb,
      @JsonKey(name: 'disk_gb') this.diskGb,
      @JsonKey(name: 'bandwidth_mb') this.bandwidthMb,
      @JsonKey(name: 'expire_at') this.expireAt,
      @JsonKey(name: 'created_at') this.createdAt,
      @JsonKey(name: 'os_name') this.osName,
      @JsonKey(name: 'os_type') this.osType,
      @JsonKey(name: 'panel_url') this.panelUrl,
      @JsonKey(name: 'vnc_url') this.vncUrl,
      final Map<String, dynamic>? metrics})
      : _metrics = metrics;

  factory _$VpsInstanceImpl.fromJson(Map<String, dynamic> json) =>
      _$$VpsInstanceImplFromJson(json);

  @override
  final int? id;
  @override
  final String? name;
  @override
  final String? status;
  @override
  final String? ip;
  @override
  @JsonKey(name: 'ipv6')
  final String? ipv6;
  @override
  @JsonKey(name: 'region_name')
  final String? regionName;
  @override
  @JsonKey(name: 'region_line')
  final String? regionLine;
  @override
  @JsonKey(name: 'package_name')
  final String? packageName;
  @override
  @JsonKey(name: 'spec_text')
  final String? specText;
  @override
  @JsonKey(name: 'cpu_cores')
  final int? cpuCores;
  @override
  @JsonKey(name: 'memory_mb')
  final int? memoryMb;
  @override
  @JsonKey(name: 'disk_gb')
  final int? diskGb;
  @override
  @JsonKey(name: 'bandwidth_mb')
  final int? bandwidthMb;
  @override
  @JsonKey(name: 'expire_at')
  final String? expireAt;
  @override
  @JsonKey(name: 'created_at')
  final String? createdAt;
  @override
  @JsonKey(name: 'os_name')
  final String? osName;
  @override
  @JsonKey(name: 'os_type')
  final String? osType;
  @override
  @JsonKey(name: 'panel_url')
  final String? panelUrl;
  @override
  @JsonKey(name: 'vnc_url')
  final String? vncUrl;
  final Map<String, dynamic>? _metrics;
  @override
  Map<String, dynamic>? get metrics {
    final value = _metrics;
    if (value == null) return null;
    if (_metrics is EqualUnmodifiableMapView) return _metrics;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableMapView(value);
  }

  @override
  String toString() {
    return 'VpsInstance(id: $id, name: $name, status: $status, ip: $ip, ipv6: $ipv6, regionName: $regionName, regionLine: $regionLine, packageName: $packageName, specText: $specText, cpuCores: $cpuCores, memoryMb: $memoryMb, diskGb: $diskGb, bandwidthMb: $bandwidthMb, expireAt: $expireAt, createdAt: $createdAt, osName: $osName, osType: $osType, panelUrl: $panelUrl, vncUrl: $vncUrl, metrics: $metrics)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$VpsInstanceImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.status, status) || other.status == status) &&
            (identical(other.ip, ip) || other.ip == ip) &&
            (identical(other.ipv6, ipv6) || other.ipv6 == ipv6) &&
            (identical(other.regionName, regionName) ||
                other.regionName == regionName) &&
            (identical(other.regionLine, regionLine) ||
                other.regionLine == regionLine) &&
            (identical(other.packageName, packageName) ||
                other.packageName == packageName) &&
            (identical(other.specText, specText) ||
                other.specText == specText) &&
            (identical(other.cpuCores, cpuCores) ||
                other.cpuCores == cpuCores) &&
            (identical(other.memoryMb, memoryMb) ||
                other.memoryMb == memoryMb) &&
            (identical(other.diskGb, diskGb) || other.diskGb == diskGb) &&
            (identical(other.bandwidthMb, bandwidthMb) ||
                other.bandwidthMb == bandwidthMb) &&
            (identical(other.expireAt, expireAt) ||
                other.expireAt == expireAt) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt) &&
            (identical(other.osName, osName) || other.osName == osName) &&
            (identical(other.osType, osType) || other.osType == osType) &&
            (identical(other.panelUrl, panelUrl) ||
                other.panelUrl == panelUrl) &&
            (identical(other.vncUrl, vncUrl) || other.vncUrl == vncUrl) &&
            const DeepCollectionEquality().equals(other._metrics, _metrics));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode => Object.hashAll([
        runtimeType,
        id,
        name,
        status,
        ip,
        ipv6,
        regionName,
        regionLine,
        packageName,
        specText,
        cpuCores,
        memoryMb,
        diskGb,
        bandwidthMb,
        expireAt,
        createdAt,
        osName,
        osType,
        panelUrl,
        vncUrl,
        const DeepCollectionEquality().hash(_metrics)
      ]);

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$VpsInstanceImplCopyWith<_$VpsInstanceImpl> get copyWith =>
      __$$VpsInstanceImplCopyWithImpl<_$VpsInstanceImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$VpsInstanceImplToJson(
      this,
    );
  }
}

abstract class _VpsInstance implements VpsInstance {
  const factory _VpsInstance(
      {final int? id,
      final String? name,
      final String? status,
      final String? ip,
      @JsonKey(name: 'ipv6') final String? ipv6,
      @JsonKey(name: 'region_name') final String? regionName,
      @JsonKey(name: 'region_line') final String? regionLine,
      @JsonKey(name: 'package_name') final String? packageName,
      @JsonKey(name: 'spec_text') final String? specText,
      @JsonKey(name: 'cpu_cores') final int? cpuCores,
      @JsonKey(name: 'memory_mb') final int? memoryMb,
      @JsonKey(name: 'disk_gb') final int? diskGb,
      @JsonKey(name: 'bandwidth_mb') final int? bandwidthMb,
      @JsonKey(name: 'expire_at') final String? expireAt,
      @JsonKey(name: 'created_at') final String? createdAt,
      @JsonKey(name: 'os_name') final String? osName,
      @JsonKey(name: 'os_type') final String? osType,
      @JsonKey(name: 'panel_url') final String? panelUrl,
      @JsonKey(name: 'vnc_url') final String? vncUrl,
      final Map<String, dynamic>? metrics}) = _$VpsInstanceImpl;

  factory _VpsInstance.fromJson(Map<String, dynamic> json) =
      _$VpsInstanceImpl.fromJson;

  @override
  int? get id;
  @override
  String? get name;
  @override
  String? get status;
  @override
  String? get ip;
  @override
  @JsonKey(name: 'ipv6')
  String? get ipv6;
  @override
  @JsonKey(name: 'region_name')
  String? get regionName;
  @override
  @JsonKey(name: 'region_line')
  String? get regionLine;
  @override
  @JsonKey(name: 'package_name')
  String? get packageName;
  @override
  @JsonKey(name: 'spec_text')
  String? get specText;
  @override
  @JsonKey(name: 'cpu_cores')
  int? get cpuCores;
  @override
  @JsonKey(name: 'memory_mb')
  int? get memoryMb;
  @override
  @JsonKey(name: 'disk_gb')
  int? get diskGb;
  @override
  @JsonKey(name: 'bandwidth_mb')
  int? get bandwidthMb;
  @override
  @JsonKey(name: 'expire_at')
  String? get expireAt;
  @override
  @JsonKey(name: 'created_at')
  String? get createdAt;
  @override
  @JsonKey(name: 'os_name')
  String? get osName;
  @override
  @JsonKey(name: 'os_type')
  String? get osType;
  @override
  @JsonKey(name: 'panel_url')
  String? get panelUrl;
  @override
  @JsonKey(name: 'vnc_url')
  String? get vncUrl;
  @override
  Map<String, dynamic>? get metrics;
  @override
  @JsonKey(ignore: true)
  _$$VpsInstanceImplCopyWith<_$VpsInstanceImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

VpsMetrics _$VpsMetricsFromJson(Map<String, dynamic> json) {
  return _VpsMetrics.fromJson(json);
}

/// @nodoc
mixin _$VpsMetrics {
  @JsonKey(name: 'cpu_usage')
  double? get cpuUsage => throw _privateConstructorUsedError;
  @JsonKey(name: 'memory_usage')
  double? get memoryUsage => throw _privateConstructorUsedError;
  @JsonKey(name: 'disk_usage')
  double? get diskUsage => throw _privateConstructorUsedError;
  @JsonKey(name: 'network_in')
  int? get networkIn => throw _privateConstructorUsedError;
  @JsonKey(name: 'network_out')
  int? get networkOut => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $VpsMetricsCopyWith<VpsMetrics> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $VpsMetricsCopyWith<$Res> {
  factory $VpsMetricsCopyWith(
          VpsMetrics value, $Res Function(VpsMetrics) then) =
      _$VpsMetricsCopyWithImpl<$Res, VpsMetrics>;
  @useResult
  $Res call(
      {@JsonKey(name: 'cpu_usage') double? cpuUsage,
      @JsonKey(name: 'memory_usage') double? memoryUsage,
      @JsonKey(name: 'disk_usage') double? diskUsage,
      @JsonKey(name: 'network_in') int? networkIn,
      @JsonKey(name: 'network_out') int? networkOut});
}

/// @nodoc
class _$VpsMetricsCopyWithImpl<$Res, $Val extends VpsMetrics>
    implements $VpsMetricsCopyWith<$Res> {
  _$VpsMetricsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? cpuUsage = freezed,
    Object? memoryUsage = freezed,
    Object? diskUsage = freezed,
    Object? networkIn = freezed,
    Object? networkOut = freezed,
  }) {
    return _then(_value.copyWith(
      cpuUsage: freezed == cpuUsage
          ? _value.cpuUsage
          : cpuUsage // ignore: cast_nullable_to_non_nullable
              as double?,
      memoryUsage: freezed == memoryUsage
          ? _value.memoryUsage
          : memoryUsage // ignore: cast_nullable_to_non_nullable
              as double?,
      diskUsage: freezed == diskUsage
          ? _value.diskUsage
          : diskUsage // ignore: cast_nullable_to_non_nullable
              as double?,
      networkIn: freezed == networkIn
          ? _value.networkIn
          : networkIn // ignore: cast_nullable_to_non_nullable
              as int?,
      networkOut: freezed == networkOut
          ? _value.networkOut
          : networkOut // ignore: cast_nullable_to_non_nullable
              as int?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$VpsMetricsImplCopyWith<$Res>
    implements $VpsMetricsCopyWith<$Res> {
  factory _$$VpsMetricsImplCopyWith(
          _$VpsMetricsImpl value, $Res Function(_$VpsMetricsImpl) then) =
      __$$VpsMetricsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {@JsonKey(name: 'cpu_usage') double? cpuUsage,
      @JsonKey(name: 'memory_usage') double? memoryUsage,
      @JsonKey(name: 'disk_usage') double? diskUsage,
      @JsonKey(name: 'network_in') int? networkIn,
      @JsonKey(name: 'network_out') int? networkOut});
}

/// @nodoc
class __$$VpsMetricsImplCopyWithImpl<$Res>
    extends _$VpsMetricsCopyWithImpl<$Res, _$VpsMetricsImpl>
    implements _$$VpsMetricsImplCopyWith<$Res> {
  __$$VpsMetricsImplCopyWithImpl(
      _$VpsMetricsImpl _value, $Res Function(_$VpsMetricsImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? cpuUsage = freezed,
    Object? memoryUsage = freezed,
    Object? diskUsage = freezed,
    Object? networkIn = freezed,
    Object? networkOut = freezed,
  }) {
    return _then(_$VpsMetricsImpl(
      cpuUsage: freezed == cpuUsage
          ? _value.cpuUsage
          : cpuUsage // ignore: cast_nullable_to_non_nullable
              as double?,
      memoryUsage: freezed == memoryUsage
          ? _value.memoryUsage
          : memoryUsage // ignore: cast_nullable_to_non_nullable
              as double?,
      diskUsage: freezed == diskUsage
          ? _value.diskUsage
          : diskUsage // ignore: cast_nullable_to_non_nullable
              as double?,
      networkIn: freezed == networkIn
          ? _value.networkIn
          : networkIn // ignore: cast_nullable_to_non_nullable
              as int?,
      networkOut: freezed == networkOut
          ? _value.networkOut
          : networkOut // ignore: cast_nullable_to_non_nullable
              as int?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$VpsMetricsImpl implements _VpsMetrics {
  const _$VpsMetricsImpl(
      {@JsonKey(name: 'cpu_usage') this.cpuUsage,
      @JsonKey(name: 'memory_usage') this.memoryUsage,
      @JsonKey(name: 'disk_usage') this.diskUsage,
      @JsonKey(name: 'network_in') this.networkIn,
      @JsonKey(name: 'network_out') this.networkOut});

  factory _$VpsMetricsImpl.fromJson(Map<String, dynamic> json) =>
      _$$VpsMetricsImplFromJson(json);

  @override
  @JsonKey(name: 'cpu_usage')
  final double? cpuUsage;
  @override
  @JsonKey(name: 'memory_usage')
  final double? memoryUsage;
  @override
  @JsonKey(name: 'disk_usage')
  final double? diskUsage;
  @override
  @JsonKey(name: 'network_in')
  final int? networkIn;
  @override
  @JsonKey(name: 'network_out')
  final int? networkOut;

  @override
  String toString() {
    return 'VpsMetrics(cpuUsage: $cpuUsage, memoryUsage: $memoryUsage, diskUsage: $diskUsage, networkIn: $networkIn, networkOut: $networkOut)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$VpsMetricsImpl &&
            (identical(other.cpuUsage, cpuUsage) ||
                other.cpuUsage == cpuUsage) &&
            (identical(other.memoryUsage, memoryUsage) ||
                other.memoryUsage == memoryUsage) &&
            (identical(other.diskUsage, diskUsage) ||
                other.diskUsage == diskUsage) &&
            (identical(other.networkIn, networkIn) ||
                other.networkIn == networkIn) &&
            (identical(other.networkOut, networkOut) ||
                other.networkOut == networkOut));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode => Object.hash(
      runtimeType, cpuUsage, memoryUsage, diskUsage, networkIn, networkOut);

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$VpsMetricsImplCopyWith<_$VpsMetricsImpl> get copyWith =>
      __$$VpsMetricsImplCopyWithImpl<_$VpsMetricsImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$VpsMetricsImplToJson(
      this,
    );
  }
}

abstract class _VpsMetrics implements VpsMetrics {
  const factory _VpsMetrics(
      {@JsonKey(name: 'cpu_usage') final double? cpuUsage,
      @JsonKey(name: 'memory_usage') final double? memoryUsage,
      @JsonKey(name: 'disk_usage') final double? diskUsage,
      @JsonKey(name: 'network_in') final int? networkIn,
      @JsonKey(name: 'network_out') final int? networkOut}) = _$VpsMetricsImpl;

  factory _VpsMetrics.fromJson(Map<String, dynamic> json) =
      _$VpsMetricsImpl.fromJson;

  @override
  @JsonKey(name: 'cpu_usage')
  double? get cpuUsage;
  @override
  @JsonKey(name: 'memory_usage')
  double? get memoryUsage;
  @override
  @JsonKey(name: 'disk_usage')
  double? get diskUsage;
  @override
  @JsonKey(name: 'network_in')
  int? get networkIn;
  @override
  @JsonKey(name: 'network_out')
  int? get networkOut;
  @override
  @JsonKey(ignore: true)
  _$$VpsMetricsImplCopyWith<_$VpsMetricsImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

Catalog _$CatalogFromJson(Map<String, dynamic> json) {
  return _Catalog.fromJson(json);
}

/// @nodoc
mixin _$Catalog {
  List<Region>? get regions => throw _privateConstructorUsedError;
  List<PackageModel>? get packages => throw _privateConstructorUsedError;
  @JsonKey(name: 'system_images')
  List<SystemImage>? get systemImages => throw _privateConstructorUsedError;
  @JsonKey(name: 'billing_cycles')
  List<BillingCycle>? get billingCycles => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $CatalogCopyWith<Catalog> get copyWith => throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CatalogCopyWith<$Res> {
  factory $CatalogCopyWith(Catalog value, $Res Function(Catalog) then) =
      _$CatalogCopyWithImpl<$Res, Catalog>;
  @useResult
  $Res call(
      {List<Region>? regions,
      List<PackageModel>? packages,
      @JsonKey(name: 'system_images') List<SystemImage>? systemImages,
      @JsonKey(name: 'billing_cycles') List<BillingCycle>? billingCycles});
}

/// @nodoc
class _$CatalogCopyWithImpl<$Res, $Val extends Catalog>
    implements $CatalogCopyWith<$Res> {
  _$CatalogCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? regions = freezed,
    Object? packages = freezed,
    Object? systemImages = freezed,
    Object? billingCycles = freezed,
  }) {
    return _then(_value.copyWith(
      regions: freezed == regions
          ? _value.regions
          : regions // ignore: cast_nullable_to_non_nullable
              as List<Region>?,
      packages: freezed == packages
          ? _value.packages
          : packages // ignore: cast_nullable_to_non_nullable
              as List<PackageModel>?,
      systemImages: freezed == systemImages
          ? _value.systemImages
          : systemImages // ignore: cast_nullable_to_non_nullable
              as List<SystemImage>?,
      billingCycles: freezed == billingCycles
          ? _value.billingCycles
          : billingCycles // ignore: cast_nullable_to_non_nullable
              as List<BillingCycle>?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$CatalogImplCopyWith<$Res> implements $CatalogCopyWith<$Res> {
  factory _$$CatalogImplCopyWith(
          _$CatalogImpl value, $Res Function(_$CatalogImpl) then) =
      __$$CatalogImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {List<Region>? regions,
      List<PackageModel>? packages,
      @JsonKey(name: 'system_images') List<SystemImage>? systemImages,
      @JsonKey(name: 'billing_cycles') List<BillingCycle>? billingCycles});
}

/// @nodoc
class __$$CatalogImplCopyWithImpl<$Res>
    extends _$CatalogCopyWithImpl<$Res, _$CatalogImpl>
    implements _$$CatalogImplCopyWith<$Res> {
  __$$CatalogImplCopyWithImpl(
      _$CatalogImpl _value, $Res Function(_$CatalogImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? regions = freezed,
    Object? packages = freezed,
    Object? systemImages = freezed,
    Object? billingCycles = freezed,
  }) {
    return _then(_$CatalogImpl(
      regions: freezed == regions
          ? _value._regions
          : regions // ignore: cast_nullable_to_non_nullable
              as List<Region>?,
      packages: freezed == packages
          ? _value._packages
          : packages // ignore: cast_nullable_to_non_nullable
              as List<PackageModel>?,
      systemImages: freezed == systemImages
          ? _value._systemImages
          : systemImages // ignore: cast_nullable_to_non_nullable
              as List<SystemImage>?,
      billingCycles: freezed == billingCycles
          ? _value._billingCycles
          : billingCycles // ignore: cast_nullable_to_non_nullable
              as List<BillingCycle>?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$CatalogImpl implements _Catalog {
  const _$CatalogImpl(
      {final List<Region>? regions,
      final List<PackageModel>? packages,
      @JsonKey(name: 'system_images') final List<SystemImage>? systemImages,
      @JsonKey(name: 'billing_cycles') final List<BillingCycle>? billingCycles})
      : _regions = regions,
        _packages = packages,
        _systemImages = systemImages,
        _billingCycles = billingCycles;

  factory _$CatalogImpl.fromJson(Map<String, dynamic> json) =>
      _$$CatalogImplFromJson(json);

  final List<Region>? _regions;
  @override
  List<Region>? get regions {
    final value = _regions;
    if (value == null) return null;
    if (_regions is EqualUnmodifiableListView) return _regions;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  final List<PackageModel>? _packages;
  @override
  List<PackageModel>? get packages {
    final value = _packages;
    if (value == null) return null;
    if (_packages is EqualUnmodifiableListView) return _packages;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  final List<SystemImage>? _systemImages;
  @override
  @JsonKey(name: 'system_images')
  List<SystemImage>? get systemImages {
    final value = _systemImages;
    if (value == null) return null;
    if (_systemImages is EqualUnmodifiableListView) return _systemImages;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  final List<BillingCycle>? _billingCycles;
  @override
  @JsonKey(name: 'billing_cycles')
  List<BillingCycle>? get billingCycles {
    final value = _billingCycles;
    if (value == null) return null;
    if (_billingCycles is EqualUnmodifiableListView) return _billingCycles;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  @override
  String toString() {
    return 'Catalog(regions: $regions, packages: $packages, systemImages: $systemImages, billingCycles: $billingCycles)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CatalogImpl &&
            const DeepCollectionEquality().equals(other._regions, _regions) &&
            const DeepCollectionEquality().equals(other._packages, _packages) &&
            const DeepCollectionEquality()
                .equals(other._systemImages, _systemImages) &&
            const DeepCollectionEquality()
                .equals(other._billingCycles, _billingCycles));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode => Object.hash(
      runtimeType,
      const DeepCollectionEquality().hash(_regions),
      const DeepCollectionEquality().hash(_packages),
      const DeepCollectionEquality().hash(_systemImages),
      const DeepCollectionEquality().hash(_billingCycles));

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$CatalogImplCopyWith<_$CatalogImpl> get copyWith =>
      __$$CatalogImplCopyWithImpl<_$CatalogImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$CatalogImplToJson(
      this,
    );
  }
}

abstract class _Catalog implements Catalog {
  const factory _Catalog(
      {final List<Region>? regions,
      final List<PackageModel>? packages,
      @JsonKey(name: 'system_images') final List<SystemImage>? systemImages,
      @JsonKey(name: 'billing_cycles')
      final List<BillingCycle>? billingCycles}) = _$CatalogImpl;

  factory _Catalog.fromJson(Map<String, dynamic> json) = _$CatalogImpl.fromJson;

  @override
  List<Region>? get regions;
  @override
  List<PackageModel>? get packages;
  @override
  @JsonKey(name: 'system_images')
  List<SystemImage>? get systemImages;
  @override
  @JsonKey(name: 'billing_cycles')
  List<BillingCycle>? get billingCycles;
  @override
  @JsonKey(ignore: true)
  _$$CatalogImplCopyWith<_$CatalogImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

Region _$RegionFromJson(Map<String, dynamic> json) {
  return _Region.fromJson(json);
}

/// @nodoc
mixin _$Region {
  int? get id => throw _privateConstructorUsedError;
  String? get name => throw _privateConstructorUsedError;
  List<RegionLine>? get lines => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $RegionCopyWith<Region> get copyWith => throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $RegionCopyWith<$Res> {
  factory $RegionCopyWith(Region value, $Res Function(Region) then) =
      _$RegionCopyWithImpl<$Res, Region>;
  @useResult
  $Res call({int? id, String? name, List<RegionLine>? lines});
}

/// @nodoc
class _$RegionCopyWithImpl<$Res, $Val extends Region>
    implements $RegionCopyWith<$Res> {
  _$RegionCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? name = freezed,
    Object? lines = freezed,
  }) {
    return _then(_value.copyWith(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      name: freezed == name
          ? _value.name
          : name // ignore: cast_nullable_to_non_nullable
              as String?,
      lines: freezed == lines
          ? _value.lines
          : lines // ignore: cast_nullable_to_non_nullable
              as List<RegionLine>?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$RegionImplCopyWith<$Res> implements $RegionCopyWith<$Res> {
  factory _$$RegionImplCopyWith(
          _$RegionImpl value, $Res Function(_$RegionImpl) then) =
      __$$RegionImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({int? id, String? name, List<RegionLine>? lines});
}

/// @nodoc
class __$$RegionImplCopyWithImpl<$Res>
    extends _$RegionCopyWithImpl<$Res, _$RegionImpl>
    implements _$$RegionImplCopyWith<$Res> {
  __$$RegionImplCopyWithImpl(
      _$RegionImpl _value, $Res Function(_$RegionImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? name = freezed,
    Object? lines = freezed,
  }) {
    return _then(_$RegionImpl(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      name: freezed == name
          ? _value.name
          : name // ignore: cast_nullable_to_non_nullable
              as String?,
      lines: freezed == lines
          ? _value._lines
          : lines // ignore: cast_nullable_to_non_nullable
              as List<RegionLine>?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$RegionImpl implements _Region {
  const _$RegionImpl({this.id, this.name, final List<RegionLine>? lines})
      : _lines = lines;

  factory _$RegionImpl.fromJson(Map<String, dynamic> json) =>
      _$$RegionImplFromJson(json);

  @override
  final int? id;
  @override
  final String? name;
  final List<RegionLine>? _lines;
  @override
  List<RegionLine>? get lines {
    final value = _lines;
    if (value == null) return null;
    if (_lines is EqualUnmodifiableListView) return _lines;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  @override
  String toString() {
    return 'Region(id: $id, name: $name, lines: $lines)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$RegionImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.name, name) || other.name == name) &&
            const DeepCollectionEquality().equals(other._lines, _lines));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode => Object.hash(
      runtimeType, id, name, const DeepCollectionEquality().hash(_lines));

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$RegionImplCopyWith<_$RegionImpl> get copyWith =>
      __$$RegionImplCopyWithImpl<_$RegionImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$RegionImplToJson(
      this,
    );
  }
}

abstract class _Region implements Region {
  const factory _Region(
      {final int? id,
      final String? name,
      final List<RegionLine>? lines}) = _$RegionImpl;

  factory _Region.fromJson(Map<String, dynamic> json) = _$RegionImpl.fromJson;

  @override
  int? get id;
  @override
  String? get name;
  @override
  List<RegionLine>? get lines;
  @override
  @JsonKey(ignore: true)
  _$$RegionImplCopyWith<_$RegionImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

RegionLine _$RegionLineFromJson(Map<String, dynamic> json) {
  return _RegionLine.fromJson(json);
}

/// @nodoc
mixin _$RegionLine {
  int? get id => throw _privateConstructorUsedError;
  String? get name => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $RegionLineCopyWith<RegionLine> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $RegionLineCopyWith<$Res> {
  factory $RegionLineCopyWith(
          RegionLine value, $Res Function(RegionLine) then) =
      _$RegionLineCopyWithImpl<$Res, RegionLine>;
  @useResult
  $Res call({int? id, String? name});
}

/// @nodoc
class _$RegionLineCopyWithImpl<$Res, $Val extends RegionLine>
    implements $RegionLineCopyWith<$Res> {
  _$RegionLineCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? name = freezed,
  }) {
    return _then(_value.copyWith(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      name: freezed == name
          ? _value.name
          : name // ignore: cast_nullable_to_non_nullable
              as String?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$RegionLineImplCopyWith<$Res>
    implements $RegionLineCopyWith<$Res> {
  factory _$$RegionLineImplCopyWith(
          _$RegionLineImpl value, $Res Function(_$RegionLineImpl) then) =
      __$$RegionLineImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({int? id, String? name});
}

/// @nodoc
class __$$RegionLineImplCopyWithImpl<$Res>
    extends _$RegionLineCopyWithImpl<$Res, _$RegionLineImpl>
    implements _$$RegionLineImplCopyWith<$Res> {
  __$$RegionLineImplCopyWithImpl(
      _$RegionLineImpl _value, $Res Function(_$RegionLineImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? name = freezed,
  }) {
    return _then(_$RegionLineImpl(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      name: freezed == name
          ? _value.name
          : name // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$RegionLineImpl implements _RegionLine {
  const _$RegionLineImpl({this.id, this.name});

  factory _$RegionLineImpl.fromJson(Map<String, dynamic> json) =>
      _$$RegionLineImplFromJson(json);

  @override
  final int? id;
  @override
  final String? name;

  @override
  String toString() {
    return 'RegionLine(id: $id, name: $name)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$RegionLineImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.name, name) || other.name == name));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode => Object.hash(runtimeType, id, name);

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$RegionLineImplCopyWith<_$RegionLineImpl> get copyWith =>
      __$$RegionLineImplCopyWithImpl<_$RegionLineImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$RegionLineImplToJson(
      this,
    );
  }
}

abstract class _RegionLine implements RegionLine {
  const factory _RegionLine({final int? id, final String? name}) =
      _$RegionLineImpl;

  factory _RegionLine.fromJson(Map<String, dynamic> json) =
      _$RegionLineImpl.fromJson;

  @override
  int? get id;
  @override
  String? get name;
  @override
  @JsonKey(ignore: true)
  _$$RegionLineImplCopyWith<_$RegionLineImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

PackageModel _$PackageModelFromJson(Map<String, dynamic> json) {
  return _PackageModel.fromJson(json);
}

/// @nodoc
mixin _$PackageModel {
  int? get id => throw _privateConstructorUsedError;
  String? get name => throw _privateConstructorUsedError;
  @JsonKey(name: 'cpu_cores')
  int? get cpuCores => throw _privateConstructorUsedError;
  @JsonKey(name: 'memory_mb')
  int? get memoryMb => throw _privateConstructorUsedError;
  @JsonKey(name: 'disk_gb')
  int? get diskGb => throw _privateConstructorUsedError;
  @JsonKey(name: 'bandwidth_mb')
  int? get bandwidthMb => throw _privateConstructorUsedError;
  @JsonKey(name: 'price_monthly')
  double? get priceMonthly => throw _privateConstructorUsedError;
  @JsonKey(name: 'price_quarterly')
  double? get priceQuarterly => throw _privateConstructorUsedError;
  @JsonKey(name: 'price_half_yearly')
  double? get priceHalfYearly => throw _privateConstructorUsedError;
  @JsonKey(name: 'price_yearly')
  double? get priceYearly => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $PackageModelCopyWith<PackageModel> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $PackageModelCopyWith<$Res> {
  factory $PackageModelCopyWith(
          PackageModel value, $Res Function(PackageModel) then) =
      _$PackageModelCopyWithImpl<$Res, PackageModel>;
  @useResult
  $Res call(
      {int? id,
      String? name,
      @JsonKey(name: 'cpu_cores') int? cpuCores,
      @JsonKey(name: 'memory_mb') int? memoryMb,
      @JsonKey(name: 'disk_gb') int? diskGb,
      @JsonKey(name: 'bandwidth_mb') int? bandwidthMb,
      @JsonKey(name: 'price_monthly') double? priceMonthly,
      @JsonKey(name: 'price_quarterly') double? priceQuarterly,
      @JsonKey(name: 'price_half_yearly') double? priceHalfYearly,
      @JsonKey(name: 'price_yearly') double? priceYearly});
}

/// @nodoc
class _$PackageModelCopyWithImpl<$Res, $Val extends PackageModel>
    implements $PackageModelCopyWith<$Res> {
  _$PackageModelCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? name = freezed,
    Object? cpuCores = freezed,
    Object? memoryMb = freezed,
    Object? diskGb = freezed,
    Object? bandwidthMb = freezed,
    Object? priceMonthly = freezed,
    Object? priceQuarterly = freezed,
    Object? priceHalfYearly = freezed,
    Object? priceYearly = freezed,
  }) {
    return _then(_value.copyWith(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      name: freezed == name
          ? _value.name
          : name // ignore: cast_nullable_to_non_nullable
              as String?,
      cpuCores: freezed == cpuCores
          ? _value.cpuCores
          : cpuCores // ignore: cast_nullable_to_non_nullable
              as int?,
      memoryMb: freezed == memoryMb
          ? _value.memoryMb
          : memoryMb // ignore: cast_nullable_to_non_nullable
              as int?,
      diskGb: freezed == diskGb
          ? _value.diskGb
          : diskGb // ignore: cast_nullable_to_non_nullable
              as int?,
      bandwidthMb: freezed == bandwidthMb
          ? _value.bandwidthMb
          : bandwidthMb // ignore: cast_nullable_to_non_nullable
              as int?,
      priceMonthly: freezed == priceMonthly
          ? _value.priceMonthly
          : priceMonthly // ignore: cast_nullable_to_non_nullable
              as double?,
      priceQuarterly: freezed == priceQuarterly
          ? _value.priceQuarterly
          : priceQuarterly // ignore: cast_nullable_to_non_nullable
              as double?,
      priceHalfYearly: freezed == priceHalfYearly
          ? _value.priceHalfYearly
          : priceHalfYearly // ignore: cast_nullable_to_non_nullable
              as double?,
      priceYearly: freezed == priceYearly
          ? _value.priceYearly
          : priceYearly // ignore: cast_nullable_to_non_nullable
              as double?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$PackageModelImplCopyWith<$Res>
    implements $PackageModelCopyWith<$Res> {
  factory _$$PackageModelImplCopyWith(
          _$PackageModelImpl value, $Res Function(_$PackageModelImpl) then) =
      __$$PackageModelImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {int? id,
      String? name,
      @JsonKey(name: 'cpu_cores') int? cpuCores,
      @JsonKey(name: 'memory_mb') int? memoryMb,
      @JsonKey(name: 'disk_gb') int? diskGb,
      @JsonKey(name: 'bandwidth_mb') int? bandwidthMb,
      @JsonKey(name: 'price_monthly') double? priceMonthly,
      @JsonKey(name: 'price_quarterly') double? priceQuarterly,
      @JsonKey(name: 'price_half_yearly') double? priceHalfYearly,
      @JsonKey(name: 'price_yearly') double? priceYearly});
}

/// @nodoc
class __$$PackageModelImplCopyWithImpl<$Res>
    extends _$PackageModelCopyWithImpl<$Res, _$PackageModelImpl>
    implements _$$PackageModelImplCopyWith<$Res> {
  __$$PackageModelImplCopyWithImpl(
      _$PackageModelImpl _value, $Res Function(_$PackageModelImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? name = freezed,
    Object? cpuCores = freezed,
    Object? memoryMb = freezed,
    Object? diskGb = freezed,
    Object? bandwidthMb = freezed,
    Object? priceMonthly = freezed,
    Object? priceQuarterly = freezed,
    Object? priceHalfYearly = freezed,
    Object? priceYearly = freezed,
  }) {
    return _then(_$PackageModelImpl(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      name: freezed == name
          ? _value.name
          : name // ignore: cast_nullable_to_non_nullable
              as String?,
      cpuCores: freezed == cpuCores
          ? _value.cpuCores
          : cpuCores // ignore: cast_nullable_to_non_nullable
              as int?,
      memoryMb: freezed == memoryMb
          ? _value.memoryMb
          : memoryMb // ignore: cast_nullable_to_non_nullable
              as int?,
      diskGb: freezed == diskGb
          ? _value.diskGb
          : diskGb // ignore: cast_nullable_to_non_nullable
              as int?,
      bandwidthMb: freezed == bandwidthMb
          ? _value.bandwidthMb
          : bandwidthMb // ignore: cast_nullable_to_non_nullable
              as int?,
      priceMonthly: freezed == priceMonthly
          ? _value.priceMonthly
          : priceMonthly // ignore: cast_nullable_to_non_nullable
              as double?,
      priceQuarterly: freezed == priceQuarterly
          ? _value.priceQuarterly
          : priceQuarterly // ignore: cast_nullable_to_non_nullable
              as double?,
      priceHalfYearly: freezed == priceHalfYearly
          ? _value.priceHalfYearly
          : priceHalfYearly // ignore: cast_nullable_to_non_nullable
              as double?,
      priceYearly: freezed == priceYearly
          ? _value.priceYearly
          : priceYearly // ignore: cast_nullable_to_non_nullable
              as double?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$PackageModelImpl implements _PackageModel {
  const _$PackageModelImpl(
      {this.id,
      this.name,
      @JsonKey(name: 'cpu_cores') this.cpuCores,
      @JsonKey(name: 'memory_mb') this.memoryMb,
      @JsonKey(name: 'disk_gb') this.diskGb,
      @JsonKey(name: 'bandwidth_mb') this.bandwidthMb,
      @JsonKey(name: 'price_monthly') this.priceMonthly,
      @JsonKey(name: 'price_quarterly') this.priceQuarterly,
      @JsonKey(name: 'price_half_yearly') this.priceHalfYearly,
      @JsonKey(name: 'price_yearly') this.priceYearly});

  factory _$PackageModelImpl.fromJson(Map<String, dynamic> json) =>
      _$$PackageModelImplFromJson(json);

  @override
  final int? id;
  @override
  final String? name;
  @override
  @JsonKey(name: 'cpu_cores')
  final int? cpuCores;
  @override
  @JsonKey(name: 'memory_mb')
  final int? memoryMb;
  @override
  @JsonKey(name: 'disk_gb')
  final int? diskGb;
  @override
  @JsonKey(name: 'bandwidth_mb')
  final int? bandwidthMb;
  @override
  @JsonKey(name: 'price_monthly')
  final double? priceMonthly;
  @override
  @JsonKey(name: 'price_quarterly')
  final double? priceQuarterly;
  @override
  @JsonKey(name: 'price_half_yearly')
  final double? priceHalfYearly;
  @override
  @JsonKey(name: 'price_yearly')
  final double? priceYearly;

  @override
  String toString() {
    return 'PackageModel(id: $id, name: $name, cpuCores: $cpuCores, memoryMb: $memoryMb, diskGb: $diskGb, bandwidthMb: $bandwidthMb, priceMonthly: $priceMonthly, priceQuarterly: $priceQuarterly, priceHalfYearly: $priceHalfYearly, priceYearly: $priceYearly)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$PackageModelImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.cpuCores, cpuCores) ||
                other.cpuCores == cpuCores) &&
            (identical(other.memoryMb, memoryMb) ||
                other.memoryMb == memoryMb) &&
            (identical(other.diskGb, diskGb) || other.diskGb == diskGb) &&
            (identical(other.bandwidthMb, bandwidthMb) ||
                other.bandwidthMb == bandwidthMb) &&
            (identical(other.priceMonthly, priceMonthly) ||
                other.priceMonthly == priceMonthly) &&
            (identical(other.priceQuarterly, priceQuarterly) ||
                other.priceQuarterly == priceQuarterly) &&
            (identical(other.priceHalfYearly, priceHalfYearly) ||
                other.priceHalfYearly == priceHalfYearly) &&
            (identical(other.priceYearly, priceYearly) ||
                other.priceYearly == priceYearly));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode => Object.hash(
      runtimeType,
      id,
      name,
      cpuCores,
      memoryMb,
      diskGb,
      bandwidthMb,
      priceMonthly,
      priceQuarterly,
      priceHalfYearly,
      priceYearly);

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$PackageModelImplCopyWith<_$PackageModelImpl> get copyWith =>
      __$$PackageModelImplCopyWithImpl<_$PackageModelImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$PackageModelImplToJson(
      this,
    );
  }
}

abstract class _PackageModel implements PackageModel {
  const factory _PackageModel(
          {final int? id,
          final String? name,
          @JsonKey(name: 'cpu_cores') final int? cpuCores,
          @JsonKey(name: 'memory_mb') final int? memoryMb,
          @JsonKey(name: 'disk_gb') final int? diskGb,
          @JsonKey(name: 'bandwidth_mb') final int? bandwidthMb,
          @JsonKey(name: 'price_monthly') final double? priceMonthly,
          @JsonKey(name: 'price_quarterly') final double? priceQuarterly,
          @JsonKey(name: 'price_half_yearly') final double? priceHalfYearly,
          @JsonKey(name: 'price_yearly') final double? priceYearly}) =
      _$PackageModelImpl;

  factory _PackageModel.fromJson(Map<String, dynamic> json) =
      _$PackageModelImpl.fromJson;

  @override
  int? get id;
  @override
  String? get name;
  @override
  @JsonKey(name: 'cpu_cores')
  int? get cpuCores;
  @override
  @JsonKey(name: 'memory_mb')
  int? get memoryMb;
  @override
  @JsonKey(name: 'disk_gb')
  int? get diskGb;
  @override
  @JsonKey(name: 'bandwidth_mb')
  int? get bandwidthMb;
  @override
  @JsonKey(name: 'price_monthly')
  double? get priceMonthly;
  @override
  @JsonKey(name: 'price_quarterly')
  double? get priceQuarterly;
  @override
  @JsonKey(name: 'price_half_yearly')
  double? get priceHalfYearly;
  @override
  @JsonKey(name: 'price_yearly')
  double? get priceYearly;
  @override
  @JsonKey(ignore: true)
  _$$PackageModelImplCopyWith<_$PackageModelImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

SystemImage _$SystemImageFromJson(Map<String, dynamic> json) {
  return _SystemImage.fromJson(json);
}

/// @nodoc
mixin _$SystemImage {
  int? get id => throw _privateConstructorUsedError;
  String? get name => throw _privateConstructorUsedError;
  String? get type => throw _privateConstructorUsedError;
  String? get version => throw _privateConstructorUsedError;
  @JsonKey(name: 'image_url')
  String? get imageUrl => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $SystemImageCopyWith<SystemImage> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $SystemImageCopyWith<$Res> {
  factory $SystemImageCopyWith(
          SystemImage value, $Res Function(SystemImage) then) =
      _$SystemImageCopyWithImpl<$Res, SystemImage>;
  @useResult
  $Res call(
      {int? id,
      String? name,
      String? type,
      String? version,
      @JsonKey(name: 'image_url') String? imageUrl});
}

/// @nodoc
class _$SystemImageCopyWithImpl<$Res, $Val extends SystemImage>
    implements $SystemImageCopyWith<$Res> {
  _$SystemImageCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? name = freezed,
    Object? type = freezed,
    Object? version = freezed,
    Object? imageUrl = freezed,
  }) {
    return _then(_value.copyWith(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      name: freezed == name
          ? _value.name
          : name // ignore: cast_nullable_to_non_nullable
              as String?,
      type: freezed == type
          ? _value.type
          : type // ignore: cast_nullable_to_non_nullable
              as String?,
      version: freezed == version
          ? _value.version
          : version // ignore: cast_nullable_to_non_nullable
              as String?,
      imageUrl: freezed == imageUrl
          ? _value.imageUrl
          : imageUrl // ignore: cast_nullable_to_non_nullable
              as String?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$SystemImageImplCopyWith<$Res>
    implements $SystemImageCopyWith<$Res> {
  factory _$$SystemImageImplCopyWith(
          _$SystemImageImpl value, $Res Function(_$SystemImageImpl) then) =
      __$$SystemImageImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {int? id,
      String? name,
      String? type,
      String? version,
      @JsonKey(name: 'image_url') String? imageUrl});
}

/// @nodoc
class __$$SystemImageImplCopyWithImpl<$Res>
    extends _$SystemImageCopyWithImpl<$Res, _$SystemImageImpl>
    implements _$$SystemImageImplCopyWith<$Res> {
  __$$SystemImageImplCopyWithImpl(
      _$SystemImageImpl _value, $Res Function(_$SystemImageImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? name = freezed,
    Object? type = freezed,
    Object? version = freezed,
    Object? imageUrl = freezed,
  }) {
    return _then(_$SystemImageImpl(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      name: freezed == name
          ? _value.name
          : name // ignore: cast_nullable_to_non_nullable
              as String?,
      type: freezed == type
          ? _value.type
          : type // ignore: cast_nullable_to_non_nullable
              as String?,
      version: freezed == version
          ? _value.version
          : version // ignore: cast_nullable_to_non_nullable
              as String?,
      imageUrl: freezed == imageUrl
          ? _value.imageUrl
          : imageUrl // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$SystemImageImpl implements _SystemImage {
  const _$SystemImageImpl(
      {this.id,
      this.name,
      this.type,
      this.version,
      @JsonKey(name: 'image_url') this.imageUrl});

  factory _$SystemImageImpl.fromJson(Map<String, dynamic> json) =>
      _$$SystemImageImplFromJson(json);

  @override
  final int? id;
  @override
  final String? name;
  @override
  final String? type;
  @override
  final String? version;
  @override
  @JsonKey(name: 'image_url')
  final String? imageUrl;

  @override
  String toString() {
    return 'SystemImage(id: $id, name: $name, type: $type, version: $version, imageUrl: $imageUrl)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$SystemImageImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.type, type) || other.type == type) &&
            (identical(other.version, version) || other.version == version) &&
            (identical(other.imageUrl, imageUrl) ||
                other.imageUrl == imageUrl));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode =>
      Object.hash(runtimeType, id, name, type, version, imageUrl);

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$SystemImageImplCopyWith<_$SystemImageImpl> get copyWith =>
      __$$SystemImageImplCopyWithImpl<_$SystemImageImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$SystemImageImplToJson(
      this,
    );
  }
}

abstract class _SystemImage implements SystemImage {
  const factory _SystemImage(
      {final int? id,
      final String? name,
      final String? type,
      final String? version,
      @JsonKey(name: 'image_url') final String? imageUrl}) = _$SystemImageImpl;

  factory _SystemImage.fromJson(Map<String, dynamic> json) =
      _$SystemImageImpl.fromJson;

  @override
  int? get id;
  @override
  String? get name;
  @override
  String? get type;
  @override
  String? get version;
  @override
  @JsonKey(name: 'image_url')
  String? get imageUrl;
  @override
  @JsonKey(ignore: true)
  _$$SystemImageImplCopyWith<_$SystemImageImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

BillingCycle _$BillingCycleFromJson(Map<String, dynamic> json) {
  return _BillingCycle.fromJson(json);
}

/// @nodoc
mixin _$BillingCycle {
  int? get id => throw _privateConstructorUsedError;
  String? get name => throw _privateConstructorUsedError;
  @JsonKey(name: 'display_name')
  String? get displayName => throw _privateConstructorUsedError;
  @JsonKey(name: 'months')
  int? get months => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $BillingCycleCopyWith<BillingCycle> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $BillingCycleCopyWith<$Res> {
  factory $BillingCycleCopyWith(
          BillingCycle value, $Res Function(BillingCycle) then) =
      _$BillingCycleCopyWithImpl<$Res, BillingCycle>;
  @useResult
  $Res call(
      {int? id,
      String? name,
      @JsonKey(name: 'display_name') String? displayName,
      @JsonKey(name: 'months') int? months});
}

/// @nodoc
class _$BillingCycleCopyWithImpl<$Res, $Val extends BillingCycle>
    implements $BillingCycleCopyWith<$Res> {
  _$BillingCycleCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? name = freezed,
    Object? displayName = freezed,
    Object? months = freezed,
  }) {
    return _then(_value.copyWith(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      name: freezed == name
          ? _value.name
          : name // ignore: cast_nullable_to_non_nullable
              as String?,
      displayName: freezed == displayName
          ? _value.displayName
          : displayName // ignore: cast_nullable_to_non_nullable
              as String?,
      months: freezed == months
          ? _value.months
          : months // ignore: cast_nullable_to_non_nullable
              as int?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$BillingCycleImplCopyWith<$Res>
    implements $BillingCycleCopyWith<$Res> {
  factory _$$BillingCycleImplCopyWith(
          _$BillingCycleImpl value, $Res Function(_$BillingCycleImpl) then) =
      __$$BillingCycleImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {int? id,
      String? name,
      @JsonKey(name: 'display_name') String? displayName,
      @JsonKey(name: 'months') int? months});
}

/// @nodoc
class __$$BillingCycleImplCopyWithImpl<$Res>
    extends _$BillingCycleCopyWithImpl<$Res, _$BillingCycleImpl>
    implements _$$BillingCycleImplCopyWith<$Res> {
  __$$BillingCycleImplCopyWithImpl(
      _$BillingCycleImpl _value, $Res Function(_$BillingCycleImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? name = freezed,
    Object? displayName = freezed,
    Object? months = freezed,
  }) {
    return _then(_$BillingCycleImpl(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      name: freezed == name
          ? _value.name
          : name // ignore: cast_nullable_to_non_nullable
              as String?,
      displayName: freezed == displayName
          ? _value.displayName
          : displayName // ignore: cast_nullable_to_non_nullable
              as String?,
      months: freezed == months
          ? _value.months
          : months // ignore: cast_nullable_to_non_nullable
              as int?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$BillingCycleImpl implements _BillingCycle {
  const _$BillingCycleImpl(
      {this.id,
      this.name,
      @JsonKey(name: 'display_name') this.displayName,
      @JsonKey(name: 'months') this.months});

  factory _$BillingCycleImpl.fromJson(Map<String, dynamic> json) =>
      _$$BillingCycleImplFromJson(json);

  @override
  final int? id;
  @override
  final String? name;
  @override
  @JsonKey(name: 'display_name')
  final String? displayName;
  @override
  @JsonKey(name: 'months')
  final int? months;

  @override
  String toString() {
    return 'BillingCycle(id: $id, name: $name, displayName: $displayName, months: $months)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$BillingCycleImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.displayName, displayName) ||
                other.displayName == displayName) &&
            (identical(other.months, months) || other.months == months));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode => Object.hash(runtimeType, id, name, displayName, months);

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$BillingCycleImplCopyWith<_$BillingCycleImpl> get copyWith =>
      __$$BillingCycleImplCopyWithImpl<_$BillingCycleImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$BillingCycleImplToJson(
      this,
    );
  }
}

abstract class _BillingCycle implements BillingCycle {
  const factory _BillingCycle(
      {final int? id,
      final String? name,
      @JsonKey(name: 'display_name') final String? displayName,
      @JsonKey(name: 'months') final int? months}) = _$BillingCycleImpl;

  factory _BillingCycle.fromJson(Map<String, dynamic> json) =
      _$BillingCycleImpl.fromJson;

  @override
  int? get id;
  @override
  String? get name;
  @override
  @JsonKey(name: 'display_name')
  String? get displayName;
  @override
  @JsonKey(name: 'months')
  int? get months;
  @override
  @JsonKey(ignore: true)
  _$$BillingCycleImplCopyWith<_$BillingCycleImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
