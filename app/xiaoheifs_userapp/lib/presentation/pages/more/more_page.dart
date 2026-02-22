import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../core/constants/app_colors.dart';
import '../../../core/constants/app_strings.dart';
import '../../../core/network/api_client.dart';
import '../../../core/utils/avatar_url.dart';
import '../../../core/utils/date_formatter.dart';
import '../../providers/auth_provider.dart';

class MorePage extends ConsumerStatefulWidget {
  const MorePage({super.key});

  @override
  ConsumerState<MorePage> createState() => _MorePageState();
}

class _MorePageState extends ConsumerState<MorePage> {
  Map<String, dynamic> _tier = const {};

  @override
  void initState() {
    super.initState();
    Future.microtask(_loadTier);
  }

  Future<void> _loadTier() async {
    try {
      final tier = await ref.read(authRepositoryProvider).getMyUserTier();
      if (!mounted) return;
      setState(() => _tier = tier);
    } catch (_) {
      // Keep silent when tier API is unavailable.
    }
  }

  @override
  Widget build(BuildContext context) {
    final ref = this.ref;
    final user = ref.watch(authProvider).user;
    final isDesktop = MediaQuery.of(context).size.width >= 1024;

    return Scaffold(
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _buildUserCard(context, user, _tier),
          if (!isDesktop) ...[
            const SizedBox(height: 16),
            _buildItem(
              context,
              icon: Icons.notifications_outlined,
              title: AppStrings.navNotifications,
              route: '/console/notifications',
            ),
            _buildItem(
              context,
              icon: Icons.account_balance_wallet_outlined,
              title: AppStrings.navWallet,
              route: '/console/billing',
            ),
            _buildItem(
              context,
              icon: Icons.support_agent_outlined,
              title: AppStrings.navTickets,
              route: '/console/tickets',
            ),
            _buildItem(
              context,
              icon: Icons.verified_user_outlined,
              title: AppStrings.navRealname,
              route: '/console/realname',
            ),
            _buildItem(
              context,
              icon: Icons.key_outlined,
              title: AppStrings.navApiManagement,
              route: '/console/api-keys',
            ),
            _buildItem(
              context,
              icon: Icons.settings_outlined,
              title: AppStrings.navProfile,
              route: '/console/profile',
            ),
          ],
          _buildItem(
            context,
            icon: Icons.security_outlined,
            title: '安全中心',
            route: '/console/profile/security',
          ),
          const SizedBox(height: 8),
          _buildLogoutItem(context, ref),
        ],
      ),
    );
  }

  Widget _buildUserCard(
    BuildContext context,
    dynamic user,
    Map<String, dynamic> tier,
  ) {
    final avatar = resolveUserAvatarUrl(
      baseUrl: ApiClient.instance.dio.options.baseUrl,
      qq: user?.qq?.toString(),
      avatarUrl: user?.avatarUrl?.toString(),
      avatar: user?.avatar?.toString(),
    );
    final username = user?.username ?? '未登录';
    final email = user?.email ?? '';
    final tierName = (tier['group_name'] ?? '').toString().trim();
    final tierColor = _resolveTierColor((tier['group_color'] ?? '').toString());
    final tierIcon = _resolveTierIcon((tier['group_icon'] ?? '').toString());
    final tierExpireRaw = (tier['expire_at'] ?? '').toString().trim();
    final tierExpireText = tierExpireRaw.isEmpty
        ? ''
        : DateFormatter.formatIso(tierExpireRaw, DateFormatter.formatCompact);

    return Card(
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: () => context.go('/console/profile'),
        child: Padding(
          padding: const EdgeInsets.all(20),
          child: Row(
            children: [
              if (avatar.isNotEmpty)
                CircleAvatar(radius: 28, backgroundImage: NetworkImage(avatar))
              else
                CircleAvatar(
                  radius: 28,
                  backgroundColor: AppColors.primaryLight,
                  child: Text(
                    username.toString().isNotEmpty
                        ? username.toString()[0].toUpperCase()
                        : 'U',
                    style: const TextStyle(
                      color: Colors.white,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      username.toString(),
                      style: const TextStyle(
                        fontSize: 16,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                    if (email.toString().isNotEmpty) ...[
                      const SizedBox(height: 4),
                      Text(
                        email.toString(),
                        style: TextStyle(
                          color: AppColors.gray500,
                          fontSize: 12,
                        ),
                      ),
                    ],
                    if (tierName.isNotEmpty) ...[
                      const SizedBox(height: 6),
                      Row(
                        children: [
                          Container(
                            padding: const EdgeInsets.symmetric(
                              horizontal: 8,
                              vertical: 3,
                            ),
                            decoration: BoxDecoration(
                              color: tierColor.withValues(alpha: 0.16),
                              borderRadius: BorderRadius.circular(999),
                              border: Border.all(
                                color: tierColor.withValues(alpha: 0.36),
                              ),
                            ),
                            child: Row(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                Icon(tierIcon, size: 12, color: tierColor),
                                const SizedBox(width: 4),
                                Text(
                                  tierName,
                                  style: TextStyle(
                                    fontSize: 11,
                                    fontWeight: FontWeight.w600,
                                    color: tierColor,
                                  ),
                                ),
                              ],
                            ),
                          ),
                        ],
                      ),
                      if (tierExpireText.isNotEmpty) ...[
                        const SizedBox(height: 3),
                        Text(
                          '到期：$tierExpireText',
                          style: const TextStyle(
                            color: AppColors.gray500,
                            fontSize: 11,
                          ),
                        ),
                      ],
                    ],
                  ],
                ),
              ),
              const Icon(Icons.chevron_right, color: AppColors.gray500),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildItem(
    BuildContext context, {
    required IconData icon,
    required String title,
    required String route,
  }) {
    return Card(
      child: ListTile(
        leading: Icon(icon),
        title: Text(title),
        trailing: const Icon(Icons.arrow_forward_ios, size: 16),
        onTap: () => context.go(route),
      ),
    );
  }

  Widget _buildLogoutItem(BuildContext context, WidgetRef ref) {
    return Card(
      child: ListTile(
        leading: const Icon(Icons.logout, color: AppColors.danger),
        title: const Text(
          AppStrings.logout,
          style: TextStyle(color: AppColors.danger),
        ),
        onTap: () async {
          await ref.read(authProvider.notifier).logout();
          if (context.mounted) {
            context.go('/login');
          }
        },
      ),
    );
  }

  Color _resolveTierColor(String raw) {
    final value = raw.trim();
    if (value.isEmpty) {
      return AppColors.primary;
    }
    final hex = value.startsWith('#') ? value.substring(1) : value;
    if (hex.length != 6 && hex.length != 8) {
      return AppColors.primary;
    }
    final parsed = int.tryParse(hex, radix: 16);
    if (parsed == null) {
      return AppColors.primary;
    }
    return hex.length == 6 ? Color(0xFF000000 | parsed) : Color(parsed);
  }

  IconData _resolveTierIcon(String raw) {
    final icon = raw.trim().toLowerCase();
    if (icon == 'vip' || icon == 'crown') return Icons.workspace_premium;
    if (icon == 'star') return Icons.star;
    if (icon == 'rocket') return Icons.rocket_launch;
    if (icon == 'diamond' || icon == 'gem') return Icons.diamond;
    if (icon == 'shield' || icon == 'badge') return Icons.verified_user;
    return Icons.military_tech;
  }
}
