import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../core/constants/app_colors.dart';
import '../../../core/constants/app_strings.dart';
import '../../../core/utils/date_formatter.dart';
import '../../../core/utils/money_formatter.dart';
import '../../providers/dashboard_provider.dart';
import '../../providers/refresh_provider.dart';
import '../../widgets/common/empty_state.dart';
import '../../widgets/charts/line_chart.dart';
import '../../widgets/charts/pie_chart.dart';

/// Dashboard 页面
class DashboardPage extends ConsumerStatefulWidget {
  const DashboardPage({super.key});

  @override
  ConsumerState<DashboardPage> createState() => _DashboardPageState();
}

class _DashboardPageState extends ConsumerState<DashboardPage> {
  ProviderSubscription<RefreshEvent?>? _refreshSub;

  @override
  void initState() {
    super.initState();
    _refreshSub = ref.listenManual<RefreshEvent?>(pageRefreshProvider, (_, next) {
      if (next?.route == '/console') {
        ref.read(dashboardProvider.notifier).refresh();
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
    final dashboardState = ref.watch(dashboardProvider);

    return Scaffold(
      body: dashboardState.loading
          ? const Center(child: CircularProgressIndicator())
          : dashboardState.error != null
              ? _buildError(context, ref, dashboardState.error!)
              : _buildContent(context, dashboardState),
    );
  }

  Widget _buildError(BuildContext context, WidgetRef ref, String error) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Text(error, style: const TextStyle(color: Colors.red)),
          const SizedBox(height: 16),
          ElevatedButton(
            onPressed: () => ref.read(dashboardProvider.notifier).refresh(),
            child: const Text(AppStrings.retry),
          ),
        ],
      ),
    );
  }

  Widget _buildContent(BuildContext context, DashboardState data) {
    return RefreshIndicator(
      onRefresh: () => ref.read(dashboardProvider.notifier).refresh(),
      child: SingleChildScrollView(
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _buildStatsCards(context, data.metrics),
            const SizedBox(height: 24),
            _buildQuickActions(context),
            const SizedBox(height: 24),
            _buildChartsSection(context, data.charts),
            const SizedBox(height: 24),
            if (data.charts.expiringList.isNotEmpty)
              _buildExpiringSection(context, data.charts.expiringList),
            if (data.charts.expiringList.isEmpty)
              const EmptyState(message: AppStrings.noData),
          ],
        ),
      ),
    );
  }

  Widget _buildStatsCards(BuildContext context, DashboardMetrics metrics) {
    final cards = <_StatCardData>[
      _StatCardData(
        title: AppStrings.accountBalance,
        value: MoneyFormatter.format(
          metrics.balance,
          currency: metrics.currency == 'CNY' ? '¥' : metrics.currency,
        ),
        icon: Icons.account_balance_wallet,
        color: AppColors.success,
        route: '/console/billing',
      ),
      _StatCardData(
        title: AppStrings.vpsCount,
        value: '${metrics.vpsTotal}',
        icon: Icons.cloud,
        color: AppColors.primary,
        route: '/console/vps',
      ),
      _StatCardData(
        title: AppStrings.orderCount,
        value: '${metrics.ordersTotal}',
        icon: Icons.receipt_long,
        color: AppColors.warning,
        route: '/console/orders',
      ),
      _StatCardData(
        title: AppStrings.spendTrend,
        value: MoneyFormatter.format(
          metrics.spend30d,
          currency: metrics.currency == 'CNY' ? '¥' : metrics.currency,
        ),
        icon: Icons.trending_up,
        color: AppColors.info,
        route: '/console/orders',
      ),
    ];

    return LayoutBuilder(
      builder: (context, constraints) {
        final width = constraints.maxWidth;
        final spacing = 16.0;
        final isMobile = width < 600;
        final canFour = width >= 900;
        if (canFour) {
          return Row(
            children: cards.asMap().entries.map((entry) {
              final idx = entry.key;
              final item = entry.value;
              return Expanded(
                child: Padding(
                  padding: EdgeInsets.only(right: idx == cards.length - 1 ? 0 : spacing),
                  child: _buildStatCard(
                    context,
                    item.title,
                    item.value,
                    item.icon,
                    item.color,
                    route: item.route,
                  ),
                ),
              );
            }).toList(),
          );
        }

        if (isMobile) {
          return GridView.count(
            crossAxisCount: 2,
            mainAxisSpacing: spacing,
            crossAxisSpacing: spacing,
            shrinkWrap: true,
            physics: const NeverScrollableScrollPhysics(),
            childAspectRatio: 1.6,
            children: cards.map((item) {
              return _buildStatCard(
                context,
                item.title,
                item.value,
                item.icon,
                item.color,
                route: item.route,
                isMobile: true,
              );
            }).toList(),
          );
        }

        return Wrap(
          spacing: spacing,
          runSpacing: spacing,
          children: cards.map((item) {
            final itemWidth = width >= 600 ? (width - spacing) / 2 : width;
            return SizedBox(
              width: itemWidth,
              child: _buildStatCard(
                context,
                item.title,
                item.value,
                item.icon,
                item.color,
                route: item.route,
              ),
            );
          }).toList(),
        );
      },
    );
  }

  Widget _buildStatCard(
    BuildContext context,
    String title,
    String value,
    IconData icon,
    Color color, {
    required String route,
    bool isMobile = false,
  }
  ) {
    return InkWell(
      onTap: () => context.go(route),
      borderRadius: BorderRadius.circular(12),
      child: Card(
        child: Padding(
          padding: EdgeInsets.all(isMobile ? 12 : 16),
          child: isMobile
              ? Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Container(
                      padding: const EdgeInsets.all(10),
                      decoration: BoxDecoration(
                        color: color.withOpacity(0.12),
                        borderRadius: BorderRadius.circular(12),
                      ),
                      child: Icon(icon, color: color, size: 22),
                    ),
                    const Spacer(),
                    Text(
                      title,
                      style: TextStyle(
                        fontSize: 12,
                        color: AppColors.gray500,
                      ),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    const SizedBox(height: 4),
                    Text(
                      value,
                      style: const TextStyle(
                        fontSize: 18,
                        fontWeight: FontWeight.bold,
                      ),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ],
                )
              : Row(
                  children: [
                    Container(
                      padding: const EdgeInsets.all(12),
                      decoration: BoxDecoration(
                        color: color.withOpacity(0.1),
                        borderRadius: BorderRadius.circular(12),
                      ),
                      child: Icon(icon, color: color, size: 24),
                    ),
                    const SizedBox(width: 16),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: [
                          Text(
                            title,
                            style: TextStyle(
                              fontSize: 12,
                              color: AppColors.gray500,
                            ),
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                          ),
                          const SizedBox(height: 4),
                          Text(
                            value,
                            style: const TextStyle(
                              fontSize: 20,
                              fontWeight: FontWeight.bold,
                            ),
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
        ),
      ),
    );
  }

  Widget _buildChartsSection(BuildContext context, DashboardCharts charts) {
    final isWide = MediaQuery.of(context).size.width > 900;
    final spendCard = _buildChartCard(
      title: '消费趋势',
      trailing: '近30天',
      child: LineChart(
        values: charts.spendValues,
        labels: charts.spendLabels,
        lineColor: AppColors.primary,
      ),
    );

    final orderCard = _buildChartCard(
      title: '订单分布',
      child: PieChart(data: charts.orderStatus),
    );

    if (isWide) {
      return Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Expanded(child: spendCard),
          const SizedBox(width: 16),
          Expanded(child: orderCard),
        ],
      );
    }

    return Column(
      children: [
        spendCard,
        const SizedBox(height: 16),
        orderCard,
      ],
    );
  }

  Widget _buildChartCard({
    required String title,
    String? trailing,
    required Widget child,
  }) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Text(
                  title,
                  style: const TextStyle(
                    fontSize: 16,
                    fontWeight: FontWeight.bold,
                  ),
                ),
                const Spacer(),
                if (trailing != null)
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                    decoration: BoxDecoration(
                      color: AppColors.primary.withOpacity(0.1),
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: Text(
                      trailing,
                      style: const TextStyle(fontSize: 12, color: AppColors.primary),
                    ),
                  ),
              ],
            ),
            const SizedBox(height: 12),
            SizedBox(
              height: 240,
              child: child,
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildQuickActions(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              AppStrings.quickActions,
              style: TextStyle(
                fontSize: 18,
                fontWeight: FontWeight.bold,
              ),
            ),
            const SizedBox(height: 16),
            Row(
              children: [
                Expanded(
                  child: _ActionButton(
                    icon: Icons.verified_user,
                    label: AppStrings.gotoRealname,
                    onTap: () => context.go('/console/realname'),
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: _ActionButton(
                    icon: Icons.shopping_cart,
                    label: AppStrings.viewCart,
                    onTap: () => context.go('/console/cart'),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildExpiringSection(BuildContext context, List<Map<String, dynamic>> instances) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              AppStrings.expiringSoon,
              style: const TextStyle(
                fontSize: 18,
                fontWeight: FontWeight.bold,
              ),
            ),
            const SizedBox(height: 16),
            ...instances.map<Widget>((instance) {
              final name = instance['name'] ?? instance['Name'] ?? 'Unknown';
              final expireAt = instance['expire_at'] ?? instance['ExpireAt'] ?? '';
              final id = instance['id'] ?? instance['ID'];
              return ListTile(
                onTap: () {
                  if (id != null) {
                    context.go('/console/vps/$id');
                  }
                },
                title: Text('$name'),
                subtitle: Text(_formatRemaining(expireAt)),
                trailing: const Icon(Icons.arrow_forward_ios, size: 16),
              );
            }).toList(),
          ],
        ),
      ),
    );
  }

  String _formatRemaining(dynamic value) {
    final dt = DateFormatter.parse(value);
    return DateFormatter.timeRemaining(dt);
  }
}

class _ActionButton extends StatelessWidget {
  final IconData icon;
  final String label;
  final VoidCallback onTap;

  const _ActionButton({
    required this.icon,
    required this.label,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(8),
      child: Container(
        padding: const EdgeInsets.all(12),
        child: Column(
          children: [
            Icon(icon, size: 32, color: AppColors.primary),
            const SizedBox(height: 8),
            Text(
              label,
              style: const TextStyle(fontSize: 12),
              textAlign: TextAlign.center,
            ),
          ],
        ),
      ),
    );
  }
}

class _StatCardData {
  final String title;
  final String value;
  final IconData icon;
  final Color color;
  final String route;

  const _StatCardData({
    required this.title,
    required this.value,
    required this.icon,
    required this.color,
    required this.route,
  });
}














