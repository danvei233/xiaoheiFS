import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:url_launcher/url_launcher.dart';

import '../app_state.dart';
import '../services/avatar.dart';
import 'user_detail_screen.dart';
import 'users_screen.dart';

class TicketDetailScreen extends StatefulWidget {
  final int ticketId;

  const TicketDetailScreen({super.key, required this.ticketId});

  @override
  State<TicketDetailScreen> createState() => _TicketDetailScreenState();
}

class _TicketDetailScreenState extends State<TicketDetailScreen> {
  bool _loading = true;
  bool _busy = false;
  final _messageController = TextEditingController();
  String _replyStatus = '';

  Map<String, dynamic> _ticket = {};
  List<Map<String, dynamic>> _messages = [];
  Map<String, dynamic> _user = {};
  String? _userError;
  List<Map<String, dynamic>> _resources = [];
  final Map<int, Map<String, dynamic>> _vpsDetails = {};

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _loadAll(showSpinner: _loading);
  }

  @override
  void dispose() {
    _messageController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (_loading) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }

    final status = _ticket['status']?.toString() ?? '';
    final statusMeta = _ticketStatusMeta(status);
    final subject = _ticket['subject']?.toString().isNotEmpty == true
        ? _ticket['subject'] as String
        : '工单 #${_ticket['id'] ?? '-'}';

    return Scaffold(
      appBar: AppBar(
        title: Text(subject, style: const TextStyle(fontSize: 16)),
        toolbarHeight: 44,
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh, size: 20),
            onPressed: _busy ? null : () => _loadAll(showSpinner: false),
          ),
        ],
      ),
      body: Column(
        children: [
          _TicketHeader(
            statusMeta: statusMeta,
            userId: _ticket['user_id']?.toString() ?? '-',
            userName: _user['username']?.toString() ?? '-',
            userEmail: _user['email']?.toString() ?? '-',
            userPhone: _user['phone']?.toString() ?? '-',
            userQq: _user['qq']?.toString() ?? '',
            avatarUrl: _qqAvatar(
              context.read<AppState>().apiClient?.baseUrl ?? '',
              _user['qq']?.toString() ?? '',
            ),
            errorText: _userError,
            onMenuTap: () => _openUserMenu(
              context,
              _user,
              _resources,
              _vpsDetails,
              _openPanel,
              _refreshVps,
              _lockVps,
              _unlockVps,
              _emergencyRenew,
            ),
            createdAt: _ticket['created_at']?.toString() ?? '',
            updatedAt: _ticket['updated_at']?.toString() ?? '',
          ),
          const SizedBox(height: 6),
          Expanded(
            child: ListView.builder(
              padding: const EdgeInsets.fromLTRB(12, 6, 12, 12),
              itemCount: _messages.length,
              itemBuilder: (context, index) {
                final msg = _messages[index];
                final isAdmin =
                    (msg['sender_role']?.toString() ?? '') == 'admin';
                return _MessageBubble(message: msg, isMe: isAdmin);
              },
            ),
          ),
          _ComposerBar(
            busy: _busy,
            controller: _messageController,
            replyStatus: _replyStatus,
            onStatusChanged: (value) =>
                setState(() => _replyStatus = value ?? ''),
            onSend: _sendMessage,
          ),
        ],
      ),
    );
  }

  Future<void> _loadAll({required bool showSpinner}) async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    if (showSpinner) setState(() => _loading = true);
    try {
      final ticketResp = await client.getJson(
        '/admin/api/v1/tickets/${widget.ticketId}',
      );
      final ticket = ticketResp['ticket'] as Map<String, dynamic>? ?? {};
      final messages = (ticketResp['messages'] as List<dynamic>? ?? [])
          .cast<Map<String, dynamic>>();
      final resources = (ticketResp['resources'] as List<dynamic>? ?? [])
          .cast<Map<String, dynamic>>();
      Map<String, dynamic> user = {};
      String? userError;
      final userId = _asInt(ticket['user_id']);
      if (userId > 0) {
        try {
          final userResp = await client.getJson('/admin/api/v1/users/$userId');
          user = userResp['user'] is Map<String, dynamic>
              ? userResp['user'] as Map<String, dynamic>
              : (userResp['data'] is Map<String, dynamic>
                    ? userResp['data'] as Map<String, dynamic>
                    : userResp);
        } catch (e) {
          userError = e.toString();
        }
      }
      if (mounted) {
        setState(() {
          _ticket = ticket;
          _messages = messages;
          _resources = resources;
          _user = user;
          _userError = userError;
          _loading = false;
        });
      }
      await _loadResourceDetails(resources);
    } catch (e) {
      if (mounted) {
        setState(() => _loading = false);
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('加载失败：$e')));
      }
    }
  }

  Future<void> _sendMessage() async {
    if (_busy) return;
    final content = _messageController.text.trim();
    if (content.isEmpty) return;
    setState(() => _busy = true);
    try {
      final client = context.read<AppState>().apiClient;
      if (client == null) return;
      await client.postJson(
        '/admin/api/v1/tickets/${widget.ticketId}/messages',
        body: {'content': content},
      );
      if (_replyStatus.isNotEmpty) {
        await client.patchJson(
          '/admin/api/v1/tickets/${widget.ticketId}',
          body: {'status': _replyStatus},
        );
      }
      _messageController.clear();
      _replyStatus = '';
      await _loadAll(showSpinner: false);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('发送失败：$e')));
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _loadResourceDetails(
    List<Map<String, dynamic>> resources,
  ) async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    for (final res in resources) {
      final type = res['resource_type']?.toString() ?? '';
      if (type != 'vps') continue;
      final id = _asInt(res['resource_id']);
      if (id <= 0 || _vpsDetails.containsKey(id)) continue;
      try {
        final detail = await client.getJson('/admin/api/v1/vps/$id');
        if (mounted) {
          setState(() {
            _vpsDetails[id] = detail;
          });
        }
      } catch (_) {
        continue;
      }
    }
  }

  Future<void> _openPanel(int vpsId) async {
    final detail = _vpsDetails[vpsId];
    if (detail == null) return;
    var panel = detail['panel_url_cache']?.toString() ?? '';
    if (panel.isEmpty) {
      final client = context.read<AppState>().apiClient;
      if (client == null) return;
      await client.postJson('/admin/api/v1/vps/$vpsId/refresh');
      try {
        final refreshed = await client.getJson('/admin/api/v1/vps/$vpsId');
        _vpsDetails[vpsId] = refreshed;
        panel = refreshed['panel_url_cache']?.toString() ?? '';
      } catch (_) {}
    }
    if (panel.isEmpty) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(const SnackBar(content: Text('未获取到面板地址')));
      }
      return;
    }
    final uri = Uri.parse(panel);
    await launchUrl(uri, mode: LaunchMode.externalApplication);
  }

  Future<void> _refreshVps(int vpsId) async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    await client.postJson('/admin/api/v1/vps/$vpsId/refresh');
  }

  Future<void> _lockVps(int vpsId) async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    await client.postJson('/admin/api/v1/vps/$vpsId/lock');
  }

  Future<void> _unlockVps(int vpsId) async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    await client.postJson('/admin/api/v1/vps/$vpsId/unlock');
  }

  Future<void> _emergencyRenew(int vpsId) async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    await client.postJson('/admin/api/v1/vps/$vpsId/emergency-renew');
  }
}

