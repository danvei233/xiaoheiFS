import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app_state.dart';
import '../services/avatar.dart';
import 'api_keys_screen.dart';
import 'audit_logs_screen.dart';
import 'catalog/catalog_hub_screen.dart';
import 'catalog/simple_crud_screen.dart';
import 'login_screen.dart';
import 'settings_kv_screen.dart';
import 'settings_extra_screens.dart';

class SettingsScreen extends StatelessWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final groups = _buildGroups();
    return Scaffold(
      appBar: AppBar(title: const Text('设置')),
      body: Consumer<AppState>(
        builder: (context, state, _) {
          return ListView(
            padding: const EdgeInsets.fromLTRB(12, 12, 12, 20),
            children: [
              _ProfileCard(username: state.session?.username ?? '管理员'),
              const SizedBox(height: 10),
              Container(
                margin: const EdgeInsets.only(bottom: 8),
                decoration: BoxDecoration(
                  color: Colors.white,
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(color: const Color(0xFFE5EAF2)),
                ),
                child: ListTile(
                  leading: const Icon(Icons.person_outline, color: Color(0xFF1E88E5)),
                  title: const Text('个人资料设置'),
                  subtitle: const Text('管理员资料与密码安全'),
                  trailing: const Icon(Icons.chevron_right_rounded),
                  onTap: () => Navigator.push(
                    context,
                    MaterialPageRoute(
                      builder: (_) => const AdminProfileSettingsScreen(),
                    ),
                  ),
                ),
              ),
              ...groups.map((g) => _GroupTile(group: g)),
              const SizedBox(height: 10),
              Card(
                color: const Color(0xFFFFF4F4),
                child: ListTile(
                  leading: const Icon(Icons.logout, color: Color(0xFFD32F2F)),
                  title: const Text('退出登录'),
                  subtitle: const Text('退出后自动返回登录页'),
                  onTap: () async {
                    final ok = await showDialog<bool>(
                      context: context,
                      builder: (context) => AlertDialog(
                        title: const Text('确认退出'),
                        content: const Text('确定退出当前账号并返回登录页吗？'),
                        actions: [
                          TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('取消')),
                          FilledButton(onPressed: () => Navigator.pop(context, true), child: const Text('确认')),
                        ],
                      ),
                    );
                    if (ok != true) return;
                    await state.logout();
                    if (!context.mounted) return;
                    Navigator.of(context).pushAndRemoveUntil(
                      MaterialPageRoute(builder: (_) => const LoginScreen()),
                      (_) => false,
                    );
                  },
                ),
              ),
            ],
          );
        },
      ),
    );
  }

  List<_SettingsGroup> _buildGroups() {
    return [
      _SettingsGroup(
        title: '商品与业务',
        icon: Icons.shopping_bag_outlined,
        items: [
          _SettingsItem(
            title: '商品设置同步',
            subtitle: '地区/线路/套餐/计费同步与维护',
            icon: Icons.sync_alt_rounded,
            builder: (_) => const CatalogHubScreen(),
          ),
          _SettingsItem(
            title: '用户等级设置',
            subtitle: '管理用户等级组',
            icon: Icons.workspace_premium_outlined,
            builder: (_) => const UserTierSettingsAdvancedScreen(),
          ),
          _SettingsItem(
            title: '优惠码设置',
            subtitle: '新增/编辑/删除优惠码',
            icon: Icons.discount_outlined,
            builder: (_) => const CouponSettingsAdvancedScreen(),
          ),
        ],
      ),
      _SettingsGroup(
        title: '安全与认证',
        icon: Icons.verified_user_outlined,
        items: [
          _SettingsItem(
            title: '注册设置',
            subtitle: '注册开关、必填项、验证策略',
            icon: Icons.app_registration_rounded,
            builder: (_) => const AuthSettingsScreen(),
          ),
          _SettingsItem(
            title: '验证码设置',
            subtitle: '图形验证码与极验配置',
            icon: Icons.verified_outlined,
            builder: (_) => const CaptchaSettingsScreen(),
          ),
          _SettingsItem(
            title: '实名设置',
            subtitle: '实名开关、服务商、拦截动作',
            icon: Icons.badge_outlined,
            builder: (_) => const RealnameConfigScreen(),
          ),
        ],
      ),
      _SettingsGroup(
        title: '站点与生命周期',
        icon: Icons.public_outlined,
        items: [
          _SettingsItem(
            title: '站点设置',
            subtitle: '站点名称、URL、SEO 信息',
            icon: Icons.language_outlined,
            builder: (_) => const SiteSettingsScreen(),
          ),
          _SettingsItem(
            title: '生命周期设置',
            subtitle: '订单/VPS 生命周期策略',
            icon: Icons.timelapse_outlined,
            builder: (_) => const LifecycleSettingsScreen(),
          ),
        ],
      ),
      _SettingsGroup(
        title: '消息与模板',
        icon: Icons.mark_email_unread_outlined,
        items: [
          _SettingsItem(
            title: '邮箱模板设置',
            subtitle: '邮件模板增删改',
            icon: Icons.email_outlined,
            builder: (_) => const EmailSettingsAdvancedScreen(),
          ),
          _SettingsItem(
            title: '短信设置',
            subtitle: '短信插件配置与模板管理',
            icon: Icons.sms_outlined,
            builder: (_) => const SmsSettingsScreen(),
          ),
        ],
      ),
      _SettingsGroup(
        title: '账号与审计',
        icon: Icons.account_circle_outlined,
        items: [
          _SettingsItem(
            title: '审计日志',
            subtitle: '管理员操作日志查询',
            icon: Icons.history,
            builder: (_) => const AuditLogsScreen(),
          ),
          _SettingsItem(
            title: 'API Keys 管理',
            subtitle: '创建与停用 API Key',
            icon: Icons.vpn_key_outlined,
            builder: (_) => const ApiKeysScreen(),
          ),
        ],
      ),
      _SettingsGroup(
        title: '内容运营',
        icon: Icons.campaign_outlined,
        items: [
          _SettingsItem(
            title: '内容分类',
            subtitle: 'CMS 分类管理',
            icon: Icons.category_outlined,
            builder: (_) => SimpleCrudScreen(
              title: '内容分类',
              listPath: '/admin/api/v1/cms/categories',
              createPath: '/admin/api/v1/cms/categories',
              updatePath: (item) => '/admin/api/v1/cms/categories/${item['id']}',
              deletePath: (item) => '/admin/api/v1/cms/categories/${item['id']}',
              fields: const [
                FieldDef(keyName: 'name', label: '名称', type: FieldType.text),
                FieldDef(keyName: 'slug', label: '标识', type: FieldType.text),
                FieldDef(keyName: 'sort_order', label: '排序', type: FieldType.number, numberIsInt: true),
                FieldDef(keyName: 'enabled', label: '启用', type: FieldType.boolValue),
              ],
              titleBuilder: (item) => (item['name'] ?? '').toString(),
              subtitleBuilder: (item) => 'slug:${(item['slug'] ?? '').toString()}',
            ),
          ),
          _SettingsItem(
            title: '内容文章',
            subtitle: 'CMS 文章管理',
            icon: Icons.article_outlined,
            builder: (_) => SimpleCrudScreen(
              title: '内容文章',
              listPath: '/admin/api/v1/cms/posts',
              createPath: '/admin/api/v1/cms/posts',
              updatePath: (item) => '/admin/api/v1/cms/posts/${item['id']}',
              deletePath: (item) => '/admin/api/v1/cms/posts/${item['id']}',
              fields: const [
                FieldDef(keyName: 'title', label: '标题', type: FieldType.text),
                FieldDef(keyName: 'slug', label: '标识', type: FieldType.text),
                FieldDef(keyName: 'content', label: '内容', type: FieldType.text),
                FieldDef(keyName: 'status', label: '状态', type: FieldType.text),
              ],
              titleBuilder: (item) => (item['title'] ?? '').toString(),
              subtitleBuilder: (item) => 'slug:${(item['slug'] ?? '').toString()}',
            ),
          ),
          _SettingsItem(
            title: '内容区块',
            subtitle: 'CMS 区块管理（暂不可用）',
            icon: Icons.view_module_outlined,
            enabled: false,
            builder: (_) => SimpleCrudScreen(
              title: '内容区块',
              listPath: '/admin/api/v1/cms/blocks',
              createPath: '/admin/api/v1/cms/blocks',
              updatePath: (item) => '/admin/api/v1/cms/blocks/${item['id']}',
              deletePath: (item) => '/admin/api/v1/cms/blocks/${item['id']}',
              fields: const [
                FieldDef(keyName: 'name', label: '名称', type: FieldType.text),
                FieldDef(keyName: 'key', label: '键名', type: FieldType.text),
                FieldDef(keyName: 'content', label: '内容', type: FieldType.text),
                FieldDef(keyName: 'enabled', label: '启用', type: FieldType.boolValue),
              ],
              titleBuilder: (item) => (item['name'] ?? item['key'] ?? '').toString(),
              subtitleBuilder: (item) => 'key:${(item['key'] ?? '').toString()}',
            ),
          ),
          _SettingsItem(
            title: '顶部导航',
            subtitle: '主页导航 JSON 设置',
            icon: Icons.menu_open_outlined,
            builder: (_) => const SettingsKvScreen(
              title: '顶部导航',
              exactKeys: ['site_nav_items'],
            ),
          ),
          _SettingsItem(
            title: '上传中心',
            subtitle: '媒体文件列表与预览',
            icon: Icons.upload_file_outlined,
            builder: (_) => const CmsUploadsSimpleScreen(),
          ),
        ],
      ),
      _SettingsGroup(
        title: '管理与权限',
        icon: Icons.admin_panel_settings_outlined,
        items: [
          _SettingsItem(
            title: '管理员列表',
            subtitle: '管理员账号与状态',
            icon: Icons.manage_accounts_outlined,
            builder: (_) => const AdminSettingsAdvancedScreen(),
          ),
          _SettingsItem(
            title: '权限组设置',
            subtitle: '权限组增删改',
            icon: Icons.shield_outlined,
            builder: (_) => const PermissionGroupSettingsAdvancedScreen(),
          ),
          _SettingsItem(
            title: '插件设置',
            subtitle: '插件状态启停',
            icon: Icons.extension_outlined,
            builder: (_) => const PluginsSettingsScreen(),
          ),
          _SettingsItem(
            title: '调试中心',
            subtitle: '调试开关与日志概览',
            icon: Icons.bug_report_outlined,
            builder: (_) => const DebugCenterScreen(),
          ),
        ],
      ),
    ];
  }
}

