import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:url_launcher/url_launcher.dart';

import '../app_state.dart';

class ServersScreen extends StatefulWidget {
  const ServersScreen({super.key});

  @override
  State<ServersScreen> createState() => _ServersScreenState();
}

class _ServersScreenState extends State<ServersScreen> {
  Future<List<VpsItem>>? _future;
  bool _loading = false;

  String _statusFilter = '';
  int _page = 1;
  int _pageSize = 20;
  int _total = 0;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final client = context.read<AppState>().apiClient;
    if (client != null) {
      _future = _load(client);
    }
  }

  Future<List<VpsItem>> _load(client) async {
    setState(() => _loading = true);
    try {
      final resp = await client.getJson('/admin/api/v1/vps', query: {
        'limit': _pageSize.toString(),
        'offset': ((_page - 1) * _pageSize).toString(),
      });
      final items = (resp['items'] as List<dynamic>? ?? [])
          .map((e) => VpsItem.fromJson(e as Map<String, dynamic>))
          .toList();
      _total = (resp['total'] as int?) ?? items.length;
      await _attachUsernames(client, items);
      if (_statusFilter.isNotEmpty) {
        return items.where((e) => e.adminStatus == _statusFilter).toList();
      }
      return items;
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  void _refresh() {
    final client = context.read<AppState>().apiClient;
    if (client != null) setState(() => _future = _load(client));
  }

  Future<void> _attachUsernames(client, List<VpsItem> items) async {
    final ids = items.map((e) => e.userId).toSet();
    for (final id in ids) {
      if (id <= 0) continue;
      try {
        final user = await client.getJson('/admin/api/v1/users/$id');
        final map = user['user'] is Map<String, dynamic>
            ? user['user'] as Map<String, dynamic>
            : (user['data'] is Map<String, dynamic> ? user['data'] as Map<String, dynamic> : user);
        final name = map['username'] as String?;
        final role = map['role'] as String?;
        if (name == null || name.isEmpty) continue;
        for (final item in items) {
          if (item.userId == id) {
            item.username = name;
            if (role != null && role.isNotEmpty) {
              item.userRole = role;
            }
          }
        }
      } catch (_) {
        continue;
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<List<VpsItem>>(
      future: _future,
      builder: (context, snapshot) {
        if (snapshot.connectionState == ConnectionState.waiting) {
          return const Center(child: CircularProgressIndicator());
        }
        if (snapshot.hasError) {
          return Center(child: Text('加载失败：$snapshot'));
        }
        final items = snapshot.data ?? [];
        return ListView(
          padding: const EdgeInsets.fromLTRB(16, 16, 16, 24),
          children: [
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text(
                  '服务器管理',
                  style: Theme.of(context).textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.w700,
                      ),
                ),
                OutlinedButton.icon(
                  onPressed: _loading ? null : _refresh,
                  icon: const Icon(Icons.refresh),
                  label: const Text('刷新'),
                )
              ],
            ),
            const SizedBox(height: 10),
            _StatusTabs(
              value: _statusFilter,
              onChanged: (value) {
                _statusFilter = value;
                _page = 1;
                _refresh();
              },
            ),
            const SizedBox(height: 12),
            if (items.isEmpty)
              const Center(child: Text('暂无实例'))
            else
              ...items.map((item) => _VpsTile(item: item, onAction: _handleAction)),
            const SizedBox(height: 12),
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
        );
      },
    );
  }

  Future<void> _handleAction(VpsItem item, String action) async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    switch (action) {
      case 'refresh':
        await client.postJson('/admin/api/v1/vps/${item.id}/refresh');
        break;
      case 'lock':
        await client.postJson('/admin/api/v1/vps/${item.id}/lock');
        break;
      case 'unlock':
        await client.postJson('/admin/api/v1/vps/${item.id}/unlock');
        break;
      case 'emergency':
        await client.postJson('/admin/api/v1/vps/${item.id}/emergency-renew');
        break;
      case 'delete':
        await client.postJson('/admin/api/v1/vps/${item.id}/delete', body: {'reason': item.deleteReason});
        break;
      case 'resize':
        await client.postJson('/admin/api/v1/vps/${item.id}/resize', body: item.resizePayload);
        break;
      case 'status':
        await client.postJson('/admin/api/v1/vps/${item.id}/status', body: item.statusPayload);
        break;
      case 'expire':
        await client.patchJson('/admin/api/v1/vps/${item.id}/expire-at', body: item.expirePayload);
        break;
      case 'edit':
        await client.patchJson('/admin/api/v1/vps/${item.id}', body: item.editPayload);
        break;
    }
    _refresh();
  }
}

