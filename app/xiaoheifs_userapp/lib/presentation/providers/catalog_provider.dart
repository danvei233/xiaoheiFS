import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../data/repositories/catalog_repository.dart';
import '../../core/utils/map_utils.dart';

class CatalogState {
  final bool loading;
  final List<Map<String, dynamic>> goodsTypes;
  final List<Map<String, dynamic>> regions;
  final List<Map<String, dynamic>> lines;
  final List<Map<String, dynamic>> planGroups;
  final List<Map<String, dynamic>> packages;
  final List<Map<String, dynamic>> systemImages;
  final List<Map<String, dynamic>> billingCycles;
  final String? error;

  const CatalogState({
    this.loading = false,
    this.goodsTypes = const [],
    this.regions = const [],
    this.lines = const [],
    this.planGroups = const [],
    this.packages = const [],
    this.systemImages = const [],
    this.billingCycles = const [],
    this.error,
  });

  CatalogState copyWith({
    bool? loading,
    List<Map<String, dynamic>>? goodsTypes,
    List<Map<String, dynamic>>? regions,
    List<Map<String, dynamic>>? lines,
    List<Map<String, dynamic>>? planGroups,
    List<Map<String, dynamic>>? packages,
    List<Map<String, dynamic>>? systemImages,
    List<Map<String, dynamic>>? billingCycles,
    String? error,
  }) {
    return CatalogState(
      loading: loading ?? this.loading,
      goodsTypes: goodsTypes ?? this.goodsTypes,
      regions: regions ?? this.regions,
      lines: lines ?? this.lines,
      planGroups: planGroups ?? this.planGroups,
      packages: packages ?? this.packages,
      systemImages: systemImages ?? this.systemImages,
      billingCycles: billingCycles ?? this.billingCycles,
      error: error,
    );
  }
}

final catalogRepositoryProvider = Provider<CatalogRepository>((ref) {
  return CatalogRepository();
});

final catalogProvider = StateNotifierProvider<CatalogNotifier, CatalogState>((ref) {
  return CatalogNotifier(ref.read(catalogRepositoryProvider));
});

class CatalogNotifier extends StateNotifier<CatalogState> {
  CatalogNotifier(this._repo) : super(const CatalogState());
  final CatalogRepository _repo;

  Future<void> fetchCatalog() async {
    state = state.copyWith(loading: true, error: null);
    try {
      final data = await _repo.fetchCatalog().timeout(const Duration(seconds: 20));
      state = state.copyWith(
        loading: false,
        goodsTypes: data.goodsTypes.map(_normalizeGoodsType).toList(),
        regions: data.regions.map(_normalizeRegion).toList(),
        lines: data.lines.map(_normalizeLine).toList(),
        planGroups: data.planGroups.map(_normalizePlanGroup).toList(),
        packages: data.packages.map(_normalizePackage).toList(),
        systemImages: data.systemImages.map(_normalizeSystemImage).toList(),
        billingCycles: data.billingCycles.map(_normalizeBillingCycle).toList(),
      );
    } catch (e) {
      state = state.copyWith(loading: false, error: e.toString());
    }
  }

  Map<String, dynamic> _normalizeGoodsType(Map<String, dynamic> raw) {
    return {
      'id': pick(raw, ['id', 'ID']),
      'code': pick(raw, ['code', 'Code']),
      'name': pick(raw, ['name', 'Name']),
      'active': pick(raw, ['active', 'Active']) ?? true,
      'sort_order': pick(raw, ['sort_order', 'SortOrder']),
      'automation_category': pick(raw, ['automation_category', 'AutomationCategory']),
      'automation_plugin_id': pick(raw, ['automation_plugin_id', 'AutomationPluginID']),
      'automation_instance_id': pick(raw, ['automation_instance_id', 'AutomationInstanceID']),
    };
  }

  Map<String, dynamic> _normalizeRegion(Map<String, dynamic> raw) {
    return {
      'id': pick(raw, ['id', 'ID']),
      'goods_type_id': pick(raw, ['goods_type_id', 'GoodsTypeID']),
      'name': pick(raw, ['name', 'Name']),
      'code': pick(raw, ['code', 'Code']),
      'active': pick(raw, ['active', 'Active']) ?? true,
    };
  }

  Map<String, dynamic> _normalizeLine(Map<String, dynamic> raw) {
    return {
      'id': pick(raw, ['id', 'ID']),
      'goods_type_id': pick(raw, ['goods_type_id', 'GoodsTypeID']),
      'region_id': pick(raw, ['region_id', 'RegionID']),
      'name': pick(raw, ['name', 'Name', 'line_name', 'LineName']),
      'line_id': pick(raw, ['line_id', 'LineID']),
      'unit_core': pick(raw, ['unit_core', 'UnitCore']),
      'unit_mem': pick(raw, ['unit_mem', 'UnitMem']),
      'unit_disk': pick(raw, ['unit_disk', 'UnitDisk']),
      'unit_bw': pick(raw, ['unit_bw', 'UnitBW']),
      'add_core_min': pick(raw, ['add_core_min', 'AddCoreMin']),
      'add_core_max': pick(raw, ['add_core_max', 'AddCoreMax']),
      'add_core_step': pick(raw, ['add_core_step', 'AddCoreStep']),
      'add_mem_min': pick(raw, ['add_mem_min', 'AddMemMin']),
      'add_mem_max': pick(raw, ['add_mem_max', 'AddMemMax']),
      'add_mem_step': pick(raw, ['add_mem_step', 'AddMemStep']),
      'add_disk_min': pick(raw, ['add_disk_min', 'AddDiskMin']),
      'add_disk_max': pick(raw, ['add_disk_max', 'AddDiskMax']),
      'add_disk_step': pick(raw, ['add_disk_step', 'AddDiskStep']),
      'add_bw_min': pick(raw, ['add_bw_min', 'AddBwMin']),
      'add_bw_max': pick(raw, ['add_bw_max', 'AddBwMax']),
      'add_bw_step': pick(raw, ['add_bw_step', 'AddBwStep']),
      'active': pick(raw, ['active', 'Active']) ?? true,
      'visible': pick(raw, ['visible', 'Visible']) ?? true,
      'capacity_remaining': pick(raw, ['capacity_remaining', 'CapacityRemaining']),
      'sort_order': pick(raw, ['sort_order', 'SortOrder']),
    };
  }

