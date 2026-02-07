import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app_state.dart';
import '../services/api_client.dart';
import 'order_detail_screen.dart';

class OrdersScreen extends StatefulWidget {
  const OrdersScreen({super.key});

  @override
  State<OrdersScreen> createState() => _OrdersScreenState();
}

class _OrdersScreenState extends State<OrdersScreen> {
  OrdersState? _state;
  bool _bound = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final client = context.read<AppState>().apiClient;
    if (!_bound && client != null) {
      _state = OrdersState(OrdersRepository(client));
      _state!.load();
      _bound = true;
    }
  }

  @override
  Widget build(BuildContext context) {
    final client = context.read<AppState>().apiClient;
    if (client == null) {
      return _ErrorState(
        title: '未登录',
        message: '请先登录管理员账号。',
        onRetry: () {},
      );
    }
    return ChangeNotifierProvider.value(
      value: _state ?? OrdersState(OrdersRepository(client)),
      child: const _OrdersView(),
    );
  }
}

class _OrdersView extends StatefulWidget {
  const _OrdersView();

  @override
  State<_OrdersView> createState() => _OrdersViewState();
}

class _OrdersViewState extends State<_OrdersView> {
  final _userIdCtl = TextEditingController();
  final _orderNoCtl = TextEditingController();
  final _keywordCtl = TextEditingController();
  bool _filterExpanded = false;

  @override
  void dispose() {
    _userIdCtl.dispose();
    _orderNoCtl.dispose();
    _keywordCtl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Consumer<OrdersState>(
      builder: (context, state, _) {
        final theme = Theme.of(context);
        final colorScheme = theme.colorScheme;

        return Scaffold(
          appBar: AppBar(
            toolbarHeight: 44,
            title: const Text('订单审核'),
            titleTextStyle: Theme.of(context)
                .textTheme
                .titleMedium
                ?.copyWith(fontWeight: FontWeight.w600, fontSize: 16),
            iconTheme: const IconThemeData(size: 20),
            actionsIconTheme: const IconThemeData(size: 20),
            actions: [
              IconButton(
                icon: const Icon(Icons.refresh_rounded),
                onPressed: state.loading ? null : state.refresh,
              ),
            ],
          ),
          body: Column(
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 0, 16, 2),
                child: _FilterCard(
                  expanded: _filterExpanded,
                  status: state.filters.status,
                  userIdCtl: _userIdCtl,
                  orderNoCtl: _orderNoCtl,
                  keywordCtl: _keywordCtl,
                  onToggleExpand: () => setState(() => _filterExpanded = !_filterExpanded),
                  onStatusChanged: (value) => state.setStatus(value),
                  onSearch: () {
                    state.setFilters(
                      userId: _userIdCtl.text.trim(),
                      orderNo: _orderNoCtl.text.trim(),
                      keyword: _keywordCtl.text.trim(),
                    );
                  },
                  onReset: () {
                    _userIdCtl.clear();
                    _orderNoCtl.clear();
                    _keywordCtl.clear();
                    state.resetFilters();
                  },
                ),
              ),
              Expanded(
                child: RefreshIndicator(
                  onRefresh: () async => state.refresh(),
                  child: _buildBody(theme, colorScheme, state),
                ),
              ),
              _PaginationBar(
                page: state.page,
                pageSize: state.pageSize,
                total: state.total,
                onPrev: state.page > 1 && !state.loading ? () => state.setPage(state.page - 1) : null,
                onNext: state.page * state.pageSize < state.total && !state.loading
                    ? () => state.setPage(state.page + 1)
                    : null,
              ),
            ],
          ),
        );
      },
    );
  }

  Widget _buildBody(ThemeData theme, ColorScheme colorScheme, OrdersState state) {
    if (state.loading && state.items.isEmpty) {
      return const _LoadingState(text: '加载订单中...');
    }
    if (state.error != null && state.items.isEmpty) {
      return _ErrorState(
        title: '加载失败',
        message: state.error!,
        onRetry: state.refresh,
      );
    }
    if (state.items.isEmpty) {
      return const _EmptyState(text: '暂无订单');
    }

    return ListView.separated(
      padding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
      itemCount: state.items.length,
      separatorBuilder: (_, __) => const SizedBox(height: 12),
      itemBuilder: (context, index) {
        final item = state.items[index];
        final username = state.usernames[item.userId] ?? item.userId.toString();
        return _OrderTile(
          item: item,
          username: username,
          busy: state.actionBusy.contains(item.id),
          onOpenDetail: () async {
            final changed = await Navigator.push<bool>(
              context,
              MaterialPageRoute(
                builder: (_) => OrderDetailScreen(orderId: item.id),
              ),
            );
            if (changed == true) {
              state.refresh();
            }
          },
          onApprove: () => state.approve(item),
          onReject: () => state.reject(item),
          onRetry: () => state.retry(item),
          onDelete: () => state.delete(item),
        );
      },
    );
  }
}

