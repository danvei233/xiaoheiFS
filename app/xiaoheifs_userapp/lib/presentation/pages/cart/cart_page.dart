import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../core/constants/app_colors.dart';
import '../../../core/constants/app_strings.dart';
import '../../../core/utils/money_formatter.dart';
import '../../providers/catalog_provider.dart';
import '../../providers/cart_provider.dart';
import '../../providers/order_provider.dart';
import '../../providers/refresh_provider.dart';
import '../../widgets/common/empty_state.dart';

/// 购物车页面
class CartPage extends ConsumerStatefulWidget {
  const CartPage({super.key});

  @override
  ConsumerState<CartPage> createState() => _CartPageState();
}

class _CartPageState extends ConsumerState<CartPage> {
  ProviderSubscription<RefreshEvent?>? _refreshSub;

  @override
  void initState() {
    super.initState();
    Future.microtask(() {
      if (ref.read(catalogProvider).packages.isEmpty) {
        ref.read(catalogProvider.notifier).fetchCatalog();
      }
      ref.read(cartProvider.notifier).fetchCart();
    });
    _refreshSub = ref.listenManual<RefreshEvent?>(pageRefreshProvider, (_, next) {
      if (next?.route == '/console/cart') {
        ref.read(cartProvider.notifier).fetchCart(force: true);
      }
    });
  }