void _openUserMenu(
  BuildContext context,
  Map<String, dynamic> user,
  List<Map<String, dynamic>> resources,
  Map<int, Map<String, dynamic>> vpsDetails,
  Future<void> Function(int) onOpenPanel,
  Future<void> Function(int) onRefresh,
  Future<void> Function(int) onLock,
  Future<void> Function(int) onUnlock,
  Future<void> Function(int) onEmergency,
) {
  final userId = _asInt(user['id']);
  showModalBottomSheet<void>(
    context: context,
    showDragHandle: true,
    isScrollControlled: true,
    builder: (context) => SafeArea(
      child: DraggableScrollableSheet(
        expand: false,
        initialChildSize: 0.7,
        minChildSize: 0.4,
        maxChildSize: 0.95,
        builder: (context, scrollController) {
          final vpsResources = resources
              .where((r) => (r['resource_type']?.toString() ?? '') == 'vps')
              .toList();
          final expanded = <int, bool>{};
          return StatefulBuilder(
            builder: (context, setLocal) {
              return ListView(
                controller: scrollController,
                padding: const EdgeInsets.all(16),
                children: [
                  Text('用户操作', style: Theme.of(context).textTheme.titleMedium),
                  const SizedBox(height: 6),
                  ListTile(
                    leading: const Icon(Icons.person),
                    title: const Text('查看用户详情'),
                    onTap: userId <= 0
                        ? null
                        : () {
                            Navigator.pop(context);
                            Navigator.push(
                              context,
                              MaterialPageRoute(
                                builder: (_) =>
                                    UserDetailScreen(userId: userId),
                              ),
                            );
                          },
                  ),
                  ListTile(
                    leading: const Icon(Icons.manage_accounts),
                    title: const Text('打开用户管理'),
                    onTap: () {
                      Navigator.pop(context);
                      Navigator.push(
                        context,
                        MaterialPageRoute(builder: (_) => const UsersScreen()),
                      );
                    },
                  ),
                  if (vpsResources.isNotEmpty) ...[
                    const SizedBox(height: 6),
                    Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        Text(
                          '关联实例',
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        Text(
                          '点“更多”展开操作',
                          style: Theme.of(context).textTheme.bodySmall
                              ?.copyWith(color: Colors.black54),
                        ),
                      ],
                    ),
                    const SizedBox(height: 6),
                    ...vpsResources.map((res) {
                      final id = _asInt(res['resource_id']);
                      final detail = vpsDetails[id] ?? {};
                      final isOpen = expanded[id] ?? false;
                      return Card(
                        child: Padding(
                          padding: const EdgeInsets.all(12),
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Row(
                                children: [
                                  Expanded(
                                    child: Text(
                                      '实例 #$id · ${res['resource_name'] ?? '-'}',
                                      style: const TextStyle(
                                        fontWeight: FontWeight.w600,
                                      ),
                                    ),
                                  ),
                                  TextButton.icon(
                                    onPressed: () =>
                                        setLocal(() => expanded[id] = !isOpen),
                                    icon: Icon(
                                      isOpen
                                          ? Icons.expand_less
                                          : Icons.expand_more,
                                    ),
                                    label: const Text('更多'),
                                  ),
                                ],
                              ),
                              const SizedBox(height: 6),
                              Text(
                                '地区 ${detail['region'] ?? '-'} · 套餐 ${detail['package_name'] ?? '-'}',
                              ),
                              Text(
                                '状态 ${detail['status'] ?? '-'} · 到期 ${_formatLocal(detail['expire_at']?.toString() ?? '')}',
                              ),
                              if (isOpen) ...[
                                const SizedBox(height: 6),
                                Wrap(
                                  spacing: 8,
                                  runSpacing: 8,
                                  children: [
                                    OutlinedButton(
                                      onPressed: () => onOpenPanel(id),
                                      child: const Text('登录面板'),
                                    ),
                                    OutlinedButton(
                                      onPressed: () => onRefresh(id),
                                      child: const Text('刷新实例'),
                                    ),
                                    OutlinedButton(
                                      onPressed: () => onEmergency(id),
                                      child: const Text('紧急续费'),
                                    ),
                                    TextButton(
                                      onPressed: () => onLock(id),
                                      child: const Text('锁定'),
                                    ),
                                    TextButton(
                                      onPressed: () => onUnlock(id),
                                      child: const Text('解锁'),
                                    ),
                                  ],
                                ),
                              ],
                            ],
                          ),
                        ),
                      );
                    }),
                  ],
                ],
              );
            },
          );
        },
      ),
    ),
  );
}

