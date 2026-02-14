import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app_state.dart';
import '../services/api_client.dart';
import '../services/avatar.dart';
import 'api_keys_screen.dart';
import 'audit_logs_screen.dart';
import 'payment_providers_screen.dart';
import 'scheduled_tasks_screen.dart';
import 'settings_kv_screen.dart';
import 'tickets_screen.dart';
import 'wallet_orders_screen.dart';
import 'catalog/catalog_hub_screen.dart';
import 'permissions_screen.dart';
import 'probes_screen.dart';

class SettingsScreen extends StatelessWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        leading: const BackButton(),
        title: const Text('设置'),
      ),
      body: Consumer<AppState>(
        builder: (context, state, _) {
          final session = state.session;
          final username = session?.username ?? '管理员';
          return ListView(
            padding: const EdgeInsets.fromLTRB(16, 16, 16, 24),
            children: [
              const _SectionHeader(title: '账户与系统', icon: Icons.account_circle),
              const SizedBox(height: 8),
              _ProfileCard(username: username, session: session),
              const SizedBox(height: 16),
              const _SectionHeader(title: '日志与接口', icon: Icons.receipt_long),
              const SizedBox(height: 8),
              _SettingTile(
                icon: Icons.description,
                title: '查看系统日志',
                subtitle: '系统运行与异常记录',
                onTap: () {
                  _showToast(context, '系统日志暂未接入');
                },
              ),
              _SettingTile(
                icon: Icons.list_alt,
                title: '查看操作日志',
                subtitle: '管理员操作记录',
                onTap: () {
                  Navigator.push(
                    context,
                    MaterialPageRoute(builder: (_) => const AuditLogsScreen()),
                  );
                },
              ),
              _SettingTile(
                icon: Icons.health_and_safety,
                title: 'API 健康检查',
                subtitle: '检查 /admin 接口状态',
                onTap: () => _checkApiHealth(context),
              ),
              const SizedBox(height: 16),
              const _SectionHeader(title: '管理模块', icon: Icons.dashboard_customize),
              const SizedBox(height: 8),
              _SettingTile(
                icon: Icons.support_agent,
                title: '工单管理',
                subtitle: '查看与回复用户工单',
                onTap: () {
                  Navigator.push(
                    context,
                    MaterialPageRoute(builder: (_) => const TicketsScreen()),
                  );
                },
              ),
              _SettingTile(
                icon: Icons.account_balance_wallet,
                title: '钱包订单',
                subtitle: '充值/退款审核',
                onTap: () {
                  Navigator.push(
                    context,
                    MaterialPageRoute(builder: (_) => const WalletOrdersScreen()),
                  );
                },
              ),
              _SettingTile(
                icon: Icons.schedule,
                title: '定时任务',
                subtitle: '任务启停与配置',
                onTap: () {
                  Navigator.push(
                    context,
                    MaterialPageRoute(
                      builder: (_) => const ScheduledTasksScreen(),
                    ),
                  );
                },
              ),
              _SettingTile(
                icon: Icons.radar,
                title: '探针管理',
                subtitle: '查看探针状态、SLA 与日志',
                onTap: () {
                  Navigator.push(
                    context,
                    MaterialPageRoute(builder: (_) => const ProbesScreen()),
                  );
                },
              ),
              _SettingTile(
                icon: Icons.vpn_key,
                title: 'API Keys',
                subtitle: '密钥创建与禁用',
                onTap: () {
                  Navigator.push(
                    context,
                    MaterialPageRoute(builder: (_) => const ApiKeysScreen()),
                  );
                },
              ),
              _SettingTile(
                icon: Icons.payments,
                title: '支付渠道',
                subtitle: '启用/停用支付方式',
                onTap: () {
                  Navigator.push(
                    context,
                    MaterialPageRoute(
                      builder: (_) => const PaymentProvidersScreen(),
                    ),
                  );
                },
              ),
              _SettingTile(
                icon: Icons.settings_applications,
                title: '系统设置',
                subtitle: '配置键值管理',
                onTap: () {
                  Navigator.push(
                    context,
                    MaterialPageRoute(builder: (_) => const SettingsKvScreen()),
                  );
                },
              ),
              _SettingTile(
                icon: Icons.category,
                title: '商品与计费',
                subtitle: '区域/线路/套餐/计费周期',
                onTap: () {
                  Navigator.push(
                    context,
                    MaterialPageRoute(builder: (_) => const CatalogHubScreen()),
                  );
                },
              ),
              _SettingTile(
                icon: Icons.rule,
                title: '权限列表',
                subtitle: '系统权限定义',
                onTap: () {
                  Navigator.push(
                    context,
                    MaterialPageRoute(builder: (_) => const PermissionsScreen()),
                  );
                },
              ),
              const SizedBox(height: 16),
              const _SectionHeader(title: '账户操作', icon: Icons.manage_accounts),
              const SizedBox(height: 8),
              _SettingTile(
                icon: Icons.manage_accounts,
                title: '编辑资料 / API 配置',
                subtitle: session?.authType == 'password'
                    ? '修改邮箱与显示名'
                    : '修改 API 地址与 Key',
                onTap: () => _openEditDialog(context, state),
              ),
              Card(
                color: const Color(0xFFFFF4F4),
                child: ListTile(
                  leading: const Icon(Icons.logout, color: Color(0xFFD32F2F)),
                  title: const Text('退出登录'),
                  subtitle: const Text('清除本地登录信息'),
                  onTap: () async {
                    await state.logout();
                  },
                ),
              ),
            ],
          );
        },
      ),
    );
  }

  void _showToast(BuildContext context, String message) {
    ScaffoldMessenger.of(
      context,
    ).showSnackBar(SnackBar(content: Text(message)));
  }

  Future<void> _checkApiHealth(BuildContext context) async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    try {
      await client.getJson('/admin/api/v1/server/status');
      if (context.mounted) {
        _showToast(context, 'API 正常');
      }
    } on ApiException catch (e) {
      if (context.mounted) {
        _showToast(context, 'API 异常：${e.message}');
      }
    } catch (_) {
      if (context.mounted) {
        _showToast(context, 'API 异常');
      }
    }
  }

  Future<void> _openEditDialog(BuildContext context, AppState state) async {
    final session = state.session;
    final isPassword = session?.authType == 'password';
    final apiUrlController = TextEditingController(text: session?.apiUrl ?? '');
    final apiKeyController = TextEditingController(text: session?.apiKey ?? '');
    final usernameController = TextEditingController(
      text: session?.username ?? '管理员',
    );
    final emailController = TextEditingController(text: session?.email ?? '');
    final formKey = GlobalKey<FormState>();

    await showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      builder: (context) {
        return Padding(
          padding: EdgeInsets.only(
            left: 20,
            right: 20,
            top: 20,
            bottom: MediaQuery.of(context).viewInsets.bottom + 20,
          ),
          child: Form(
            key: formKey,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  isPassword ? '编辑资料' : '编辑 API 配置',
                  style: Theme.of(
                    context,
                  ).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w600),
                ),
                const SizedBox(height: 16),
                if (!isPassword) ...[
                  TextFormField(
                    controller: apiUrlController,
                    decoration: const InputDecoration(labelText: 'API 地址'),
                    validator: (value) {
                      if (value == null || value.trim().isEmpty) {
                        return '请输入 API 地址';
                      }
                      return null;
                    },
                  ),
                  const SizedBox(height: 12),
                  TextFormField(
                    controller: apiKeyController,
                    decoration: const InputDecoration(labelText: 'API Key'),
                    validator: (value) {
                      if (value == null || value.trim().isEmpty) {
                        return '请输入 API Key';
                      }
                      return null;
                    },
                  ),
                  const SizedBox(height: 12),
                ],
                TextFormField(
                  controller: usernameController,
                  decoration: const InputDecoration(labelText: '显示名'),
                ),
                if (isPassword) ...[
                  const SizedBox(height: 12),
                  TextFormField(
                    controller: emailController,
                    decoration: const InputDecoration(labelText: '邮箱'),
                  ),
                ],
                const SizedBox(height: 20),
                SizedBox(
                  width: double.infinity,
                  child: FilledButton(
                    onPressed: () async {
                      if (!formKey.currentState!.validate()) return;
                      if (isPassword) {
                        await state.updateProfile(
                          username: usernameController.text.trim(),
                          email: emailController.text.trim(),
                        );
                        final client = state.apiClient;
                        if (client != null) {
                          await client.patchJson(
                            '/admin/api/v1/profile',
                            body: {
                              'email': emailController.text.trim(),
                              'qq': '',
                            },
                          );
                        }
                      } else {
                        await state.updateProfile(
                          apiUrl: apiUrlController.text.trim(),
                          apiKey: apiKeyController.text.trim(),
                          username: usernameController.text.trim(),
                        );
                      }
                      if (context.mounted) {
                        Navigator.pop(context);
                      }
                    },
                    child: const Text('保存'),
                  ),
                ),
              ],
            ),
          ),
        );
      },
    );

    apiUrlController.dispose();
    apiKeyController.dispose();
    usernameController.dispose();
    emailController.dispose();
  }
}