class _SettingsGroupScreen extends StatelessWidget {
  final _SettingsGroup group;
  const _SettingsGroupScreen({required this.group});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(group.title)),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(12, 12, 12, 20),
        children: group.items
            .map(
              (item) => Container(
                margin: const EdgeInsets.only(bottom: 8),
                decoration: BoxDecoration(
                  color: Colors.white,
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(color: const Color(0xFFE5EAF2)),
                ),
                child: ListTile(
                  leading: Icon(
                    item.icon,
                    color: item.enabled ? const Color(0xFF1E88E5) : const Color(0xFF94A3B8),
                  ),
                  title: Text(
                    item.title,
                    style: TextStyle(
                      color: item.enabled ? null : const Color(0xFF94A3B8),
                    ),
                  ),
                  subtitle: Text(
                    item.subtitle,
                    style: TextStyle(
                      color: item.enabled ? null : const Color(0xFF94A3B8),
                    ),
                  ),
                  trailing: const Icon(Icons.chevron_right_rounded),
                  onTap: () {
                    if (!item.enabled) {
                      ScaffoldMessenger.of(context).showSnackBar(
                        const SnackBar(content: Text('该功能暂不可用')),
                      );
                      return;
                    }
                    Navigator.push(
                      context,
                      MaterialPageRoute(builder: item.builder),
                    );
                  },
                ),
              ),
            )
            .toList(),
      ),
    );
  }
}