class _TicketHeader extends StatelessWidget {
  final _StatusMeta statusMeta;
  final String userId;
  final String userName;
  final String userEmail;
  final String userPhone;
  final String userQq;
  final String avatarUrl;
  final String? errorText;
  final VoidCallback onMenuTap;
  final String createdAt;
  final String updatedAt;

  const _TicketHeader({
    required this.statusMeta,
    required this.userId,
    required this.userName,
    required this.userEmail,
    required this.userPhone,
    required this.userQq,
    required this.avatarUrl,
    required this.onMenuTap,
    required this.createdAt,
    required this.updatedAt,
    this.errorText,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    return Container(
      margin: const EdgeInsets.fromLTRB(10, 4, 10, 0),
      padding: const EdgeInsets.all(6),
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: colorScheme.outlineVariant.withOpacity(0.5)),
        boxShadow: [
          BoxShadow(
            color: colorScheme.shadow.withOpacity(0.05),
            blurRadius: 6,
            offset: const Offset(0, 1),
          ),
        ],
      ),
      child: Row(
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
                        '用户 $userName · ID $userId',
                        style: theme.textTheme.titleSmall?.copyWith(
                          fontWeight: FontWeight.w700,
                          fontSize: 12,
                        ),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                    _StatusPill(label: statusMeta.label, color: statusMeta.color),
                  ],
                ),
                const SizedBox(height: 3),
                Wrap(
                  spacing: 4,
                  runSpacing: 3,
                  children: [
                    if (userEmail.isNotEmpty)
                      _InfoChip(icon: Icons.email_outlined, text: userEmail),
                    if (userPhone.isNotEmpty)
                      _InfoChip(icon: Icons.phone_outlined, text: userPhone),
                    _InfoChip(
                      icon: Icons.chat_outlined,
                      text: userQq.isEmpty ? '-' : userQq,
                    ),
                  ],
                ),
                const SizedBox(height: 3),
                Text(
                  '创建 ${_formatLocal(createdAt)} · 更新 ${_formatLocal(updatedAt)}',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: colorScheme.onSurfaceVariant,
                    fontSize: 10,
                  ),
                ),
                if (errorText != null)
                  Padding(
                    padding: const EdgeInsets.only(top: 2),
                    child: Text(
                      '用户信息加载失败：$errorText',
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: colorScheme.error,
                        fontSize: 10,
                      ),
                    ),
                  ),
              ],
            ),
          ),
          IconButton(
            onPressed: onMenuTap,
            icon: const Icon(Icons.more_horiz, size: 20),
          ),
        ],
      ),
    );
  }
}