class _FilterCard extends StatelessWidget {
  final bool expanded;
  final String status;
  final TextEditingController userIdCtl;
  final TextEditingController orderNoCtl;
  final TextEditingController keywordCtl;
  final VoidCallback onToggleExpand;
  final ValueChanged<String> onStatusChanged;
  final VoidCallback onSearch;
  final VoidCallback onReset;

  const _FilterCard({
    required this.expanded,
    required this.status,
    required this.userIdCtl,
    required this.orderNoCtl,
    required this.keywordCtl,
    required this.onToggleExpand,
    required this.onStatusChanged,
    required this.onSearch,
    required this.onReset,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    return Container(
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: colorScheme.outlineVariant.withOpacity(0.5)),
      ),
      child: Padding(
        padding: const EdgeInsets.all(8),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.tune_rounded, color: colorScheme.primary, size: 14),
                const SizedBox(width: 4),
                Text('筛选条件', style: theme.textTheme.labelMedium?.copyWith(fontWeight: FontWeight.w600)),
                const Spacer(),
                TextButton.icon(
                  onPressed: onToggleExpand,
                  icon: Icon(expanded ? Icons.expand_less : Icons.expand_more, size: 16),
                  label: Text(expanded ? '收起' : '更多', style: const TextStyle(fontSize: 11)),
                  style: TextButton.styleFrom(
                    padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                    minimumSize: const Size(0, 26),
                    tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                  ),
                )
              ],
            ),
            const SizedBox(height: 4),
            SizedBox(
              height: 32,
              child: ListView.separated(
                scrollDirection: Axis.horizontal,
                itemCount: 9,
                separatorBuilder: (_, __) => const SizedBox(width: 4),
                itemBuilder: (context, index) {
                  switch (index) {
                    case 0:
                      return _StatusChip(value: '', label: '全部', current: status, onChanged: onStatusChanged);
                    case 1:
                      return _StatusChip(value: 'pending_review', label: '待审核', current: status, onChanged: onStatusChanged);
                    case 2:
                      return _StatusChip(value: 'provisioning', label: '开通中', current: status, onChanged: onStatusChanged);
                    case 3:
                      return _StatusChip(value: 'failed', label: '失败', current: status, onChanged: onStatusChanged);
                    case 4:
                      return _StatusChip(value: 'pending_payment', label: '待支付', current: status, onChanged: onStatusChanged);
                    case 5:
                      return _StatusChip(value: 'approved', label: '已通过', current: status, onChanged: onStatusChanged);
                    case 6:
                      return _StatusChip(value: 'active', label: '已完成', current: status, onChanged: onStatusChanged);
                    case 7:
                      return _StatusChip(value: 'rejected', label: '已驳回', current: status, onChanged: onStatusChanged);
                    default:
                      return _StatusChip(value: 'canceled', label: '已取消', current: status, onChanged: onStatusChanged);
                  }
                },
              ),
            ),
            if (expanded) ...[
              const SizedBox(height: 6),
              TextField(
                controller: userIdCtl,
                keyboardType: TextInputType.number,
                decoration: const InputDecoration(
                  labelText: '用户 ID',
                  prefixIcon: Icon(Icons.person_outline),
                  isDense: true,
                  contentPadding: EdgeInsets.symmetric(horizontal: 10, vertical: 10),
                ),
              ),
              const SizedBox(height: 4),
              TextField(
                controller: orderNoCtl,
                decoration: const InputDecoration(
                  labelText: '订单号',
                  prefixIcon: Icon(Icons.receipt_long),
                  isDense: true,
                  contentPadding: EdgeInsets.symmetric(horizontal: 10, vertical: 10),
                ),
              ),
              const SizedBox(height: 4),
              TextField(
                controller: keywordCtl,
                decoration: const InputDecoration(
                  labelText: '关键词 (ID/订单号)',
                  prefixIcon: Icon(Icons.search),
                  isDense: true,
                  contentPadding: EdgeInsets.symmetric(horizontal: 10, vertical: 10),
                ),
              ),
            ],
            const SizedBox(height: 6),
            Row(
              children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed: onReset,
                    style: OutlinedButton.styleFrom(
                      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                      minimumSize: const Size(0, 28),
                      tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                      textStyle: const TextStyle(fontSize: 11),
                    ),
                    child: const Text('重置'),
                  ),
                ),
                const SizedBox(width: 6),
                Expanded(
                  child: FilledButton(
                    onPressed: onSearch,
                    style: FilledButton.styleFrom(
                      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                      minimumSize: const Size(0, 28),
                      tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                      textStyle: const TextStyle(fontSize: 11),
                    ),
                    child: const Text('搜索'),
                  ),
                ),
              ],
            )
          ],
        ),
      ),
    );
  }
}