class _VpsTile extends StatelessWidget {
  final VpsItem item;
  final Future<void> Function(VpsItem, String) onAction;

  const _VpsTile({required this.item, required this.onAction});

  void _openActionSheet(
    BuildContext context,
    VpsItem item,
    Future<void> Function(VpsItem, String) onAction,
  ) {
    showModalBottomSheet<void>(
      context: context,
      showDragHandle: true,
      isScrollControlled: true,
      builder: (context) => SafeArea(
        child: DraggableScrollableSheet(
          expand: false,
          initialChildSize: 0.7,
          minChildSize: 0.35,
          maxChildSize: 0.95,
          builder: (context, controller) => ListView(
            controller: controller,
            padding: const EdgeInsets.all(16),
            children: [
              Text('实例 #${item.id}', style: Theme.of(context).textTheme.titleMedium),
              const SizedBox(height: 4),
              Text('用户 ${item.username ?? item.userId} · ${_statusLabel(item.status)}',
                  style: Theme.of(context).textTheme.bodySmall?.copyWith(color: Colors.black54)),
              const SizedBox(height: 12),
              _ActionGroup(
                title: '常用操作',
                children: [
                  _ActionTile(
                    icon: Icons.login,
                    title: '登录面板',
                    subtitle: '打开实例控制面板',
                    onTap: () async {
                      Navigator.pop(context);
                      await _openPanel(context, item);
                    },
                  ),
                  _ActionTile(
                    icon: Icons.refresh,
                    title: '刷新实例',
                    subtitle: '同步最新状态',
                    onTap: () async {
                      Navigator.pop(context);
                      await onAction(item, 'refresh');
                    },
                  ),
                  _ActionTile(
                    icon: Icons.schedule,
                    title: '修改到期',
                    subtitle: '调整实例到期时间',
                    onTap: () async {
                      Navigator.pop(context);
                      await _openExpire(context, item);
                    },
                  ),
                ],
              ),
              const SizedBox(height: 12),
              _ActionGroup(
                title: '配置与状态',
                children: [
                  _ActionTile(
                    icon: Icons.tune,
                    title: '编辑配置',
                    subtitle: '调整套餐/规格等字段',
                    onTap: () async {
                      Navigator.pop(context);
                      await _openEdit(context, item);
                    },
                  ),
                  _ActionTile(
                    icon: Icons.swap_horiz,
                    title: '改配',
                    subtitle: '调整 CPU/内存/磁盘/带宽',
                    onTap: () async {
                      Navigator.pop(context);
                      await _openResize(context, item);
                    },
                  ),
                  _ActionTile(
                    icon: Icons.rule_folder,
                    title: '设置状态',
                    subtitle: 'normal/abuse/fraud/locked',
                    onTap: () async {
                      Navigator.pop(context);
                      await _openStatus(context, item);
                    },
                  ),
                ],
              ),
              const SizedBox(height: 12),
              _ActionGroup(
                title: '风险操作',
                children: [
                  _ActionTile(
                    icon: Icons.warning_amber,
                    title: '紧急续费',
                    subtitle: '立即创建紧急续费订单',
                    color: Colors.orange,
                    onTap: () async {
                      final ok = await _confirm(context, '紧急续费', '确认对该实例执行紧急续费？');
                      if (ok) {
                        Navigator.pop(context);
                        await onAction(item, 'emergency');
                      }
                    },
                  ),
                  _ActionTile(
                    icon: Icons.lock,
                    title: '锁定',
                    subtitle: '禁止实例操作',
                    color: Colors.orange,
                    onTap: () async {
                      final ok = await _confirm(context, '锁定实例', '确认锁定该实例？');
                      if (ok) {
                        Navigator.pop(context);
                        await onAction(item, 'lock');
                      }
                    },
                  ),
                  _ActionTile(
                    icon: Icons.lock_open,
                    title: '解锁',
                    subtitle: '恢复实例操作',
                    color: Colors.orange,
                    onTap: () async {
                      final ok = await _confirm(context, '解锁实例', '确认解锁该实例？');
                      if (ok) {
                        Navigator.pop(context);
                        await onAction(item, 'unlock');
                      }
                    },
                  ),
                  _ActionTile(
                    icon: Icons.delete_forever,
                    title: '删除实例',
                    subtitle: '不可恢复',
                    color: Colors.red,
                    onTap: () async {
                      Navigator.pop(context);
                      await _openDelete(context, item);
                    },
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final statusChip = _StatusChip(
      label: _statusLabel(item.status),
      color: _statusColor(item.status),
    );
    final adminChip = _StatusChip(
      label: _adminStatusLabel(item.adminStatus),
      color: _adminStatusColor(item.adminStatus),
    );
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    return Material(
      color: Colors.transparent,
      child: InkWell(
        borderRadius: BorderRadius.circular(16),
        onTap: () => _openActionSheet(context, item, onAction),
        child: Container(
          margin: const EdgeInsets.only(bottom: 12),
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: colorScheme.surface,
            borderRadius: BorderRadius.circular(16),
            border: Border.all(color: colorScheme.outlineVariant.withOpacity(0.5)),
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
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Container(
                    padding: const EdgeInsets.all(8),
                    decoration: BoxDecoration(
                      color: colorScheme.primary.withOpacity(0.1),
                      borderRadius: BorderRadius.circular(10),
                    ),
                    child: Icon(Icons.dns, color: colorScheme.primary),
                  ),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          '实例 #${item.id}',
                          style: theme.textTheme.titleSmall?.copyWith(
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                        const SizedBox(height: 2),
                        Text(
                          '用户 ${item.username ?? item.userId}',
                          style: theme.textTheme.bodySmall?.copyWith(
                            color: colorScheme.onSurfaceVariant,
                          ),
                        ),
                      ],
                    ),
                  ),
                  IconButton(
                    icon: const Icon(Icons.more_horiz),
                    onPressed: () => _openActionSheet(context, item, onAction),
                  ),
                ],
              ),
              const SizedBox(height: 8),
              Wrap(
                spacing: 8,
                runSpacing: 6,
                children: [statusChip, adminChip],
              ),
              const SizedBox(height: 8),
              Row(
                children: [
                  Icon(Icons.place_outlined,
                      size: 16, color: colorScheme.onSurfaceVariant),
                  const SizedBox(width: 6),
                  Expanded(
                    child: Text(
                      '${item.region.isEmpty ? '-' : item.region} · ${item.packageName.isEmpty ? '-' : item.packageName}',
                      style: theme.textTheme.bodySmall,
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 4),
              Row(
                children: [
                  Icon(Icons.tune_rounded,
                      size: 16, color: colorScheme.onSurfaceVariant),
                  const SizedBox(width: 6),
                  Expanded(
                    child: Text(
                      '${item.cpu}C / ${item.memoryGb}G / ${item.diskGb}G / ${item.bandwidthMbps}Mbps',
                      style: theme.textTheme.bodySmall,
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 4),
              Row(
                children: [
                  Icon(Icons.payments_outlined,
                      size: 16, color: colorScheme.onSurfaceVariant),
                  const SizedBox(width: 6),
                  Expanded(
                    child: Text(
                      '月费 ￥${item.monthlyPrice.toStringAsFixed(2)} · 到期 ${_formatLocal(item.expireAt)}',
                      style: theme.textTheme.bodySmall,
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _openStatus(BuildContext context, VpsItem item) async {
    String status = item.adminStatus.isNotEmpty ? item.adminStatus : 'normal';
    final reasonCtl = TextEditingController();
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('设置状态'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            DropdownButtonFormField<String>(
              value: status,
              items: const [
                DropdownMenuItem(value: 'normal', child: Text('normal')),
                DropdownMenuItem(value: 'abuse', child: Text('abuse')),
                DropdownMenuItem(value: 'fraud', child: Text('fraud')),
                DropdownMenuItem(value: 'locked', child: Text('locked')),
              ],
              onChanged: (value) => status = value ?? status,
              decoration: const InputDecoration(labelText: '状态'),
            ),
            const SizedBox(height: 8),
            TextField(
              controller: reasonCtl,
              decoration: const InputDecoration(labelText: '原因'),
              maxLines: 3,
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(context, true), child: const Text('保存')),
        ],
      ),
    );
    if (ok == true) {
      item.statusPayload = {
        'status': status,
        'reason': reasonCtl.text.trim(),
      };
      await onAction(item, 'status');
    }
  }

  Future<void> _openResize(BuildContext context, VpsItem item) async {
    final cpu = TextEditingController(text: '0');
    final mem = TextEditingController(text: '0');
    final disk = TextEditingController(text: '0');
    final bw = TextEditingController(text: '0');
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('改配'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(controller: cpu, decoration: const InputDecoration(labelText: 'CPU'), keyboardType: TextInputType.number),
            TextField(controller: mem, decoration: const InputDecoration(labelText: '内存GB'), keyboardType: TextInputType.number),
            TextField(controller: disk, decoration: const InputDecoration(labelText: '磁盘GB'), keyboardType: TextInputType.number),
            TextField(controller: bw, decoration: const InputDecoration(labelText: '带宽Mbps'), keyboardType: TextInputType.number),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(context, true), child: const Text('保存')),
        ],
      ),
    );
    if (ok == true) {
      item.resizePayload = {
        'cpu': int.tryParse(cpu.text.trim()) ?? 0,
        'memory_gb': int.tryParse(mem.text.trim()) ?? 0,
        'disk_gb': int.tryParse(disk.text.trim()) ?? 0,
        'bandwidth_mbps': int.tryParse(bw.text.trim()) ?? 0,
      };
      await onAction(item, 'resize');
    }
  }

  Future<void> _openExpire(BuildContext context, VpsItem item) async {
    DateTime? selected;
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('修改到期时间'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            FilledButton.icon(
              onPressed: () async {
                final date = await showDatePicker(
                  context: context,
                  initialDate: DateTime.now(),
                  firstDate: DateTime(2000),
                  lastDate: DateTime(2100),
                );
                if (date == null) return;
                final time = await showTimePicker(
                  context: context,
                  initialTime: TimeOfDay.fromDateTime(DateTime.now()),
                );
                if (time == null) return;
                selected = DateTime(date.year, date.month, date.day, time.hour, time.minute);
              },
              icon: const Icon(Icons.calendar_today),
              label: const Text('选择日期时间'),
            ),
            const SizedBox(height: 8),
            Text('当前：${_formatLocal(item.expireAt)}'),
            Text('新值：${selected == null ? '-' : _formatLocal(selected!.toIso8601String())}'),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(context, true), child: const Text('保存')),
        ],
      ),
    );
    if (ok == true) {
      if (selected == null) return;
      item.expirePayload = {'expire_at': selected!.toUtc().toIso8601String()};
      await onAction(item, 'expire');
    }
  }

  Future<void> _openEdit(BuildContext context, VpsItem item) async {
    final monthly = TextEditingController(text: item.monthlyPrice.toString());
    final pkgName = TextEditingController(text: item.packageName);
    final cpu = TextEditingController(text: item.cpu.toString());
    final mem = TextEditingController(text: item.memoryGb.toString());
    final disk = TextEditingController(text: item.diskGb.toString());
    final bw = TextEditingController(text: item.bandwidthMbps.toString());
    final port = TextEditingController(text: item.portNum.toString());
    String status = item.status.isNotEmpty ? item.status : 'running';
    String adminStatus = item.adminStatus.isNotEmpty ? item.adminStatus : 'normal';
    final systemId = TextEditingController(text: item.systemId.toString());
    final packageId = TextEditingController(text: item.packageId.toString());
    String syncMode = 'local';

    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('编辑 VPS'),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              DropdownButtonFormField<String>(
                value: syncMode,
                items: const [
                  DropdownMenuItem(value: 'local', child: Text('只修改本地')),
                  DropdownMenuItem(value: 'automation', child: Text('同步到自动化')),
                ],
                onChanged: (value) => syncMode = value ?? syncMode,
                decoration: const InputDecoration(labelText: '同步模式'),
              ),
              const SizedBox(height: 8),
              TextField(controller: packageId, decoration: const InputDecoration(labelText: '套餐 ID'), keyboardType: TextInputType.number),
              TextField(controller: monthly, decoration: const InputDecoration(labelText: '月费'), keyboardType: TextInputType.number),
              TextField(controller: pkgName, decoration: const InputDecoration(labelText: '套餐名')),
              TextField(controller: cpu, decoration: const InputDecoration(labelText: 'CPU'), keyboardType: TextInputType.number),
              TextField(controller: mem, decoration: const InputDecoration(labelText: '内存GB'), keyboardType: TextInputType.number),
              TextField(controller: disk, decoration: const InputDecoration(labelText: '磁盘GB'), keyboardType: TextInputType.number),
              TextField(controller: bw, decoration: const InputDecoration(labelText: '带宽Mbps'), keyboardType: TextInputType.number),
              TextField(controller: port, decoration: const InputDecoration(labelText: '端口数'), keyboardType: TextInputType.number),
              DropdownButtonFormField<String>(
                value: status,
                items: const [
                  DropdownMenuItem(value: 'running', child: Text('运行中')),
                  DropdownMenuItem(value: 'stopped', child: Text('已关机')),
                  DropdownMenuItem(value: 'provisioning', child: Text('开通中')),
                ],
                onChanged: (value) => status = value ?? status,
                decoration: const InputDecoration(labelText: '状态'),
              ),
              DropdownButtonFormField<String>(
                value: adminStatus,
                items: const [
                  DropdownMenuItem(value: 'normal', child: Text('normal')),
                  DropdownMenuItem(value: 'abuse', child: Text('abuse')),
                  DropdownMenuItem(value: 'fraud', child: Text('fraud')),
                  DropdownMenuItem(value: 'locked', child: Text('locked')),
                ],
                onChanged: (value) => adminStatus = value ?? adminStatus,
                decoration: const InputDecoration(labelText: '管理状态'),
              ),
              TextField(controller: systemId, decoration: const InputDecoration(labelText: '系统镜像ID'), keyboardType: TextInputType.number),
            ],
          ),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(context, true), child: const Text('保存')),
        ],
      ),
    );
    if (ok == true) {
      item.editPayload = {
        'sync_mode': syncMode,
        'package_id': int.tryParse(packageId.text.trim()) ?? 0,
        'monthly_price': double.tryParse(monthly.text.trim()) ?? 0,
        'package_name': pkgName.text.trim(),
        'cpu': int.tryParse(cpu.text.trim()) ?? 0,
        'memory_gb': int.tryParse(mem.text.trim()) ?? 0,
        'disk_gb': int.tryParse(disk.text.trim()) ?? 0,
        'bandwidth_mbps': int.tryParse(bw.text.trim()) ?? 0,
        'port_num': int.tryParse(port.text.trim()) ?? 0,
        'status': status,
        'admin_status': adminStatus,
        'system_id': int.tryParse(systemId.text.trim()) ?? 0,
      };
      await onAction(item, 'edit');
    }
  }

  Future<void> _openDelete(BuildContext context, VpsItem item) async {
    final reason = TextEditingController();
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('删除实例'),
        content: TextField(controller: reason, decoration: const InputDecoration(labelText: '删除原因')),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(context, true), child: const Text('删除')),
        ],
      ),
    );
    if (ok == true) {
      item.deleteReason = reason.text.trim();
      await onAction(item, 'delete');
    }
  }