  @override
  void dispose() {
    _refreshSub?.close();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final cartState = ref.watch(cartProvider);
    final catalog = ref.watch(catalogProvider);
    final items = cartState.items;

    return Scaffold(
      body: RefreshIndicator(
        onRefresh: () => ref.read(cartProvider.notifier).fetchCart(force: true),
        child: LayoutBuilder(
          builder: (context, constraints) {
            final isWide = constraints.maxWidth >= 1024;
            return SingleChildScrollView(
              physics: const AlwaysScrollableScrollPhysics(),
              padding: EdgeInsets.all(isWide ? 24 : 16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  _buildHeader(
                    itemCount: items.length,
                    loading: cartState.loading,
                    onRefresh: () => ref.read(cartProvider.notifier).fetchCart(force: true),
                  ),
                  const SizedBox(height: 16),
                  if (cartState.loading && items.isEmpty)
                    const Center(child: CircularProgressIndicator())
                  else if (items.isEmpty)
                    EmptyState(
                      message: AppStrings.cartEmpty,
                      icon: Icons.shopping_cart_outlined,
                      actionLabel: AppStrings.vpsBuy,
                      onAction: () => context.go('/console/buy'),
                    )
                  else if (isWide)
                    Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Expanded(child: _buildItemsList(context, ref, items, catalog)),
                        const SizedBox(width: 24),
                        SizedBox(
                          width: 340,
                          child: _buildSummaryCard(context, ref, items),
                        ),
                      ],
                    )
                  else
                    Column(
                      children: [
                        _buildItemsList(context, ref, items, catalog),
                        const SizedBox(height: 16),
                        _buildSummaryCard(context, ref, items),
                      ],
                    ),
                ],
              ),
            );
          },
        ),
      ),
    );
  }

  Widget _buildHeader({
    required int itemCount,
    required bool loading,
    required VoidCallback onRefresh,
  }) {
    return Row(
      children: [
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text(
                AppStrings.shoppingCart,
                style: TextStyle(fontSize: 22, fontWeight: FontWeight.w700),
              ),
              const SizedBox(height: 4),
              Text(
                '$itemCount 件商品',
                style: const TextStyle(fontSize: 13, color: AppColors.gray500),
              ),
            ],
          ),
        ),
        TextButton.icon(
          onPressed: loading ? null : onRefresh,
          icon: loading
              ? const SizedBox(
                  width: 14,
                  height: 14,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : const Icon(Icons.refresh),
          label: const Text(AppStrings.refresh),
        ),
      ],
    );
  }

  Widget _buildItemsList(
    BuildContext context,
    WidgetRef ref,
    List<Map<String, dynamic>> items,
    CatalogState catalog,
  ) {
    return Column(
      children: List.generate(items.length, (index) {
        final item = items[index];
        final pkg = _findPackage(catalog.packages, item['package_id']);
        final system = _findSystemImage(catalog.systemImages, item['system_id']);
        return Container(
          margin: const EdgeInsets.only(bottom: 16),
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: AppColors.darkSurface.withOpacity(0.7),
            borderRadius: BorderRadius.circular(16),
            border: Border.all(color: AppColors.gray700.withOpacity(0.5)),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Container(
                    width: 40,
                    height: 40,
                    decoration: BoxDecoration(
                      color: AppColors.primary.withOpacity(0.15),
                      borderRadius: BorderRadius.circular(10),
                    ),
                    child: const Icon(Icons.desktop_windows, color: AppColors.primary),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          pkg?['name']?.toString() ?? '套餐 #${item['package_id']}',
                          style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w600),
                        ),
                        const SizedBox(height: 2),
                        Text(
                          'ID: ${item['package_id'] ?? '-'}',
                          style: const TextStyle(fontSize: 12, color: AppColors.gray500),
                        ),
                      ],
                    ),
                  ),
                  IconButton(
                    icon: const Icon(Icons.delete_outline, color: AppColors.danger),
                    onPressed: () => _confirmRemove(context, ref, item['id']),
                  ),
                ],
              ),
              const SizedBox(height: 12),
              Row(
                children: [
                  const Icon(Icons.code, size: 16, color: AppColors.gray400),
                  const SizedBox(width: 6),
                  Expanded(
                    child: Text(
                      system?['name']?.toString() ?? '系统 #${item['system_id'] ?? '-'}',
                      style: const TextStyle(fontSize: 13, color: AppColors.gray300),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 12),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  _buildSpecChip('CPU', '${_getCpu(item, pkg)} 核', AppColors.primary),
                  _buildSpecChip('内存', '${_getMemory(item, pkg)}G', AppColors.success),
                  _buildSpecChip('磁盘', '${_getDisk(item, pkg)}G', AppColors.info),
                  _buildSpecChip('带宽', '${_getBandwidth(item, pkg)}M', AppColors.warning),
                  _buildSpecChip('周期', _getDuration(item), AppColors.gray400),
                ],
              ),
              const SizedBox(height: 12),
              Row(
                children: [
                  _buildQtyStepper(
                    value: _getQty(item),
                    onChanged: (v) => _updateQty(ref, item, v),
                  ),
                  const Spacer(),
                  Text(
                    MoneyFormatter.format(_itemTotal(item)),
                    style: const TextStyle(
                      fontSize: 18,
                      fontWeight: FontWeight.w700,
                      color: AppColors.primary,
                    ),
                  ),
                ],
              ),
            ],
          ),
        );
      }),
    );
  }

  Widget _buildSpecChip(String label, String value, Color color) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: AppColors.gray800.withOpacity(0.7),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: AppColors.gray700.withOpacity(0.5)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 6,
            height: 6,
            decoration: BoxDecoration(color: color, shape: BoxShape.circle),
          ),
          const SizedBox(width: 6),
          Text(
            '$label $value',
            style: const TextStyle(fontSize: 12, color: AppColors.gray200),
          ),
        ],
      ),
    );
  }

  Widget _buildQtyStepper({
    required int value,
    required ValueChanged<int> onChanged,
  }) {
    return Container(
      decoration: BoxDecoration(
        color: AppColors.darkSurface.withOpacity(0.6),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: AppColors.gray700.withOpacity(0.5)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          IconButton(
            icon: const Icon(Icons.remove, size: 18),
            onPressed: value > 1 ? () => onChanged(value - 1) : null,
          ),
          Text(
            '$value',
            style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w600),
          ),
          IconButton(
            icon: const Icon(Icons.add, size: 18),
            onPressed: value < 99 ? () => onChanged(value + 1) : null,
          ),
        ],
      ),
    );
  }

  Widget _buildSummaryCard(BuildContext context, WidgetRef ref, List<Map<String, dynamic>> items) {
    final total = _calcTotal(items);
    final totalItems = _totalQty(items);
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: AppColors.darkSurface.withOpacity(0.85),
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: AppColors.gray700.withOpacity(0.5)),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withOpacity(0.2),
            blurRadius: 12,
            offset: const Offset(0, 6),
          )
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const Row(
            children: [
              Icon(Icons.receipt_long, color: AppColors.primary),
              SizedBox(width: 8),
              Text(
                '订单摘要',
                style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
              ),
            ],
          ),
          const SizedBox(height: 16),
          _buildSummaryRow('商品数量', '$totalItems'),
          _buildSummaryRow('小计', MoneyFormatter.format(total)),
          const Divider(height: 24, color: AppColors.gray700),
          _buildSummaryRow(
            '总计',
            MoneyFormatter.format(total),
            valueStyle: const TextStyle(
              fontSize: 20,
              fontWeight: FontWeight.w700,
              color: AppColors.primary,
            ),
          ),
          const SizedBox(height: 16),
          ElevatedButton.icon(
            onPressed: () => _checkout(context, ref),
            icon: const Icon(Icons.check_circle_outline),
            label: const Text('立即下单'),
            style: ElevatedButton.styleFrom(
              padding: const EdgeInsets.symmetric(vertical: 14),
              backgroundColor: AppColors.primary,
              shape: const StadiumBorder(),
            ),
          ),
          const SizedBox(height: 12),
          Container(
            padding: const EdgeInsets.symmetric(vertical: 10),
            decoration: BoxDecoration(
              color: AppColors.gray800.withOpacity(0.7),
              borderRadius: BorderRadius.circular(12),
            ),
            child: const Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Icon(Icons.verified_user_outlined, size: 14, color: AppColors.success),
                SizedBox(width: 6),
                Text(
                  '安全支付 · 即时开通',
                  style: TextStyle(fontSize: 12, color: AppColors.gray300),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildSummaryRow(String label, String value, {TextStyle? valueStyle}) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        children: [
          Text(label, style: const TextStyle(color: AppColors.gray400)),
          const Spacer(),
          Text(value, style: valueStyle ?? const TextStyle(fontWeight: FontWeight.w600)),
        ],
      ),
    );
  }

  Map<String, dynamic>? _findPackage(List<Map<String, dynamic>> packages, dynamic id) {
    if (id == null) return null;
    for (final pkg in packages) {
      if ('${pkg['id']}' == '$id') return pkg;
    }
    return null;
  }

  Map<String, dynamic>? _findSystemImage(List<Map<String, dynamic>> images, dynamic id) {
    if (id == null) return null;
    for (final img in images) {
      if ('${img['id']}' == '$id') return img;
    }
    return null;
  }

  Map<String, dynamic> _specOf(Map<String, dynamic> item) {
    final spec = item['spec'];
    if (spec is Map) {
      return Map<String, dynamic>.from(spec);
    }
    return const {};
  }

  int _getCpu(Map<String, dynamic> item, Map<String, dynamic>? pkg) {
    final base = int.tryParse('${pkg?['cores'] ?? pkg?['cpu'] ?? 0}') ?? 0;
    final add = int.tryParse('${_specOf(item)['add_cores'] ?? 0}') ?? 0;
    return base + add;
  }

  int _getMemory(Map<String, dynamic> item, Map<String, dynamic>? pkg) {
    final base = int.tryParse('${pkg?['memory_gb'] ?? 0}') ?? 0;
    final add = int.tryParse('${_specOf(item)['add_mem_gb'] ?? 0}') ?? 0;
    return base + add;
  }

  int _getDisk(Map<String, dynamic> item, Map<String, dynamic>? pkg) {
    final base = int.tryParse('${pkg?['disk_gb'] ?? 0}') ?? 0;
    final add = int.tryParse('${_specOf(item)['add_disk_gb'] ?? 0}') ?? 0;
    return base + add;
  }

  int _getBandwidth(Map<String, dynamic> item, Map<String, dynamic>? pkg) {
    final base = int.tryParse('${pkg?['bandwidth_mbps'] ?? 0}') ?? 0;
    final add = int.tryParse('${_specOf(item)['add_bw_mbps'] ?? 0}') ?? 0;
    return base + add;
  }

  String _getDuration(Map<String, dynamic> item) {
    final spec = _specOf(item);
    if (spec['duration_months'] != null) {
      return '${spec['duration_months']}个月';
    }
    if (spec['cycle_qty'] != null) {
      return '周期 ${spec['cycle_qty']}';
    }
    return '-';
  }

  int _getQty(Map<String, dynamic> item) {
    return int.tryParse('${item['qty'] ?? item['quantity'] ?? 1}') ?? 1;
  }

  double _itemTotal(Map<String, dynamic> item) {
    final price = double.tryParse('${item['amount'] ?? item['price'] ?? 0}') ?? 0;
    return price * _getQty(item);
  }

  int _totalQty(List<Map<String, dynamic>> items) {
    var sum = 0;
    for (final item in items) {
      sum += _getQty(item);
    }
    return sum;
  }

  Future<void> _confirmRemove(BuildContext context, WidgetRef ref, dynamic id) async {
    if (id == null) return;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('移除商品'),
        content: const Text('确定要从购物车移除该商品吗？'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('取消')),
          TextButton(onPressed: () => Navigator.pop(context, true), child: const Text('移除')),
        ],
      ),
    );
    if (confirmed == true) {
      await ref.read(cartProvider.notifier).removeItem(int.parse('$id'));
    }
  }

  double _calcTotal(List<Map<String, dynamic>> items) {
    double total = 0;
    for (final item in items) {
      final price = double.tryParse('${item['price'] ?? item['amount'] ?? 0}') ?? 0;
      final qty = int.tryParse('${item['quantity'] ?? item['qty'] ?? 1}') ?? 1;
      total += price * qty;
    }
    return total;
  }

  Future<void> _checkout(BuildContext context, WidgetRef ref) async {
    try {
      final orderRepo = ref.read(orderRepositoryProvider);
      final res = await orderRepo.createOrderFromCart();
      final orderId = res['order']?['id'] ?? res['id'] ?? res['order_id'];
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('订单创建成功')),
        );
        if (orderId != null) {
          context.go('/console/orders/$orderId');
        }
      }
      await ref.read(cartProvider.notifier).fetchCart();
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(e.toString())),
        );
      }
    }
  }

  Future<void> _updateQty(WidgetRef ref, Map<String, dynamic> item, int qty) async {
    final id = item['id'];
    if (id == null) return;
    await ref.read(cartProvider.notifier).updateItem(
      int.parse('$id'),
      {
        'spec': item['spec'],
        'qty': qty,
      },
    );
  }
}