class _GroupTile extends StatelessWidget {
  final _SettingsGroup group;
  const _GroupTile({required this.group});

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: const Color(0xFFE5EAF2)),
      ),
      child: ListTile(
        leading: Icon(group.icon, color: const Color(0xFF1E88E5)),
        title: Text(group.title),
        subtitle: Text('${group.items.length} 个子项'),
        trailing: const Icon(Icons.chevron_right_rounded),
        onTap: () => Navigator.push(
          context,
          MaterialPageRoute(builder: (_) => _SettingsGroupScreen(group: group)),
        ),
      ),
    );
  }
}

class _ProfileCard extends StatefulWidget {
  final String username;
  const _ProfileCard({required this.username});

  @override
  State<_ProfileCard> createState() => _ProfileCardState();
}

class _ProfileCardState extends State<_ProfileCard> {
  Future<Map<String, dynamic>>? _future;
  String _baseUrl = '';
  Map<String, String> _headers = const {};

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final app = context.read<AppState>();
    final client = app.apiClient;
    if (client != null) {
      _baseUrl = client.baseUrl;
      _headers = avatarHeaders(
        token: app.session?.token,
        apiKey: app.session?.apiKey,
      );
      _future ??= client.getJson('/admin/api/v1/profile');
    }
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<Map<String, dynamic>>(
      future: _future,
      builder: (context, snapshot) {
        final data = snapshot.data ?? {};
        final username = (data['username'] ?? widget.username).toString();
        final email = (data['email'] ?? '').toString();
        final avatarUrl = resolveAvatarUrl(
          baseUrl: _baseUrl,
          qq: data['qq']?.toString(),
          avatarUrl: data['avatar_url']?.toString(),
        );
        return Container(
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            gradient: const LinearGradient(
              colors: [Color(0xFF1E88E5), Color(0xFF42A5F5)],
              begin: Alignment.topLeft,
              end: Alignment.bottomRight,
            ),
            borderRadius: BorderRadius.circular(14),
          ),
          child: Row(
            children: [
              CircleAvatar(
                radius: 24,
                backgroundColor: Colors.white24,
                child: avatarUrl.isNotEmpty
                    ? ClipOval(
                        child: Image.network(
                          avatarUrl,
                          width: 48,
                          height: 48,
                          fit: BoxFit.cover,
                          headers: _headers.isEmpty ? null : _headers,
                          errorBuilder: (_, __, ___) => const Icon(Icons.person, color: Colors.white),
                        ),
                      )
                    : Text(
                        username.isNotEmpty ? username.characters.first : '管',
                        style: const TextStyle(color: Colors.white, fontWeight: FontWeight.w700),
                      ),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(username, style: const TextStyle(color: Colors.white, fontWeight: FontWeight.w700, fontSize: 16)),
                    const SizedBox(height: 2),
                    Text(
                      email.isEmpty ? '管理员设置中心' : email,
                      style: const TextStyle(color: Colors.white70, fontSize: 12),
                    ),
                  ],
                ),
              ),
            ],
          ),
        );
      },
    );
  }
}

class _SettingsGroup {
  final String title;
  final IconData icon;
  final List<_SettingsItem> items;
  _SettingsGroup({required this.title, required this.icon, required this.items});
}

class _SettingsItem {
  final String title;
  final String subtitle;
  final IconData icon;
  final WidgetBuilder builder;
  final bool enabled;
  _SettingsItem({
    required this.title,
    required this.subtitle,
    required this.icon,
    required this.builder,
    this.enabled = true,
  });
}
