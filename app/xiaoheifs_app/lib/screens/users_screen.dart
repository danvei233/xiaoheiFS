import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:url_launcher/url_launcher.dart';

import '../app_state.dart';
import '../services/api_client.dart';
import '../services/avatar.dart';
import 'user_detail_screen.dart';

class UsersScreen extends StatefulWidget {
  const UsersScreen({super.key});

  @override
  State<UsersScreen> createState() => _UsersScreenState();
}

class _UsersScreenState extends State<UsersScreen> {
  ApiClient? _client;
  bool _loading = false;
  List<UserItem> _items = [];

  final _keywordController = TextEditingController();
  String _statusFilter = '';
  int _page = 1;
  int _pageSize = 20;
  int _total = 0;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final client = context.read<AppState>().apiClient;
    if (client?.baseUrl != _client?.baseUrl ||
        client?.apiKey != _client?.apiKey ||
        client?.token != _client?.token) {
      _client = client;
      if (client != null) {
        _load(client, showSpinner: _items.isEmpty);
      }
    }
  }

  @override
  void dispose() {
    _keywordController.dispose();
    super.dispose();
  }

  Future<void> _load(ApiClient client, {bool showSpinner = false}) async {
    if (showSpinner) {
      setState(() => _loading = true);
    } else {
      _loading = true;
    }
    try {
      final query = <String, String>{
        'limit': _pageSize.toString(),
        'offset': ((_page - 1) * _pageSize).toString(),
      };
      final resp = await client.getJson('/admin/api/v1/users', query: query);
      var items = (resp['items'] as List<dynamic>? ?? [])
          .map((e) => UserItem.fromJson(e as Map<String, dynamic>))
          .where((e) => e.role != 'admin')
          .toList();
      _total = (resp['total'] as int?) ?? items.length;
      final kw = _keywordController.text.trim();
      if (kw.isNotEmpty) {
        items = items
            .where(
              (e) =>
                  e.id.toString().contains(kw) ||
                  e.username.contains(kw) ||
                  e.email.contains(kw) ||
                  e.phone.contains(kw),
            )
            .toList();
      }
      if (_statusFilter.isNotEmpty) {
        items = items.where((e) => e.status == _statusFilter).toList();
      }
      if (mounted) {
        setState(() {
          _items = items;
          _loading = false;
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() => _loading = false);
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('加载失败：$e')));
      }
    }
  }

  void _refresh() {
    final client = context.read<AppState>().apiClient;
    if (client != null) _load(client);
  }

  void _resetFilters() {
    _keywordController.clear();
    _statusFilter = '';
    _page = 1;
    _refresh();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Material(
      child: Stack(
        children: [
          ListView(
            padding: const EdgeInsets.fromLTRB(16, 10, 16, 16),
            children: [
              Row(
                children: [
                  Text(
                    '用户管理',
                    style: theme.textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.w700,
                      fontSize: 16,
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 6),
              _FilterBar(
                keywordController: _keywordController,
                status: _statusFilter,
                onStatusChanged: (value) => _statusFilter = value,
                onSearch: () {
                  _page = 1;
                  _refresh();
                },
                onReset: _resetFilters,
                onRefresh: _refresh,
              ),
              const SizedBox(height: 8),
              if (_items.isEmpty && !_loading)
                const _EmptyState()
              else
                ..._items.map(
                  (item) => _UserItem(
                    item: item,
                    onDetail: () => _openDetail(item),
                    onEdit: () => _openEdit(item),
                    onResetPassword: () => _openReset(item),
                    onToggle: () => _toggleStatus(item),
                    onImpersonate: () => _impersonate(item),
                  ),
                ),
              const SizedBox(height: 8),
              _PaginationBar(
                page: _page,
                pageSize: _pageSize,
                total: _total,
                onPrev: _page > 1
                    ? () {
                        _page -= 1;
                        _refresh();
                      }
                    : null,
                onNext: _page * _pageSize < _total
                    ? () {
                        _page += 1;
                        _refresh();
                      }
                    : null,
              ),
            ],
          ),
          if (_loading)
            const Positioned(
              left: 0,
              right: 0,
              top: 0,
              child: LinearProgressIndicator(minHeight: 2),
            ),
          Positioned(
            right: 12,
            bottom: 16,
            child: FloatingActionButton.extended(
              extendedPadding: const EdgeInsets.symmetric(horizontal: 12),
              onPressed: _loading ? null : _openCreate,
              icon: const Icon(Icons.person_add, size: 18),
              label: const Text('新增用户'),
            ),
          ),
        ],
      ),
    );
  }

Future<void> _openDetail(UserItem item) async {
    final changed = await Navigator.push<bool>(
      context,
      MaterialPageRoute(builder: (_) => UserDetailScreen(userId: item.id)),
    );
    if (changed == true) _refresh();
  }

  Future<void> _openCreate() async {
    final qqController = TextEditingController();
    final username = TextEditingController();
    final email = TextEditingController();
    final password = TextEditingController(text: _randomPassword());
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('创建用户'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: qqController,
              decoration: const InputDecoration(labelText: 'QQ 号'),
            ),
            TextField(
              controller: username,
              decoration: const InputDecoration(labelText: '用户名'),
            ),
            TextField(
              controller: email,
              decoration: const InputDecoration(labelText: '邮箱'),
            ),
            Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: password,
                    obscureText: true,
                    decoration: const InputDecoration(labelText: '密码'),
                  ),
                ),
                const SizedBox(width: 4),
                OutlinedButton(
                  onPressed: () => password.text = _randomPassword(),
                  child: const Text('随机生成'),
                ),
              ],
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('取消'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('创建'),
          ),
        ],
      ),
    );
    if (ok == true) {
      final client = context.read<AppState>().apiClient;
      if (client == null) return;
      final qq = qqController.text.trim();
      final resolvedUsername = username.text.trim().isNotEmpty
          ? username.text.trim()
          : (qq.isNotEmpty ? qq : '');
      final resolvedEmail = email.text.trim().isNotEmpty
          ? email.text.trim()
          : (qq.isNotEmpty ? '$qq@qq.com' : '');
      final resolvedPassword = password.text.isNotEmpty
          ? password.text
          : _randomPassword();
      if (resolvedUsername.isEmpty || resolvedPassword.isEmpty) {
        if (mounted) {
          ScaffoldMessenger.of(
            context,
          ).showSnackBar(const SnackBar(content: Text('用户名或密码不能为空')));
        }
        return;
      }
      try {
        await client.postJson(
          '/admin/api/v1/users',
          body: {
            'username': resolvedUsername,
            'email': resolvedEmail,
            'password': resolvedPassword,
            'qq': qq,
          },
        );
        final apiUrl = context.read<AppState>().session?.apiUrl ?? '';
        final clip = [
          '账号：$resolvedUsername',
          '密码：$resolvedPassword',
          '登录地址：$apiUrl',
        ].join('\n');
        await Clipboard.setData(ClipboardData(text: clip));
        if (mounted) {
          ScaffoldMessenger.of(
            context,
          ).showSnackBar(const SnackBar(content: Text('已创建用户并复制账号信息')));
        }
        _page = 1;
        _refresh();
      } catch (e) {
        if (mounted) {
          ScaffoldMessenger.of(
            context,
          ).showSnackBar(SnackBar(content: Text('创建失败：$e')));
        }
      }
    }
  }

  Future<void> _openEdit(UserItem item) async {
    final username = TextEditingController(text: item.username);
    final email = TextEditingController(text: item.email);
    final qq = TextEditingController(text: item.qq);
    final avatar = TextEditingController(text: item.avatarUrl);
    String status = item.status.isEmpty ? 'active' : item.status;
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('编辑用户'),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: username,
                decoration: const InputDecoration(labelText: '用户名'),
              ),
              TextField(
                controller: email,
                decoration: const InputDecoration(labelText: '邮箱'),
              ),
              TextField(
                controller: qq,
                decoration: const InputDecoration(labelText: 'QQ'),
              ),
              TextField(
                controller: avatar,
                decoration: const InputDecoration(labelText: '头像 URL'),
              ),
              const SizedBox(height: 6),
              DropdownButtonFormField<String>(
                value: status,
                items: const [
                  DropdownMenuItem(value: 'active', child: Text('active')),
                  DropdownMenuItem(value: 'blocked', child: Text('blocked')),
                ],
                onChanged: (value) => status = value ?? status,
                decoration: const InputDecoration(labelText: '状态'),
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('取消'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('保存'),
          ),
        ],
      ),
    );
    if (ok == true) {
      final client = context.read<AppState>().apiClient;
      if (client == null) return;
      await client.patchJson(
        '/admin/api/v1/users/${item.id}',
        body: {
          'username': username.text.trim(),
          'email': email.text.trim(),
          'qq': qq.text.trim(),
          'avatar': avatar.text.trim(),
          'status': status,
        },
      );
      _refresh();
    }
  }

  Future<void> _openReset(UserItem item) async {
    final password = TextEditingController();
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('重置密码'),
        content: TextField(
          controller: password,
          obscureText: true,
          decoration: const InputDecoration(labelText: '新密码'),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('取消'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('确认'),
          ),
        ],
      ),
    );
    if (ok == true) {
      final client = context.read<AppState>().apiClient;
      if (client == null) return;
      await client.postJson(
        '/admin/api/v1/users/${item.id}/reset-password',
        body: {'password': password.text},
      );
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('已重置密码')));
    }
  }

  Future<void> _toggleStatus(UserItem item) async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    final status = item.status == 'active' ? 'blocked' : 'active';
    await client.patchJson(
      '/admin/api/v1/users/${item.id}/status',
      body: {'status': status},
    );
    _refresh();
  }

  Future<void> _impersonate(UserItem item) async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    final resp = await client.postJson(
      '/admin/api/v1/users/${item.id}/impersonate',
    );
    final token = resp['access_token'] as String?;
    if (token == null || token.isEmpty) {
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('未获取到用户令牌')));
      return;
    }
    await Clipboard.setData(ClipboardData(text: token));
    final baseUrl = context.read<AppState>().apiClient?.baseUrl ?? '';
    final url = _buildImpersonateConsoleUri(baseUrl, token);
    final opened = await launchUrl(url, mode: LaunchMode.externalApplication);
    if (!opened && mounted) {
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('无法打开浏览器，请检查系统设置')));
    }
  }
}

