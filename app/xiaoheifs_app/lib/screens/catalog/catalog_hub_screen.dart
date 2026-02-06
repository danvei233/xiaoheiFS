import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../app_state.dart';
import '../../services/api_client.dart';

class CatalogHubScreen extends StatefulWidget {
  const CatalogHubScreen({super.key});

  @override
  State<CatalogHubScreen> createState() => _CatalogHubScreenState();
}

class _CatalogHubScreenState extends State<CatalogHubScreen>
    with SingleTickerProviderStateMixin {
  ApiClient? _client;
  bool _loading = false;

  List<Map<String, dynamic>> _goodsTypes = [];
  List<Map<String, dynamic>> _regions = [];
  List<Map<String, dynamic>> _lines = [];
  List<Map<String, dynamic>> _packages = [];
  List<Map<String, dynamic>> _systemImages = [];
  List<Map<String, dynamic>> _billingCycles = [];

  int? _goodsTypeId;
  int? _packageLineId;
  int? _imageLineId;

  final Set<int> _selectedRegionIds = {};
  final Set<int> _selectedLineIds = {};
  final Set<int> _selectedPackageIds = {};
  final Set<int> _selectedImageIds = {};
  final Set<int> _selectedCycleIds = {};

  late final TabController _tabController;

  final Map<String, dynamic> _batchForm = {
    'plan_group_id': null,
    'cpu_min': 1,
    'cpu_max': 8,
    'cpu_step': 1,
    'memory_ratio': 2,
    'memory_min': 1,
    'memory_max': 128,
    'disk_min': 20,
    'disk_max': 200,
    'disk_step': 20,
    'bw_min': 1,
    'bw_max': 100,
    'bw_step': 5,
    'port_num': 30,
    'cpu_model': '',
    'price_multiplier': 1,
    'total_cost': 0,
    'active': true,
    'visible': true,
    'total_cores': 0,
    'total_mem': 0,
    'total_disk': 0,
    'total_bw': 0,
    'overcommit_enabled': false,
    'overcommit_ratio': 1,
  };

  List<Map<String, dynamic>> _generatedPackages = [];

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 6, vsync: this);
  }

  @override
  void dispose() {
    _tabController.dispose();
    super.dispose();
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final client = context.read<AppState>().apiClient;
    if (client?.baseUrl != _client?.baseUrl ||
        client?.token != _client?.token ||
        client?.apiKey != _client?.apiKey) {
      _client = client;
      if (client != null) {
        _loadGoodsTypes();
      }
    }
  }

  int? _int(dynamic value) {
    if (value == null) return null;
    if (value is int) return value;
    if (value is double) return value.toInt();
    if (value is String) return int.tryParse(value);
    return null;
  }

  double _double(dynamic value) {
    if (value == null) return 0;
    if (value is num) return value.toDouble();
    if (value is String) return double.tryParse(value) ?? 0;
    return 0;
  }

  bool _bool(dynamic value) {
    if (value == null) return false;
    if (value is bool) return value;
    if (value is num) return value != 0;
    if (value is String) {
      final v = value.toLowerCase();
      return v == 'true' || v == '1' || v == 'yes';
    }
    return false;
  }

  String _str(dynamic value) => value?.toString() ?? '';
  Map<String, dynamic> _mapGoodsType(dynamic row) {
    final m = row is Map<String, dynamic> ? row : <String, dynamic>{};
    return {
      'id': _int(m['id'] ?? m['ID']),
      'code': _str(m['code'] ?? m['Code']),
      'name': _str(m['name'] ?? m['Name']),
      'active': _bool(m['active'] ?? m['Active']),
      'sort_order': _int(m['sort_order'] ?? m['SortOrder']) ?? 0,
      'automation_plugin_id': _str(
        m['automation_plugin_id'] ?? m['AutomationPluginID'],
      ),
      'automation_instance_id': _str(
        m['automation_instance_id'] ?? m['AutomationInstanceID'],
      ),
    };
  }

  Map<String, dynamic> _mapRegion(dynamic row) {
    final m = row is Map<String, dynamic> ? row : <String, dynamic>{};
    return {
      'id': _int(m['id'] ?? m['ID']),
      'goods_type_id': _int(m['goods_type_id'] ?? m['GoodsTypeID']),
      'name': _str(m['name'] ?? m['Name']),
      'code': _str(m['code'] ?? m['Code']),
      'active': _bool(m['active'] ?? m['Active']),
    };
  }

  Map<String, dynamic> _mapLine(dynamic row) {
    final m = row is Map<String, dynamic> ? row : <String, dynamic>{};
    return {
      'id': _int(m['id'] ?? m['ID']),
      'goods_type_id': _int(m['goods_type_id'] ?? m['GoodsTypeID']),
      'region_id': _int(m['region_id'] ?? m['RegionID']),
      'name': _str(m['name'] ?? m['Name'] ?? m['line_name'] ?? m['LineName']),
      'line_id': _str(m['line_id'] ?? m['LineID']),
      'unit_core': _double(m['unit_core'] ?? m['UnitCore']),
      'unit_mem': _double(m['unit_mem'] ?? m['UnitMem']),
      'unit_disk': _double(m['unit_disk'] ?? m['UnitDisk']),
      'unit_bw': _double(m['unit_bw'] ?? m['UnitBW']),
      'add_core_min': _double(m['add_core_min'] ?? m['AddCoreMin']),
      'add_core_max': _double(m['add_core_max'] ?? m['AddCoreMax']),
      'add_core_step': _double(m['add_core_step'] ?? m['AddCoreStep']),
      'add_mem_min': _double(m['add_mem_min'] ?? m['AddMemMin']),
      'add_mem_max': _double(m['add_mem_max'] ?? m['AddMemMax']),
      'add_mem_step': _double(m['add_mem_step'] ?? m['AddMemStep']),
      'add_disk_min': _double(m['add_disk_min'] ?? m['AddDiskMin']),
      'add_disk_max': _double(m['add_disk_max'] ?? m['AddDiskMax']),
      'add_disk_step': _double(m['add_disk_step'] ?? m['AddDiskStep']),
      'add_bw_min': _double(m['add_bw_min'] ?? m['AddBwMin']),
      'add_bw_max': _double(m['add_bw_max'] ?? m['AddBwMax']),
      'add_bw_step': _double(m['add_bw_step'] ?? m['AddBwStep']),
      'active': _bool(m['active'] ?? m['Active']),
      'visible': _bool(m['visible'] ?? m['Visible']),
      'capacity_remaining': _double(
        m['capacity_remaining'] ?? m['CapacityRemaining'],
      ),
    };
  }

  Map<String, dynamic> _mapPackage(dynamic row) {
    final m = row is Map<String, dynamic> ? row : <String, dynamic>{};
    return {
      'id': _int(m['id'] ?? m['ID']),
      'name': _str(m['name'] ?? m['Name']),
      'goods_type_id': _int(m['goods_type_id'] ?? m['GoodsTypeID']),
      'plan_group_id': _int(m['plan_group_id'] ?? m['PlanGroupID']),
      'cores': _double(m['cores'] ?? m['Cores']),
      'memory_gb': _double(m['memory_gb'] ?? m['MemoryGB']),
      'disk_gb': _double(m['disk_gb'] ?? m['DiskGB']),
      'bandwidth_mbps': _double(m['bandwidth_mbps'] ?? m['BandwidthMB']),
      'cpu_model': _str(m['cpu_model'] ?? m['CPUModel']),
      'port_num': _double(m['port_num'] ?? m['PortNum']),
      'monthly_price': _double(m['monthly_price'] ?? m['Monthly']),
      'active': _bool(m['active'] ?? m['Active']),
      'visible': _bool(m['visible'] ?? m['Visible']),
      'capacity_remaining': _double(
        m['capacity_remaining'] ?? m['CapacityRemaining'],
      ),
    };
  }

  Map<String, dynamic> _mapImage(dynamic row) {
    final m = row is Map<String, dynamic> ? row : <String, dynamic>{};
    return {
      'id': _int(m['id'] ?? m['ID']),
      'image_id': _str(m['image_id'] ?? m['ImageID']),
      'name': _str(m['name'] ?? m['Name']),
      'type': _str(m['type'] ?? m['Type']),
      'enabled': _bool(m['enabled'] ?? m['Enabled']),
    };
  }

  Map<String, dynamic> _mapCycle(dynamic row) {
    final m = row is Map<String, dynamic> ? row : <String, dynamic>{};
    return {
      'id': _int(m['id'] ?? m['ID']),
      'name': _str(m['name'] ?? m['Name']),
      'months': _double(m['months'] ?? m['Months']),
      'multiplier': _double(m['multiplier'] ?? m['Multiplier']),
      'min_qty': _double(m['min_qty'] ?? m['MinQty']),
      'max_qty': _double(m['max_qty'] ?? m['MaxQty']),
      'active': _bool(m['active'] ?? m['Active']),
    };
  }

  String _regionNameById(int? id) {
    if (id == null) return '-';
    final region = _regions.firstWhere(
      (item) => item['id'] == id,
      orElse: () => {},
    );
    final name = region['name']?.toString() ?? '';
    return name.isNotEmpty ? name : id.toString();
  }

  String _lineNameById(int? id) {
    if (id == null) return '-';
    final line = _lines.firstWhere(
      (item) => item['id'] == id,
      orElse: () => {},
    );
    final name = line['name']?.toString() ?? '';
    return name.isNotEmpty ? name : id.toString();
  }

  String _goodsTypeNameById(int? id) {
    if (id == null) return '-';
    final type = _goodsTypes.firstWhere(
      (item) => item['id'] == id,
      orElse: () => {},
    );
    final name = type['name']?.toString() ?? '';
    return name.isNotEmpty ? name : id.toString();
  }

  String? _resolveCloudLineId(int lineId) {
    final line = _lines.firstWhere(
      (item) => item['id'] == lineId,
      orElse: () => {},
    );
    final raw = line['line_id'];
    final text = raw?.toString().trim() ?? '';
    if (text.isNotEmpty) return text;
    return lineId.toString();
  }

  String _formatCapacity(dynamic value) {
    final num = _double(value);
    if (num.isNaN) return '-';
    if (num < 0) return '不限';
    if (num == 0) return '售罄';
    if (num == num.roundToDouble()) return num.toInt().toString();
    return num.toString();
  }

  Color _capacityColor(dynamic value) {
    final num = _double(value);
    if (num.isNaN) return Colors.grey;
    if (num < 0) return Colors.green;
    if (num == 0) return Colors.red;
    return Colors.blue;
  }

  bool _isWindowsType(String value) => value.toLowerCase().contains('win');
  bool _isLinuxType(String value) => value.toLowerCase().contains('linux');

  Color _imageTypeColor(String value) {
    if (_isWindowsType(value)) return Colors.blue;
    if (_isLinuxType(value)) return Colors.green;
    return Colors.grey;
  }

  String _formatImageType(String value) {
    if (value.trim().isEmpty) return '-';
    if (_isWindowsType(value)) return 'Windows';
    if (_isLinuxType(value)) return 'Linux';
    return value;
  }

  Future<void> _loadGoodsTypes() async {
    if (_client == null) return;
    setState(() => _loading = true);
    try {
      final resp = await _client!.getJson('/admin/api/v1/goods-types');
      final items = (resp['items'] as List<dynamic>? ?? [])
          .map(_mapGoodsType)
          .where((item) => item['id'] != null)
          .toList();
      setState(() {
        _goodsTypes = items;
        if (_goodsTypeId == null && items.isNotEmpty) {
          _goodsTypeId = items.first['id'] as int;
        }
      });
      await _loadAll();
    } catch (e) {
      _showError(e);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _loadAll() async {
    if (_client == null) return;
    if (_goodsTypeId == null) {
      setState(() {
        _regions = [];
        _lines = [];
        _packages = [];
      });
      return;
    }
    setState(() => _loading = true);
    try {
      final goodsTypeParam = {'goods_type_id': _goodsTypeId.toString()};
      final results = await Future.wait([
        _client!.getJson('/admin/api/v1/regions', query: goodsTypeParam),
        _client!.getJson('/admin/api/v1/lines', query: goodsTypeParam),
        _client!.getJson('/admin/api/v1/packages', query: goodsTypeParam),
        _client!.getJson('/admin/api/v1/system-images'),
        _client!.getJson('/admin/api/v1/billing-cycles'),
      ]);

      final regions = (results[0]['items'] as List<dynamic>? ?? [])
          .map(_mapRegion)
          .toList();
      final lines = (results[1]['items'] as List<dynamic>? ?? [])
          .map(_mapLine)
          .toList();
      final packages = (results[2]['items'] as List<dynamic>? ?? [])
          .map(_mapPackage)
          .toList();
      final images = (results[3]['items'] as List<dynamic>? ?? [])
          .map(_mapImage)
          .toList();
      final cycles = (results[4]['items'] as List<dynamic>? ?? [])
          .map(_mapCycle)
          .toList();

      setState(() {
        _regions = regions;
        _lines = lines;
        _packages = packages;
        _systemImages = images;
        _billingCycles = cycles;
      });
    } catch (e) {
      _showError(e);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  void _showMessage(String text) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(text)));
  }

  void _showError(Object e) {
    _showMessage('操作失败：$e');
  }

  Future<bool> _confirm(String title) async {
    final result = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(title),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('取消'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('确认'),
          ),
        ],
      ),
    );
    return result ?? false;
  }

  Future<void> _syncCurrentGoodsType() async {
    if (_client == null || _goodsTypeId == null) return;
    try {
      await _client!.postJson(
        '/admin/api/v1/goods-types/${_goodsTypeId!}/sync-automation',
        query: {'mode': 'merge'},
      );
      _showMessage('已同步当前类型');
      await _loadAll();
    } catch (e) {
      _showError(e);
    }
  }

  Future<void> _syncGoodsType(Map<String, dynamic> record) async {
    final id = record['id'];
    if (_client == null || id == null) return;
    try {
      await _client!.postJson(
        '/admin/api/v1/goods-types/$id/sync-automation',
        query: {'mode': 'merge'},
      );
      _showMessage('已同步');
      await _loadAll();
    } catch (e) {
      _showError(e);
    }
  }

  Future<void> _deleteGoodsType(Map<String, dynamic> record) async {
    final id = record['id'];
    if (_client == null || id == null) return;
    if (!await _confirm('确认删除该商品类型?')) return;
    try {
      await _client!.deleteJson('/admin/api/v1/goods-types/$id');
      _showMessage('已删除');
      await _loadGoodsTypes();
    } catch (e) {
      _showError(e);
    }
  }

  Future<void> _deleteRegion(Map<String, dynamic> record) async {
    final id = record['id'];
    if (_client == null || id == null) return;
    if (!await _confirm('确认删除该地区?')) return;
    try {
      await _client!.deleteJson('/admin/api/v1/regions/$id');
      _showMessage('已删除');
      await _loadAll();
    } catch (e) {
      _showError(e);
    }
  }

  Future<void> _bulkDeleteRegions() async {
    if (_client == null || _selectedRegionIds.isEmpty) return;
    if (!await _confirm('确认删除所选 ${_selectedRegionIds.length} 个地区?')) {
      return;
    }
    try {
      await _client!.postJson(
        '/admin/api/v1/regions/bulk-delete',
        body: {'ids': _selectedRegionIds.toList()},
      );
      _selectedRegionIds.clear();
      _showMessage('已批量删除');
      await _loadAll();
    } catch (e) {
      _showError(e);
    }
  }

  Future<void> _deleteLine(Map<String, dynamic> record) async {
    final id = record['id'];
    if (_client == null || id == null) return;
    if (!await _confirm('确认删除该线路?')) return;
    try {
      await _client!.deleteJson('/admin/api/v1/lines/$id');
      _showMessage('已删除');
      await _loadAll();
    } catch (e) {
      _showError(e);
    }
  }

  Future<void> _bulkDeleteLines() async {
    if (_client == null || _selectedLineIds.isEmpty) return;
    if (!await _confirm('确认删除所选 ${_selectedLineIds.length} 条线路?')) {
      return;
    }
    try {
      await _client!.postJson(
        '/admin/api/v1/lines/bulk-delete',
        body: {'ids': _selectedLineIds.toList()},
      );
      _selectedLineIds.clear();
      _showMessage('已批量删除');
      await _loadAll();
    } catch (e) {
      _showError(e);
    }
  }

  Future<void> _deletePackage(Map<String, dynamic> record) async {
    final id = record['id'];
    if (_client == null || id == null) return;
    if (!await _confirm('确认删除该套餐?')) return;
    try {
      await _client!.deleteJson('/admin/api/v1/packages/$id');
      _showMessage('已删除');
      await _loadAll();
    } catch (e) {
      _showError(e);
    }
  }

  Future<void> _bulkDeletePackages() async {
    if (_client == null || _selectedPackageIds.isEmpty) return;
    if (!await _confirm('确认删除所选 ${_selectedPackageIds.length} 个套餐?')) {
      return;
    }
    try {
      await _client!.postJson(
        '/admin/api/v1/packages/bulk-delete',
        body: {'ids': _selectedPackageIds.toList()},
      );
      _selectedPackageIds.clear();
      _showMessage('已批量删除');
      await _loadAll();
    } catch (e) {
      _showError(e);
    }
  }

  Future<void> _deleteImage(Map<String, dynamic> record) async {
    final id = record['id'];
    if (_client == null || id == null) return;
    if (!await _confirm('确认删除该镜像?')) return;
    try {
      await _client!.deleteJson('/admin/api/v1/system-images/$id');
      _showMessage('已删除');
      await _loadAll();
    } catch (e) {
      _showError(e);
    }
  }

  Future<void> _bulkDeleteImages() async {
    if (_client == null || _selectedImageIds.isEmpty) return;
    if (!await _confirm('确认删除所选 ${_selectedImageIds.length} 个镜像?')) {
      return;
    }
    try {
      await _client!.postJson(
        '/admin/api/v1/system-images/bulk-delete',
        body: {'ids': _selectedImageIds.toList()},
      );
      _selectedImageIds.clear();
      _showMessage('已批量删除');
      await _loadAll();
    } catch (e) {
      _showError(e);
    }
  }

  Future<void> _deleteCycle(Map<String, dynamic> record) async {
    final id = record['id'];
    if (_client == null || id == null) return;
    if (!await _confirm('确认删除该周期?')) return;
    try {
      await _client!.deleteJson('/admin/api/v1/billing-cycles/$id');
      _showMessage('已删除');
      await _loadAll();
    } catch (e) {
      _showError(e);
    }
  }

  Future<void> _bulkDeleteCycles() async {
    if (_client == null || _selectedCycleIds.isEmpty) return;
    if (!await _confirm('确认删除所选 ${_selectedCycleIds.length} 个周期?')) {
      return;
    }
    try {
      await _client!.postJson(
        '/admin/api/v1/billing-cycles/bulk-delete',
        body: {'ids': _selectedCycleIds.toList()},
      );
      _selectedCycleIds.clear();
      _showMessage('已批量删除');
      await _loadAll();
    } catch (e) {
      _showError(e);
    }
  }

  Future<void> _syncImages() async {
    if (_client == null || _imageLineId == null) return;
    final cloudLineId = _resolveCloudLineId(_imageLineId!);
    if (cloudLineId == null || cloudLineId.isEmpty) {
      _showMessage('无法解析线路 ID');
      return;
    }
    try {
      await _client!.postJson(
        '/admin/api/v1/system-images/sync',
        query: {'line_id': cloudLineId},
      );
      _showMessage('已触发同步');
      await _loadAll();
    } catch (e) {
      _showError(e);
    }
  }

  List<Map<String, dynamic>> get _filteredPackages {
    if (_packageLineId == null) return _packages;
    return _packages
        .where((item) => item['plan_group_id'] == _packageLineId)
        .toList();
  }

  List<Map<String, dynamic>> get _filteredImages {
    return _systemImages;
  }

  Future<List<int>> _loadLineImageIds(int lineId) async {
    if (_client == null) return [];
    final cloudLineId = _resolveCloudLineId(lineId);
    if (cloudLineId == null || cloudLineId.isEmpty) return [];
    try {
      final resp = await _client!.getJson(
        '/admin/api/v1/system-images',
        query: {'line_id': cloudLineId},
      );
      final items = (resp['items'] as List<dynamic>? ?? []);
      return items
          .map((row) => _int((row as Map<String, dynamic>)['id'] ?? row['ID']))
          .whereType<int>()
          .toList();
    } catch (_) {
      return [];
    }
  }

  bool _isCompact(BuildContext context) =>
      MediaQuery.of(context).size.width < 900;

  Widget _simpleTag(String text, Color color) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: color.withOpacity(0.15),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        text,
        style: TextStyle(color: color, fontWeight: FontWeight.w600),
      ),
    );
  }

  Widget _statusTag(bool value, {String yes = '启用', String no = '停用'}) {
    return _simpleTag(value ? yes : no, value ? Colors.green : Colors.red);
  }

  Widget _visibilityTag(bool value) {
    return _simpleTag(value ? '可见' : '隐藏', value ? Colors.green : Colors.grey);
  }

  Widget _capacityTag(dynamic value) {
    final text = _formatCapacity(value);
    final color = _capacityColor(value);
    return _simpleTag(text, color);
  }

  Widget _kvRow(String label, String value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 2),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 96,
            child: Text(label, style: const TextStyle(color: Colors.black54)),
          ),
          Expanded(child: Text(value.isEmpty ? '-' : value)),
        ],
      ),
    );
  }

  Widget _actionRow(List<Widget> actions) {
    return Wrap(spacing: 8, runSpacing: 8, children: actions);
  }

  Widget _mobileCard({required Widget child}) {
    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: Padding(padding: const EdgeInsets.all(16), child: child),
    );
  }

  void _toggleSelection(Set<int> set, int id, bool selected) {
    setState(() {
      if (selected) {
        set.add(id);
      } else {
        set.remove(id);
      }
    });
  }

  Future<void> _setLineSystemImages(int lineId, List<int> imageIds) async {
    if (_client == null) return;
    try {
      await _client!.postJson(
        '/admin/api/v1/lines/$lineId/system-images',
        body: {'image_ids': imageIds},
      );
    } catch (e) {
      _showError(e);
    }
  }

  Widget _numberField(String label, TextEditingController controller) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: TextField(
        controller: controller,
        decoration: InputDecoration(labelText: label),
        keyboardType: const TextInputType.numberWithOptions(decimal: true),
      ),
    );
  }

  Future<void> _openGoodsTypeEditor([Map<String, dynamic>? record]) async {
    final nameCtrl = TextEditingController(text: _str(record?['name']));
    final codeCtrl = TextEditingController(text: _str(record?['code']));
    final sortCtrl = TextEditingController(
      text: _int(record?['sort_order'])?.toString() ?? '0',
    );
    final pluginCtrl = TextEditingController(
      text: _str(record?['automation_plugin_id']).isNotEmpty
          ? _str(record?['automation_plugin_id'])
          : 'lightboat',
    );
    final instanceCtrl = TextEditingController(
      text: _str(record?['automation_instance_id']).isNotEmpty
          ? _str(record?['automation_instance_id'])
          : 'default',
    );
    bool active = record?['active'] ?? true;
    bool saving = false;

    await showDialog<void>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setState) => AlertDialog(
          title: const Text('商品类型'),
          content: SizedBox(
            width: 520,
            child: SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  TextField(
                    controller: nameCtrl,
                    decoration: const InputDecoration(labelText: '名称'),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: codeCtrl,
                    decoration: const InputDecoration(labelText: '代码'),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: sortCtrl,
                    decoration: const InputDecoration(labelText: '排序'),
                    keyboardType: TextInputType.number,
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: pluginCtrl,
                    decoration: const InputDecoration(
                      labelText: 'automation_plugin_id',
                    ),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: instanceCtrl,
                    decoration: const InputDecoration(
                      labelText: 'automation_instance_id',
                    ),
                  ),
                  const SizedBox(height: 8),
                  SwitchListTile(
                    contentPadding: EdgeInsets.zero,
                    value: active,
                    title: const Text('启用'),
                    onChanged: (val) => setState(() => active = val),
                  ),
                ],
              ),
            ),
          ),
          actions: [
            TextButton(
              onPressed: saving ? null : () => Navigator.pop(context),
              child: const Text('取消'),
            ),
            FilledButton(
              onPressed: saving
                  ? null
                  : () async {
                      if (_client == null) return;
                      setState(() => saving = true);
                      final payload = {
                        'name': nameCtrl.text.trim(),
                        'code': codeCtrl.text.trim(),
                        'sort_order': int.tryParse(sortCtrl.text) ?? 0,
                        'automation_plugin_id': pluginCtrl.text.trim(),
                        'automation_instance_id': instanceCtrl.text.trim(),
                        'active': active,
                      };
                      try {
                        final id = record?['id'];
                        if (id != null) {
                          await _client!.patchJson(
                            '/admin/api/v1/goods-types/$id',
                            body: payload,
                          );
                        } else {
                          await _client!.postJson(
                            '/admin/api/v1/goods-types',
                            body: payload,
                          );
                        }
                        if (mounted) Navigator.pop(context);
                        _showMessage('已保存');
                        await _loadGoodsTypes();
                      } catch (e) {
                        _showError(e);
                      } finally {
                        if (mounted) setState(() => saving = false);
                      }
                    },
              child: const Text('保存'),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _openRegionEditor([Map<String, dynamic>? record]) async {
    final nameCtrl = TextEditingController(text: _str(record?['name']));
    final codeCtrl = TextEditingController(text: _str(record?['code']));
    int? goodsTypeId = _int(record?['goods_type_id']) ?? _goodsTypeId;
    bool active = record?['active'] ?? true;
    bool saving = false;

    await showDialog<void>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setState) => AlertDialog(
          title: const Text('地区'),
          content: SizedBox(
            width: 520,
            child: SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  DropdownButtonFormField<int>(
                    value: goodsTypeId,
                    decoration: const InputDecoration(labelText: '商品类型'),
                    items: _goodsTypes
                        .map(
                          (item) => DropdownMenuItem<int>(
                            value: item['id'] as int,
                            child: Text(item['name']?.toString() ?? '-'),
                          ),
                        )
                        .toList(),
                    onChanged: (val) => setState(() => goodsTypeId = val),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: nameCtrl,
                    decoration: const InputDecoration(labelText: '名称'),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: codeCtrl,
                    decoration: const InputDecoration(labelText: '代码'),
                  ),
                  const SizedBox(height: 8),
                  SwitchListTile(
                    contentPadding: EdgeInsets.zero,
                    value: active,
                    title: const Text('启用'),
                    onChanged: (val) => setState(() => active = val),
                  ),
                ],
              ),
            ),
          ),
          actions: [
            TextButton(
              onPressed: saving ? null : () => Navigator.pop(context),
              child: const Text('取消'),
            ),
            FilledButton(
              onPressed: saving
                  ? null
                  : () async {
                      if (_client == null) return;
                      setState(() => saving = true);
                      final payload = {
                        'goods_type_id': goodsTypeId,
                        'name': nameCtrl.text.trim(),
                        'code': codeCtrl.text.trim(),
                        'active': active,
                      };
                      try {
                        final id = record?['id'];
                        if (id != null) {
                          await _client!.patchJson(
                            '/admin/api/v1/regions/$id',
                            body: payload,
                          );
                        } else {
                          await _client!.postJson(
                            '/admin/api/v1/regions',
                            body: payload,
                          );
                        }
                        if (mounted) Navigator.pop(context);
                        _showMessage('已保存地区');
                        await _loadAll();
                      } catch (e) {
                        _showError(e);
                      } finally {
                        if (mounted) setState(() => saving = false);
                      }
                    },
              child: const Text('保存'),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _openLineEditor([Map<String, dynamic>? record]) async {
    final nameCtrl = TextEditingController(text: _str(record?['name']));
    final lineIdCtrl = TextEditingController(text: _str(record?['line_id']));
    final unitCoreCtrl = TextEditingController(
      text: _str(record?['unit_core']),
    );
    final unitMemCtrl = TextEditingController(text: _str(record?['unit_mem']));
    final unitDiskCtrl = TextEditingController(
      text: _str(record?['unit_disk']),
    );
    final unitBwCtrl = TextEditingController(text: _str(record?['unit_bw']));
    final addCoreMinCtrl = TextEditingController(
      text: _str(record?['add_core_min']),
    );
    final addCoreMaxCtrl = TextEditingController(
      text: _str(record?['add_core_max']),
    );
    final addCoreStepCtrl = TextEditingController(
      text: _str(record?['add_core_step']),
    );
    final addMemMinCtrl = TextEditingController(
      text: _str(record?['add_mem_min']),
    );
    final addMemMaxCtrl = TextEditingController(
      text: _str(record?['add_mem_max']),
    );
    final addMemStepCtrl = TextEditingController(
      text: _str(record?['add_mem_step']),
    );
    final addDiskMinCtrl = TextEditingController(
      text: _str(record?['add_disk_min']),
    );
    final addDiskMaxCtrl = TextEditingController(
      text: _str(record?['add_disk_max']),
    );
    final addDiskStepCtrl = TextEditingController(
      text: _str(record?['add_disk_step']),
    );
    final addBwMinCtrl = TextEditingController(
      text: _str(record?['add_bw_min']),
    );
    final addBwMaxCtrl = TextEditingController(
      text: _str(record?['add_bw_max']),
    );
    final addBwStepCtrl = TextEditingController(
      text: _str(record?['add_bw_step']),
    );
    final capacityCtrl = TextEditingController(
      text: _str(record?['capacity_remaining']).isNotEmpty
          ? _str(record?['capacity_remaining'])
          : '-1',
    );

    int? regionId = _int(record?['region_id']);
    bool active = record?['active'] ?? true;
    bool visible = record?['visible'] ?? true;
    bool saving = false;
    final imageIds = <int>{};

    if (record?['id'] != null) {
      final ids = await _loadLineImageIds(record!['id'] as int);
      imageIds.addAll(ids);
    }

    await showDialog<void>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setState) => AlertDialog(
          title: const Text('线路'),
          content: SizedBox(
            width: 640,
            child: SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  DropdownButtonFormField<int>(
                    value: regionId,
                    decoration: const InputDecoration(labelText: '地区'),
                    items: _regions
                        .map(
                          (item) => DropdownMenuItem<int>(
                            value: item['id'] as int,
                            child: Text(item['name']?.toString() ?? '-'),
                          ),
                        )
                        .toList(),
                    onChanged: (val) => setState(() => regionId = val),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: nameCtrl,
                    decoration: const InputDecoration(labelText: '线路名称'),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: lineIdCtrl,
                    decoration: const InputDecoration(labelText: '云线路 ID'),
                  ),
                  const SizedBox(height: 16),
                  const Align(
                    alignment: Alignment.centerLeft,
                    child: Text(
                      '单价设置',
                      style: TextStyle(fontWeight: FontWeight.w600),
                    ),
                  ),
                  const SizedBox(height: 8),
                  _numberField('CPU 单价', unitCoreCtrl),
                  _numberField('内存单价', unitMemCtrl),
                  _numberField('磁盘单价', unitDiskCtrl),
                  _numberField('带宽单价', unitBwCtrl),
                  const SizedBox(height: 16),
                  const Align(
                    alignment: Alignment.centerLeft,
                    child: Text(
                      '附加项范围',
                      style: TextStyle(fontWeight: FontWeight.w600),
                    ),
                  ),
                  const SizedBox(height: 8),
                  _numberField('CPU 最小', addCoreMinCtrl),
                  _numberField('CPU 最大', addCoreMaxCtrl),
                  _numberField('CPU 步进', addCoreStepCtrl),
                  _numberField('内存最小', addMemMinCtrl),
                  _numberField('内存最大', addMemMaxCtrl),
                  _numberField('内存步进', addMemStepCtrl),
                  _numberField('磁盘最小', addDiskMinCtrl),
                  _numberField('磁盘最大', addDiskMaxCtrl),
                  _numberField('磁盘步进', addDiskStepCtrl),
                  _numberField('带宽最小', addBwMinCtrl),
                  _numberField('带宽最大', addBwMaxCtrl),
                  _numberField('带宽步进', addBwStepCtrl),
                  const SizedBox(height: 8),
                  SwitchListTile(
                    contentPadding: EdgeInsets.zero,
                    value: active,
                    title: const Text('启用'),
                    onChanged: (val) => setState(() => active = val),
                  ),
                  SwitchListTile(
                    contentPadding: EdgeInsets.zero,
                    value: visible,
                    title: const Text('可见'),
                    onChanged: (val) => setState(() => visible = val),
                  ),
                  TextField(
                    controller: capacityCtrl,
                    decoration: const InputDecoration(
                      labelText: '余量 (负数表示不限，0 表示售罄)',
                    ),
                    keyboardType: TextInputType.number,
                  ),
                  const SizedBox(height: 12),
                  const Align(
                    alignment: Alignment.centerLeft,
                    child: Text(
                      '可用镜像',
                      style: TextStyle(fontWeight: FontWeight.w600),
                    ),
                  ),
                  const SizedBox(height: 4),
                  ..._systemImages.map(
                    (img) => CheckboxListTile(
                      contentPadding: EdgeInsets.zero,
                      value: imageIds.contains(img['id']),
                      title: Text('${img['name'] ?? '-'}'),
                      subtitle: Text(_formatImageType(_str(img['type']))),
                      onChanged: (val) {
                        final id = img['id'] as int?;
                        if (id == null) return;
                        setState(() {
                          if (val == true) {
                            imageIds.add(id);
                          } else {
                            imageIds.remove(id);
                          }
                        });
                      },
                    ),
                  ),
                ],
              ),
            ),
          ),
          actions: [
            TextButton(
              onPressed: saving ? null : () => Navigator.pop(context),
              child: const Text('取消'),
            ),
            FilledButton(
              onPressed: saving
                  ? null
                  : () async {
                      if (_client == null) return;
                      setState(() => saving = true);
                      final payload = {
                        'goods_type_id': _goodsTypeId,
                        'region_id': regionId,
                        'name': nameCtrl.text.trim(),
                        'line_id': lineIdCtrl.text.trim(),
                        'unit_core': _double(unitCoreCtrl.text),
                        'unit_mem': _double(unitMemCtrl.text),
                        'unit_disk': _double(unitDiskCtrl.text),
                        'unit_bw': _double(unitBwCtrl.text),
                        'add_core_min': _double(addCoreMinCtrl.text),
                        'add_core_max': _double(addCoreMaxCtrl.text),
                        'add_core_step': _double(addCoreStepCtrl.text),
                        'add_mem_min': _double(addMemMinCtrl.text),
                        'add_mem_max': _double(addMemMaxCtrl.text),
                        'add_mem_step': _double(addMemStepCtrl.text),
                        'add_disk_min': _double(addDiskMinCtrl.text),
                        'add_disk_max': _double(addDiskMaxCtrl.text),
                        'add_disk_step': _double(addDiskStepCtrl.text),
                        'add_bw_min': _double(addBwMinCtrl.text),
                        'add_bw_max': _double(addBwMaxCtrl.text),
                        'add_bw_step': _double(addBwStepCtrl.text),
                        'active': active,
                        'visible': visible,
                        'capacity_remaining': _double(capacityCtrl.text),
                      };
                      try {
                        final id = record?['id'];
                        int? lineId;
                        if (id != null) {
                          await _client!.patchJson(
                            '/admin/api/v1/lines/$id',
                            body: payload,
                          );
                          lineId = id as int;
                        } else {
                          final res = await _client!.postJson(
                            '/admin/api/v1/lines',
                            body: payload,
                          );
                          lineId = _int(
                            res['id'] ?? res['data']?['id'] ?? res['ID'],
                          );
                        }
                        if (lineId != null) {
                          await _setLineSystemImages(lineId, imageIds.toList());
                        }
                        if (mounted) Navigator.pop(context);
                        _showMessage('已保存线路');
                        await _loadAll();
                      } catch (e) {
                        _showError(e);
                      } finally {
                        if (mounted) setState(() => saving = false);
                      }
                    },
              child: const Text('保存'),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _openPackageEditor([Map<String, dynamic>? record]) async {
    final nameCtrl = TextEditingController(text: _str(record?['name']));
    final coresCtrl = TextEditingController(text: _str(record?['cores']));
    final memoryCtrl = TextEditingController(text: _str(record?['memory_gb']));
    final diskCtrl = TextEditingController(text: _str(record?['disk_gb']));
    final bwCtrl = TextEditingController(text: _str(record?['bandwidth_mbps']));
    final cpuModelCtrl = TextEditingController(
      text: _str(record?['cpu_model']),
    );
    final portCtrl = TextEditingController(text: _str(record?['port_num']));
    final priceCtrl = TextEditingController(
      text: _str(record?['monthly_price']),
    );
    final capacityCtrl = TextEditingController(
      text: _str(record?['capacity_remaining']).isNotEmpty
          ? _str(record?['capacity_remaining'])
          : '-1',
    );

    int? planGroupId =
        _int(record?['plan_group_id']) ??
        (_packageLineId == null ? null : _packageLineId);
    bool active = record?['active'] ?? true;
    bool visible = record?['visible'] ?? true;
    bool saving = false;

    await showDialog<void>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setState) => AlertDialog(
          title: const Text('套餐'),
          content: SizedBox(
            width: 520,
            child: SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  TextField(
                    controller: nameCtrl,
                    decoration: const InputDecoration(labelText: '名称'),
                  ),
                  const SizedBox(height: 12),
                  DropdownButtonFormField<int>(
                    value: planGroupId,
                    decoration: const InputDecoration(labelText: '线路'),
                    items: _lines
                        .map(
                          (item) => DropdownMenuItem<int>(
                            value: item['id'] as int,
                            child: Text(item['name']?.toString() ?? '-'),
                          ),
                        )
                        .toList(),
                    onChanged: (val) => setState(() => planGroupId = val),
                  ),
                  _numberField('CPU', coresCtrl),
                  _numberField('内存(GB)', memoryCtrl),
                  _numberField('磁盘(GB)', diskCtrl),
                  _numberField('带宽(Mbps)', bwCtrl),
                  _numberField('端口数', portCtrl),
                  TextField(
                    controller: cpuModelCtrl,
                    decoration: const InputDecoration(labelText: 'CPU 型号'),
                  ),
                  _numberField('月费', priceCtrl),
                  SwitchListTile(
                    contentPadding: EdgeInsets.zero,
                    value: active,
                    title: const Text('启用'),
                    onChanged: (val) => setState(() => active = val),
                  ),
                  SwitchListTile(
                    contentPadding: EdgeInsets.zero,
                    value: visible,
                    title: const Text('可见'),
                    onChanged: (val) => setState(() => visible = val),
                  ),
                  TextField(
                    controller: capacityCtrl,
                    decoration: const InputDecoration(
                      labelText: '余量 (负数表示不限，0 表示售罄)',
                    ),
                    keyboardType: TextInputType.number,
                  ),
                ],
              ),
            ),
          ),
          actions: [
            TextButton(
              onPressed: saving ? null : () => Navigator.pop(context),
              child: const Text('取消'),
            ),
            FilledButton(
              onPressed: saving
                  ? null
                  : () async {
                      if (_client == null) return;
                      setState(() => saving = true);
                      final payload = {
                        'goods_type_id': _goodsTypeId,
                        'name': nameCtrl.text.trim(),
                        'plan_group_id': planGroupId,
                        'cores': _double(coresCtrl.text),
                        'memory_gb': _double(memoryCtrl.text),
                        'disk_gb': _double(diskCtrl.text),
                        'bandwidth_mbps': _double(bwCtrl.text),
                        'cpu_model': cpuModelCtrl.text.trim(),
                        'port_num': _double(portCtrl.text),
                        'monthly_price': _double(priceCtrl.text),
                        'active': active,
                        'visible': visible,
                        'capacity_remaining': _double(capacityCtrl.text),
                      };
                      try {
                        final id = record?['id'];
                        if (id != null) {
                          await _client!.patchJson(
                            '/admin/api/v1/packages/$id',
                            body: payload,
                          );
                        } else {
                          await _client!.postJson(
                            '/admin/api/v1/packages',
                            body: payload,
                          );
                        }
                        if (mounted) Navigator.pop(context);
                        _showMessage('已保存套餐');
                        await _loadAll();
                      } catch (e) {
                        _showError(e);
                      } finally {
                        if (mounted) setState(() => saving = false);
                      }
                    },
              child: const Text('保存'),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _openImageEditor([Map<String, dynamic>? record]) async {
    final imageIdCtrl = TextEditingController(text: _str(record?['image_id']));
    final nameCtrl = TextEditingController(text: _str(record?['name']));
    final typeCtrl = TextEditingController(text: _str(record?['type']));
    bool enabled = record?['enabled'] ?? true;
    bool saving = false;

    await showDialog<void>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setState) => AlertDialog(
          title: const Text('系统镜像'),
          content: SizedBox(
            width: 420,
            child: SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  TextField(
                    controller: imageIdCtrl,
                    decoration: const InputDecoration(labelText: '镜像 ID'),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: nameCtrl,
                    decoration: const InputDecoration(labelText: '名称'),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: typeCtrl,
                    decoration: const InputDecoration(labelText: '类型'),
                  ),
                  const SizedBox(height: 8),
                  SwitchListTile(
                    contentPadding: EdgeInsets.zero,
                    value: enabled,
                    title: const Text('启用'),
                    onChanged: (val) => setState(() => enabled = val),
                  ),
                ],
              ),
            ),
          ),
          actions: [
            TextButton(
              onPressed: saving ? null : () => Navigator.pop(context),
              child: const Text('取消'),
            ),
            FilledButton(
              onPressed: saving
                  ? null
                  : () async {
                      if (_client == null) return;
                      setState(() => saving = true);
                      final payload = {
                        'image_id': imageIdCtrl.text.trim(),
                        'name': nameCtrl.text.trim(),
                        'type': typeCtrl.text.trim(),
                        'enabled': enabled,
                      };
                      try {
                        final id = record?['id'];
                        if (id != null) {
                          await _client!.patchJson(
                            '/admin/api/v1/system-images/$id',
                            body: payload,
                          );
                        } else {
                          await _client!.postJson(
                            '/admin/api/v1/system-images',
                            body: payload,
                          );
                        }
                        if (mounted) Navigator.pop(context);
                        _showMessage('已保存镜像');
                        await _loadAll();
                      } catch (e) {
                        _showError(e);
                      } finally {
                        if (mounted) setState(() => saving = false);
                      }
                    },
              child: const Text('保存'),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _openCycleEditor([Map<String, dynamic>? record]) async {
    final nameCtrl = TextEditingController(text: _str(record?['name']));
    final monthsCtrl = TextEditingController(text: _str(record?['months']));
    final multiplierCtrl = TextEditingController(
      text: _str(record?['multiplier']),
    );
    final minQtyCtrl = TextEditingController(text: _str(record?['min_qty']));
    final maxQtyCtrl = TextEditingController(text: _str(record?['max_qty']));
    bool active = record?['active'] ?? true;
    bool saving = false;

    await showDialog<void>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setState) => AlertDialog(
          title: const Text('计费周期'),
          content: SizedBox(
            width: 420,
            child: SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  TextField(
                    controller: nameCtrl,
                    decoration: const InputDecoration(labelText: '名称'),
                  ),
                  _numberField('月数', monthsCtrl),
                  _numberField('倍率', multiplierCtrl),
                  _numberField('最小数量', minQtyCtrl),
                  _numberField('最大数量', maxQtyCtrl),
                  SwitchListTile(
                    contentPadding: EdgeInsets.zero,
                    value: active,
                    title: const Text('启用'),
                    onChanged: (val) => setState(() => active = val),
                  ),
                ],
              ),
            ),
          ),
          actions: [
            TextButton(
              onPressed: saving ? null : () => Navigator.pop(context),
              child: const Text('取消'),
            ),
            FilledButton(
              onPressed: saving
                  ? null
                  : () async {
                      if (_client == null) return;
                      setState(() => saving = true);
                      final payload = {
                        'name': nameCtrl.text.trim(),
                        'months': _double(monthsCtrl.text),
                        'multiplier': _double(multiplierCtrl.text),
                        'min_qty': _double(minQtyCtrl.text),
                        'max_qty': _double(maxQtyCtrl.text),
                        'active': active,
                      };
                      try {
                        final id = record?['id'];
                        if (id != null) {
                          await _client!.patchJson(
                            '/admin/api/v1/billing-cycles/$id',
                            body: payload,
                          );
                        } else {
                          await _client!.postJson(
                            '/admin/api/v1/billing-cycles',
                            body: payload,
                          );
                        }
                        if (mounted) Navigator.pop(context);
                        _showMessage('已保存周期');
                        await _loadAll();
                      } catch (e) {
                        _showError(e);
                      } finally {
                        if (mounted) setState(() => saving = false);
                      }
                    },
              child: const Text('保存'),
            ),
          ],
        ),
      ),
    );
  }

  int _calcCapacity(
    double cores,
    double memory,
    double disk,
    double bandwidth,
  ) {
    final multiplier = _bool(_batchForm['overcommit_enabled'])
        ? _double(_batchForm['overcommit_ratio'])
        : 1;
    final totalCores = _double(_batchForm['total_cores']) * multiplier;
    final totalMem = _double(_batchForm['total_mem']) * multiplier;
    final totalDisk = _double(_batchForm['total_disk']) * multiplier;
    final totalBw = _double(_batchForm['total_bw']) * multiplier;
    final candidates = <int>[];
    if (totalCores > 0 && cores > 0) {
      candidates.add((totalCores / cores).floor());
    }
    if (totalMem > 0 && memory > 0) {
      candidates.add((totalMem / memory).floor());
    }
    if (totalDisk > 0 && disk > 0) {
      candidates.add((totalDisk / disk).floor());
    }
    if (totalBw > 0 && bandwidth > 0) {
      candidates.add((totalBw / bandwidth).floor());
    }
    if (candidates.isEmpty) return -1;
    candidates.sort();
    return candidates.first < 0 ? 0 : candidates.first;
  }

  void _generatePackages() {
    final planGroupId = _int(_batchForm['plan_group_id']);
    if (planGroupId == null) {
      _showMessage('请选择线路');
      return;
    }
    final line = _lines.firstWhere(
      (item) => item['id'] == planGroupId,
      orElse: () => {},
    );
    if (line.isEmpty) {
      _showMessage('线路信息未加载');
      return;
    }

    final cpuMin = _double(_batchForm['cpu_min']);
    final cpuMax = _double(_batchForm['cpu_max']);
    final cpuStep = _double(_batchForm['cpu_step']).clamp(1, 1000);
    final diskMin = _double(_batchForm['disk_min']);
    final diskMax = _double(_batchForm['disk_max']);
    final diskStep = _double(_batchForm['disk_step']).clamp(1, 10000);
    final bwMin = _double(_batchForm['bw_min']);
    final bwMax = _double(_batchForm['bw_max']);
    final bwStep = _double(_batchForm['bw_step']).clamp(1, 10000);
    final memoryRatio = _double(_batchForm['memory_ratio']);
    final memoryMin = _double(_batchForm['memory_min']);
    final memoryMax = _double(_batchForm['memory_max']);

    if (cpuMin <= 0 || cpuMax <= 0 || cpuMax < cpuMin) {
      _showMessage('CPU 范围不正确');
      return;
    }
    if (diskMin <= 0 || diskMax <= 0 || diskMax < diskMin) {
      _showMessage('存储范围不正确');
      return;
    }
    if (bwMin <= 0 || bwMax <= 0 || bwMax < bwMin) {
      _showMessage('带宽范围不正确');
      return;
    }
    if (memoryRatio <= 0) {
      _showMessage('内存比率需大于 0');
      return;
    }

    var priceMultiplier = _double(_batchForm['price_multiplier']);
    if (priceMultiplier <= 0) priceMultiplier = 1;
    final totalCost = _double(_batchForm['total_cost']);
    if (totalCost > 0) {
      final baseCost =
          _double(line['unit_core']) * _double(_batchForm['total_cores']) +
          _double(line['unit_mem']) * _double(_batchForm['total_mem']) +
          _double(line['unit_disk']) * _double(_batchForm['total_disk']) +
          _double(line['unit_bw']) * _double(_batchForm['total_bw']);
      if (baseCost > 0) {
        priceMultiplier = totalCost / baseCost;
      }
    }

    final items = <Map<String, dynamic>>[];
    for (double cpu = cpuMin; cpu <= cpuMax; cpu += cpuStep) {
      var memory = (cpu * memoryRatio).roundToDouble();
      if (memoryMin > 0 && memory < memoryMin) memory = memoryMin;
      if (memoryMax > 0 && memory > memoryMax) continue;
      for (double disk = diskMin; disk <= diskMax; disk += diskStep) {
        for (double bw = bwMin; bw <= bwMax; bw += bwStep) {
          final monthlyBase =
              _double(line['unit_core']) * cpu +
              _double(line['unit_mem']) * memory +
              _double(line['unit_disk']) * disk +
              _double(line['unit_bw']) * bw;
          final monthlyPrice = double.parse(
            (monthlyBase * priceMultiplier).toStringAsFixed(2),
          );
          final capacity = _calcCapacity(cpu, memory, disk, bw);
          items.add({
            'name':
                '${cpu.toInt()}C${memory.toInt()}G ${disk.toInt()}G ${bw.toInt()}M',
            'plan_group_id': planGroupId,
            'cores': cpu,
            'memory_gb': memory,
            'disk_gb': disk,
            'bandwidth_mbps': bw,
            'cpu_model': _batchForm['cpu_model'] ?? '',
            'port_num': _double(_batchForm['port_num']),
            'monthly_price': monthlyPrice,
            'active': _batchForm['active'] ?? true,
            'visible': _batchForm['visible'] ?? true,
            'capacity_remaining': capacity,
          });
        }
      }
    }

    if (items.isEmpty) {
      _showMessage('未生成任何套餐，请检查条件');
      setState(() => _generatedPackages = []);
      return;
    }

    if (items.length > 200) {
      _showMessage('生成数量过多，已截断至 200 条');
      items.removeRange(200, items.length);
    }

    setState(() => _generatedPackages = items);
  }

  Future<void> _applyGenerated() async {
    if (_client == null || _generatedPackages.isEmpty) return;
    if (!await _confirm('确认批量创建 ${_generatedPackages.length} 个套餐?')) {
      return;
    }
    try {
      for (final item in _generatedPackages) {
        await _client!.postJson('/admin/api/v1/packages', body: item);
      }
      _showMessage('已批量创建套餐');
      setState(() => _generatedPackages = []);
      if (mounted) Navigator.pop(context);
      await _loadAll();
    } catch (e) {
      _showError(e);
    }
  }

  Future<void> _openBatchDialog() async {
    if (_packageLineId != null) {
      _batchForm['plan_group_id'] = _packageLineId;
    }

    final cpuMinCtrl = TextEditingController(text: _str(_batchForm['cpu_min']));
    final cpuMaxCtrl = TextEditingController(text: _str(_batchForm['cpu_max']));
    final cpuStepCtrl = TextEditingController(
      text: _str(_batchForm['cpu_step']),
    );
    final memRatioCtrl = TextEditingController(
      text: _str(_batchForm['memory_ratio']),
    );
    final memMinCtrl = TextEditingController(
      text: _str(_batchForm['memory_min']),
    );
    final memMaxCtrl = TextEditingController(
      text: _str(_batchForm['memory_max']),
    );
    final diskMinCtrl = TextEditingController(
      text: _str(_batchForm['disk_min']),
    );
    final diskMaxCtrl = TextEditingController(
      text: _str(_batchForm['disk_max']),
    );
    final diskStepCtrl = TextEditingController(
      text: _str(_batchForm['disk_step']),
    );
    final bwMinCtrl = TextEditingController(text: _str(_batchForm['bw_min']));
    final bwMaxCtrl = TextEditingController(text: _str(_batchForm['bw_max']));
    final bwStepCtrl = TextEditingController(text: _str(_batchForm['bw_step']));
    final portCtrl = TextEditingController(text: _str(_batchForm['port_num']));
    final cpuModelCtrl = TextEditingController(
      text: _str(_batchForm['cpu_model']),
    );
    final priceMultiplierCtrl = TextEditingController(
      text: _str(_batchForm['price_multiplier']),
    );
    final totalCostCtrl = TextEditingController(
      text: _str(_batchForm['total_cost']),
    );
    final totalCoresCtrl = TextEditingController(
      text: _str(_batchForm['total_cores']),
    );
    final totalMemCtrl = TextEditingController(
      text: _str(_batchForm['total_mem']),
    );
    final totalDiskCtrl = TextEditingController(
      text: _str(_batchForm['total_disk']),
    );
    final totalBwCtrl = TextEditingController(
      text: _str(_batchForm['total_bw']),
    );
    final overcommitRatioCtrl = TextEditingController(
      text: _str(_batchForm['overcommit_ratio']),
    );

    bool active = _bool(_batchForm['active']);
    bool visible = _bool(_batchForm['visible']);
    bool overcommit = _bool(_batchForm['overcommit_enabled']);

    void syncBatchForm() {
      _batchForm['cpu_min'] = _double(cpuMinCtrl.text);
      _batchForm['cpu_max'] = _double(cpuMaxCtrl.text);
      _batchForm['cpu_step'] = _double(cpuStepCtrl.text);
      _batchForm['memory_ratio'] = _double(memRatioCtrl.text);
      _batchForm['memory_min'] = _double(memMinCtrl.text);
      _batchForm['memory_max'] = _double(memMaxCtrl.text);
      _batchForm['disk_min'] = _double(diskMinCtrl.text);
      _batchForm['disk_max'] = _double(diskMaxCtrl.text);
      _batchForm['disk_step'] = _double(diskStepCtrl.text);
      _batchForm['bw_min'] = _double(bwMinCtrl.text);
      _batchForm['bw_max'] = _double(bwMaxCtrl.text);
      _batchForm['bw_step'] = _double(bwStepCtrl.text);
      _batchForm['port_num'] = _double(portCtrl.text);
      _batchForm['cpu_model'] = cpuModelCtrl.text.trim();
      _batchForm['price_multiplier'] = _double(priceMultiplierCtrl.text);
      _batchForm['total_cost'] = _double(totalCostCtrl.text);
      _batchForm['total_cores'] = _double(totalCoresCtrl.text);
      _batchForm['total_mem'] = _double(totalMemCtrl.text);
      _batchForm['total_disk'] = _double(totalDiskCtrl.text);
      _batchForm['total_bw'] = _double(totalBwCtrl.text);
      _batchForm['overcommit_ratio'] = _double(overcommitRatioCtrl.text);
      _batchForm['active'] = active;
      _batchForm['visible'] = visible;
      _batchForm['overcommit_enabled'] = overcommit;
    }

    await showDialog<void>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setState) => AlertDialog(
          title: const Text('批量生成套餐'),
          content: SizedBox(
            width: 720,
            child: SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  DropdownButtonFormField<int?>(
                    value: _int(_batchForm['plan_group_id']),
                    decoration: const InputDecoration(labelText: '选择线路'),
                    items: _lines
                        .map(
                          (item) => DropdownMenuItem<int?>(
                            value: item['id'] as int,
                            child: Text(item['name']?.toString() ?? '-'),
                          ),
                        )
                        .toList(),
                    onChanged: (val) =>
                        setState(() => _batchForm['plan_group_id'] = val),
                  ),
                  _numberField('CPU 最小', cpuMinCtrl),
                  _numberField('CPU 最大', cpuMaxCtrl),
                  _numberField('CPU 步进', cpuStepCtrl),
                  _numberField('内存比例', memRatioCtrl),
                  _numberField('内存最小', memMinCtrl),
                  _numberField('内存最大', memMaxCtrl),
                  _numberField('磁盘最小', diskMinCtrl),
                  _numberField('磁盘最大', diskMaxCtrl),
                  _numberField('磁盘步进', diskStepCtrl),
                  _numberField('带宽最小', bwMinCtrl),
                  _numberField('带宽最大', bwMaxCtrl),
                  _numberField('带宽步进', bwStepCtrl),
                  _numberField('端口数', portCtrl),
                  TextField(
                    controller: cpuModelCtrl,
                    decoration: const InputDecoration(labelText: 'CPU 型号'),
                  ),
                  _numberField('价格倍率', priceMultiplierCtrl),
                  _numberField('总成本', totalCostCtrl),
                  _numberField('总核数', totalCoresCtrl),
                  _numberField('总内存', totalMemCtrl),
                  _numberField('总磁盘', totalDiskCtrl),
                  _numberField('总带宽', totalBwCtrl),
                  SwitchListTile(
                    contentPadding: EdgeInsets.zero,
                    value: active,
                    title: const Text('启用'),
                    onChanged: (val) => setState(() => active = val),
                  ),
                  SwitchListTile(
                    contentPadding: EdgeInsets.zero,
                    value: visible,
                    title: const Text('可见'),
                    onChanged: (val) => setState(() => visible = val),
                  ),
                  SwitchListTile(
                    contentPadding: EdgeInsets.zero,
                    value: overcommit,
                    title: const Text('开启超售'),
                    onChanged: (val) => setState(() => overcommit = val),
                  ),
                  if (overcommit) _numberField('超售比例', overcommitRatioCtrl),
                  const SizedBox(height: 12),
                  if (_generatedPackages.isNotEmpty) ...[
                    Align(
                      alignment: Alignment.centerLeft,
                      child: Text('预览 (${_generatedPackages.length} 条)'),
                    ),
                    const SizedBox(height: 8),
                    ..._generatedPackages
                        .take(6)
                        .map(
                          (item) => Padding(
                            padding: const EdgeInsets.only(bottom: 6),
                            child: Text(
                              '${item['name']}  月费 ${item['monthly_price']}',
                            ),
                          ),
                        ),
                    const SizedBox(height: 8),
                    FilledButton(
                      onPressed: () async {
                        await _applyGenerated();
                        setState(() {});
                      },
                      child: const Text('应用生成'),
                    ),
                  ],
                ],
              ),
            ),
          ),
          actions: [
            TextButton(
              onPressed: () {
                Navigator.pop(context);
              },
              child: const Text('关闭'),
            ),
            FilledButton(
              onPressed: () {
                syncBatchForm();
                _generatePackages();
                setState(() {});
              },
              child: const Text('生成预览'),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildSectionHeader(
    BuildContext context, {
    required String title,
    String? subtitle,
    List<Widget> actions = const [],
    Widget? leading,
  }) {
    final compact = _isCompact(context);
    final titleWidget = Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(title, style: Theme.of(context).textTheme.titleLarge),
        if (subtitle != null)
          Text(subtitle, style: const TextStyle(color: Colors.black54)),
      ],
    );

    if (compact) {
      return Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (leading != null) leading,
          titleWidget,
          const SizedBox(height: 12),
          _actionRow(actions),
        ],
      );
    }

    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (leading != null) ...[leading, const SizedBox(width: 12)],
        Expanded(child: titleWidget),
        _actionRow(actions),
      ],
    );
  }

  Widget _buildGoodsTypesTab(BuildContext context) {
    final compact = _isCompact(context);
    if (compact) {
      return ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _buildSectionHeader(
            context,
            title: '商品类型',
            subtitle: '维护商品类型与自动化插件',
            actions: [
              FilledButton(
                onPressed: () => _openGoodsTypeEditor(),
                child: const Text('新增商品类型'),
              ),
            ],
          ),
          const SizedBox(height: 16),
          ..._goodsTypes.map(
            (item) => _mobileCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Expanded(
                        child: Text(
                          item['name']?.toString() ?? '-',
                          style: const TextStyle(
                            fontSize: 16,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ),
                      _statusTag(_bool(item['active'])),
                    ],
                  ),
                  const SizedBox(height: 8),
                  _kvRow('代码', _str(item['code'])),
                  _kvRow('排序', _str(item['sort_order'])),
                  _kvRow('插件', _str(item['automation_plugin_id'])),
                  _kvRow('实例', _str(item['automation_instance_id'])),
                  const SizedBox(height: 8),
                  _actionRow([
                    OutlinedButton(
                      onPressed: () => _openGoodsTypeEditor(item),
                      child: const Text('编辑'),
                    ),
                    OutlinedButton(
                      onPressed: () => _syncGoodsType(item),
                      child: const Text('同步'),
                    ),
                    OutlinedButton(
                      onPressed: () => _deleteGoodsType(item),
                      style: OutlinedButton.styleFrom(
                        foregroundColor: Colors.red,
                      ),
                      child: const Text('删除'),
                    ),
                  ]),
                ],
              ),
            ),
          ),
        ],
      );
    }

    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        _buildSectionHeader(
          context,
          title: '商品类型',
          subtitle: '维护商品类型与自动化插件',
          actions: [
            FilledButton(
              onPressed: () => _openGoodsTypeEditor(),
              child: const Text('新增商品类型'),
            ),
          ],
        ),
        const SizedBox(height: 16),
        Card(
          child: SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            child: DataTable(
              columns: const [
                DataColumn(label: Text('ID')),
                DataColumn(label: Text('名称')),
                DataColumn(label: Text('代码')),
                DataColumn(label: Text('排序')),
                DataColumn(label: Text('插件')),
                DataColumn(label: Text('实例')),
                DataColumn(label: Text('状态')),
                DataColumn(label: Text('操作')),
              ],
              rows: _goodsTypes.map((item) {
                return DataRow(
                  cells: [
                    DataCell(Text(_str(item['id']))),
                    DataCell(Text(_str(item['name']))),
                    DataCell(Text(_str(item['code']))),
                    DataCell(Text(_str(item['sort_order']))),
                    DataCell(Text(_str(item['automation_plugin_id']))),
                    DataCell(Text(_str(item['automation_instance_id']))),
                    DataCell(_statusTag(_bool(item['active']))),
                    DataCell(
                      _actionRow([
                        TextButton(
                          onPressed: () => _openGoodsTypeEditor(item),
                          child: const Text('编辑'),
                        ),
                        TextButton(
                          onPressed: () => _syncGoodsType(item),
                          child: const Text('同步'),
                        ),
                        TextButton(
                          onPressed: () => _deleteGoodsType(item),
                          style: TextButton.styleFrom(
                            foregroundColor: Colors.red,
                          ),
                          child: const Text('删除'),
                        ),
                      ]),
                    ),
                  ],
                );
              }).toList(),
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildRegionsTab(BuildContext context) {
    final compact = _isCompact(context);
    final actions = [
      OutlinedButton(
        onPressed: _selectedRegionIds.isEmpty ? null : _bulkDeleteRegions,
        style: OutlinedButton.styleFrom(foregroundColor: Colors.red),
        child: const Text('批量删除'),
      ),
      FilledButton(
        onPressed: () => _openRegionEditor(),
        child: const Text('新增地区'),
      ),
    ];

    if (compact) {
      return ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _buildSectionHeader(
            context,
            title: '地区',
            subtitle: '维护区域与商品类型关联',
            actions: actions,
          ),
          const SizedBox(height: 16),
          ..._regions.map((item) {
            final id = item['id'] as int?;
            if (id == null) return const SizedBox.shrink();
            final selected = _selectedRegionIds.contains(id);
            return _mobileCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Checkbox(
                        value: selected,
                        onChanged: (val) => _toggleSelection(
                          _selectedRegionIds,
                          id,
                          val ?? false,
                        ),
                      ),
                      Expanded(
                        child: Text(
                          item['name']?.toString() ?? '-',
                          style: const TextStyle(
                            fontSize: 16,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ),
                      _statusTag(_bool(item['active'])),
                    ],
                  ),
                  _kvRow('代码', _str(item['code'])),
                  _kvRow(
                    '商品类型',
                    _goodsTypeNameById(_int(item['goods_type_id'])),
                  ),
                  const SizedBox(height: 8),
                  _actionRow([
                    OutlinedButton(
                      onPressed: () => _openRegionEditor(item),
                      child: const Text('编辑'),
                    ),
                    OutlinedButton(
                      onPressed: () => _deleteRegion(item),
                      style: OutlinedButton.styleFrom(
                        foregroundColor: Colors.red,
                      ),
                      child: const Text('删除'),
                    ),
                  ]),
                ],
              ),
            );
          }).toList(),
        ],
      );
    }

    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        _buildSectionHeader(
          context,
          title: '地区',
          subtitle: '维护区域与商品类型关联',
          actions: actions,
        ),
        const SizedBox(height: 16),
        Card(
          child: SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            child: DataTable(
              columns: const [
                DataColumn(label: Text('选择')),
                DataColumn(label: Text('ID')),
                DataColumn(label: Text('名称')),
                DataColumn(label: Text('代码')),
                DataColumn(label: Text('状态')),
                DataColumn(label: Text('操作')),
              ],
              rows: _regions.map((item) {
                final id = item['id'] as int?;
                if (id == null) {
                  return const DataRow(
                    cells: [
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                    ],
                  );
                }
                return DataRow(
                  cells: [
                    DataCell(
                      Checkbox(
                        value: _selectedRegionIds.contains(id),
                        onChanged: (val) => _toggleSelection(
                          _selectedRegionIds,
                          id,
                          val ?? false,
                        ),
                      ),
                    ),
                    DataCell(Text(_str(item['id']))),
                    DataCell(Text(_str(item['name']))),
                    DataCell(Text(_str(item['code']))),
                    DataCell(_statusTag(_bool(item['active']))),
                    DataCell(
                      _actionRow([
                        TextButton(
                          onPressed: () => _openRegionEditor(item),
                          child: const Text('编辑'),
                        ),
                        TextButton(
                          onPressed: () => _deleteRegion(item),
                          style: TextButton.styleFrom(
                            foregroundColor: Colors.red,
                          ),
                          child: const Text('删除'),
                        ),
                      ]),
                    ),
                  ],
                );
              }).toList(),
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildLinesTab(BuildContext context) {
    final compact = _isCompact(context);
    final actions = [
      OutlinedButton(
        onPressed: _selectedLineIds.isEmpty ? null : _bulkDeleteLines,
        style: OutlinedButton.styleFrom(foregroundColor: Colors.red),
        child: const Text('批量删除'),
      ),
      FilledButton(
        onPressed: () => _openLineEditor(),
        child: const Text('新增线路'),
      ),
    ];

    if (compact) {
      return ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _buildSectionHeader(
            context,
            title: '线路/附加项',
            subtitle: '维护线路成本、范围与可见性',
            actions: actions,
          ),
          const SizedBox(height: 16),
          ..._lines.map((item) {
            final id = item['id'] as int?;
            if (id == null) return const SizedBox.shrink();
            final selected = _selectedLineIds.contains(id);
            return _mobileCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Checkbox(
                        value: selected,
                        onChanged: (val) => _toggleSelection(
                          _selectedLineIds,
                          id,
                          val ?? false,
                        ),
                      ),
                      Expanded(
                        child: Text(
                          item['name']?.toString() ?? '-',
                          style: const TextStyle(
                            fontSize: 16,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ),
                      _statusTag(_bool(item['active'])),
                    ],
                  ),
                  _kvRow('地区', _regionNameById(_int(item['region_id']))),
                  _kvRow('云线路 ID', _str(item['line_id'])),
                  _kvRow(
                    '单价',
                    'CPU ${_str(item['unit_core'])} / 内存 ${_str(item['unit_mem'])} / 磁盘 ${_str(item['unit_disk'])} / 带宽 ${_str(item['unit_bw'])}',
                  ),
                  _kvRow('可见性', _bool(item['visible']) ? '可见' : '隐藏'),
                  _kvRow('余量', _formatCapacity(item['capacity_remaining'])),
                  const SizedBox(height: 8),
                  _actionRow([
                    OutlinedButton(
                      onPressed: () => _openLineEditor(item),
                      child: const Text('编辑'),
                    ),
                    OutlinedButton(
                      onPressed: () => _deleteLine(item),
                      style: OutlinedButton.styleFrom(
                        foregroundColor: Colors.red,
                      ),
                      child: const Text('删除'),
                    ),
                  ]),
                ],
              ),
            );
          }).toList(),
        ],
      );
    }

    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        _buildSectionHeader(
          context,
          title: '线路/附加项',
          subtitle: '维护线路成本、范围与可见性',
          actions: actions,
        ),
        const SizedBox(height: 16),
        Card(
          child: SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            child: DataTable(
              columns: const [
                DataColumn(label: Text('选择')),
                DataColumn(label: Text('ID')),
                DataColumn(label: Text('名称')),
                DataColumn(label: Text('地区')),
                DataColumn(label: Text('云线路')),
                DataColumn(label: Text('启用')),
                DataColumn(label: Text('可见')),
                DataColumn(label: Text('余量')),
                DataColumn(label: Text('操作')),
              ],
              rows: _lines.map((item) {
                final id = item['id'] as int?;
                if (id == null) {
                  return const DataRow(
                    cells: [
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                    ],
                  );
                }
                return DataRow(
                  cells: [
                    DataCell(
                      Checkbox(
                        value: _selectedLineIds.contains(id),
                        onChanged: (val) => _toggleSelection(
                          _selectedLineIds,
                          id,
                          val ?? false,
                        ),
                      ),
                    ),
                    DataCell(Text(_str(item['id']))),
                    DataCell(Text(_str(item['name']))),
                    DataCell(Text(_regionNameById(_int(item['region_id'])))),
                    DataCell(Text(_str(item['line_id']))),
                    DataCell(_statusTag(_bool(item['active']))),
                    DataCell(_visibilityTag(_bool(item['visible']))),
                    DataCell(_capacityTag(item['capacity_remaining'])),
                    DataCell(
                      _actionRow([
                        TextButton(
                          onPressed: () => _openLineEditor(item),
                          child: const Text('编辑'),
                        ),
                        TextButton(
                          onPressed: () => _deleteLine(item),
                          style: TextButton.styleFrom(
                            foregroundColor: Colors.red,
                          ),
                          child: const Text('删除'),
                        ),
                      ]),
                    ),
                  ],
                );
              }).toList(),
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildPackagesTab(BuildContext context) {
    final compact = _isCompact(context);
    final filterDropdown = DropdownButtonFormField<int?>(
      value: _packageLineId,
      decoration: const InputDecoration(labelText: '筛选线路'),
      items: [
        const DropdownMenuItem<int?>(value: null, child: Text('全部线路')),
        ..._lines.map(
          (item) => DropdownMenuItem<int?>(
            value: item['id'] as int,
            child: Text(item['name']?.toString() ?? '-'),
          ),
        ),
      ],
      onChanged: (val) => setState(() => _packageLineId = val),
    );

    final actions = [
      OutlinedButton(
        onPressed: _selectedPackageIds.isEmpty ? null : _bulkDeletePackages,
        style: OutlinedButton.styleFrom(foregroundColor: Colors.red),
        child: const Text('批量删除'),
      ),
      OutlinedButton(onPressed: _openBatchDialog, child: const Text('批量生成')),
      FilledButton(
        onPressed: () => _openPackageEditor(),
        child: const Text('新增套餐'),
      ),
    ];

    if (compact) {
      return ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _buildSectionHeader(
            context,
            title: '套餐',
            subtitle: '管理套餐规格与价格',
            actions: actions,
            leading: filterDropdown,
          ),
          const SizedBox(height: 16),
          ..._filteredPackages.map((item) {
            final id = item['id'] as int?;
            if (id == null) return const SizedBox.shrink();
            final selected = _selectedPackageIds.contains(id);
            return _mobileCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Checkbox(
                        value: selected,
                        onChanged: (val) => _toggleSelection(
                          _selectedPackageIds,
                          id,
                          val ?? false,
                        ),
                      ),
                      Expanded(
                        child: Text(
                          item['name']?.toString() ?? '-',
                          style: const TextStyle(
                            fontSize: 16,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ),
                      _statusTag(_bool(item['active'])),
                    ],
                  ),
                  _kvRow('线路', _lineNameById(_int(item['plan_group_id']))),
                  _kvRow(
                    '规格',
                    '${_str(item['cores'])}C / ${_str(item['memory_gb'])}G / ${_str(item['disk_gb'])}G / ${_str(item['bandwidth_mbps'])}M',
                  ),
                  _kvRow('月费', _str(item['monthly_price'])),
                  _kvRow('可见性', _bool(item['visible']) ? '可见' : '隐藏'),
                  _kvRow('余量', _formatCapacity(item['capacity_remaining'])),
                  const SizedBox(height: 8),
                  _actionRow([
                    OutlinedButton(
                      onPressed: () => _openPackageEditor(item),
                      child: const Text('编辑'),
                    ),
                    OutlinedButton(
                      onPressed: () => _deletePackage(item),
                      style: OutlinedButton.styleFrom(
                        foregroundColor: Colors.red,
                      ),
                      child: const Text('删除'),
                    ),
                  ]),
                ],
              ),
            );
          }).toList(),
        ],
      );
    }

    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        _buildSectionHeader(
          context,
          title: '套餐',
          subtitle: '管理套餐规格与价格',
          actions: actions,
          leading: SizedBox(width: 260, child: filterDropdown),
        ),
        const SizedBox(height: 16),
        Card(
          child: SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            child: DataTable(
              columns: const [
                DataColumn(label: Text('选择')),
                DataColumn(label: Text('ID')),
                DataColumn(label: Text('名称')),
                DataColumn(label: Text('线路')),
                DataColumn(label: Text('CPU')),
                DataColumn(label: Text('内存')),
                DataColumn(label: Text('磁盘')),
                DataColumn(label: Text('带宽')),
                DataColumn(label: Text('月费')),
                DataColumn(label: Text('启用')),
                DataColumn(label: Text('可见')),
                DataColumn(label: Text('余量')),
                DataColumn(label: Text('操作')),
              ],
              rows: _filteredPackages.map((item) {
                final id = item['id'] as int?;
                if (id == null) {
                  return const DataRow(
                    cells: [
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                    ],
                  );
                }
                return DataRow(
                  cells: [
                    DataCell(
                      Checkbox(
                        value: _selectedPackageIds.contains(id),
                        onChanged: (val) => _toggleSelection(
                          _selectedPackageIds,
                          id,
                          val ?? false,
                        ),
                      ),
                    ),
                    DataCell(Text(_str(item['id']))),
                    DataCell(Text(_str(item['name']))),
                    DataCell(Text(_lineNameById(_int(item['plan_group_id'])))),
                    DataCell(Text(_str(item['cores']))),
                    DataCell(Text(_str(item['memory_gb']))),
                    DataCell(Text(_str(item['disk_gb']))),
                    DataCell(Text(_str(item['bandwidth_mbps']))),
                    DataCell(Text(_str(item['monthly_price']))),
                    DataCell(_statusTag(_bool(item['active']))),
                    DataCell(_visibilityTag(_bool(item['visible']))),
                    DataCell(_capacityTag(item['capacity_remaining'])),
                    DataCell(
                      _actionRow([
                        TextButton(
                          onPressed: () => _openPackageEditor(item),
                          child: const Text('编辑'),
                        ),
                        TextButton(
                          onPressed: () => _deletePackage(item),
                          style: TextButton.styleFrom(
                            foregroundColor: Colors.red,
                          ),
                          child: const Text('删除'),
                        ),
                      ]),
                    ),
                  ],
                );
              }).toList(),
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildImagesTab(BuildContext context) {
    final compact = _isCompact(context);
    final lineDropdown = DropdownButtonFormField<int?>(
      value: _imageLineId,
      decoration: const InputDecoration(labelText: '选择同步线路'),
      items: _lines
          .map(
            (item) => DropdownMenuItem<int?>(
              value: item['id'] as int,
              child: Text(
                '${item['name'] ?? '-'} (${item['line_id'] ?? item['id']})',
              ),
            ),
          )
          .toList(),
      onChanged: (val) => setState(() => _imageLineId = val),
    );

    final actions = [
      OutlinedButton(
        onPressed: _selectedImageIds.isEmpty ? null : _bulkDeleteImages,
        style: OutlinedButton.styleFrom(foregroundColor: Colors.red),
        child: const Text('批量删除'),
      ),
      OutlinedButton(onPressed: _syncImages, child: const Text('同步镜像')),
      FilledButton(
        onPressed: () => _openImageEditor(),
        child: const Text('新增镜像'),
      ),
    ];

    if (compact) {
      return ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _buildSectionHeader(
            context,
            title: '系统镜像',
            subtitle: '维护镜像列表与启用状态',
            actions: actions,
            leading: lineDropdown,
          ),
          const SizedBox(height: 16),
          ..._filteredImages.map((item) {
            final id = item['id'] as int?;
            if (id == null) return const SizedBox.shrink();
            final selected = _selectedImageIds.contains(id);
            return _mobileCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Checkbox(
                        value: selected,
                        onChanged: (val) => _toggleSelection(
                          _selectedImageIds,
                          id,
                          val ?? false,
                        ),
                      ),
                      Expanded(
                        child: Text(
                          item['name']?.toString() ?? '-',
                          style: const TextStyle(
                            fontSize: 16,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ),
                      _statusTag(_bool(item['enabled'])),
                    ],
                  ),
                  _kvRow('镜像 ID', _str(item['image_id'])),
                  _kvRow('类型', _formatImageType(_str(item['type']))),
                  const SizedBox(height: 8),
                  _actionRow([
                    OutlinedButton(
                      onPressed: () => _openImageEditor(item),
                      child: const Text('编辑'),
                    ),
                    OutlinedButton(
                      onPressed: () => _deleteImage(item),
                      style: OutlinedButton.styleFrom(
                        foregroundColor: Colors.red,
                      ),
                      child: const Text('删除'),
                    ),
                  ]),
                ],
              ),
            );
          }).toList(),
        ],
      );
    }

    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        _buildSectionHeader(
          context,
          title: '系统镜像',
          subtitle: '维护镜像列表与启用状态',
          actions: actions,
          leading: SizedBox(width: 300, child: lineDropdown),
        ),
        const SizedBox(height: 16),
        Card(
          child: SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            child: DataTable(
              columns: const [
                DataColumn(label: Text('选择')),
                DataColumn(label: Text('ID')),
                DataColumn(label: Text('镜像 ID')),
                DataColumn(label: Text('名称')),
                DataColumn(label: Text('类型')),
                DataColumn(label: Text('启用')),
                DataColumn(label: Text('操作')),
              ],
              rows: _filteredImages.map((item) {
                final id = item['id'] as int?;
                if (id == null) {
                  return const DataRow(
                    cells: [
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                    ],
                  );
                }
                return DataRow(
                  cells: [
                    DataCell(
                      Checkbox(
                        value: _selectedImageIds.contains(id),
                        onChanged: (val) => _toggleSelection(
                          _selectedImageIds,
                          id,
                          val ?? false,
                        ),
                      ),
                    ),
                    DataCell(Text(_str(item['id']))),
                    DataCell(Text(_str(item['image_id']))),
                    DataCell(Text(_str(item['name']))),
                    DataCell(
                      _simpleTag(
                        _formatImageType(_str(item['type'])),
                        _imageTypeColor(_str(item['type'])),
                      ),
                    ),
                    DataCell(_statusTag(_bool(item['enabled']))),
                    DataCell(
                      _actionRow([
                        TextButton(
                          onPressed: () => _openImageEditor(item),
                          child: const Text('编辑'),
                        ),
                        TextButton(
                          onPressed: () => _deleteImage(item),
                          style: TextButton.styleFrom(
                            foregroundColor: Colors.red,
                          ),
                          child: const Text('删除'),
                        ),
                      ]),
                    ),
                  ],
                );
              }).toList(),
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildCyclesTab(BuildContext context) {
    final compact = _isCompact(context);
    final actions = [
      OutlinedButton(
        onPressed: _selectedCycleIds.isEmpty ? null : _bulkDeleteCycles,
        style: OutlinedButton.styleFrom(foregroundColor: Colors.red),
        child: const Text('批量删除'),
      ),
      FilledButton(
        onPressed: () => _openCycleEditor(),
        child: const Text('新增周期'),
      ),
    ];

    if (compact) {
      return ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _buildSectionHeader(
            context,
            title: '计费周期',
            subtitle: '维护周期倍率与范围',
            actions: actions,
          ),
          const SizedBox(height: 16),
          ..._billingCycles.map((item) {
            final id = item['id'] as int?;
            if (id == null) return const SizedBox.shrink();
            final selected = _selectedCycleIds.contains(id);
            return _mobileCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Checkbox(
                        value: selected,
                        onChanged: (val) => _toggleSelection(
                          _selectedCycleIds,
                          id,
                          val ?? false,
                        ),
                      ),
                      Expanded(
                        child: Text(
                          item['name']?.toString() ?? '-',
                          style: const TextStyle(
                            fontSize: 16,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ),
                      _statusTag(_bool(item['active'])),
                    ],
                  ),
                  _kvRow('月数', _str(item['months'])),
                  _kvRow('倍率', _str(item['multiplier'])),
                  _kvRow('最小数量', _str(item['min_qty'])),
                  _kvRow('最大数量', _str(item['max_qty'])),
                  const SizedBox(height: 8),
                  _actionRow([
                    OutlinedButton(
                      onPressed: () => _openCycleEditor(item),
                      child: const Text('编辑'),
                    ),
                    OutlinedButton(
                      onPressed: () => _deleteCycle(item),
                      style: OutlinedButton.styleFrom(
                        foregroundColor: Colors.red,
                      ),
                      child: const Text('删除'),
                    ),
                  ]),
                ],
              ),
            );
          }).toList(),
        ],
      );
    }

    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        _buildSectionHeader(
          context,
          title: '计费周期',
          subtitle: '维护周期倍率与范围',
          actions: actions,
        ),
        const SizedBox(height: 16),
        Card(
          child: SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            child: DataTable(
              columns: const [
                DataColumn(label: Text('选择')),
                DataColumn(label: Text('ID')),
                DataColumn(label: Text('名称')),
                DataColumn(label: Text('月数')),
                DataColumn(label: Text('倍率')),
                DataColumn(label: Text('最小数量')),
                DataColumn(label: Text('最大数量')),
                DataColumn(label: Text('启用')),
                DataColumn(label: Text('操作')),
              ],
              rows: _billingCycles.map((item) {
                final id = item['id'] as int?;
                if (id == null) {
                  return const DataRow(
                    cells: [
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                      DataCell(Text('-')),
                    ],
                  );
                }
                return DataRow(
                  cells: [
                    DataCell(
                      Checkbox(
                        value: _selectedCycleIds.contains(id),
                        onChanged: (val) => _toggleSelection(
                          _selectedCycleIds,
                          id,
                          val ?? false,
                        ),
                      ),
                    ),
                    DataCell(Text(_str(item['id']))),
                    DataCell(Text(_str(item['name']))),
                    DataCell(Text(_str(item['months']))),
                    DataCell(Text(_str(item['multiplier']))),
                    DataCell(Text(_str(item['min_qty']))),
                    DataCell(Text(_str(item['max_qty']))),
                    DataCell(_statusTag(_bool(item['active']))),
                    DataCell(
                      _actionRow([
                        TextButton(
                          onPressed: () => _openCycleEditor(item),
                          child: const Text('编辑'),
                        ),
                        TextButton(
                          onPressed: () => _deleteCycle(item),
                          style: TextButton.styleFrom(
                            foregroundColor: Colors.red,
                          ),
                          child: const Text('删除'),
                        ),
                      ]),
                    ),
                  ],
                );
              }).toList(),
            ),
          ),
        ),
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    final compact = _isCompact(context);
    final goodsTypeDropdown = DropdownButtonFormField<int>(
      value: _goodsTypeId,
      decoration: const InputDecoration(labelText: '商品类型'),
      items: _goodsTypes
          .map(
            (item) => DropdownMenuItem<int>(
              value: item['id'] as int,
              child: Text(item['name']?.toString() ?? '-'),
            ),
          )
          .toList(),
      onChanged: (val) async {
        setState(() => _goodsTypeId = val);
        await _loadAll();
      },
    );

    final syncButton = FilledButton(
      onPressed: _goodsTypeId == null ? null : _syncCurrentGoodsType,
      child: const Text('同步当前类型'),
    );

    return Scaffold(
      appBar: AppBar(title: const Text('售卖配置')),
      body: SafeArea(
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
              child: compact
                  ? Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          '地区、线路、套餐与计费策略维护',
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        const SizedBox(height: 12),
                        goodsTypeDropdown,
                        const SizedBox(height: 12),
                        syncButton,
                      ],
                    )
                  : Row(
                      children: [
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(
                                '售卖配置',
                                style: Theme.of(
                                  context,
                                ).textTheme.headlineSmall,
                              ),
                              const SizedBox(height: 4),
                              const Text(
                                '地区、线路、套餐与计费策略维护',
                                style: TextStyle(color: Colors.black54),
                              ),
                            ],
                          ),
                        ),
                        SizedBox(width: 260, child: goodsTypeDropdown),
                        const SizedBox(width: 12),
                        syncButton,
                      ],
                    ),
            ),
            if (_loading)
              const Padding(
                padding: EdgeInsets.only(top: 8),
                child: LinearProgressIndicator(minHeight: 2),
              ),
            const SizedBox(height: 8),
            TabBar(
              controller: _tabController,
              isScrollable: true,
              tabs: const [
                Tab(text: '商品类型'),
                Tab(text: '地区'),
                Tab(text: '线路/附加项'),
                Tab(text: '套餐'),
                Tab(text: '系统镜像'),
                Tab(text: '计费周期'),
              ],
            ),
            const SizedBox(height: 8),
            Expanded(
              child: TabBarView(
                controller: _tabController,
                children: [
                  _buildGoodsTypesTab(context),
                  _buildRegionsTab(context),
                  _buildLinesTab(context),
                  _buildPackagesTab(context),
                  _buildImagesTab(context),
                  _buildCyclesTab(context),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