  Map<String, dynamic> _normalizePlanGroup(Map<String, dynamic> raw) {
    return {
      'id': pick(raw, ['id', 'ID']),
      'goods_type_id': pick(raw, ['goods_type_id', 'GoodsTypeID']),
      'region_id': pick(raw, ['region_id', 'RegionID']),
      'line_id': pick(raw, ['line_id', 'LineID']),
      'name': pick(raw, ['name', 'Name', 'line_name', 'LineName']),
      'unit_core': pick(raw, ['unit_core', 'UnitCore']),
      'unit_mem': pick(raw, ['unit_mem', 'UnitMem']),
      'unit_disk': pick(raw, ['unit_disk', 'UnitDisk']),
      'unit_bw': pick(raw, ['unit_bw', 'UnitBW']),
      'add_core_min': pick(raw, ['add_core_min', 'AddCoreMin']),
      'add_core_max': pick(raw, ['add_core_max', 'AddCoreMax']),
      'add_core_step': pick(raw, ['add_core_step', 'AddCoreStep']),
      'add_mem_min': pick(raw, ['add_mem_min', 'AddMemMin']),
      'add_mem_max': pick(raw, ['add_mem_max', 'AddMemMax']),
      'add_mem_step': pick(raw, ['add_mem_step', 'AddMemStep']),
      'add_disk_min': pick(raw, ['add_disk_min', 'AddDiskMin']),
      'add_disk_max': pick(raw, ['add_disk_max', 'AddDiskMax']),
      'add_disk_step': pick(raw, ['add_disk_step', 'AddDiskStep']),
      'add_bw_min': pick(raw, ['add_bw_min', 'AddBwMin']),
      'add_bw_max': pick(raw, ['add_bw_max', 'AddBwMax']),
      'add_bw_step': pick(raw, ['add_bw_step', 'AddBwStep']),
      'active': pick(raw, ['active', 'Active']) ?? true,
      'visible': pick(raw, ['visible', 'Visible']) ?? true,
      'capacity_remaining': pick(raw, ['capacity_remaining', 'CapacityRemaining']),
    };
  }

  Map<String, dynamic> _normalizePackage(Map<String, dynamic> raw) {
    return {
      'id': pick(raw, ['id', 'ID']),
      'goods_type_id': pick(raw, ['goods_type_id', 'GoodsTypeID']),
      'product_id': pick(raw, ['product_id', 'ProductID']),
      'plan_group_id': pick(raw, ['plan_group_id', 'PlanGroupID', 'planGroupId']),
      'name': pick(raw, ['name', 'Name']),
      'cores': pick(raw, ['cores', 'Cores', 'cpu', 'CPU']),
      'memory_gb': pick(raw, ['memory_gb', 'MemoryGB']),
      'disk_gb': pick(raw, ['disk_gb', 'DiskGB']),
      'bandwidth_mbps': pick(raw, ['bandwidth_mbps', 'BandwidthMB', 'bandwidth']),
      'cpu_model': pick(raw, ['cpu_model', 'CPUModel']),
      'monthly_price': pick(raw, ['monthly_price', 'Monthly', 'price_monthly']),
      'port_num': pick(raw, ['port_num', 'PortNum']),
      'active': pick(raw, ['active', 'Active']) ?? true,
      'visible': pick(raw, ['visible', 'Visible']) ?? true,
      'capacity_remaining': pick(raw, ['capacity_remaining', 'CapacityRemaining']),
    };
  }

  Map<String, dynamic> _normalizeSystemImage(Map<String, dynamic> raw) {
    return {
      'id': pick(raw, ['id', 'ID']),
      'line_id': pick(raw, ['line_id', 'LineID']),
      'plan_group_id': pick(raw, ['plan_group_id', 'PlanGroupID']),
      'image_id': pick(raw, ['image_id', 'ImageID']),
      'name': pick(raw, ['name', 'Name']),
      'type': pick(raw, ['type', 'Type']),
      'enabled': pick(raw, ['enabled', 'Enabled']) ?? true,
    };
  }

  Map<String, dynamic> _normalizeBillingCycle(Map<String, dynamic> raw) {
    return {
      'id': pick(raw, ['id', 'ID']),
      'name': pick(raw, ['name', 'Name']),
      'months': pick(raw, ['months', 'Months']),
      'multiplier': pick(raw, ['multiplier', 'Multiplier']),
      'min_qty': pick(raw, ['min_qty', 'MinQty']),
      'max_qty': pick(raw, ['max_qty', 'MaxQty']),
      'active': pick(raw, ['active', 'Active']) ?? true,
      'sort_order': pick(raw, ['sort_order', 'SortOrder']),
    };
  }
}