  Future<void> _openPanel(BuildContext context, VpsItem item) async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    final session = context.read<AppState>().session;
    if (item.userId <= 0) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('用户信息缺失')));
      }
      return;
    }
    try {
      Future<bool> launchPanelUrl(String raw) async {
        if (raw.trim().isEmpty) return false;
        Uri? uri = Uri.tryParse(raw);
        if (uri == null) return false;
        if (!uri.hasScheme) {
          final base = Uri.tryParse(client.baseUrl);
          if (base == null) return false;
          uri = base.resolveUri(uri);
        }
        final ok = await launchUrl(uri, mode: LaunchMode.externalApplication);
        if (!ok && context.mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('无法打开面板链接，请检查系统浏览器设置')),
          );
        }
        return ok;
      }

      if (item.userRole == 'admin') {
        if (item.panelUrlCache.isEmpty) {
          final detail = await client.getJson('/admin/api/v1/vps/${item.id}');
          final data = detail['vps'] is Map<String, dynamic>
              ? detail['vps'] as Map<String, dynamic>
              : (detail['data'] is Map<String, dynamic> ? detail['data'] as Map<String, dynamic> : detail);
          item.panelUrlCache = data['panel_url_cache']?.toString() ?? '';
        }
        if (item.panelUrlCache.isNotEmpty) {
          final ok = await launchPanelUrl(item.panelUrlCache);
          if (ok) return;
        }

        final adminToken = session?.token ?? '';
        if (adminToken.isEmpty) {
          if (context.mounted) {
            ScaffoldMessenger.of(context).showSnackBar(
              const SnackBar(content: Text('面板地址为空，且未获取到管理员登录令牌')),
            );
          }
          return;
        }
        final normalizedBase = client.baseUrl.endsWith('/')
            ? client.baseUrl.substring(0, client.baseUrl.length - 1)
            : client.baseUrl;
        final adminPanelUri = Uri.parse('$normalizedBase/api/v1/vps/${item.id}/panel')
            .replace(queryParameters: {'token': adminToken});
        await launchPanelUrl(adminPanelUri.toString());
        return;
      }

      final resp = await client.postJson('/admin/api/v1/users/${item.userId}/impersonate');
      final token = resp['access_token']?.toString() ?? '';
      if (token.isEmpty) {
        if (context.mounted) {
          ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('未获取到用户令牌')));
        }
        return;
      }
      final normalizedBase = client.baseUrl.endsWith('/')
          ? client.baseUrl.substring(0, client.baseUrl.length - 1)
          : client.baseUrl;
      final uri = Uri.parse('$normalizedBase/api/v1/vps/${item.id}/panel')
          .replace(queryParameters: {'token': token});
      await launchPanelUrl(uri.toString());
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('打开面板失败：$e')));
      }
    }
  }
}