Uri _buildImpersonateConsoleUri(String rawBaseUrl, String token) {
  final normalized = rawBaseUrl.endsWith('/')
      ? rawBaseUrl.substring(0, rawBaseUrl.length - 1)
      : rawBaseUrl;
  return Uri.parse(
    '$normalized/console#impersonate_token=${Uri.encodeComponent(token)}',
  );
}

class _FilterBar extends StatelessWidget {
  final TextEditingController keywordController;
  final String status;
  final ValueChanged<String> onStatusChanged;
  final VoidCallback onSearch;
  final VoidCallback onReset;
  final VoidCallback onRefresh;

  const _FilterBar({
    required this.keywordController,
    required this.status,
    required this.onStatusChanged,
    required this.onSearch,
    required this.onReset,
    required this.onRefresh,
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
        boxShadow: [
          BoxShadow(
            color: colorScheme.shadow.withOpacity(0.05),
            blurRadius: 8,
            offset: const Offset(0, 2),
          ),
        ],
      ),
      child: Padding(
        padding: const EdgeInsets.all(8),
        child: Column(
          children: [
            Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: keywordController,
                    decoration: const InputDecoration(
                      hintText: '关键词（ID/用户名/邮箱/手机号）',
                      prefixIcon: Icon(Icons.search, size: 18),
                      isDense: true,
                      contentPadding: EdgeInsets.symmetric(horizontal: 10, vertical: 10),
                    ),
                  ),
                ),
                const SizedBox(width: 4),
                FilledButton.icon(
                  style: FilledButton.styleFrom(
                    padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                    minimumSize: const Size(0, 30),
                    tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                    textStyle: const TextStyle(fontSize: 12),
                  ),
                  onPressed: onSearch,
                  icon: const Icon(Icons.search_rounded),
                  label: const Text('搜索'),
                ),
              ],
            ),
            const SizedBox(height: 6),
            SizedBox(
              height: 30,
              child: ListView(
                scrollDirection: Axis.horizontal,
                children: [
                  _FilterChip(
                    label: '全部',
                    selected: status.isEmpty,
                    onTap: () => onStatusChanged(''),
                  ),
                  const SizedBox(width: 4),
                  _FilterChip(
                    label: '正常',
                    selected: status == 'active',
                    onTap: () => onStatusChanged('active'),
                  ),
                  const SizedBox(width: 4),
                  _FilterChip(
                    label: '已禁用',
                    selected: status == 'blocked',
                    onTap: () => onStatusChanged('blocked'),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 6),
            Row(
              children: [
                Expanded(
                  child: OutlinedButton.icon(
                    style: OutlinedButton.styleFrom(
                      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                      minimumSize: const Size(0, 30),
                      tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                      textStyle: const TextStyle(fontSize: 12),
                    ),
                    onPressed: onReset,
                    icon: const Icon(Icons.restart_alt_rounded),
                    label: const Text('重置'),
                  ),
                ),
                const SizedBox(width: 4),
                Expanded(
                  child: OutlinedButton.icon(
                    style: OutlinedButton.styleFrom(
                      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                      minimumSize: const Size(0, 30),
                      tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                      textStyle: const TextStyle(fontSize: 12),
                    ),
                    onPressed: onRefresh,
                    icon: const Icon(Icons.refresh_rounded),
                    label: const Text('刷新'),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _FilterChip extends StatelessWidget {
  final String label;
  final bool selected;
  final VoidCallback onTap;

  const _FilterChip({
    required this.label,
    required this.selected,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final bgColor = selected
        ? colorScheme.primaryContainer.withOpacity(0.7)
        : colorScheme.surface;
    final borderColor = selected
        ? colorScheme.primary
        : colorScheme.outlineVariant.withOpacity(0.7);
    final textColor =
        selected ? colorScheme.primary : colorScheme.onSurface;

    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(8),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
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
                const SizedBox(width: 4),
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

class _UserItem extends StatelessWidget {
  final UserItem item;
  final VoidCallback onDetail;
  final VoidCallback onEdit;
  final VoidCallback onResetPassword;
  final VoidCallback onToggle;
  final VoidCallback onImpersonate;

  const _UserItem({
    required this.item,
    required this.onDetail,
    required this.onEdit,
    required this.onResetPassword,
    required this.onToggle,
    required this.onImpersonate,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final statusColor = item.statusColor;
    final baseUrl = context.read<AppState>().apiClient?.baseUrl ?? '';
    final avatarUrl = resolveAvatarUrl(
      baseUrl: baseUrl,
      qq: item.qq,
      avatarUrl: item.avatarUrl.isNotEmpty ? item.avatarUrl : null,
    );
    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: colorScheme.outlineVariant.withOpacity(0.5)),
        boxShadow: [
          BoxShadow(
            color: colorScheme.shadow.withOpacity(0.05),
            blurRadius: 8,
            offset: const Offset(0, 2),
          ),
        ],
      ),
      child: InkWell(
        onTap: onDetail,
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.all(8),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  _Avatar(url: avatarUrl, radius: 14),
                  const SizedBox(width: 6),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Row(
                          children: [
                            Expanded(
                              child: Text(
                                item.username,
                                style: theme.textTheme.titleSmall?.copyWith(
                                  fontWeight: FontWeight.w700,
                    fontSize: 16,
                  ),
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                              ),
                            ),
                            _StatusPill(
                              label: item.statusLabel,
                              color: statusColor,
                            ),
                          ],
                        ),
                        const SizedBox(height: 2),
                        Text(
                          item.contact,
                          style: theme.textTheme.bodySmall?.copyWith(
                            color: colorScheme.onSurfaceVariant,
                            fontSize: 11,
                          ),
                        ),
                        const SizedBox(height: 2),
                        Text(
                          'ID ${item.id} · ${item.roleLabel}',
                          style: theme.textTheme.labelSmall?.copyWith(
                            color: colorScheme.onSurfaceVariant,
                            fontSize: 10,
                          ),
                        ),
                      ],
                    ),
                  ),
                  PopupMenuButton<String>(
                    onSelected: (value) {
                      switch (value) {
                        case 'detail':
                          onDetail();
                          break;
                        case 'edit':
                          onEdit();
                          break;
                        case 'reset':
                          onResetPassword();
                          break;
                        case 'toggle':
                          onToggle();
                          break;
                        case 'impersonate':
                          onImpersonate();
                          break;
                      }
                    },
                    itemBuilder: (context) => const [
                      PopupMenuItem(value: 'detail', child: Text('详情')),
                      PopupMenuItem(value: 'edit', child: Text('编辑')),
                      PopupMenuItem(value: 'impersonate', child: Text('以此用户登录')),
                      PopupMenuItem(value: 'toggle', child: Text('禁用/启用')),
                      PopupMenuItem(value: 'reset', child: Text('重置密码')),
                    ],
                  ),
                ],
              ),
              const SizedBox(height: 6),
              Wrap(
                spacing: 6,
                runSpacing: 6,
                children: [
                  _MiniActionButton(
                    icon: Icons.visibility_outlined,
                    label: '详情',
                    onTap: onDetail,
                  ),
                  _MiniActionButton(
                    icon: Icons.edit_outlined,
                    label: '编辑',
                    onTap: onEdit,
                  ),
                  _MiniActionButton(
                    icon: Icons.lock_reset_outlined,
                    label: '重置密码',
                    onTap: onResetPassword,
                  ),
                  _MiniActionButton(
                    icon: item.status == 'active'
                        ? Icons.block_outlined
                        : Icons.check_circle_outline,
                    label: item.status == 'active' ? '禁用' : '启用',
                    onTap: onToggle,
                  ),
                  _MiniActionButton(
                    icon: Icons.login_outlined,
                    label: '以此登录',
                    onTap: onImpersonate,
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _StatusPill extends StatelessWidget {
  final String label;
  final Color color;

  const _StatusPill({required this.label, required this.color});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: color.withOpacity(0.12),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        label,
        style: TextStyle(
          color: color,
          fontWeight: FontWeight.w600,
          fontSize: 10,
        ),
      ),
    );
  }
}

class _MiniActionButton extends StatelessWidget {
  final IconData icon;
  final String label;
  final VoidCallback onTap;

  const _MiniActionButton({
    required this.icon,
    required this.label,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return Material(
      color: colorScheme.surfaceContainerHighest.withOpacity(0.35),
      borderRadius: BorderRadius.circular(8),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(8),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(icon, size: 12, color: colorScheme.primary),
              const SizedBox(width: 4),
              Text(
                label,
                style: TextStyle(
                  color: colorScheme.primary,
                  fontWeight: FontWeight.w600,
                  fontSize: 10,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class UserItem {
  final int id;
  final String username;
  final String email;
  final String phone;
  final String role;
  final String status;
  final String qq;
  final String avatarUrl;

  UserItem({
    required this.id,
    required this.username,
    required this.email,
    required this.phone,
    required this.role,
    required this.status,
    required this.qq,
    required this.avatarUrl,
  });

  factory UserItem.fromJson(Map<String, dynamic> json) {
    return UserItem(
      id: _asInt(json['id']),
      username: json['username'] as String? ?? '未知用户',
      email: json['email'] as String? ?? '',
      phone: json['phone'] as String? ?? '',
      role: json['role'] as String? ?? '',
      status: json['status'] as String? ?? '',
      qq: json['qq'] as String? ?? '',
      avatarUrl: json['avatar_url'] as String? ?? '',
    );
  }

  String get contact {
    if (phone.isNotEmpty) return phone;
    if (email.isNotEmpty) return email;
    return '未填写';
  }

  String get roleLabel {
    if (role == 'admin') return '管理员';
    return '用户';
  }

  String get statusLabel {
    switch (status) {
      case 'active':
        return '正常';
      case 'pending':
        return '待审核';
      case 'blocked':
        return '已禁用';
      default:
        return status.isEmpty ? '未知' : status;
    }
  }

  Color get statusColor {
    switch (status) {
      case 'active':
        return const Color(0xFF00A68C);
      case 'pending':
        return const Color(0xFFEF6C00);
      case 'blocked':
        return const Color(0xFFD32F2F);
      default:
        return Colors.black54;
    }
  }
}

class _Avatar extends StatelessWidget {
  final String url;
  final double radius;

  const _Avatar({required this.url, required this.radius});

  @override
  Widget build(BuildContext context) {
    final size = radius * 2;
    if (url.isEmpty) {
      return CircleAvatar(radius: radius, child: const Icon(Icons.person));
    }
    final session = context.read<AppState>().session;
    final headers = avatarHeaders(
      token: session?.token,
      apiKey: session?.apiKey,
    );
    return CircleAvatar(
      radius: radius,
      backgroundColor: Theme.of(context).colorScheme.surface,
      child: ClipOval(
        child: Image.network(
          url,
          width: size,
          height: size,
          fit: BoxFit.cover,
          headers: headers.isEmpty ? null : headers,
          errorBuilder: (context, error, stack) {
            return SizedBox(
              width: size,
              height: size,
              child: const Icon(Icons.person),
            );
          },
        ),
      ),
    );
  }
}

String _randomPassword() {
  const chars = 'ABCDEFGHJKMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789';
  final now = DateTime.now().millisecondsSinceEpoch;
  final sb = StringBuffer();
  for (var i = 0; i < 10; i++) {
    final idx = (now + i * 37) % chars.length;
    sb.write(chars[idx]);
  }
  return sb.toString();
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
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: [
        Text('第 $page / $totalPages 页 · 共 $total 条'),
        Row(
          children: [
            OutlinedButton(onPressed: onPrev, child: const Text('上一页')),
            const SizedBox(width: 4),
            OutlinedButton(onPressed: onNext, child: const Text('下一页')),
          ],
        ),
      ],
    );
  }
}

class _EmptyState extends StatelessWidget {
  const _EmptyState();

  @override
  Widget build(BuildContext context) {
    return const Center(
      child: Padding(padding: EdgeInsets.all(16), child: Text('暂无用户')),
    );
  }
}

int _asInt(dynamic value) {
  if (value is int) return value;
  if (value is num) return value.toInt();
  if (value is String) return int.tryParse(value) ?? 0;
  return 0;
}


