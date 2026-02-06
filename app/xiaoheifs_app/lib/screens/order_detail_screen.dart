import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app_state.dart';
import '../services/api_client.dart';
import '../services/avatar.dart';

class OrderDetailScreen extends StatefulWidget {
  final int orderId;

  const OrderDetailScreen({super.key, required this.orderId});

  @override
  State<OrderDetailScreen> createState() => _OrderDetailScreenState();
}

class _OrderDetailScreenState extends State<OrderDetailScreen>
    with TickerProviderStateMixin {
  Future<Map<String, dynamic>>? _future;
  Future<Map<String, dynamic>?>? _userFuture;
  Future<_Catalog>? _catalogFuture;
  bool _busy = false;
  bool _changed = false;
  late AnimationController _fadeController;
  late Animation<double> _fadeAnimation;

  @override
  void initState() {
    super.initState();
    _fadeController = AnimationController(
      duration: const Duration(milliseconds: 300),
      vsync: this,
    );
    _fadeAnimation = Tween<double>(begin: 0.0, end: 1.0).animate(
      CurvedAnimation(parent: _fadeController, curve: Curves.easeOut),
    );
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _load();
  }

  @override
  void dispose() {
    _fadeController.dispose();
    super.dispose();
  }

  void _load() {
    final client = context.read<AppState>().apiClient;
    if (client != null) {
      setState(() {
        _future = client.getJson('/admin/api/v1/orders/${widget.orderId}');
        _userFuture = _future!.then((data) async {
          final order = data['order'] as Map<String, dynamic>? ?? {};
          final userId = _asInt(order['user_id']);
          if (userId <= 0) return null;
          try {
            return await client.getJson('/admin/api/v1/users/$userId');
          } catch (e) {
            return {'_error': e.toString()};
          }
        });
        _catalogFuture ??= _loadCatalog(client);
      });
      _fadeController.forward();
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    return WillPopScope(
      onWillPop: () async {
        Navigator.pop(context, _changed);
        return false;
      },
      child: FutureBuilder<Map<String, dynamic>>(
        future: _future,
        builder: (context, snapshot) {
          if (snapshot.connectionState == ConnectionState.waiting) {
            return Scaffold(
              body: Center(
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    _LoadingIndicator(color: colorScheme.primary),
                    const SizedBox(height: 24),
                    Text(
                      '加载订单详情...',
                      style: theme.textTheme.bodyLarge?.copyWith(
                        color: colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),
            );
          }
          if (snapshot.hasError) {
            return Scaffold(
              backgroundColor: colorScheme.surface,
              appBar: AppBar(
                title: const Text('订单详情'),
                backgroundColor: colorScheme.surface,
                elevation: 0,
              ),
              body: Center(
                child: Padding(
                  padding: const EdgeInsets.all(32),
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      Container(
                        padding: const EdgeInsets.all(24),
                        decoration: BoxDecoration(
                          color: colorScheme.errorContainer.withOpacity(0.3),
                          shape: BoxShape.circle,
                        ),
                        child: Icon(
                          Icons.error_outline_rounded,
                          size: 56,
                          color: colorScheme.error,
                        ),
                      ),
                      const SizedBox(height: 24),
                      Text(
                        '加载失败',
                        style: theme.textTheme.headlineSmall?.copyWith(
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                      const SizedBox(height: 12),
                      Text(
                        snapshot.error.toString(),
                        style: theme.textTheme.bodyMedium?.copyWith(
                          color: colorScheme.onSurfaceVariant,
                        ),
                        textAlign: TextAlign.center,
                      ),
                      const SizedBox(height: 32),
                      FilledButton.icon(
                        onPressed: _load,
                        icon: const Icon(Icons.refresh_rounded),
                        label: const Text('重新加载'),
                      ),
                    ],
                  ),
                ),
              ),
            );
          }
          final data = snapshot.data ?? {};
          final order = data['order'] as Map<String, dynamic>? ?? {};
          final items =
              (data['items'] as List<dynamic>? ?? []).cast<Map<String, dynamic>>();
          final payments = (data['payments'] as List<dynamic>? ?? [])
              .cast<Map<String, dynamic>>();
          final events = _normalizeEvents(
            data['events'] as List<dynamic>? ?? [],
          );

          final status = order['status']?.toString() ?? '';
          final statusInfo = _getStatusInfo(status);

          return Scaffold(
            backgroundColor: colorScheme.surface,
            appBar: AppBar(
              title: const Text('订单详情'),
              backgroundColor: colorScheme.surface,
              elevation: 0,
              actions: [
                if (_busy)
                  Padding(
                    padding: const EdgeInsets.only(right: 16),
                    child: Center(
                      child: SizedBox(
                        width: 20,
                        height: 20,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          color: colorScheme.primary,
                        ),
                      ),
                    ),
                  )
                else
                  IconButton(
                    icon: const Icon(Icons.refresh_rounded),
                    onPressed: () {
                      _fadeController.reset();
                      _load();
                    },
                  ),
              ],
            ),
            bottomNavigationBar: SafeArea(
              top: false,
              child: Container(
                padding: const EdgeInsets.fromLTRB(16, 10, 16, 12),
                decoration: BoxDecoration(
                  color: colorScheme.surface,
                  border: Border(
                    top: BorderSide(
                      color: colorScheme.outlineVariant.withOpacity(0.5),
                    ),
                  ),
                ),
                child: _ActionBar(
                  busy: _busy,
                  status: status,
                  onApprove: () => _perform(
                    '/admin/api/v1/orders/${widget.orderId}/approve',
                  ),
                  onReject: _rejectOrder,
                  onRetry: () =>
                      _perform('/admin/api/v1/orders/${widget.orderId}/retry'),
                  onDelete: () => _perform(
                    '/admin/api/v1/orders/${widget.orderId}',
                    method: _ActionMethod.delete,
                  ),
                ),
              ),
            ),
            body: FadeTransition(
              opacity: _fadeAnimation,
              child: DefaultTabController(
                length: 4,
                child: Column(
                  children: [
                    _OrderStatusHeader(
                      orderId: order['id'] ?? '-',
                      status: status,
                      statusInfo: statusInfo,
                      amount: order['total_amount'],
                      currency: order['currency'] ?? 'CNY',
                    ),
                    Padding(
                      padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
                      child: Container(
                        decoration: BoxDecoration(
                          color: colorScheme.surface,
                          borderRadius: BorderRadius.circular(14),
                          border: Border.all(
                            color: colorScheme.outlineVariant.withOpacity(0.5),
                          ),
                        ),
                        child: TabBar(
                          labelColor: colorScheme.primary,
                          unselectedLabelColor: colorScheme.onSurfaceVariant,
                          indicatorColor: colorScheme.primary,
                          indicatorSize: TabBarIndicatorSize.tab,
                          tabs: const [
                            Tab(text: '概览'),
                            Tab(text: '付款'),
                            Tab(text: '订单项'),
                            Tab(text: '事件'),
                          ],
                        ),
                      ),
                    ),
                    Expanded(
                      child: TabBarView(
                        children: [
                          ListView(
                            padding: const EdgeInsets.fromLTRB(16, 8, 16, 88),
                            children: [
                              _OrderInfoCard(
                                order: order,
                                status: status,
                                statusInfo: statusInfo,
                              ),
                              const SizedBox(height: 12),
                              _UserInfoCard(
                                orderId: order['user_id'] ?? '-',
                                userFuture: _userFuture,
                              ),
                            ],
                          ),
                          ListView(
                            padding: const EdgeInsets.fromLTRB(16, 8, 16, 88),
                            children: [
                              _SectionHeader(
                                title: '付款信息',
                                icon: Icons.payments_rounded,
                                count: payments.length,
                              ),
                              const SizedBox(height: 12),
                              if (payments.isEmpty)
                                const _EmptyState(
                                  icon: Icons.payments_outlined,
                                  text: '暂无付款记录',
                                )
                              else
                                ...payments.map((p) => _PaymentCard(payment: p)),
                            ],
                          ),
                          ListView(
                            padding: const EdgeInsets.fromLTRB(16, 8, 16, 88),
                            children: [
                              _SectionHeader(
                                title: '订单项',
                                icon: Icons.shopping_cart_rounded,
                                count: items.length,
                              ),
                              const SizedBox(height: 12),
                              if (items.isEmpty)
                                const _EmptyState(
                                  icon: Icons.shopping_basket_outlined,
                                  text: '暂无订单项',
                                )
                              else
                                FutureBuilder<_Catalog>(
                                  future: _catalogFuture,
                                  builder: (context, catSnap) {
                                    final catalog = catSnap.data;
                                    return Column(
                                      children: items.map((item) {
                                        final pkgId = _asInt(item['package_id']);
                                        final pkg = catalog?.packages[pkgId];
                                        final plan = catalog?.planGroups[
                                            _asInt(pkg?['plan_group_id'])];
                                        final region = catalog
                                            ?.regions[_asInt(plan?['region_id'])];
                                        final regionName =
                                            (region?['name'] ?? '').toString();
                                        final lineName =
                                            (plan?['name'] ?? '').toString();
                                        final pkgName =
                                            (pkg?['name'] ?? '').toString();
                                        final specText =
                                            _specSummary(item['spec'], pkg);
                                        return _OrderItemCard(
                                          pkgName: pkgName,
                                          regionName: regionName,
                                          lineName: lineName,
                                          item: item,
                                          specText: specText,
                                        );
                                      }).toList(),
                                    );
                                  },
                                ),
                            ],
                          ),
                          ListView(
                            padding: const EdgeInsets.fromLTRB(16, 8, 16, 88),
                            children: [
                              _SectionHeader(
                                title: '事件流',
                                icon: Icons.event_rounded,
                                count: events.length,
                              ),
                              const SizedBox(height: 12),
                              if (events.isEmpty)
                                const _EmptyState(
                                  icon: Icons.history_outlined,
                                  text: '暂无事件记录',
                                )
                              else
                                ...events.map((ev) => _EventTile(ev: ev)),
                            ],
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            ),
          );
        },
      ),
    );
  }

  Future<void> _perform(
    String path, {
    _ActionMethod method = _ActionMethod.post,
    Map<String, dynamic>? body,
  }) async {
    if (_busy) return;
    setState(() => _busy = true);
    try {
      final client = context.read<AppState>().apiClient;
      if (client == null) return;
      if (method == _ActionMethod.post) {
        await client.postJson(path, body: body);
      } else if (method == _ActionMethod.delete) {
        await client.deleteJson(path);
      }
      _changed = true;
      _userFuture = null;
      _load();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          _ErrorSnackBar(message: '操作失败：$e'),
        );
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _rejectOrder() async {
    final controller = TextEditingController();
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => _RejectDialog(controller: controller),
    );
    if (ok == true && mounted) {
      await _perform(
        '/admin/api/v1/orders/${widget.orderId}/reject',
        body: {
          'reason': controller.text.trim().isEmpty
              ? 'manual'
              : controller.text.trim(),
        },
      );
    }
  }
}

// =============================================================================
// 状态头部卡片
// =============================================================================

class _OrderStatusHeader extends StatelessWidget {
  final dynamic orderId;
  final String status;
  final _StatusInfo statusInfo;
  final dynamic amount;
  final String currency;

  const _OrderStatusHeader({
    required this.orderId,
    required this.status,
    required this.statusInfo,
    required this.amount,
    required this.currency,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    return Container(
      margin: const EdgeInsets.fromLTRB(16, 8, 16, 16),
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        gradient: LinearGradient(
          colors: [
            statusInfo.color.withOpacity(0.15),
            statusInfo.color.withOpacity(0.05),
          ],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(20),
        border: Border.all(
          color: statusInfo.color.withOpacity(0.3),
          width: 1.5,
        ),
      ),
      child: Row(
        children: [
          _StatusBadge(icon: statusInfo.icon, color: statusInfo.color),
          const SizedBox(width: 20),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  '订单 #$orderId',
                  style: theme.textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.w700,
                    color: colorScheme.onSurface,
                  ),
                ),
                const SizedBox(height: 8),
                Row(
                  children: [
                    _StatusLabel(status: status, statusInfo: statusInfo),
                    const SizedBox(width: 12),
                    Text(
                      '${_money(amount)} $currency',
                      style: theme.textTheme.titleLarge?.copyWith(
                        fontWeight: FontWeight.w800,
                        color: colorScheme.primary,
                        fontSize: 22,
                      ),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _StatusBadge extends StatelessWidget {
  final IconData icon;
  final Color color;

  const _StatusBadge({required this.icon, required this.color});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: color.withOpacity(0.2),
        shape: BoxShape.circle,
      ),
      child: Icon(
        icon,
        color: color,
        size: 32,
      ),
    );
  }
}

class _StatusLabel extends StatelessWidget {
  final String status;
  final _StatusInfo statusInfo;

  const _StatusLabel({
    required this.status,
    required this.statusInfo,
  });

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: statusInfo.color.withOpacity(0.2),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Text(
        _statusText(status),
        style: TextStyle(
          color: statusInfo.color,
          fontWeight: FontWeight.w700,
          fontSize: 13,
        ),
      ),
    );
  }
}

// =============================================================================
// 订单信息卡片
// =============================================================================

class _OrderInfoCard extends StatelessWidget {
  final Map<String, dynamic> order;
  final String status;
  final _StatusInfo statusInfo;

  const _OrderInfoCard({
    required this.order,
    required this.status,
    required this.statusInfo,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    return _InfoCard(
      title: '订单信息',
      icon: Icons.receipt_long_rounded,
      dense: true,
      columns: 2,
      lines: [
        _InfoLine(
          label: '订单号',
          value: order['order_no']?.toString() ?? '-',
          icon: Icons.confirmation_number_rounded,
        ),
        _InfoLine(
          label: '状态',
          value: _statusText(status),
          valueColor: statusInfo.color,
          icon: Icons.info_rounded,
        ),
        _InfoLine(
          label: '金额',
          value: '${_money(order['total_amount'])} ${order['currency'] ?? 'CNY'}',
          valueColor: colorScheme.primary,
          valueBold: true,
          icon: Icons.payments_rounded,
        ),
        _InfoLine(
          label: '创建时间',
          value: _formatLocal(order['created_at']?.toString() ?? ''),
          icon: Icons.access_time_rounded,
        ),
        if (order['approved_by'] != null)
          _InfoLine(
            label: '审核人',
            value: order['approved_by'].toString(),
            icon: Icons.person_rounded,
          ),
        if (order['approved_at'] != null)
          _InfoLine(
            label: '审核时间',
            value: _formatLocal(order['approved_at'].toString()),
            icon: Icons.event_available_rounded,
          ),
        if ((order['rejected_reason'] ?? '').toString().isNotEmpty)
          _InfoLine(
            label: '驳回原因',
            value: order['rejected_reason'].toString(),
            valueColor: colorScheme.error,
            icon: Icons.cancel_rounded,
            fullWidth: true,
          ),
      ],
    );
  }
}

// =============================================================================
// 用户信息卡片
// =============================================================================

class _UserInfoCard extends StatelessWidget {
  final dynamic orderId;
  final Future<Map<String, dynamic>?>? userFuture;

  const _UserInfoCard({
    required this.orderId,
    required this.userFuture,
  });

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

    return FutureBuilder<Map<String, dynamic>?>(
      future: userFuture,
      builder: (context, userSnap) {
        final raw = userSnap.data ?? {};
        final user = raw['user'] is Map<String, dynamic>
            ? raw['user'] as Map<String, dynamic>
            : (raw['data'] is Map<String, dynamic>
                  ? raw['data'] as Map<String, dynamic>
                  : raw);
        final err = raw['_error']?.toString();
        final qq = (user['qq'] ?? '').toString();
        final baseUrl =
            context.read<AppState>().apiClient?.baseUrl ?? '';
        final avatarUrl = _resolveAvatar(
          user['avatar_url'] ?? user['avatar'],
          qq,
          baseUrl,
        );

        return _InfoCard(
          title: '用户信息',
          icon: Icons.person_rounded,
          leading: _Avatar(url: avatarUrl, radius: 20),
          dense: true,
          columns: 2,
          lines: [
            _InfoLine(
              label: '用户ID',
              value: orderId.toString(),
              icon: Icons.tag_rounded,
            ),
            _InfoLine(
              label: '用户名',
              value: user['username']?.toString() ?? '-',
              icon: Icons.person_rounded,
            ),
            _InfoLine(
              label: '邮箱',
              value: user['email']?.toString() ?? '-',
              icon: Icons.email_rounded,
            ),
            _InfoLine(
              label: '手机号',
              value: user['phone']?.toString() ?? '-',
              icon: Icons.phone_rounded,
            ),
            _InfoLine(
              label: 'QQ',
              value: qq.isEmpty ? '-' : qq,
              icon: Icons.chat_rounded,
            ),
            if (err != null)
              _InfoLine(
                label: '错误',
                value: '用户信息加载失败：$err',
                valueColor: colorScheme.error,
                icon: Icons.error_outline_rounded,
                fullWidth: true,
              ),
          ],
        );
      },
    );
  }
}

// =============================================================================
// 操作按钮栏
// =============================================================================

enum _ActionMethod { post, delete }

class _ActionBar extends StatelessWidget {
  final bool busy;
  final String status;
  final VoidCallback onApprove;
  final VoidCallback onReject;
  final VoidCallback onRetry;
  final VoidCallback onDelete;

  const _ActionBar({
    required this.busy,
    required this.status,
    required this.onApprove,
    required this.onReject,
    required this.onRetry,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    final canApprove = status == 'pending_review' || status == 'rejected';
    final canReject = status == 'pending_review';
    final canRetry = status == 'failed';
    final canDelete = status == 'pending_review' ||
        status == 'rejected' ||
        status == 'failed' ||
        status == 'canceled';
    final isDisabled = status == 'active' || status == 'completed';

    return Container(
      padding: const EdgeInsets.all(4),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surface,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(
          color: Theme.of(context).colorScheme.outlineVariant.withOpacity(0.5),
          width: 1,
        ),
      ),
      child: Row(
        children: [
          Expanded(
            child: _ActionButton(
              label: '通过',
              icon: Icons.check_circle_rounded,
              isPrimary: true,
              isDisabled: isDisabled || !canApprove,
              isLoading: busy && canApprove,
              onPressed: onApprove,
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: _ActionButton(
              label: '驳回',
              icon: Icons.cancel_rounded,
              isDisabled: isDisabled || !canReject,
              isLoading: busy && canReject,
              onPressed: onReject,
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: _ActionButton(
              label: '重试',
              icon: Icons.refresh_rounded,
              isDisabled: isDisabled || !canRetry,
              isLoading: busy && canRetry,
              onPressed: onRetry,
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: _ActionButton(
              label: '删除',
              icon: Icons.delete_rounded,
              isDestructive: true,
              isDisabled: isDisabled || !canDelete,
              isLoading: busy && canDelete,
              onPressed: onDelete,
            ),
          ),
        ],
      ),
    );
  }
}

class _ActionButton extends StatelessWidget {
  final String label;
  final IconData icon;
  final bool isPrimary;
  final bool isDestructive;
  final bool isDisabled;
  final bool isLoading;
  final VoidCallback onPressed;

  const _ActionButton({
    required this.label,
    required this.icon,
    this.isPrimary = false,
    this.isDestructive = false,
    this.isDisabled = false,
    this.isLoading = false,
    required this.onPressed,
  });

  Color _bgColor(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    if (isDisabled) return colorScheme.surfaceContainerHighest;
    if (isDestructive) return colorScheme.errorContainer;
    if (isPrimary) return colorScheme.primary;
    return colorScheme.secondaryContainer;
  }

  Color _textColor(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    if (isDisabled) return colorScheme.onSurface.withOpacity(0.38);
    if (isDestructive) return colorScheme.error;
    if (isPrimary) return colorScheme.onPrimary;
    return colorScheme.onSecondaryContainer;
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final bgColor = _bgColor(context);
    final textColor = _textColor(context);

    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: isDisabled || isLoading ? null : onPressed,
        borderRadius: BorderRadius.circular(12),
        child: Container(
          padding: const EdgeInsets.symmetric(vertical: 12),
          decoration: BoxDecoration(
            color: bgColor,
            borderRadius: BorderRadius.circular(12),
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              if (isLoading)
                SizedBox(
                  width: 20,
                  height: 20,
                  child: CircularProgressIndicator(
                    strokeWidth: 2,
                    valueColor: AlwaysStoppedAnimation<Color>(textColor),
                  ),
                )
              else
                Icon(
                  icon,
                  color: textColor,
                  size: 20,
                ),
              const SizedBox(height: 4),
              Text(
                label,
                style: theme.textTheme.labelSmall?.copyWith(
                  color: textColor,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// =============================================================================
// 驳回对话框
// =============================================================================

class _RejectDialog extends StatelessWidget {
  final TextEditingController controller;

  const _RejectDialog({required this.controller});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    return AlertDialog(
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(20),
      ),
      title: Row(
        children: [
          Container(
            padding: const EdgeInsets.all(10),
            decoration: BoxDecoration(
              color: colorScheme.errorContainer.withOpacity(0.3),
              borderRadius: BorderRadius.circular(10),
            ),
            child: Icon(
              Icons.cancel_rounded,
              color: colorScheme.error,
              size: 22,
            ),
          ),
          const SizedBox(width: 12),
          const Text('驳回订单'),
        ],
      ),
      content: TextField(
        controller: controller,
        decoration: InputDecoration(
          hintText: '请输入驳回原因',
          filled: true,
          border: OutlineInputBorder(
            borderRadius: BorderRadius.circular(12),
          ),
        ),
        maxLines: 3,
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context, false),
          child: const Text('取消'),
        ),
        FilledButton(
          onPressed: () => Navigator.pop(context, true),
          style: FilledButton.styleFrom(
            backgroundColor: colorScheme.error,
          ),
          child: const Text('确认驳回'),
        ),
      ],
    );
  }
}

// =============================================================================
// 分节标题
// =============================================================================

class _SectionHeader extends StatelessWidget {
  final String title;
  final IconData icon;
  final int count;

  const _SectionHeader({
    required this.title,
    required this.icon,
    this.count = 0,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    return Row(
      children: [
        Container(
          padding: const EdgeInsets.all(10),
          decoration: BoxDecoration(
            color: colorScheme.primaryContainer.withOpacity(0.5),
            borderRadius: BorderRadius.circular(10),
          ),
          child: Icon(
            icon,
            size: 20,
            color: colorScheme.primary,
          ),
        ),
        const SizedBox(width: 12),
        Text(
          title,
          style: theme.textTheme.titleMedium?.copyWith(
            fontWeight: FontWeight.w700,
            color: colorScheme.onSurface,
          ),
        ),
        if (count > 0) ...[
          const SizedBox(width: 8),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
            decoration: BoxDecoration(
              color: colorScheme.primaryContainer.withOpacity(0.5),
              borderRadius: BorderRadius.circular(10),
            ),
            child: Text(
              '$count',
              style: theme.textTheme.labelSmall?.copyWith(
                color: colorScheme.primary,
                fontWeight: FontWeight.w700,
              ),
            ),
          ),
        ],
      ],
    );
  }
}

// =============================================================================
// 信息卡片基类
// =============================================================================

class _InfoCard extends StatelessWidget {
  final String title;
  final IconData icon;
  final List<_InfoLine> lines;
  final Widget? leading;
  final bool dense;
  final int columns;

  const _InfoCard({
    required this.title,
    required this.lines,
    required this.icon,
    this.leading,
    this.dense = false,
    this.columns = 1,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    final headerPadding =
        dense ? const EdgeInsets.fromLTRB(12, 12, 12, 10) : const EdgeInsets.fromLTRB(16, 16, 16, 12);
    final contentPadding = dense ? const EdgeInsets.all(12) : const EdgeInsets.all(16);

    return Container(
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(
          color: colorScheme.outlineVariant.withOpacity(0.5),
          width: 1,
        ),
        boxShadow: [
          BoxShadow(
            color: colorScheme.shadow.withOpacity(0.05),
            blurRadius: 8,
            offset: const Offset(0, 2),
          ),
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // 标题栏
          Container(
            padding: headerPadding,
            decoration: BoxDecoration(
              color: colorScheme.surfaceContainerHighest.withOpacity(0.4),
              borderRadius: const BorderRadius.only(
                topLeft: Radius.circular(16),
                topRight: Radius.circular(16),
              ),
            ),
            child: Row(
              children: [
                Container(
                  padding: const EdgeInsets.all(8),
                  decoration: BoxDecoration(
                    color: colorScheme.primaryContainer.withOpacity(0.5),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Icon(
                    icon,
                    size: 18,
                    color: colorScheme.primary,
                  ),
                ),
                const SizedBox(width: 12),
                if (leading != null) ...[leading!, const SizedBox(width: 12)],
                Text(
                  title,
                  style: theme.textTheme.titleSmall?.copyWith(
                    fontWeight: FontWeight.w700,
                    color: colorScheme.onSurface,
                  ),
                ),
              ],
            ),
          ),
          // 内容
          Padding(
            padding: contentPadding,
            child: columns <= 1
                ? Column(
                    children: [
                      for (var i = 0; i < lines.length; i++) ...[
                        lines[i],
                        if (i < lines.length - 1)
                          Padding(
                            padding: const EdgeInsets.only(
                              left: 80,
                              top: 12,
                              bottom: 12,
                            ),
                            child: Divider(
                              height: 1,
                              thickness: 1,
                              color: colorScheme.outlineVariant.withOpacity(0.3),
                            ),
                          ),
                      ],
                    ],
                  )
                : LayoutBuilder(
                    builder: (context, constraints) {
                      final spacing = 12.0;
                      final colWidth =
                          (constraints.maxWidth - spacing * (columns - 1)) / columns;
                      final compactLines =
                          lines.where((line) => !line.fullWidth).toList();
                      final fullLines =
                          lines.where((line) => line.fullWidth).toList();

                      return Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Wrap(
                            spacing: spacing,
                            runSpacing: 10,
                            children: compactLines
                                .map(
                                  (line) => SizedBox(
                                    width: colWidth,
                                    child: line,
                                  ),
                                )
                                .toList(),
                          ),
                          if (fullLines.isNotEmpty) ...[
                            const SizedBox(height: 12),
                            ...fullLines.map(
                              (line) => Padding(
                                padding: const EdgeInsets.only(bottom: 10),
                                child: line,
                              ),
                            ),
                          ],
                        ],
                      );
                    },
                  ),
          ),
        ],
      ),
    );
  }
}

class _InfoLine extends StatelessWidget {
  final String label;
  final String value;
  final Color? valueColor;
  final bool valueBold;
  final IconData? icon;
  final bool fullWidth;

  const _InfoLine({
    required this.label,
    required this.value,
    this.valueColor,
    this.valueBold = false,
    this.icon,
    this.fullWidth = false,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (icon != null)
          Container(
            padding: const EdgeInsets.all(6),
            decoration: BoxDecoration(
              color: colorScheme.surfaceContainerHighest.withOpacity(0.5),
              borderRadius: BorderRadius.circular(6),
            ),
            child: Icon(
              icon,
              size: 14,
              color: colorScheme.onSurfaceVariant,
            ),
          ),
        if (icon != null) const SizedBox(width: 10),
        SizedBox(
          width: icon == null ? 70 : 60,
          child: Text(
            label,
            style: theme.textTheme.bodySmall?.copyWith(
              color: colorScheme.onSurfaceVariant,
              fontWeight: FontWeight.w500,
            ),
          ),
        ),
        const SizedBox(width: 10),
        Expanded(
          child: Text(
            value,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: valueColor ?? colorScheme.onSurface,
              fontWeight: valueBold ? FontWeight.w700 : FontWeight.w500,
            ),
          ),
        ),
      ],
    );
  }
}

// =============================================================================
// 空状态
// =============================================================================

class _EmptyState extends StatelessWidget {
  final IconData icon;
  final String text;

  const _EmptyState({
    required this.icon,
    required this.text,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    return Container(
      padding: const EdgeInsets.all(32),
      decoration: BoxDecoration(
        color: colorScheme.surfaceContainerHighest.withOpacity(0.3),
        borderRadius: BorderRadius.circular(16),
        border: Border.all(
          color: colorScheme.outlineVariant.withOpacity(0.3),
          width: 1,
        ),
      ),
      child: Column(
        children: [
          Icon(
            icon,
            size: 48,
            color: colorScheme.onSurfaceVariant.withOpacity(0.5),
          ),
          const SizedBox(height: 12),
          Text(
            text,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: colorScheme.onSurfaceVariant,
            ),
          ),
        ],
      ),
    );
  }
}

// =============================================================================
// 付款卡片
// =============================================================================

class _PaymentCard extends StatelessWidget {
  final Map<String, dynamic> payment;

  const _PaymentCard({required this.payment});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    final status = payment['status']?.toString() ?? '';
    final statusColor = _getPaymentStatusColor(status);

    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(
          color: colorScheme.outlineVariant.withOpacity(0.5),
          width: 1,
        ),
        boxShadow: [
          BoxShadow(
            color: colorScheme.shadow.withOpacity(0.05),
            blurRadius: 8,
            offset: const Offset(0, 2),
          ),
        ],
      ),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Container(
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: colorScheme.primaryContainer.withOpacity(0.5),
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Icon(
                    Icons.payment_rounded,
                    size: 22,
                    color: colorScheme.primary,
                  ),
                ),
                const SizedBox(width: 14),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        payment['method']?.toString() ?? '-',
                        style: theme.textTheme.titleSmall?.copyWith(
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                      const SizedBox(height: 4),
                      Container(
                        padding: const EdgeInsets.symmetric(
                          horizontal: 10,
                          vertical: 4,
                        ),
                        decoration: BoxDecoration(
                          color: statusColor.withOpacity(0.12),
                          borderRadius: BorderRadius.circular(8),
                        ),
                        child: Text(
                          _paymentStatusText(status),
                          style: TextStyle(
                            color: statusColor,
                            fontSize: 12,
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
                Text(
                  '¥${_money(payment['amount'])}',
                  style: theme.textTheme.titleLarge?.copyWith(
                    fontWeight: FontWeight.w800,
                    color: colorScheme.primary,
                  ),
                ),
              ],
            ),
            if ((payment['trade_no'] ?? '').toString().isNotEmpty ||
                (payment['note'] ?? '').toString().isNotEmpty ||
                payment['created_at'] != null)
              Padding(
                padding: const EdgeInsets.only(top: 14, left: 48),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    if ((payment['trade_no'] ?? '').toString().isNotEmpty)
                      _DetailRow(
                        icon: Icons.confirmation_number_rounded,
                        label: '交易号',
                        value: payment['trade_no'].toString(),
                      ),
                    if ((payment['note'] ?? '').toString().isNotEmpty)
                      _DetailRow(
                        icon: Icons.note_rounded,
                        label: '备注',
                        value: payment['note'].toString(),
                      ),
                    if (payment['created_at'] != null)
                      _DetailRow(
                        icon: Icons.access_time_rounded,
                        label: '创建时间',
                        value: _formatLocal(payment['created_at'].toString()),
                      ),
                  ],
                ),
              ),
          ],
        ),
      ),
    );
  }
}

// =============================================================================
// 订单项卡片
// =============================================================================

class _OrderItemCard extends StatelessWidget {
  final String pkgName;
  final String regionName;
  final String lineName;
  final Map<String, dynamic> item;
  final String specText;

  const _OrderItemCard({
    required this.pkgName,
    required this.regionName,
    required this.lineName,
    required this.item,
    required this.specText,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(
          color: colorScheme.outlineVariant.withOpacity(0.5),
          width: 1,
        ),
        boxShadow: [
          BoxShadow(
            color: colorScheme.shadow.withOpacity(0.05),
            blurRadius: 8,
            offset: const Offset(0, 2),
          ),
        ],
      ),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // 标题行
            Row(
              children: [
                Container(
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: colorScheme.secondaryContainer.withOpacity(0.5),
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Icon(
                    Icons.inventory_2_rounded,
                    size: 22,
                    color: colorScheme.secondary,
                  ),
                ),
                const SizedBox(width: 14),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        pkgName.isNotEmpty ? pkgName : '套餐',
                        style: theme.textTheme.titleSmall?.copyWith(
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                      const SizedBox(height: 2),
                      Text(
                        'ID: ${item['package_id'] ?? '-'} · 系统: ${item['system_id'] ?? '-'}',
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: colorScheme.onSurfaceVariant,
                          fontFamily: 'monospace',
                        ),
                      ),
                    ],
                  ),
                ),
                Text(
                  '¥${_money(item['amount'])}',
                  style: theme.textTheme.titleLarge?.copyWith(
                    fontWeight: FontWeight.w800,
                    color: colorScheme.primary,
                  ),
                ),
              ],
            ),
            // 详细信息
            Padding(
              padding: const EdgeInsets.only(top: 14, left: 48),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  _DetailRow(
                    icon: Icons.format_list_numbered_rounded,
                    label: '数量',
                    value: item['qty']?.toString() ?? '-',
                  ),
                  if (regionName.isNotEmpty || lineName.isNotEmpty)
                    _DetailRow(
                      icon: Icons.public_rounded,
                      label: '地区/线路',
                      value:
                          '${regionName.isNotEmpty ? regionName : '-'} / ${lineName.isNotEmpty ? lineName : '-'}',
                    ),
                  if ((item['action'] ?? '').toString().isNotEmpty)
                    _DetailRow(
                      icon: Icons.bolt_rounded,
                      label: '动作',
                      value: item['action'].toString(),
                    ),
                  if ((item['duration_months'] ?? '').toString().isNotEmpty)
                    _DetailRow(
                      icon: Icons.calendar_month_rounded,
                      label: '时长',
                      value: '${item['duration_months']} 个月',
                    ),
                  if ((item['automation_instance_id'] ?? '').toString().isNotEmpty)
                    _DetailRow(
                      icon: Icons.cloud_rounded,
                      label: '实例',
                      value: item['automation_instance_id'].toString(),
                    ),
                  _DetailRow(
                    icon: Icons.memory_rounded,
                    label: '规格',
                    value: specText,
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _DetailRow extends StatelessWidget {
  final IconData icon;
  final String label;
  final String value;

  const _DetailRow({
    required this.icon,
    required this.label,
    required this.value,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(
            icon,
            size: 16,
            color: colorScheme.onSurfaceVariant,
          ),
          const SizedBox(width: 10),
          Text(
            '$label：',
            style: theme.textTheme.bodySmall?.copyWith(
              color: colorScheme.onSurfaceVariant,
              fontWeight: FontWeight.w500,
            ),
          ),
          Expanded(
            child: Text(
              value,
              style: theme.textTheme.bodySmall?.copyWith(
                color: colorScheme.onSurface,
                fontWeight: FontWeight.w500,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

// =============================================================================
// 事件卡片
// =============================================================================

class _EventTile extends StatelessWidget {
  final Map<String, dynamic> ev;

  const _EventTile({required this.ev});

  @override
  Widget build(BuildContext context) {
    final type = ev['type']?.toString() ?? '';
    final meta = _eventMeta(type);
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    return Container(
      margin: const EdgeInsets.only(bottom: 10),
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(
          color: colorScheme.outlineVariant.withOpacity(0.5),
          width: 1,
        ),
      ),
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Row(
          children: [
            Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: meta.color.withOpacity(0.12),
                borderRadius: BorderRadius.circular(12),
              ),
              child: Icon(
                meta.icon,
                color: meta.color,
                size: 20,
              ),
            ),
            const SizedBox(width: 14),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    meta.label,
                    style: theme.textTheme.titleSmall?.copyWith(
                      fontWeight: FontWeight.w700,
                      color: colorScheme.onSurface,
                    ),
                  ),
                  if (_eventSummary(ev).isNotEmpty)
                    Padding(
                      padding: const EdgeInsets.only(top: 2),
                      child: Text(
                        _eventSummary(ev),
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: colorScheme.onSurfaceVariant,
                        ),
                      ),
                    ),
                ],
              ),
            ),
            Text(
              _formatTime(ev['created_at']?.toString() ?? ''),
              style: theme.textTheme.bodySmall?.copyWith(
                color: colorScheme.onSurfaceVariant,
                fontFamily: 'monospace',
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// =============================================================================
// 头像组件
// =============================================================================

class _Avatar extends StatelessWidget {
  final String url;
  final double radius;

  const _Avatar({required this.url, required this.radius});

  @override
  Widget build(BuildContext context) {
    final size = radius * 2;
    final colorScheme = Theme.of(context).colorScheme;

    if (url.isEmpty) {
      return Container(
        width: size,
        height: size,
        decoration: BoxDecoration(
          color: colorScheme.primaryContainer.withOpacity(0.3),
          shape: BoxShape.circle,
        ),
        child: Icon(
          Icons.person_rounded,
          size: radius,
          color: colorScheme.primary,
        ),
      );
    }
    final session = context.read<AppState>().session;
    final headers = avatarHeaders(
      token: session?.token,
      apiKey: session?.apiKey,
    );
    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        border: Border.all(
          color: colorScheme.outlineVariant.withOpacity(0.5),
          width: 2,
        ),
      ),
      child: ClipOval(
        child: Image.network(
          url,
          width: size,
          height: size,
          fit: BoxFit.cover,
          headers: headers.isEmpty ? null : headers,
          errorBuilder: (context, error, stack) {
            return Container(
              width: size,
              height: size,
              color: colorScheme.primaryContainer.withOpacity(0.3),
              child: Icon(
                Icons.person_rounded,
                size: radius,
                color: colorScheme.primary,
              ),
            );
          },
        ),
      ),
    );
  }
}

// =============================================================================
// 加载指示器
// =============================================================================

class _LoadingIndicator extends StatelessWidget {
  final Color color;

  const _LoadingIndicator({required this.color});

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 48,
      height: 48,
      child: CircularProgressIndicator(
        strokeWidth: 3,
        color: color,
      ),
    );
  }
}

// =============================================================================
// 错误提示条
// =============================================================================

class _ErrorSnackBar extends SnackBar {
  final String message;

  _ErrorSnackBar({required this.message})
      : super(
          content: Row(
            children: const [
              Icon(Icons.error_outline_rounded, color: Colors.white),
              SizedBox(width: 12),
              Expanded(child: Text('操作失败，请稍后重试')),
            ],
          ),
          backgroundColor: Colors.red,
          behavior: SnackBarBehavior.floating,
        );
}

// =============================================================================
// 状态和格式化工具类
// =============================================================================

class _StatusInfo {
  final String label;
  final IconData icon;
  final Color color;
  const _StatusInfo(this.label, this.icon, this.color);
}

class _EventMeta {
  final String label;
  final IconData icon;
  final Color color;
  const _EventMeta(this.label, this.icon, this.color);
}

class _Catalog {
  final Map<int, Map<String, dynamic>> packages;
  final Map<int, Map<String, dynamic>> planGroups;
  final Map<int, Map<String, dynamic>> regions;

  const _Catalog({
    required this.packages,
    required this.planGroups,
    required this.regions,
  });
}

// =============================================================================
// 工具函数
// =============================================================================

_StatusInfo _getStatusInfo(String status) {
  switch (status) {
    case 'pending_payment':
      return const _StatusInfo('待支付', Icons.schedule_rounded, Color(0xFFEF6C00));
    case 'pending_review':
      return const _StatusInfo(
          '待审核', Icons.hourglass_top_rounded, Color(0xFFEF6C00));
    case 'approved':
      return const _StatusInfo('已通过', Icons.check_circle_rounded, Color(0xFF00A68C));
    case 'provisioning':
      return const _StatusInfo(
          '开通中', Icons.rocket_launch_rounded, Color(0xFF1E88E5));
    case 'active':
      return const _StatusInfo('已完成', Icons.verified_rounded, Color(0xFF00A68C));
    case 'failed':
      return const _StatusInfo('失败', Icons.error_rounded, Color(0xFFD32F2F));
    case 'rejected':
      return const _StatusInfo('已驳回', Icons.cancel_rounded, Color(0xFFD32F2F));
    case 'canceled':
      return const _StatusInfo('已取消', Icons.block_rounded, Color(0xFF757575));
    default:
      return _StatusInfo(status, Icons.info_rounded, const Color(0xFF546E7A));
  }
}

String _statusText(String status) {
  switch (status) {
    case 'pending_payment':
      return '待支付';
    case 'pending_review':
      return '待审核';
    case 'approved':
      return '已通过';
    case 'provisioning':
      return '开通中';
    case 'active':
      return '已完成';
    case 'failed':
      return '失败';
    case 'rejected':
      return '已驳回';
    case 'canceled':
      return '已取消';
    default:
      return status;
  }
}

String _paymentStatusText(String status) {
  switch (status) {
    case 'pending':
      return '待支付';
    case 'pending_review':
      return '待审核';
    case 'approved':
      return '已通过';
    case 'rejected':
      return '已驳回';
    case 'paid':
      return '已支付';
    default:
      return status;
  }
}

Color _getPaymentStatusColor(String status) {
  switch (status) {
    case 'paid':
      return const Color(0xFF00A68C);
    case 'approved':
      return const Color(0xFF00A68C);
    case 'pending':
      return const Color(0xFFEF6C00);
    case 'pending_review':
      return const Color(0xFFEF6C00);
    case 'rejected':
      return const Color(0xFFD32F2F);
    default:
      return const Color(0xFF546E7A);
  }
}

_EventMeta _eventMeta(String type) {
  switch (type) {
    case 'order.pending_payment':
      return const _EventMeta('待支付', Icons.schedule, Color(0xFFEF6C00));
    case 'order.pending_review':
      return const _EventMeta('待审核', Icons.hourglass_top, Color(0xFFEF6C00));
    case 'order.approved':
    case 'order_approved':
      return const _EventMeta('已通过', Icons.check_circle, Color(0xFF00A68C));
    case 'order.provisioning':
    case 'provisioning_started':
    case 'provisioning_progress':
      return const _EventMeta('开通中', Icons.rocket_launch, Color(0xFF1E88E5));
    case 'order.completed':
    case 'provisioning_completed':
      return const _EventMeta('已完成', Icons.verified, Color(0xFF00A68C));
    case 'order.failed':
    case 'provisioning_failed':
      return const _EventMeta('开通失败', Icons.error, Color(0xFFD32F2F));
    case 'order.rejected':
    case 'order_rejected':
      return const _EventMeta('已驳回', Icons.cancel, Color(0xFFD32F2F));
    case 'order.canceled':
      return const _EventMeta('已取消', Icons.block, Color(0xFF757575));
    case 'order_created':
      return const _EventMeta('订单创建', Icons.receipt_long, Color(0xFF546E7A));
    case 'order_paid':
      return const _EventMeta('订单支付', Icons.payments, Color(0xFF2E7D32));
    case 'payment_created':
      return const _EventMeta('支付创建', Icons.payments, Color(0xFF2E7D32));
    case 'payment_approved':
      return const _EventMeta('支付通过', Icons.check_circle, Color(0xFF00A68C));
    case 'payment_rejected':
      return const _EventMeta('支付驳回', Icons.cancel, Color(0xFFD32F2F));
    case 'status_changed':
      return const _EventMeta('状态变更', Icons.sync, Color(0xFF546E7A));
    default:
      return _EventMeta(
        _eventTypeText(type),
        Icons.info,
        const Color(0xFF546E7A),
      );
  }
}

String _eventTypeText(String type) {
  const map = {
    'order.pending_payment': '待支付',
    'order.pending_review': '待审核',
    'order.approved': '已通过',
    'order.provisioning': '开通中',
    'order.completed': '已完成',
    'order.failed': '开通失败',
    'order.rejected': '已驳回',
    'order.canceled': '已取消',
    'order_created': '订单创建',
    'order_paid': '订单支付',
    'order_approved': '订单通过',
    'order_rejected': '订单驳回',
    'provisioning_started': '开始开通',
    'provisioning_progress': '开通进度',
    'provisioning_completed': '开通完成',
    'provisioning_failed': '开通失败',
    'payment_created': '支付创建',
    'payment_approved': '支付通过',
    'payment_rejected': '支付驳回',
    'status_changed': '状态变更',
  };
  return map[type] ?? type;
}

String _eventSummary(Map<String, dynamic> ev) {
  final data = ev['data'];
  if (data is Map<String, dynamic>) {
    final reason = data['reason'];
    final message = data['message'];
    if (reason != null) return '原因：$reason';
    if (message != null) return '消息：$message';
  }
  return '';
}

List<Map<String, dynamic>> _normalizeEvents(List<dynamic> raw) {
  return raw.map((ev) {
    final map = Map<String, dynamic>.from(ev as Map);
    if (map['data'] is String) {
      try {
        map['data'] = jsonDecode(map['data']);
      } catch (_) {}
    }
    return map;
  }).toList();
}

String _formatLocal(String raw) {
  if (raw.isEmpty) return '';
  final dt = DateTime.tryParse(raw);
  if (dt == null) return raw;
  final local = dt.toLocal();
  return '${local.year}-${_pad2(local.month)}-${_pad2(local.day)} ${_pad2(local.hour)}:${_pad2(local.minute)}:${_pad2(local.second)}';
}

String _formatTime(String raw) {
  if (raw.isEmpty) return '';
  final dt = DateTime.tryParse(raw);
  if (dt == null) return raw;
  final local = dt.toLocal();
  return '${_pad2(local.hour)}:${_pad2(local.minute)}:${_pad2(local.second)}';
}

String _pad2(int v) => v.toString().padLeft(2, '0');

String _money(dynamic value) {
  if (value is num) return value.toStringAsFixed(2);
  return value?.toString() ?? '0.00';
}

String _specSummary(dynamic spec, Map<String, dynamic>? pkg) {
  if (spec == null) return '-';
  dynamic parsed = spec;
  if (spec is String) {
    try {
      parsed = jsonDecode(spec);
    } catch (_) {
      parsed = spec;
    }
  }
  if (parsed is Map<String, dynamic>) {
    final addC = _asInt(parsed['add_cores']);
    final addMem = _asInt(parsed['add_mem_gb']);
    final addDisk = _asInt(parsed['add_disk_gb']);
    final addBw = _asInt(parsed['add_bw_mbps']);
    int cpu = _asInt(parsed['cpu'] ?? parsed['cores']);
    int mem = _asInt(parsed['memory_gb'] ?? parsed['memory']);
    int disk = _asInt(parsed['disk_gb'] ?? parsed['disk']);
    int bw = _asInt(parsed['bandwidth_mbps'] ?? parsed['bandwidth']);
    final baseCpu = _asInt(pkg?['cores']);
    final baseMem = _asInt(pkg?['memory_gb']);
    final baseDisk = _asInt(pkg?['disk_gb']);
    final baseBw = _asInt(pkg?['bandwidth_mbps']);
    if (cpu == 0) cpu = baseCpu + addC;
    if (mem == 0) mem = baseMem + addMem;
    if (disk == 0) disk = baseDisk + addDisk;
    if (bw == 0) bw = baseBw + addBw;
    final duration = parsed['duration_months'];
    final billing = parsed['billing_cycle_id'];
    final cycleQty = parsed['cycle_qty'];
    final hasAny = cpu > 0 || mem > 0 || disk > 0 || bw > 0;
    final parts = <String>[];
    parts.add(hasAny ? '$cpu''C ${mem}G ${disk}G ${bw}M' : '默认规格');
    if (duration != null) parts.add('时长 $duration 个月');
    if (billing != null) parts.add('计费ID $billing');
    if (cycleQty != null) parts.add('周期 $cycleQty');
    return parts.join(' · ');
  }
  return parsed.toString();
}

String _resolveAvatar(dynamic avatarValue, String qq, String baseUrl) {
  return resolveAvatarUrl(
    baseUrl: baseUrl,
    qq: qq,
    avatarUrl: avatarValue?.toString(),
  );
}

Future<_Catalog> _loadCatalog(ApiClient client) async {
  final packagesResp = await client.getJson('/admin/api/v1/packages');
  final planResp = await client.getJson('/admin/api/v1/plan-groups');
  final regionsResp = await client.getJson('/admin/api/v1/regions');
  final packages = <int, Map<String, dynamic>>{};
  for (final item in (packagesResp['items'] as List<dynamic>? ?? [])) {
    final map = (item as Map).cast<String, dynamic>();
    packages[_asInt(map['id'])] = map;
  }
  final planGroups = <int, Map<String, dynamic>>{};
  for (final item in (planResp['items'] as List<dynamic>? ?? [])) {
    final map = (item as Map).cast<String, dynamic>();
    planGroups[_asInt(map['id'])] = map;
  }
  final regions = <int, Map<String, dynamic>>{};
  for (final item in (regionsResp['items'] as List<dynamic>? ?? [])) {
    final map = (item as Map).cast<String, dynamic>();
    regions[_asInt(map['id'])] = map;
  }
  return _Catalog(packages: packages, planGroups: planGroups, regions: regions);
}

int _asInt(dynamic value) {
  if (value is int) return value;
  if (value is num) return value.toInt();
  if (value is String) return int.tryParse(value) ?? 0;
  return 0;
}
