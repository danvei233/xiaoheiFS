import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../core/constants/app_colors.dart';
import '../../../core/constants/app_strings.dart';
import '../../../core/network/api_client.dart';
import '../../../core/utils/avatar_url.dart';
import '../../providers/auth_provider.dart';

class MorePage extends ConsumerWidget {
  const MorePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final user = ref.watch(authProvider).user;
    final isDesktop = MediaQuery.of(context).size.width >= 1024;

    return Scaffold(
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _buildUserCard(context, user),
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
              icon: Icons.settings_outlined,
              title: AppStrings.navProfile,
              route: '/console/profile',
            ),
          ],
          const SizedBox(height: 8),
          _buildLogoutItem(context, ref),
        ],
      ),
    );
  }

  Widget _buildUserCard(BuildContext context, dynamic user) {
    final avatar = resolveUserAvatarUrl(
      baseUrl: ApiClient.instance.dio.options.baseUrl,
      qq: user?.qq?.toString(),
      avatarUrl: user?.avatarUrl?.toString(),
      avatar: user?.avatar?.toString(),
    );
    final username = user?.username ?? '未登录';
    final email = user?.email ?? '';
    return Card(
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: () => context.go('/console/profile'),
        child: Padding(
          padding: const EdgeInsets.all(20),
          child: Row(
            children: [
              if (avatar.isNotEmpty)
                CircleAvatar(
                  radius: 28,
                  backgroundImage: NetworkImage(avatar),
                )
              else
                CircleAvatar(
                  radius: 28,
                  backgroundColor: AppColors.primaryLight,
                  child: Text(
                    username.toString().isNotEmpty ? username.toString()[0].toUpperCase() : 'U',
                    style: const TextStyle(color: Colors.white, fontWeight: FontWeight.bold),
                  ),
                ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      username.toString(),
                      style: const TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
                    ),
                    if (email.toString().isNotEmpty) ...[
                      const SizedBox(height: 4),
                      Text(
                        email.toString(),
                        style: TextStyle(color: AppColors.gray500, fontSize: 12),
                      ),
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
        title: const Text(AppStrings.logout, style: TextStyle(color: AppColors.danger)),
        onTap: () async {
          await ref.read(authProvider.notifier).logout();
          if (context.mounted) {
            context.go('/login');
          }
        },
      ),
    );
  }
}



