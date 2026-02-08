import 'package:dio/dio.dart';
import '../../core/constants/api_endpoints.dart';
import '../../core/network/api_client.dart';
import '../../core/utils/map_utils.dart';

class CatalogData {
  final List<Map<String, dynamic>> goodsTypes;
  final List<Map<String, dynamic>> regions;
  final List<Map<String, dynamic>> lines;
  final List<Map<String, dynamic>> planGroups;
  final List<Map<String, dynamic>> packages;
  final List<Map<String, dynamic>> systemImages;
  final List<Map<String, dynamic>> billingCycles;

  CatalogData({
    required this.goodsTypes,
    required this.regions,
    required this.lines,
    required this.planGroups,
    required this.packages,
    required this.systemImages,
    required this.billingCycles,
  });
}

class CatalogRepository {
  final Dio _dio = ApiClient.instance.dio;

  Future<CatalogData> fetchCatalog() async {
    final response = await _dio.get(ApiEndpoints.catalog);
    final data = ensureMap(response.data);

    final goodsTypes = _normalizeList(data['goods_types']);
    final regions = _normalizeList(data['regions']);
    final lines = _normalizeList(data['lines']);
    final planGroups = _normalizeList(data['plan_groups']);
    final packages = _normalizeList(data['packages']);
    final systemImages = _normalizeList(data['system_images']);
    final billingCycles = _normalizeList(data['billing_cycles']);

    return CatalogData(
      goodsTypes: goodsTypes,
      regions: regions,
      lines: lines,
      planGroups: planGroups,
      packages: packages,
      systemImages: systemImages,
      billingCycles: billingCycles,
    );
  }

  Future<List<Map<String, dynamic>>> listSystemImages({
    int? lineId,
    int? planGroupId,
  }) async {
    final response = await _dio.get(
      ApiEndpoints.systemImages,
      queryParameters: {
        if (lineId != null) 'line_id': lineId,
        if (planGroupId != null) 'plan_group_id': planGroupId,
      },
    );
    final data = ensureMap(response.data);
    final payload = data['data'] ?? data['items'] ?? data['system_images'] ?? data;
    if (payload is Map && payload['items'] is List) {
      return _normalizeList(payload['items']);
    }
    return _normalizeList(payload);
  }

  List<Map<String, dynamic>> _normalizeList(dynamic raw) {
    if (raw is List) {
      return raw.map((e) => ensureMap(e)).toList();
    }
    return [];
  }
}