class VpsItem {
  final int id;
  final int userId;
  String? username;
  String? userRole;
  final int packageId;
  final String region;
  final String packageName;
  final double monthlyPrice;
  final String status;
  final String adminStatus;
  final String expireAt;
  final int cpu;
  final int memoryGb;
  final int diskGb;
  final int bandwidthMbps;
  final int portNum;
  final int systemId;
  final String automationInstanceId;
  final int automationState;
  String panelUrlCache;

  Map<String, dynamic>? resizePayload;
  Map<String, dynamic>? statusPayload;
  Map<String, dynamic>? expirePayload;
  Map<String, dynamic>? editPayload;
  String deleteReason = '';

  VpsItem({
    required this.id,
    required this.userId,
    required this.region,
    required this.packageName,
    required this.monthlyPrice,
    required this.status,
    required this.adminStatus,
    required this.expireAt,
    required this.cpu,
    required this.memoryGb,
    required this.diskGb,
    required this.bandwidthMbps,
    required this.portNum,
    required this.systemId,
    required this.automationInstanceId,
    required this.automationState,
    required this.packageId,
    this.username,
    this.userRole,
    this.panelUrlCache = '',
  });

  factory VpsItem.fromJson(Map<String, dynamic> json) {
    final rawAutomation = json['automation_state'] as int? ?? 0;
    final resolvedStatus = _statusFromAutomation(rawAutomation, json['status']?.toString() ?? '');
    return VpsItem(
      id: json['id'] as int? ?? 0,
      userId: json['user_id'] as int? ?? 0,
      region: json['region'] as String? ?? '',
      packageName: json['package_name'] as String? ?? '',
      monthlyPrice: (json['monthly_price'] as num?)?.toDouble() ?? 0,
      status: resolvedStatus,
      adminStatus: json['admin_status'] as String? ?? '',
      expireAt: json['expire_at']?.toString() ?? '',
      cpu: json['cpu'] as int? ?? 0,
      memoryGb: json['memory_gb'] as int? ?? 0,
      diskGb: json['disk_gb'] as int? ?? 0,
      bandwidthMbps: json['bandwidth_mbps'] as int? ?? 0,
      portNum: json['port_num'] as int? ?? 0,
      systemId: json['system_id'] as int? ?? 0,
      automationInstanceId: json['automation_instance_id'] as String? ?? '',
      automationState: rawAutomation,
      packageId: json['package_id'] as int? ?? 0,
      panelUrlCache: json['panel_url_cache']?.toString() ?? '',
    );
  }
}