class _ProfileCard extends StatefulWidget {
  final String username;
  final dynamic session;

  const _ProfileCard({required this.username, required this.session});

  @override
  State<_ProfileCard> createState() => _ProfileCardState();
}

class _ProfileCardState extends State<_ProfileCard> {
  Future<Map<String, dynamic>>? _future;
  ApiClient? _client;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final client = context.read<AppState>().apiClient;
    if (client?.baseUrl != _client?.baseUrl ||
        client?.apiKey != _client?.apiKey ||
        client?.token != _client?.token) {
      _client = client;
      if (client != null) {
        _future = client.getJson('/admin/api/v1/profile');
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: FutureBuilder<Map<String, dynamic>>(
          future: _future,
          builder: (context, snapshot) {
            final data = snapshot.data;
            final username = (data?['username'] as String?) ?? widget.username;
            final email = data?['email'] as String?;
            final baseUrl = context.read<AppState>().apiClient?.baseUrl ?? '';
            final session = context.read<AppState>().session;
            final headers = avatarHeaders(
              token: session?.token,
              apiKey: session?.apiKey,
            );
            final avatarUrl = resolveAvatarUrl(
              baseUrl: baseUrl,
              qq: data?['qq']?.toString(),
              avatarUrl: data?['avatar_url'] as String?,
            );
            return Row(
              children: [
                CircleAvatar(
                  radius: 28,
                  backgroundColor: const Color(0xFF00BFA6),
                  child: avatarUrl.isNotEmpty
                      ? ClipOval(
                          child: Image.network(
                            avatarUrl,
                            width: 56,
                            height: 56,
                            fit: BoxFit.cover,
                            headers: headers.isEmpty ? null : headers,
                            errorBuilder: (context, error, stack) {
                              return const Icon(
                                Icons.person,
                                color: Colors.white,
                              );
                            },
                          ),
                        )
                      : Text(
                          username.isNotEmpty ? username.characters.first : '管',
                          style: const TextStyle(
                            color: Colors.white,
                            fontSize: 20,
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        username,
                        style: theme.textTheme.titleMedium?.copyWith(
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      const SizedBox(height: 4),
                      Text(
                        email ?? (widget.session?.apiUrl ?? '未配置 API 地址'),
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: Colors.black54,
                        ),
                      ),
                    ],
                  ),
                ),
                Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 10,
                    vertical: 6,
                  ),
                  decoration: BoxDecoration(
                    color: const Color(0xFFEFF7F6),
                    borderRadius: BorderRadius.circular(999),
                  ),
                  child: Text(
                    widget.session?.authType == 'password' ? '账号' : 'API Key',
                    style: theme.textTheme.labelSmall?.copyWith(
                      color: const Color(0xFF009A83),
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
              ],
            );
          },
        ),
      ),
    );
  }
}