class _StatusChip extends StatelessWidget {
  final String value;
  final String label;
  final String current;
  final ValueChanged<String> onChanged;

  const _StatusChip({
    required this.value,
    required this.label,
    required this.current,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    final selected = current == value || (current.isEmpty && value.isEmpty);
    final colorScheme = Theme.of(context).colorScheme;
    final bgColor = selected
        ? colorScheme.primaryContainer.withOpacity(0.6)
        : colorScheme.surface;
    final borderColor = selected
        ? colorScheme.primary
        : colorScheme.outlineVariant.withOpacity(0.7);
    final textColor = selected ? colorScheme.primary : colorScheme.onSurface;

    return Material(
      color: Colors.transparent,
      child: InkWell(
        borderRadius: BorderRadius.circular(8),
        onTap: () => onChanged(value),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
          decoration: BoxDecoration(
            color: bgColor,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: borderColor),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              if (selected) ...[
                Icon(Icons.check_rounded, size: 12, color: textColor),
                const SizedBox(width: 3),
              ],
              Text(
                label,
                style: TextStyle(
                  color: textColor,
                  fontWeight: FontWeight.w600,
                  fontSize: 11,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _OrderTile extends StatelessWidget {
  final OrderListItem item;
  final String username;
  final bool busy;
  final VoidCallback onOpenDetail;
  final VoidCallback onApprove;
  final VoidCallback onReject;
  final VoidCallback onRetry;
  final VoidCallback onDelete;

  const _OrderTile({
    required this.item,
    required this.username,
    required this.busy,
    required this.onOpenDetail,
    required this.onApprove,
    required this.onReject,
    required this.onRetry,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final statusMeta = _statusMeta(item.status, colorScheme);

    return Container(
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: colorScheme.outlineVariant.withOpacity(0.5)),
        boxShadow: [
          BoxShadow(
            color: colorScheme.shadow.withOpacity(0.06),
            blurRadius: 12,
            offset: const Offset(0, 4),
          ),
        ],
      ),
      child: ClipRRect(
        borderRadius: BorderRadius.circular(16),
        child: Row(
          children: [
            // 左侧状态颜色条
            Container(
              width: 3,
              height: 92,
              color: statusMeta.color,
            ),
            Expanded(
              child: Material(
                color: Colors.transparent,
                child: InkWell(
                  onTap: onOpenDetail,
                  child: Padding(
                    padding: const EdgeInsets.all(8),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Row(
                          children: [
                            Container(
                              padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                              decoration: BoxDecoration(
                                color: statusMeta.color.withOpacity(0.12),
                                borderRadius: BorderRadius.circular(6),
                                border: Border.all(
                                  color: statusMeta.color.withOpacity(0.3),
                                  width: 1,
                                ),
                              ),
                              child: Row(
                                mainAxisSize: MainAxisSize.min,
                                children: [
                                  Icon(
                                    _statusIcon(item.status),
                                    size: 10,
                                    color: statusMeta.color,
                                  ),
                                  const SizedBox(width: 3),
                                  Text(
                                    statusMeta.label,
                                    style: theme.textTheme.bodySmall?.copyWith(
                                      color: statusMeta.color,
                                      fontWeight: FontWeight.w700,
                                      letterSpacing: 0.1,
                                      fontSize: 10,
                                    ),
                                  ),
                                ],
                              ),
                            ),
                            const Spacer(),
                            if (busy)
                              const SizedBox(
                                width: 18,
                                height: 18,
                                child: CircularProgressIndicator(strokeWidth: 2),
                              )
                            else
                              Icon(
                                Icons.chevron_right_rounded,
                                size: 18,
                                color: colorScheme.onSurfaceVariant,
                              ),
                          ],
                        ),
                        const SizedBox(height: 4),
                        Text(
                          item.orderNo.isEmpty ? '订单 #${item.id}' : item.orderNo,
                          style: theme.textTheme.bodyLarge?.copyWith(
                            fontWeight: FontWeight.w700,
                            letterSpacing: -0.2,
                            fontSize: 14,
                          ),
                        ),
                        const SizedBox(height: 3),
                        Wrap(
                          spacing: 10,
                          runSpacing: 2,
                          children: [
                            _InfoChip(icon: Icons.person_outline, text: username),
                            _InfoChip(icon: Icons.badge_outlined, text: 'ID ${item.userId}'),
                            _InfoChip(icon: Icons.schedule, text: _formatLocal(item.createdAt)),
                          ],
                        ),
                        const SizedBox(height: 4),
                        Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Container(
                              padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 3),
                              decoration: BoxDecoration(
                                gradient: LinearGradient(
                                  colors: [
                                    colorScheme.primary.withOpacity(0.1),
                                    colorScheme.primary.withOpacity(0.05),
                                  ],
                                ),
                                borderRadius: BorderRadius.circular(6),
                              ),
                              child: Text(
                                '¥${item.totalAmount.toStringAsFixed(2)} ${item.currency}',
                                style: theme.textTheme.titleLarge?.copyWith(
                                  fontWeight: FontWeight.w800,
                                  color: colorScheme.primary,
                                  fontSize: 14,
                                ),
                              ),
                            ),
                            const SizedBox(height: 4),
                            Wrap(
                              spacing: 6,
                              runSpacing: 3,
                              children: [
                                _ActionButton(
                                  label: '详情',
                                  icon: Icons.visibility_outlined,
                                  onPressed: onOpenDetail,
                                  isOutlined: true,
                                ),
                                _ActionButton(
                                  label: '通过',
                                  icon: Icons.check_circle_outline,
                                  onPressed: item.canReview && !busy ? onApprove : null,
                                  color: const Color(0xFF00A68C),
                                ),
                                _ActionButton(
                                  label: '驳回',
                                  icon: Icons.cancel_outlined,
                                  onPressed: item.canReview && !busy ? onReject : null,
                                  color: const Color(0xFFEF6C00),
                                ),
                                _ActionButton(
                                  label: '重试',
                                  icon: Icons.refresh_outlined,
                                  onPressed: !busy ? onRetry : null,
                                  isOutlined: true,
                                ),
                                _ActionButton(
                                  label: '删除',
                                  icon: Icons.delete_outline,
                                  onPressed: !busy ? onDelete : null,
                                  color: const Color(0xFFD32F2F),
                                ),
                              ],
                            ),
                          ],
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _InfoChip extends StatelessWidget {
  final IconData icon;
  final String text;

  const _InfoChip({required this.icon, required this.text});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(icon, size: 11, color: colorScheme.onSurfaceVariant),
        const SizedBox(width: 2),
        Text(
          text,
          style: theme.textTheme.bodySmall?.copyWith(
            color: colorScheme.onSurfaceVariant,
            fontSize: 10,
          ),
        ),
      ],
    );
  }
}

class _ActionButton extends StatelessWidget {
  final String label;
  final IconData icon;
  final VoidCallback? onPressed;
  final Color? color;
  final bool isOutlined;

  const _ActionButton({
    required this.label,
    required this.icon,
    required this.onPressed,
    this.color,
    this.isOutlined = false,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final foreground = color ?? colorScheme.primary;
    final style = ButtonStyle(
      padding: WidgetStateProperty.all(
        const EdgeInsets.symmetric(horizontal: 6, vertical: 4),
      ),
      minimumSize: WidgetStateProperty.all(const Size(0, 24)),
      tapTargetSize: MaterialTapTargetSize.shrinkWrap,
      textStyle: WidgetStateProperty.all(
        theme.textTheme.labelSmall?.copyWith(fontWeight: FontWeight.w600, fontSize: 10),
      ),
    );

    final content = Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(icon, size: 12),
        const SizedBox(width: 3),
        Text(label),
      ],
    );

    if (isOutlined) {
      return OutlinedButton(
        onPressed: onPressed,
        style: style.copyWith(
          foregroundColor: WidgetStateProperty.resolveWith(
            (states) => states.contains(WidgetState.disabled)
                ? colorScheme.onSurfaceVariant
                : foreground,
          ),
          side: WidgetStateProperty.resolveWith(
            (states) => BorderSide(
              color: states.contains(WidgetState.disabled)
                  ? colorScheme.outlineVariant
                  : foreground,
            ),
          ),
        ),
        child: content,
      );
    }

    return FilledButton(
      onPressed: onPressed,
      style: style.copyWith(
        backgroundColor: WidgetStateProperty.resolveWith(
          (states) => states.contains(WidgetState.disabled)
              ? colorScheme.surfaceVariant
              : foreground,
        ),
        foregroundColor: WidgetStateProperty.resolveWith(
          (states) => states.contains(WidgetState.disabled)
              ? colorScheme.onSurfaceVariant
              : Colors.white,
        ),
      ),
      child: content,
    );
  }
}

class _LoadingState extends StatelessWidget {
  final String text;

  const _LoadingState({required this.text});

  @override
  Widget build(BuildContext context) {
    return ListView(
      physics: const AlwaysScrollableScrollPhysics(),
      children: [
        const SizedBox(height: 120),
        Column(
          children: [
            const CircularProgressIndicator(),
            const SizedBox(height: 12),
            Text(text),
          ],
        ),
      ],
    );
  }
}

class _EmptyState extends StatelessWidget {
  final String text;

  const _EmptyState({required this.text});

  @override
  Widget build(BuildContext context) {
    return ListView(
      physics: const AlwaysScrollableScrollPhysics(),
      children: [
        const SizedBox(height: 120),
        Center(child: Text(text)),
      ],
    );
  }
}

class _ErrorState extends StatelessWidget {
  final String title;
  final String message;
  final VoidCallback onRetry;

  const _ErrorState({
    required this.title,
    required this.message,
    required this.onRetry,
  });

  @override
  Widget build(BuildContext context) {
    return ListView(
      physics: const AlwaysScrollableScrollPhysics(),
      children: [
        const SizedBox(height: 100),
        Column(
          children: [
            const Icon(Icons.error_outline, size: 48),
            const SizedBox(height: 12),
            Text(title, style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 8),
            Text(message, textAlign: TextAlign.center),
            const SizedBox(height: 12),
            FilledButton(onPressed: onRetry, child: const Text('重试')),
          ],
        )
      ],
    );
  }
}

class _PaginationBar extends StatelessWidget {
  final int page;
  final int pageSize;
  final int total;
  final VoidCallback? onPrev;
  final VoidCallback? onNext;

  const _PaginationBar({
    required this.page,
    required this.pageSize,
    required this.total,
    this.onPrev,
    this.onNext,
  });

  @override
  Widget build(BuildContext context) {
    final totalPages = (total / pageSize).ceil();
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 2, 16, 10),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Text(
            '第 $page / $totalPages 页 · 共 $total 条',
            style: Theme.of(context).textTheme.bodySmall,
          ),
          Row(
            children: [
              OutlinedButton(
                onPressed: onPrev,
                style: OutlinedButton.styleFrom(
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                  minimumSize: const Size(0, 32),
                  tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                ),
                child: const Text('上一页', style: TextStyle(fontSize: 12)),
              ),
              const SizedBox(width: 6),
              OutlinedButton(
                onPressed: onNext,
                style: OutlinedButton.styleFrom(
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                  minimumSize: const Size(0, 32),
                  tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                ),
                child: const Text('下一页', style: TextStyle(fontSize: 12)),
              ),
            ],
          )
        ],
      ),
    );
  }
}

class OrdersRepository {
  OrdersRepository(this.client);

  final ApiClient client;
  final Map<int, String> _usernameCache = {};

  Future<OrderListResponse> fetchOrders({
    required int limit,
    required int offset,
    String status = '',
    String userId = '',
    String orderNo = '',
    String keyword = '',
  }) async {
    final query = <String, String>{
      'limit': limit.toString(),
      'offset': offset.toString(),
      if (status.isNotEmpty) 'status': status,
      if (userId.isNotEmpty) 'user_id': userId,
      if (orderNo.isNotEmpty) 'order_no': orderNo,
    };
    final resp = await client.getJson('/admin/api/v1/orders', query: query);
    final items = (resp['items'] as List<dynamic>? ?? [])
        .map((e) => OrderListItem.fromJson(e as Map<String, dynamic>))
        .toList();
    final total = resp['total'] as int? ?? items.length;
    final filtered = keyword.isEmpty
        ? items
        : items.where((item) {
            final k = keyword.toLowerCase();
            return item.id.toString().contains(k) ||
                item.orderNo.toLowerCase().contains(k);
          }).toList();
    return OrderListResponse(items: filtered, total: total);
  }

  Future<String> getUsername(int userId) async {
    if (_usernameCache.containsKey(userId)) return _usernameCache[userId]!;
    final resp = await client.getJson('/admin/api/v1/users/$userId');
    final map = resp['user'] is Map<String, dynamic>
        ? resp['user'] as Map<String, dynamic>
        : (resp['data'] is Map<String, dynamic> ? resp['data'] as Map<String, dynamic> : resp);
    final name = (map['username'] ?? userId.toString()).toString();
    _usernameCache[userId] = name;
    return name;
  }

  Future<void> approve(int orderId) async {
    await client.postJson('/admin/api/v1/orders/$orderId/approve');
  }

  Future<void> reject(int orderId, String reason) async {
    await client.postJson('/admin/api/v1/orders/$orderId/reject', body: {'reason': reason});
  }

  Future<void> retry(int orderId) async {
    await client.postJson('/admin/api/v1/orders/$orderId/retry');
  }

  Future<void> delete(int orderId) async {
    await client.deleteJson('/admin/api/v1/orders/$orderId');
  }
}

class OrdersState extends ChangeNotifier {
  OrdersState(this.repo);

  final OrdersRepository repo;

  List<OrderListItem> items = [];
  int total = 0;
  int page = 1;
  int pageSize = 20;
  bool loading = false;
  String? error;

  final Map<int, String> usernames = {};
  final Set<int> actionBusy = {};
  OrderFilters filters = OrderFilters.empty();

  Future<void> load() async {
    loading = true;
    error = null;
    notifyListeners();
    try {
      final resp = await repo.fetchOrders(
        limit: pageSize,
        offset: (page - 1) * pageSize,
        status: filters.status,
        userId: filters.userId,
        orderNo: filters.orderNo,
        keyword: filters.keyword,
      );
      items = resp.items;
      total = resp.total;
      await _loadUsernames(items);
    } catch (e) {
      error = e.toString();
    } finally {
      loading = false;
      notifyListeners();
    }
  }

  Future<void> refresh() => load();

  void setPage(int newPage) {
    page = newPage;
    load();
  }

  void setStatus(String value) {
    filters = filters.copyWith(status: value);
    page = 1;
    load();
  }

  void setFilters({String? userId, String? orderNo, String? keyword}) {
    filters = filters.copyWith(
      userId: userId ?? filters.userId,
      orderNo: orderNo ?? filters.orderNo,
      keyword: keyword ?? filters.keyword,
    );
    page = 1;
    load();
  }

  void resetFilters() {
    filters = OrderFilters.empty();
    page = 1;
    load();
  }

  Future<void> approve(OrderListItem item) async {
    if (!item.canReview || actionBusy.contains(item.id)) return;
    actionBusy.add(item.id);
    notifyListeners();
    try {
      await repo.approve(item.id);
      await load();
    } finally {
      actionBusy.remove(item.id);
      notifyListeners();
    }
  }

  Future<void> reject(OrderListItem item) async {
    if (!item.canReview || actionBusy.contains(item.id)) return;
    actionBusy.add(item.id);
    notifyListeners();
    try {
      await repo.reject(item.id, 'manual');
      await load();
    } finally {
      actionBusy.remove(item.id);
      notifyListeners();
    }
  }

  Future<void> retry(OrderListItem item) async {
    if (actionBusy.contains(item.id)) return;
    actionBusy.add(item.id);
    notifyListeners();
    try {
      await repo.retry(item.id);
      await load();
    } finally {
      actionBusy.remove(item.id);
      notifyListeners();
    }
  }

  Future<void> delete(OrderListItem item) async {
    if (actionBusy.contains(item.id)) return;
    actionBusy.add(item.id);
    notifyListeners();
    try {
      await repo.delete(item.id);
      await load();
    } finally {
      actionBusy.remove(item.id);
      notifyListeners();
    }
  }

  Future<void> _loadUsernames(List<OrderListItem> items) async {
    final ids = items.map((e) => e.userId).where((e) => e > 0).toSet();
    if (ids.isEmpty) return;
    for (final id in ids) {
      if (usernames.containsKey(id)) continue;
      try {
        final name = await repo.getUsername(id);
        usernames[id] = name;
      } catch (_) {
        usernames[id] = id.toString();
      }
    }
  }
}

class OrderFilters {
  final String status;
  final String userId;
  final String orderNo;
  final String keyword;

  const OrderFilters({
    required this.status,
    required this.userId,
    required this.orderNo,
    required this.keyword,
  });

  static OrderFilters empty() => const OrderFilters(status: '', userId: '', orderNo: '', keyword: '');

  OrderFilters copyWith({String? status, String? userId, String? orderNo, String? keyword}) {
    return OrderFilters(
      status: status ?? this.status,
      userId: userId ?? this.userId,
      orderNo: orderNo ?? this.orderNo,
      keyword: keyword ?? this.keyword,
    );
  }
}

class OrderListResponse {
  final List<OrderListItem> items;
  final int total;

  const OrderListResponse({required this.items, required this.total});
}

class OrderListItem {
  final int id;
  final int userId;
  final String orderNo;
  final String status;
  final double totalAmount;
  final String currency;
  final String createdAt;
  final bool canReview;

  const OrderListItem({
    required this.id,
    required this.userId,
    required this.orderNo,
    required this.status,
    required this.totalAmount,
    required this.currency,
    required this.createdAt,
    required this.canReview,
  });

  factory OrderListItem.fromJson(Map<String, dynamic> json) {
    return OrderListItem(
      id: _asInt(json['id'] ?? json['ID']),
      userId: _asInt(json['user_id'] ?? json['UserID']),
      orderNo: (json['order_no'] ?? json['OrderNo'] ?? '').toString(),
      status: (json['status'] ?? json['Status'] ?? '').toString(),
      totalAmount: _asDouble(json['total_amount'] ?? json['TotalAmount']),
      currency: (json['currency'] ?? json['Currency'] ?? 'CNY').toString(),
      createdAt: (json['created_at'] ?? json['CreatedAt'] ?? '').toString(),
      canReview: (json['can_review'] ?? json['CanReview'] ?? false) == true,
    );
  }
}

class _StatusMeta {
  final String label;
  final Color color;

  const _StatusMeta(this.label, this.color);
}

IconData _statusIcon(String status) {
  switch (status) {
    case 'pending_payment':
      return Icons.payments_outlined;
    case 'pending_review':
      return Icons.hourglass_bottom_rounded;
    case 'approved':
      return Icons.check_circle_outline;
    case 'provisioning':
      return Icons.autorenew_rounded;
    case 'active':
      return Icons.verified_rounded;
    case 'failed':
      return Icons.error_outline;
    case 'rejected':
      return Icons.block_outlined;
    case 'canceled':
      return Icons.cancel_outlined;
    default:
      return Icons.help_outline;
  }
}

_StatusMeta _statusMeta(String status, ColorScheme scheme) {
  switch (status) {
    case 'pending_payment':
      return _StatusMeta('待支付', const Color(0xFFEF6C00));
    case 'pending_review':
      return _StatusMeta('待审核', const Color(0xFFEF6C00));
    case 'approved':
      return _StatusMeta('已通过', const Color(0xFF00A68C));
    case 'provisioning':
      return _StatusMeta('开通中', const Color(0xFF1E88E5));
    case 'active':
      return _StatusMeta('已完成', const Color(0xFF00A68C));
    case 'failed':
      return _StatusMeta('失败', const Color(0xFFD32F2F));
    case 'rejected':
      return _StatusMeta('已驳回', const Color(0xFFD32F2F));
    case 'canceled':
      return _StatusMeta('已取消', const Color(0xFF757575));
    default:
      return _StatusMeta(status.isEmpty ? '未知' : status, scheme.outline);
  }
}

String _formatLocal(String raw) {
  if (raw.isEmpty) return '-';
  final dt = DateTime.tryParse(raw);
  if (dt == null) return raw;
  final local = dt.toLocal();
  return '${local.year}-${_pad2(local.month)}-${_pad2(local.day)} ${_pad2(local.hour)}:${_pad2(local.minute)}:${_pad2(local.second)}';
}

String _pad2(int v) => v.toString().padLeft(2, '0');

int _asInt(dynamic value) {
  if (value is int) return value;
  if (value is num) return value.toInt();
  if (value is String) return int.tryParse(value) ?? 0;
  return 0;
}

double _asDouble(dynamic value) {
  if (value is double) return value;
  if (value is int) return value.toDouble();
  if (value is num) return value.toDouble();
  if (value is String) return double.tryParse(value) ?? 0;
  return 0;
}