String _statusFromAutomation(int state, String fallback) {
  switch (state) {
    case 1:
    case 13:
      return 'provisioning';
    case 2:
      return 'running';
    case 3:
      return 'stopped';
    case 4:
      return 'reinstalling';
    case 5:
      return 'reinstall_failed';
    case 10:
      return 'locked';
    case 11:
      return 'failed';
    case 12:
      return 'deleting';
    default:
      return fallback;
  }
}

class _StatusTabs extends StatelessWidget {
  final String value;
  final ValueChanged<String> onChanged;

  const _StatusTabs({required this.value, required this.onChanged});

  @override
  Widget build(BuildContext context) {
    final tabs = const [
      {'label': 'normal', 'value': 'normal'},
      {'label': 'abuse', 'value': 'abuse'},
      {'label': 'fraud', 'value': 'fraud'},
      {'label': 'locked', 'value': 'locked'},
    ];
    return SizedBox(
      height: 40,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        itemCount: tabs.length + 1,
        separatorBuilder: (_, __) => const SizedBox(width: 8),
        itemBuilder: (context, index) {
          if (index == 0) {
            return _StatusFilterChip(
              label: '全部',
              selected: value.isEmpty,
              onTap: () => onChanged(''),
            );
          }
          final t = tabs[index - 1];
          return _StatusFilterChip(
            label: t['label']!,
            selected: value == t['value'],
            onTap: () => onChanged(t['value']!),
          );
        },
      ),
    );
  }
}

