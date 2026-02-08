import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../core/constants/app_colors.dart';
import '../../../core/constants/app_strings.dart';
import '../../../core/utils/date_formatter.dart';
import '../../../core/utils/money_formatter.dart';
import '../../providers/order_provider.dart';
import '../../providers/refresh_provider.dart';
import '../../widgets/common/pagination_bar.dart';
import '../../widgets/common/empty_state.dart';
import '../../widgets/common/status_tag.dart';

/// 订单列表页面
class OrdersListPage extends ConsumerStatefulWidget {
  const OrdersListPage({super.key});

  @override
  ConsumerState<OrdersListPage> createState() => _OrdersListPageState();
}

class _OrdersListPageState extends ConsumerState<OrdersListPage> {
  ProviderSubscription<RefreshEvent?>? _refreshSub;
  int _page = 1;
  int _pageSize = 10;

  @override
  void initState() {
    super.initState();
    Future.microtask(() => _fetch());
    _refreshSub = ref.listenManual<RefreshEvent?>(pageRefreshProvider, (_, next) {
      if (next?.route == '/console/orders') {
        _fetch(force: true);
      }
    });
  }

  @override
  void dispose() {
    _refreshSub?.close();
    super.dispose();
  }

  Future<void> _fetch({bool force = false}) async {
    await ref.read(orderListProvider.notifier).fetchOrders(
          limit: _pageSize,
          offset: (_page - 1) * _pageSize,
          force: force,
        );
  }

  @override
  Widget build(BuildContext context) {
    final orderListState = ref.watch(orderListProvider);

    return Scaffold(
      body: orderListState.loading
          ? const Center(child: CircularProgressIndicator())
          : orderListState.items.isEmpty
              ? const EmptyState(
                  message: AppStrings.noOrders,
                  icon: Icons.receipt_long_outlined,
                )
              : _buildOrderList(context, ref, orderListState.items, orderListState.total),
    );
  }

  Widget _buildOrderList(
    BuildContext context,
    WidgetRef ref,
    List<Map<String, dynamic>> orders,
    int total,
  ) {
    return Column(
      children: [
        Expanded(
          child: RefreshIndicator(
            onRefresh: () => _fetch(force: true),
            child: ListView.builder(
              padding: const EdgeInsets.all(24),
              itemCount: orders.length,
              itemBuilder: (context, index) {
                final order = orders[index];
                final id = order['id'] ?? order['ID'];
                final orderNo = order['order_no'] ?? order['orderNo'] ?? '';
                final status = order['status'] ?? order['Status'] ?? '';
                final totalAmount = order['total_amount'] ?? order['totalAmount'] ?? 0;
                final createdAt = order['created_at'] ?? order['CreatedAt'];
                final totalValue = totalAmount is num
                    ? totalAmount.toDouble()
                    : double.tryParse(totalAmount.toString()) ?? 0;
                final amountColor = totalValue < 0 ? AppColors.success : AppColors.primary;
                return Card(
                  margin: const EdgeInsets.only(bottom: 16),
                  child: ListTile(
                    contentPadding: const EdgeInsets.all(16),
                    title: Row(
                      children: [
                        Expanded(
                          child: Text(
                            '$orderNo',
                            style: const TextStyle(
                              fontSize: 14,
                              fontWeight: FontWeight.bold,
                            ),
                          ),
                        ),
                        StatusTag.order('$status'),
                      ],
                    ),
                    subtitle: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const SizedBox(height: 8),
                        Text(
                          MoneyFormatter.format(totalAmount),
                          style: TextStyle(
                            fontSize: 16,
                            fontWeight: FontWeight.bold,
                            color: amountColor,
                          ),
                        ),
                        const SizedBox(height: 4),
                        Text(
                          DateFormatter.formatIso(createdAt),
                          style: TextStyle(
                            fontSize: 12,
                            color: AppColors.gray500,
                          ),
                        ),
                      ],
                    ),
                    onTap: () {
                      if (id != null) {
                        context.go('/console/orders/$id');
                      }
                    },
                  ),
                );
              },
            ),
          ),
        ),
        Padding(
          padding: const EdgeInsets.fromLTRB(24, 0, 24, 16),
          child: PaginationBar(
            currentPage: _page,
            pageSize: _pageSize,
            totalItems: total,
            onPageChanged: (page) async {
              setState(() => _page = page);
              await _fetch();
            },
            onPageSizeChanged: (size) async {
              setState(() {
                _pageSize = size;
                _page = 1;
              });
              await _fetch(force: true);
            },
          ),
        ),
      ],
    );
  }
}