class _MessageBubble extends StatelessWidget {
  final Map<String, dynamic> message;
  final bool isMe;

  const _MessageBubble({required this.message, required this.isMe});

  @override
  Widget build(BuildContext context) {
    final role = message['sender_role']?.toString() ?? '';
    final qq = message['sender_qq']?.toString() ?? '';
    final avatar = _qqAvatar(
      context.read<AppState>().apiClient?.baseUrl ?? '',
      qq,
    );
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final bg = isMe ? const Color(0xFF00BFA6) : colorScheme.surface;
    final fg = isMe ? Colors.white : colorScheme.onSurface;
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisAlignment: isMe
            ? MainAxisAlignment.end
            : MainAxisAlignment.start,
        children: [
          if (!isMe) _Avatar(url: avatar, radius: 14),
          if (!isMe) const SizedBox(width: 6),
          Flexible(
            child: Column(
              crossAxisAlignment: isMe
                  ? CrossAxisAlignment.end
                  : CrossAxisAlignment.start,
              children: [
                ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 300),
                  child: Container(
                    padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
                    decoration: BoxDecoration(
                      color: bg,
                      borderRadius: BorderRadius.only(
                        topLeft: const Radius.circular(12),
                        topRight: const Radius.circular(12),
                        bottomLeft: Radius.circular(isMe ? 12 : 4),
                        bottomRight: Radius.circular(isMe ? 4 : 12),
                      ),
                      border: Border.all(
                        color: colorScheme.outlineVariant.withOpacity(0.5),
                      ),
                    ),
                    child: Text(
                      message['content']?.toString() ?? '',
                      style: TextStyle(color: fg, height: 1.35, fontSize: 12),
                    ),
                  ),
                ),
                const SizedBox(height: 3),
                Text(
                  '${role.isEmpty ? '-' : role} · ${_formatLocal(message['created_at']?.toString() ?? '')}',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: colorScheme.onSurfaceVariant,
                    fontSize: 10,
                  ),
                ),
              ],
            ),
          ),
          if (isMe) const SizedBox(width: 6),
          if (isMe) _Avatar(url: avatar, radius: 14),
        ],
      ),
    );
  }
}

class _ComposerBar extends StatelessWidget {
  final bool busy;
  final TextEditingController controller;
  final String replyStatus;
  final ValueChanged<String?> onStatusChanged;
  final VoidCallback onSend;

