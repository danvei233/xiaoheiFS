import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app_state.dart';
import '../services/api_client.dart';

class VPSDetailScreen extends StatefulWidget {
  final int vpsId;

  const VPSDetailScreen({super.key, required this.vpsId});

  @override
  State<VPSDetailScreen> createState() => _VPSDetailScreenState();
}

class _VPSDetailScreenState extends State<VPSDetailScreen> {
  Future<Map<String, dynamic>>? _future;
  bool _busy = false;

  /// 请求代次守卫：每次重新加载（客户端/会话变化）时递增。只有「本次代次」
  /// 的响应允许成为 `_future` 的结果——旧账号/旧会话的慢响应即使先完成也会
  /// 被丢弃，防止 logout/login 或快速切换后旧数据写入新会话。
  int _generation = 0;

  /// 标记代次失效的哨兵异常，FutureBuilder 会把过期响应当作错误忽略。
  static const _stale = 'stale_response_generation';

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final client = context.read<AppState>().apiClient;
    if (client != null) {
      final gen = ++_generation;
      final future = client.getJson('/admin/api/v1/vps/${widget.vpsId}');
      _future = future.then((value) {
        if (gen != _generation) throw _stale;
        return value;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<Map<String, dynamic>>(
      future: _future,
      builder: (context, snapshot) {
        if (snapshot.connectionState == ConnectionState.waiting) {
          return const Scaffold(
            body: Center(child: CircularProgressIndicator()),
          );
        }
        if (snapshot.hasError) {
          return Scaffold(
            appBar: AppBar(title: const Text('服务器详情')),
            body: Center(child: Text('加载失败：$snapshot')),
          );
        }
        final data = snapshot.data ?? {};
        return Scaffold(
          appBar: AppBar(title: const Text('服务器详情')),
          body: ListView(
            padding: const EdgeInsets.all(16),
            children: [
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(16),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        data['name'] ?? '',
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                      const SizedBox(height: 8),
                      Text('区域：${data['region'] ?? ''}'),
                      Text('状态：${data['status'] ?? ''}'),
                      Text('管理员状态：${data['admin_status'] ?? ''}'),
                      Text(
                        '配置：${data['cpu'] ?? ''}核 · ${data['memory_gb'] ?? ''}G · ${data['disk_gb'] ?? ''}G',
                      ),
                      Text('带宽：${data['bandwidth_mbps'] ?? ''} Mbps'),
                      Text('到期：${data['expire_at'] ?? ''}'),
                    ],
                  ),
                ),
              ),
              const SizedBox(height: 12),
              Wrap(
                spacing: 8,
                children: [
                  FilledButton(
                    onPressed: _busy ? null : () => _postAction('refresh'),
                    child: const Text('刷新'),
                  ),
                  OutlinedButton(
                    onPressed: _busy ? null : () => _postAction('lock'),
                    child: const Text('锁定'),
                  ),
                  OutlinedButton(
                    onPressed: _busy ? null : () => _postAction('unlock'),
                    child: const Text('解锁'),
                  ),
                ],
              ),
            ],
          ),
        );
      },
    );
  }

  Future<void> _postAction(String action) async {
    if (_busy) return;
    setState(() => _busy = true);
    try {
      final client = context.read<AppState>().apiClient;
      if (client == null) return;
      await client.postJson('/admin/api/v1/vps/${widget.vpsId}/$action');
      if (mounted) {
        final gen = ++_generation;
        final reload = client.getJson('/admin/api/v1/vps/${widget.vpsId}');
        setState(() {
          _future = reload.then((value) {
            if (gen != _generation) throw _stale;
            return value;
          });
        });
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('操作失败：$e')));
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }
}