class _SettingTile extends StatelessWidget {
  final IconData icon;
  final String title;
  final String subtitle;
  final VoidCallback? onTap;

  const _SettingTile({
    required this.icon,
    required this.title,
    required this.subtitle,
    this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: colorScheme.outlineVariant.withOpacity(0.5)),
      ),
      child: ListTile(
        leading: Icon(icon, color: colorScheme.primary),
        title: Text(title),
        subtitle: Text(subtitle),
        trailing: const Icon(Icons.chevron_right),
        onTap:
            onTap ??
            () {
              ScaffoldMessenger.of(
                context,
              ).showSnackBar(SnackBar(content: Text('打开 $title')));
            },
      ),
    );
  }
}

class _SectionHeader extends StatelessWidget {
  final String title;
  final IconData icon;

  const _SectionHeader({required this.title, required this.icon});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    return Row(
      children: [
        Container(
          padding: const EdgeInsets.all(8),
          decoration: BoxDecoration(
            color: colorScheme.primaryContainer.withOpacity(0.5),
            borderRadius: BorderRadius.circular(10),
          ),
          child: Icon(icon, size: 18, color: colorScheme.primary),
        ),
        const SizedBox(width: 10),
        Text(
          title,
          style: theme.textTheme.titleSmall?.copyWith(
            fontWeight: FontWeight.w700,
          ),
        ),
      ],
    );
  }
}
