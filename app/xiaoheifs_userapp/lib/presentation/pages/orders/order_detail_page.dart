import 'dart:async';
import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../core/constants/app_colors.dart';
import '../../../core/constants/app_strings.dart';
import '../../../core/constants/input_limits.dart';
import '../../../core/config/api_config.dart';
import '../../../core/utils/date_formatter.dart';
import '../../../core/utils/money_formatter.dart';
import '../../providers/catalog_provider.dart';
import '../../providers/order_provider.dart';
import '../../providers/vps_provider.dart';
import '../../widgets/common/status_tag.dart';

class OrderDetailPage extends ConsumerStatefulWidget {
  final int id;
  const OrderDetailPage({super.key, required this.id});

  @override
  ConsumerState<OrderDetailPage> createState() => _OrderDetailPageState();
}

class _OrderDetailPageState extends ConsumerState<OrderDetailPage> {
  static final RegExp _moneyPattern = RegExp(r'^\d+(\.\d{1,2})?$');
  List<Map<String, dynamic>> _paymentProviders = [];
  Timer? _pollingTimer;
  bool _autoNavigated = false;
  ProviderSubscription<OrderDetailState>? _detailSub;

  String _resolveBackendOrigin() {
    final configured = Uri.tryParse(ApiConfig.baseUrl.trim());
    if (configured != null &&
        (configured.scheme == 'http' || configured.scheme == 'https') &&
        configured.host.isNotEmpty) {
      final port = configured.hasPort ? ':${configured.port}' : '';
      return '${configured.scheme}://${configured.host}$port';
    }
    final base = Uri.base;
    if (base.scheme == 'http' || base.scheme == 'https') {
      return base.origin;
    }
    return '';
  }

  @override
  void initState() {
    super.initState();
    Future.microtask(() async {
      await ref.read(orderDetailProvider.notifier).fetchDetail(widget.id);
      ref.read(catalogProvider.notifier).fetchCatalog();
      await _loadProviders();
    });
    _detailSub = ref.listenManual<OrderDetailState>(orderDetailProvider, (prev, next) {
      final isProv = _isProvisioning(next);
      if (isProv) {
        _startPolling();
      } else {
        _stopPolling();
      }
      if (!_autoNavigated) {
        _tryAutoNavigate(prev, next);
      }
    });
  }

  @override
  void dispose() {
    _stopPolling();
    _detailSub?.close();
    super.dispose();
  }

  Future<void> _loadProviders() async {
    try {
      final res = await ref.read(orderRepositoryProvider).listPaymentProviders();
      final items = res['items'];
      if (items is List) {
        setState(() {
          _paymentProviders =
              items.map((e) => e is Map<String, dynamic> ? e : <String, dynamic>{}).toList();
        });
      }
    } catch (_) {}
  }

  Map<String, dynamic>? _findProvider(String? key) {
    if (key == null) return null;
    for (final provider in _paymentProviders) {
      final providerKey = provider['key']?.toString() ?? provider['code']?.toString();
      if (providerKey == key) return provider;
    }
    return null;
  }

  List<Map<String, dynamic>> _normalizeSchemaFields(String? schemaJson) {
    if (schemaJson == null || schemaJson.isEmpty) return [];
    try {
      final parsed = jsonDecode(schemaJson);
      if (parsed is List) {
        return parsed.cast<Map<String, dynamic>>();
      }
      if (parsed is Map && parsed['fields'] is List) {
        return (parsed['fields'] as List).cast<Map<String, dynamic>>();
      }
      if (parsed is Map) {
        final props = parsed['properties'] is Map ? parsed['properties'] as Map : {};
        final required = parsed['required'] is List ? Set.from(parsed['required']) : <dynamic>{};
        return props.keys.map<Map<String, dynamic>>((key) {
          final prop = props[key] is Map ? props[key] as Map : {};
          final enumValues = prop['enum'] is List ? prop['enum'] as List : null;
          final type = enumValues != null
              ? 'select'
              : prop['format'] == 'password'
                  ? 'password'
                  : prop['format'] == 'textarea'
                      ? 'textarea'
                      : prop['type'] == 'number' || prop['type'] == 'integer'
                          ? 'number'
                          : prop['type'] == 'boolean'
                              ? 'boolean'
                              : 'text';
          return {
            'key': key,
            'label': prop['title'] ?? prop['label'] ?? key,
            'type': type,
            'required': required.contains(key),
            'placeholder': prop['description'] ?? prop['placeholder'] ?? '',
            'default': prop['default'],
            'options': enumValues != null
                ? enumValues.map((value) => {'label': value.toString(), 'value': value}).toList()
                : <Map<String, dynamic>>[],
          };
        }).toList();
      }
    } catch (_) {}
    return [];
  }