  const _ComposerBar({
    required this.busy,
    required this.controller,
    required this.replyStatus,
    required this.onStatusChanged,
    required this.onSend,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    return SafeArea(
      top: false,
      child: Container(
        padding: const EdgeInsets.fromLTRB(8, 6, 8, 8),
        decoration: BoxDecoration(
          color: colorScheme.surface,
          border: Border(
            top: BorderSide(color: colorScheme.outlineVariant.withOpacity(0.6)),
          ),
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Row(
              children: [
                Text(
                  '\u56de\u590d\u540e\u72b6\u6001',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: colorScheme.onSurfaceVariant,
                    fontSize: 9,
                  ),
                ),
                const SizedBox(width: 3),
                SizedBox(
                  width: 110,
                  child: DropdownButtonFormField<String>(
                    value: replyStatus.isEmpty ? null : replyStatus,
                    menuMaxHeight: 240,
                    style: const TextStyle(fontSize: 10),
                    isDense: true,
                    iconSize: 16,
                    dropdownColor: null,
                    items: const [
                      DropdownMenuItem(value: '', child: Text('\u4e0d\u4fee\u6539')),
                      DropdownMenuItem(value: 'open', child: Text('\u5f85\u5904\u7406')),
                      DropdownMenuItem(
                        value: 'waiting_user',
                        child: Text('\u7b49\u5f85\u7528\u6237'),
                      ),
                      DropdownMenuItem(
                        value: 'waiting_admin',
                        child: Text('\u5904\u7406\u4e2d'),
                      ),
                      DropdownMenuItem(value: 'closed', child: Text('\u5df2\u5173\u95ed')),
                    ],
                    onChanged: onStatusChanged,
                    decoration: const InputDecoration(
                      isDense: true,
                      contentPadding: EdgeInsets.symmetric(
                        horizontal: 6,
                        vertical: 2,
                      ),
                      border: OutlineInputBorder(),
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 4),
            Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: controller,
                    maxLines: 2,
                    decoration: const InputDecoration(
                      hintText: '\u8f93\u5165\u56de\u590d\u5185\u5bb9...',
                      prefixIcon: Icon(Icons.edit_outlined, size: 14),
                      isDense: true,
                      contentPadding: EdgeInsets.symmetric(horizontal: 8, vertical: 8),
                    ),
                  ),
                ),
                const SizedBox(width: 3),
                FilledButton.icon(
                  onPressed: busy ? null : onSend,
                  style: FilledButton.styleFrom(
                    padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                    minimumSize: const Size(0, 28),
                    tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                    textStyle: const TextStyle(fontSize: 11),
                  ),
                  icon: const Icon(Icons.send_rounded, size: 14),
                  label: const Text('\u53d1\u9001'),
                ),
              ],
            ),

          ],
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
      padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 2),
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

class _InfoChip extends StatelessWidget {
  final IconData icon;
  final String text;

  const _InfoChip({required this.icon, required this.text});

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 2),
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: colorScheme.outlineVariant.withOpacity(0.5)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 11, color: colorScheme.onSurfaceVariant),
          const SizedBox(width: 3),
          Text(
            text,
            style: TextStyle(
              color: colorScheme.onSurfaceVariant,
              fontSize: 10,
              fontWeight: FontWeight.w500,
            ),
          ),
        ],
      ),
    );
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

class _StatusMeta {
  final String label;
  final IconData icon;
  final Color color;

  const _StatusMeta(this.label, this.icon, this.color);
}

_StatusMeta _ticketStatusMeta(String status) {
  switch (status) {
    case 'open':
      return const _StatusMeta('待处理', Icons.report, Color(0xFF1E88E5));
    case 'waiting_user':
      return const _StatusMeta(
        '等待用户',
        Icons.hourglass_bottom,
        Color(0xFFEF6C00),
      );
    case 'waiting_admin':
      return const _StatusMeta('处理中', Icons.support_agent, Color(0xFF7B1FA2));
    case 'closed':
      return const _StatusMeta('已关闭', Icons.check_circle, Color(0xFF00A68C));
    default:
      return _StatusMeta(
        status.isEmpty ? '未知' : status,
        Icons.info,
        const Color(0xFF546E7A),
      );
  }
}

String _formatLocal(String raw) {
  if (raw.isEmpty) return '-';
  final dt = DateTime.tryParse(raw);
  if (dt == null) return raw;
  final local = dt.toLocal();
  return '${local.year}-${_pad2(local.month)}-${_pad2(local.day)} ${_pad2(local.hour)}:${_pad2(local.minute)}';
}

String _pad2(int v) => v.toString().padLeft(2, '0');

String _qqAvatar(String baseUrl, String qq) {
  if (qq.isEmpty) return '';
  return resolveAvatarUrl(baseUrl: baseUrl, qq: qq);
}

int _asInt(dynamic value) {
  if (value is int) return value;
  if (value is num) return value.toInt();
  if (value is String) return int.tryParse(value) ?? 0;
  return 0;
}