class _StatusFilterChip extends StatelessWidget {
  final String label;
  final bool selected;
  final VoidCallback onTap;

  const _StatusFilterChip({
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
        borderRadius: BorderRadius.circular(12),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          decoration: BoxDecoration(
            color: bgColor,
            borderRadius: BorderRadius.circular(12),
            border: Border.all(color: borderColor),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              if (selected) ...[
                Icon(Icons.check_rounded, size: 16, color: textColor),
                const SizedBox(width: 6),
              ],
              Text(
                label,
                style: TextStyle(
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

class _StatusChip extends StatelessWidget {
  final String label;
  final Color color;

  const _StatusChip({required this.label, required this.color});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: color.withOpacity(0.12),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        label,
        style: TextStyle(color: color, fontSize: 12, fontWeight: FontWeight.w600),
      ),
    );
  }
}

String _statusLabel(String status) {
  switch (status) {
    case 'running':
      return '运行中';
    case 'stopped':
      return '已停止';
    case 'provisioning':
      return '开通中';
    case 'reinstalling':
      return '重装中';
    case 'reinstall_failed':
      return '重装失败';
    case 'locked':
      return '已锁定';
    case 'failed':
      return '失败';
    case 'deleting':
      return '删除中';
    default:
      return status.isEmpty ? '未知' : status;
  }
}

Color _statusColor(String status) {
  switch (status) {
    case 'running':
      return const Color(0xFF00A68C);
    case 'stopped':
      return const Color(0xFF546E7A);
    case 'provisioning':
    case 'reinstalling':
    case 'deleting':
      return const Color(0xFF1E88E5);
    case 'reinstall_failed':
    case 'failed':
      return const Color(0xFFD32F2F);
    case 'locked':
      return const Color(0xFFEF6C00);
    default:
      return Colors.black54;
  }
}

String _adminStatusLabel(String status) {
  switch (status) {
    case 'normal':
      return '正常';
    case 'abuse':
      return '滥用';
    case 'fraud':
      return '欺诈';
    case 'locked':
      return '锁定';
    default:
      return status.isEmpty ? '未知' : status;
  }
}

Color _adminStatusColor(String status) {
  switch (status) {
    case 'normal':
      return const Color(0xFF00A68C);
    case 'abuse':
      return const Color(0xFFEF6C00);
    case 'fraud':
      return const Color(0xFFD32F2F);
    case 'locked':
      return const Color(0xFF546E7A);
    default:
      return Colors.black54;
  }
}

class _ActionGroup extends StatelessWidget {
  final String title;
  final List<Widget> children;

  const _ActionGroup({required this.title, required this.children});

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(title, style: Theme.of(context).textTheme.bodySmall?.copyWith(color: Colors.black54)),
        const SizedBox(height: 6),
        ...children,
      ],
    );
  }
}

class _ActionTile extends StatelessWidget {
  final IconData icon;
  final String title;
  final String subtitle;
  final VoidCallback onTap;
  final Color? color;

  const _ActionTile({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.onTap,
    this.color,
  });

  @override
  Widget build(BuildContext context) {
    final tint = color ?? Theme.of(context).colorScheme.primary;
    return ListTile(
      contentPadding: EdgeInsets.zero,
      leading: CircleAvatar(
        backgroundColor: tint.withOpacity(0.12),
        child: Icon(icon, color: tint),
      ),
      title: Text(title),
      subtitle: Text(subtitle),
      onTap: onTap,
    );
  }
}

Future<bool> _confirm(BuildContext context, String title, String message) async {
  final ok = await showDialog<bool>(
    context: context,
    builder: (context) => AlertDialog(
      title: Text(title),
      content: Text(message),
      actions: [
        TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('取消')),
        FilledButton(onPressed: () => Navigator.pop(context, true), child: const Text('确认')),
      ],
    ),
  );
  return ok == true;
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
            const SizedBox(width: 8),
            OutlinedButton(onPressed: onNext, child: const Text('下一页')),
          ],
        )
      ],
    );
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