  String _providerInstructions(Map<String, dynamic>? provider) {
    final configJson = provider?['config_json']?.toString();
    if (configJson == null || configJson.isEmpty) return '';
    try {
      final parsed = jsonDecode(configJson);
      if (parsed is Map) {
        return parsed['instructions']?.toString() ?? parsed['notice']?.toString() ?? '';
      }
    } catch (_) {}
    return '';
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(orderDetailProvider);

    return Scaffold(
      body: state.loading
          ? const Center(child: CircularProgressIndicator())
          : state.error != null
              ? Center(child: Text(state.error!))
              : _buildContent(context, state),
    );
  }

  Widget _buildContent(BuildContext context, OrderDetailState state) {
    final order = state.order ?? {};
    final status = order['status'] ?? order['Status'] ?? '';
    final orderNo = order['order_no'] ?? order['OrderNo'] ?? '';
    final total = order['total_amount'] ?? order['TotalAmount'] ?? 0;
    final createdAt = order['created_at'] ?? order['CreatedAt'] ?? '';
    final totalValue = total is num ? total.toDouble() : double.tryParse(total.toString()) ?? 0;
    final amountColor = totalValue < 0 ? AppColors.success : AppColors.primary;
    final packages = ref.watch(catalogProvider).packages;
    final stepIndex = _orderStepIndex(status);

    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _buildHeader(orderNo, status),
          const SizedBox(height: 16),
          _buildStatsBanner(
            orderNo: orderNo.toString(),
            total: MoneyFormatter.format(total),
            createdAt: DateFormatter.formatIso(createdAt),
            amountColor: amountColor,
          ),
          const SizedBox(height: 16),
          _buildProgressSection(stepIndex),
          const SizedBox(height: 16),
          _buildActions(context, status, total),
          const SizedBox(height: 16),
          _buildItems(state.items, packages),
          const SizedBox(height: 16),
          _buildPayments(state.payments),
        ],
      ),
    );
  }

  Widget _buildHeader(String orderNo, String status) {
    return Row(
      children: [
        const Icon(Icons.shopping_cart, color: AppColors.primary),
        const SizedBox(width: 8),
        const Expanded(
          child: Text(
            '订单详情',
            style: TextStyle(fontSize: 18, fontWeight: FontWeight.w700),
          ),
        ),
        StatusTag.order(status),
      ],
    );
  }

  Widget _buildStatsBanner({
    required String orderNo,
    required String total,
    required String createdAt,
    required Color amountColor,
  }) {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppColors.darkSurface.withOpacity(0.5),
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: AppColors.gray700.withOpacity(0.3)),
      ),
      child: Row(
        children: [
          _statItem(
            Icons.description,
            '订单号',
            orderNo,
            onValueTap: orderNo.isEmpty ? null : () => _copyOrderNo(orderNo),
          ),
          _divider(),
          _statItem(Icons.payments, '订单金额', total, valueColor: amountColor, valueSize: 18),
          _divider(),
          _statItem(Icons.schedule, '创建时间', createdAt),
        ],
      ),
    );
  }

  Widget _statItem(
    IconData icon,
    String label,
    String value, {
    Color? valueColor,
    double valueSize = 14,
    VoidCallback? onValueTap,
  }) {
    return Expanded(
      child: Row(
        children: [
          Icon(icon, size: 18, color: AppColors.gray500),
          const SizedBox(width: 8),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(label, style: const TextStyle(fontSize: 12, color: AppColors.gray500)),
                const SizedBox(height: 4),
                GestureDetector(
                  onTap: onValueTap,
                  behavior: HitTestBehavior.opaque,
                  child: Row(
                    children: [
                      Expanded(
                        child: Text(
                          value.isEmpty ? '-' : value,
                          style: TextStyle(
                            fontWeight: FontWeight.w600,
                            fontSize: valueSize,
                            color: valueColor ?? AppColors.gray100,
                          ),
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                      if (onValueTap != null) ...[
                        const SizedBox(width: 4),
                        const Icon(Icons.copy, size: 14, color: AppColors.gray500),
                      ],
                    ],
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Future<void> _copyOrderNo(String orderNo) async {
    await Clipboard.setData(ClipboardData(text: orderNo));
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('订单号已复制')),
    );
  }

  Widget _divider() {
    return Container(
      width: 1,
      height: 36,
      margin: const EdgeInsets.symmetric(horizontal: 12),
      color: AppColors.gray700.withOpacity(0.4),
    );
  }

  Widget _buildProgressSection(int stepIndex) {
    final steps = const [
      '草稿',
      '待支付',
      '待审核',
      '已通过',
      '开通中',
      '已生效',
    ];
    final current = stepIndex.clamp(0, steps.length - 1);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const Text('订单进度', style: TextStyle(fontWeight: FontWeight.w700)),
                const Spacer(),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                  decoration: BoxDecoration(
                    color: AppColors.primary.withOpacity(0.15),
                    borderRadius: BorderRadius.circular(999),
                  ),
                  child: Text(
                    '当前：${steps[current]}',
                    style: const TextStyle(
                      color: AppColors.primary,
                      fontSize: 12,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 14),
            Row(
              children: List.generate(steps.length * 2 - 1, (i) {
                if (i.isOdd) {
                  final connectorIndex = i ~/ 2;
                  final done = connectorIndex < current;
                  return Expanded(
                    child: Container(
                      height: 2,
                      margin: const EdgeInsets.only(bottom: 18),
                      color: done ? AppColors.success : AppColors.gray700,
                    ),
                  );
                }
                final index = i ~/ 2;
                final isDone = index < current;
                final isCurrent = index == current;
                return Expanded(
                  child: _progressNode(
                    label: steps[index],
                    index: index,
                    isDone: isDone,
                    isCurrent: isCurrent,
                  ),
                );
              }),
            ),
          ],
        ),
      ),
    );
  }

  Widget _progressNode({
    required String label,
    required int index,
    required bool isDone,
    required bool isCurrent,
  }) {
    final bg = isDone
        ? AppColors.success
        : isCurrent
            ? AppColors.primary
            : AppColors.darkSurface.withOpacity(0.8);
    final border = isDone
        ? AppColors.success
        : isCurrent
            ? AppColors.primary
            : AppColors.gray700;
    final labelColor = isDone
        ? AppColors.success
        : isCurrent
            ? AppColors.primary
            : AppColors.gray500;
    return Column(
      children: [
        Container(
          width: 28,
          height: 28,
          decoration: BoxDecoration(
            color: bg,
            shape: BoxShape.circle,
            border: Border.all(color: border),
          ),
          alignment: Alignment.center,
          child: isDone
              ? const Icon(Icons.check, size: 16, color: Colors.white)
              : Text(
                  '${index + 1}',
                  style: TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w700,
                    color: isCurrent ? Colors.white : AppColors.gray400,
                  ),
                ),
        ),
        const SizedBox(height: 8),
        Text(
          label,
          style: TextStyle(
            fontSize: 12,
            fontWeight: isCurrent ? FontWeight.w700 : FontWeight.w500,
            color: labelColor,
          ),
        ),
      ],
    );
  }


  Widget _buildActions(BuildContext context, String status, dynamic total) {
    return Wrap(
      spacing: 12,
      runSpacing: 8,
      children: [
        ElevatedButton.icon(
          onPressed: () => ref.read(orderDetailProvider.notifier).refresh(widget.id),
          icon: const Icon(Icons.refresh),
          label: const Text(AppStrings.refresh),
        ),
        OutlinedButton.icon(
          onPressed: status == 'pending_payment' || status == 'pending_review'
              ? () => _cancelOrder(context)
              : null,
          icon: const Icon(Icons.cancel),
          label: const Text(AppStrings.cancelOrder),
        ),
        ElevatedButton.icon(
          onPressed: status == 'pending_payment' ? () => _openPayDialog(context, total) : null,
          icon: const Icon(Icons.payments),
          label: const Text(AppStrings.payNow),
        ),
      ],
    );
  }

  Widget _buildItems(List<Map<String, dynamic>> items, List<Map<String, dynamic>> packages) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const Icon(Icons.list_alt, size: 18, color: AppColors.primary),
                const SizedBox(width: 6),
                const Text('商品详情', style: TextStyle(fontWeight: FontWeight.bold)),
                const Spacer(),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                  decoration: BoxDecoration(
                    color: AppColors.primary.withOpacity(0.15),
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: Text('${items.length} 个商品', style: const TextStyle(fontSize: 12)),
                ),
              ],
            ),
            const SizedBox(height: 12),
            if (items.isEmpty)
              const Text('暂无订单项')
            else
              ...items.map((item) {
                final name = item['name'] ?? item['Name'] ?? item['product_name'] ?? '商品';
                final amount = item['amount'] ?? item['price'] ?? 0;
                final qty = item['qty'] ?? item['quantity'] ?? 1;
                final status = item['status'] ?? item['Status'] ?? '';
                final specText = _formatSpec(item, packages);
                return ListTile(
                  contentPadding: const EdgeInsets.symmetric(vertical: 6, horizontal: 8),
                  title: Text('$name', style: const TextStyle(fontWeight: FontWeight.w600)),
                  subtitle: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const SizedBox(height: 4),
                      Text(specText, style: const TextStyle(color: AppColors.gray500)),
                      const SizedBox(height: 4),
                      Text('数量: $qty', style: const TextStyle(color: AppColors.gray500)),
                    ],
                  ),
                  trailing: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    crossAxisAlignment: CrossAxisAlignment.end,
                    children: [
                      Text(MoneyFormatter.format(amount),
                          style: const TextStyle(fontWeight: FontWeight.w600)),
                      const SizedBox(height: 6),
                      StatusTag.order(status.toString()),
                    ],
                  ),
                );
              }),
          ],
        ),
      ),
    );
  }

  Widget _buildPayments(List<Map<String, dynamic>> payments) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('支付记录', style: TextStyle(fontWeight: FontWeight.bold)),
            const SizedBox(height: 12),
            if (payments.isEmpty)
              const Text('暂无支付记录')
            else
              ...payments.map((p) {
                final method = p['method'] ?? p['Method'] ?? '';
                final amount = p['amount'] ?? p['Amount'] ?? 0;
                final status = p['status'] ?? p['Status'] ?? '';
                return ListTile(
                  title: Text('$method'),
                  subtitle: Text('状态: $status'),
                  trailing: Text(MoneyFormatter.format(amount)),
                );
              }),
          ],
        ),
      ),
    );
  }

  bool _isProvisioning(OrderDetailState state) {
    final status = state.order?['status'] ?? state.order?['Status'] ?? '';
    if (status.toString().toLowerCase() == 'provisioning') return true;
    for (final item in state.items) {
      final itemStatus = item['status'] ?? item['Status'] ?? '';
      if (itemStatus.toString().toLowerCase() == 'provisioning') return true;
    }
    return false;
  }

  void _startPolling() {
    if (_pollingTimer != null) return;
    _pollingTimer = Timer.periodic(const Duration(seconds: 3), (_) {
      ref.read(orderDetailProvider.notifier).fetchDetail(widget.id);
    });
  }

  void _stopPolling() {
    _pollingTimer?.cancel();
    _pollingTimer = null;
  }

  Future<void> _tryAutoNavigate(OrderDetailState? prev, OrderDetailState next) async {
    final prevStatus = prev?.order?['status'] ?? prev?.order?['Status'];
    final nextStatus = next.order?['status'] ?? next.order?['Status'];
    if (prevStatus?.toString() != 'provisioning' || nextStatus?.toString() != 'active') {
      return;
    }
    if (next.items.length != 1) return;
    final orderItemId = next.items.first['id'] ?? next.items.first['ID'];
    if (orderItemId == null) return;
    try {
      final vpsList = await ref.read(vpsRepositoryProvider).listVps();
      final vps = vpsList.firstWhere(
        (row) => row['order_item_id']?.toString() == orderItemId.toString(),
        orElse: () => {},
      );
      final vpsId = vps['id'] ?? vps['ID'];
      if (vpsId == null || _autoNavigated) return;
      _autoNavigated = true;
      _stopPolling();
      if (mounted) {
        context.go('/console/vps/$vpsId');
      }
    } catch (_) {}
  }

  int _orderStepIndex(String? status) {
    final steps = [
      'draft',
      'pending_payment',
      'pending_review',
      'approved',
      'provisioning',
      'active',
    ];
    final idx = steps.indexOf((status ?? '').toLowerCase());
    return idx < 0 ? 0 : idx;
  }

  Map<String, dynamic>? _findPackage(List<Map<String, dynamic>> packages, dynamic packageId) {
    if (packageId == null) return null;
    final pid = packageId.toString();
    for (final pkg in packages) {
      if ((pkg['id'] ?? pkg['ID'] ?? '').toString() == pid) return pkg;
    }
    return null;
  }

  String _formatSpec(Map<String, dynamic> item, List<Map<String, dynamic>> packages) {
    final rawSpec = item['spec'] ?? item['Spec'];
    Map<String, dynamic> specMap = {};
    if (rawSpec is String) {
      try {
        specMap = jsonDecode(rawSpec) is Map ? Map<String, dynamic>.from(jsonDecode(rawSpec)) : {};
      } catch (_) {}
    } else if (rawSpec is Map) {
      specMap = rawSpec.cast<String, dynamic>();
    }

    final packageId = item['package_id'] ?? item['PackageID'];
    final pkg = _findPackage(packages, packageId);
    final baseCores = int.tryParse('${pkg?['cores'] ?? pkg?['cpu'] ?? 0}') ?? 0;
    final baseMem = int.tryParse('${pkg?['memory_gb'] ?? pkg?['mem_gb'] ?? 0}') ?? 0;
    final baseDisk = int.tryParse('${pkg?['disk_gb'] ?? 0}') ?? 0;
    final baseBw = int.tryParse('${pkg?['bandwidth_mbps'] ?? 0}') ?? 0;

    final addCores = int.tryParse('${specMap['add_cores'] ?? 0}') ?? 0;
    final addMem = int.tryParse('${specMap['add_mem_gb'] ?? 0}') ?? 0;
    final addDisk = int.tryParse('${specMap['add_disk_gb'] ?? 0}') ?? 0;
    final addBw = int.tryParse('${specMap['add_bw_mbps'] ?? 0}') ?? 0;

    final totalCores = baseCores + addCores;
    final totalMem = baseMem + addMem;
    final totalDisk = baseDisk + addDisk;
    final totalBw = baseBw + addBw;

    final duration =
        specMap['duration_months'] ?? item['duration_months'] ?? item['DurationMonths'];

    final parts = <String>[];
    if (totalCores > 0 || totalMem > 0 || totalDisk > 0 || totalBw > 0) {
      parts.add('CPU $totalCores');
      parts.add('内存 ${totalMem}G');
      parts.add('磁盘 ${totalDisk}G');
      parts.add('带宽 ${totalBw}M');
    }
    if (duration != null && duration.toString().isNotEmpty) {
      parts.add('时长 $duration 个月');
    }
    return parts.isEmpty ? '-' : parts.join(' / ');
  }

  Future<void> _cancelOrder(BuildContext context) async {
    try {
      await ref.read(orderRepositoryProvider).cancelOrder(widget.id);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('订单已取消')));
        await ref.read(orderDetailProvider.notifier).fetchDetail(widget.id);
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(e.toString())));
      }
    }
  }

  Future<void> _openPayDialog(BuildContext context, dynamic total) async {
    String? method;
    final amountController = TextEditingController(text: '${total ?? 0}');
    final noteController = TextEditingController();
    final Map<String, dynamic> extraValues = {};
    final Map<String, TextEditingController> fieldControllers = {};
    List<Map<String, dynamic>> schemaFields = [];
    String instructions = '';

    int? schemaFieldLimit(String key) {
      final lower = key.toLowerCase();
      if (lower == 'method') return InputLimits.paymentMethod;
      if (lower == 'note') return InputLimits.paymentNote;
      if (lower == 'approval' || lower.contains('approval') || lower.contains('reason')) {
        return InputLimits.approval;
      }
      if (lower == 'screenshot_url') return InputLimits.screenshotUrl;
      return null;
    }

    void setProviderFields(String? key) {
      method = key;
      final provider = _findProvider(key);
      instructions = _providerInstructions(provider);
      if (provider != null &&
          ['approval', 'balance', 'custom', 'yipay'].contains(provider['key'])) {
        schemaFields = [];
      } else {
        schemaFields = _normalizeSchemaFields(provider?['schema_json']?.toString());
      }
      fieldControllers.clear();
      extraValues.clear();
      for (final field in schemaFields) {
        final fieldKey = field['key']?.toString() ?? '';
        final defaultValue = field['default'];
        extraValues[fieldKey] = defaultValue;
        if (field['type'] != 'boolean' && field['type'] != 'select') {
          fieldControllers[fieldKey] =
              TextEditingController(text: defaultValue?.toString() ?? '');
        }
      }
    }

    await showDialog(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setModalState) => AlertDialog(
          title: const Text('发起支付'),
          content: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                DropdownButtonFormField<String>(
                  value: method,
                  decoration: const InputDecoration(labelText: '支付方式'),
                  items: _paymentProviders
                      .map((e) => DropdownMenuItem<String>(
                            value: e['key']?.toString() ?? e['code']?.toString(),
                            child: Text(e['name']?.toString() ?? e['label']?.toString() ?? '方式'),
                          ))
                      .toList(),
                  onChanged: (value) {
                    setModalState(() {
                      setProviderFields(value);
                    });
                  },
                ),
                if (instructions.isNotEmpty) ...[
                  const SizedBox(height: 8),
                  Container(
                    width: double.infinity,
                    padding: const EdgeInsets.all(8),
                    decoration: BoxDecoration(
                      color: AppColors.primary.withOpacity(0.08),
                      borderRadius: BorderRadius.circular(6),
                    ),
                    child: Text(instructions),
                  ),
                ],
                const SizedBox(height: 12),
                TextField(
                  controller: amountController,
                  keyboardType: TextInputType.number,
                  decoration: const InputDecoration(labelText: '支付金额'),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: noteController,
                  maxLength: InputLimits.paymentNote,
                  decoration: const InputDecoration(labelText: '备注'),
                ),
                if (schemaFields.isNotEmpty) const SizedBox(height: 12),
                ...schemaFields.map((field) {
                  final fieldKey = field['key']?.toString() ?? '';
                  final fieldLabel = field['label']?.toString() ?? fieldKey;
                  final fieldType = field['type']?.toString() ?? 'text';
                  final fieldPlaceholder = field['placeholder']?.toString() ?? '';
                  final isRequired = field['required'] == true;
                  if (fieldType == 'select') {
                    final options = field['options'] is List ? field['options'] as List : [];
                    return Padding(
                      padding: const EdgeInsets.only(bottom: 12),
                      child: DropdownButtonFormField<dynamic>(
                        value: extraValues[fieldKey],
                        decoration: InputDecoration(labelText: fieldLabel),
                        items: options
                            .map((opt) => DropdownMenuItem<dynamic>(
                                  value: opt['value'],
                                  child: Text(opt['label']?.toString() ?? opt['value']?.toString() ?? ''),
                                ))
                            .toList(),
                        onChanged: (value) => setModalState(() {
                          extraValues[fieldKey] = value;
                        }),
                      ),
                    );
                  }
                  if (fieldType == 'boolean') {
                    final value = extraValues[fieldKey] == true;
                    return SwitchListTile(
                      contentPadding: EdgeInsets.zero,
                      title: Text(fieldLabel),
                      value: value,
                      onChanged: (val) => setModalState(() {
                        extraValues[fieldKey] = val;
                      }),
                    );
                  }

                  final controller = fieldControllers[fieldKey] ??
                      TextEditingController(text: extraValues[fieldKey]?.toString() ?? '');
                  fieldControllers[fieldKey] = controller;
                  final maxLength = schemaFieldLimit(fieldKey);
                  return Padding(
                    padding: const EdgeInsets.only(bottom: 12),
                    child: TextField(
                      controller: controller,
                      keyboardType: fieldType == 'number'
                          ? TextInputType.number
                          : TextInputType.text,
                      obscureText: fieldType == 'password',
                      maxLines: fieldType == 'textarea' ? 3 : 1,
                      maxLength: maxLength,
                      decoration: InputDecoration(
                        labelText: isRequired ? '$fieldLabel *' : fieldLabel,
                        hintText: fieldPlaceholder,
                      ),
                      onChanged: (value) => extraValues[fieldKey] = value,
                    ),
                  );
                }).toList(),
              ],
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text(AppStrings.cancel),
            ),
            TextButton(
              onPressed: () async {
                if (method == null || method!.isEmpty) {
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('请选择支付方式')),
                  );
                  return;
                }
                final amountText = amountController.text.trim();
                if (!_moneyPattern.hasMatch(amountText)) {
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('金额格式不正确，最多保留2位小数')),
                  );
                  return;
                }
                final amount = double.parse(amountText);
                if (amount <= 0) {
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('请输入有效金额')),
                  );
                  return;
                }
                final note = noteController.text.trim();
                if (runeLength(note) > InputLimits.paymentNote) {
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('备注长度不能超过 500 个字符')),
                  );
                  return;
                }
                for (final field in schemaFields) {
                  final key = field['key']?.toString() ?? '';
                  final limit = schemaFieldLimit(key);
                  final val = extraValues[key];
                  if (limit != null && val is String && runeLength(val.trim()) > limit) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      SnackBar(content: Text('${field['label'] ?? key}长度不能超过 $limit 个字符')),
                    );
                    return;
                  }
                  if (field['required'] == true) {
                    final val = extraValues[key];
                    if (val == null || (val is String && val.trim().isEmpty)) {
                      ScaffoldMessenger.of(context).showSnackBar(
                        SnackBar(content: Text('请填写${field['label'] ?? key}')),
                      );
                      return;
                    }
                  }
                }
                try {
                  if (method == 'approval') {
                    await ref.read(orderRepositoryProvider).submitOrderPayment(
                          widget.id,
                          {
                            'method': method,
                            'amount': amount,
                            if (note.isNotEmpty) 'note': note,
                          },
                          idempotencyKey: 'pay-${DateTime.now().millisecondsSinceEpoch}',
                        );
                    if (context.mounted) {
                      Navigator.pop(context);
                      ScaffoldMessenger.of(context).showSnackBar(
                        const SnackBar(content: Text('已提交审核支付信息')),
                      );
                      await ref.read(orderDetailProvider.notifier).fetchDetail(widget.id);
                    }
                    return;
                  }

                  final extra = <String, dynamic>{};
                  for (final field in schemaFields) {
                    final key = field['key']?.toString() ?? '';
                    final value = extraValues[key];
                    if (value != null && (!(value is String) || value.trim().isNotEmpty)) {
                      extra[key] = value;
                    }
                  }

                  final base = _resolveBackendOrigin();
                  if (base.isEmpty) {
                    throw Exception('无法确定后端地址，请先在设置中配置 API 地址');
                  }
                  final payload = {
                    'method': method,
                    'return_url': '$base/console/orders/${widget.id}',
                    'notify_url': '$base/api/v1/payments/notify/$method',
                    'extra': extra,
                  };
                  final res =
                      await ref.read(orderRepositoryProvider).createOrderPayment(widget.id, payload);
                  final result = res['data'] ?? res;
                  if (context.mounted) {
                    Navigator.pop(context);
                    await _handlePaymentResult(context, method ?? '', result);
                    await ref.read(orderDetailProvider.notifier).fetchDetail(widget.id);
                  }
                } catch (e) {
                  if (context.mounted) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      SnackBar(content: Text(e.toString())),
                    );
                  }
                }
              },
              child: const Text(AppStrings.confirm),
            ),
          ],
        ),
      ),
    );

    for (final controller in fieldControllers.values) {
      controller.dispose();
    }
  }

  Future<void> _handlePaymentResult(
      BuildContext context, String method, Map<String, dynamic> result) async {
    final extra = result['extra'] is Map<String, dynamic>
        ? result['extra'] as Map<String, dynamic>
        : <String, dynamic>{};
    final payKind = (extra['pay_kind'] ?? '').toString().toLowerCase();
    final payUrl = extra['code_url'] ?? result['pay_url'] ?? result['payUrl'] ?? result['url'];
    final urlScheme = extra['urlscheme'] ?? result['urlscheme'];
    final instructions = extra['instructions']?.toString();

    // balance/approval 不跳转 payUrl，直接提示结果
    if (method == 'balance' || method == 'approval') {
      if (instructions != null && instructions.isNotEmpty && context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(instructions)),
        );
      }
      return;
    }

    String manualLink = '';
    if (payKind == 'urlscheme' && urlScheme != null && urlScheme.toString().isNotEmpty) {
      manualLink = urlScheme.toString();
    } else if (payKind == 'redirect' &&
        extra['pay_url'] != null &&
        extra['pay_url'].toString().isNotEmpty) {
      manualLink = extra['pay_url'].toString();
    } else if (payUrl != null && payUrl.toString().isNotEmpty) {
      manualLink = payUrl.toString();
    }

    if (manualLink.isNotEmpty) {
      await _showPaymentCopyDialog(context, manualLink);
      return;
    }

    if (context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('支付地址为空')),
      );
    }
    return;
  }

  Future<void> _showPaymentCopyDialog(BuildContext context, String link) async {
    final text = link.trim();
    if (text.isEmpty) return;
    await showDialog(
      context: context,
      builder: (dialogContext) {
        return AlertDialog(
          title: const Text('复制支付链接'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text('请复制下方链接，发送到微信/对应App聊天框后点击打开完成支付。'),
              const SizedBox(height: 12),
              SelectableText(text),
            ],
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(dialogContext).pop(),
              child: const Text('关闭'),
            ),
            FilledButton(
              onPressed: () async {
                await Clipboard.setData(ClipboardData(text: text));
                if (dialogContext.mounted) {
                  ScaffoldMessenger.of(dialogContext).showSnackBar(
                    const SnackBar(content: Text('支付链接已复制')),
                  );
                }
              },
              child: const Text('复制链接'),
            ),
          ],
        );
      },
    );
  }
}

