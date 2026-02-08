import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../core/constants/app_colors.dart';
import '../../../core/constants/app_strings.dart';
import '../../../core/utils/date_formatter.dart';
import '../../providers/vps_provider.dart';
import '../../providers/refresh_provider.dart';
import '../../widgets/common/empty_state.dart';
import '../../widgets/common/pagination_bar.dart';
import '../../widgets/common/status_tag.dart';

/// VPS 列表页面
class VpsListPage extends ConsumerStatefulWidget {
  const VpsListPage({super.key});

  @override
  ConsumerState<VpsListPage> createState() => _VpsListPageState();
}

class _VpsListPageState extends ConsumerState<VpsListPage> {
  ProviderSubscription<RefreshEvent?>? _refreshSub;
  int _page = 1;
  int _pageSize = 10;
  static const double _paginationBarHeight = 72;
  static const double _paginationBarBottomPadding = 20;
  static const double _paginationFabOffset =
      _paginationBarHeight + _paginationBarBottomPadding + 12; // height + 12px

  @override
  void initState() {
    super.initState();
    Future.microtask(() => ref.read(vpsListProvider.notifier).fetch());
    _refreshSub = ref.listenManual<RefreshEvent?>(pageRefreshProvider, (_, next) {
      if (next?.route == '/console/vps') {
        ref.read(vpsListProvider.notifier).refresh();
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
    final vpsListState = ref.watch(vpsListProvider);

    return Scaffold(
      body: vpsListState.loading
          ? const Center(child: CircularProgressIndicator())
          : vpsListState.error != null
              ? _buildError(context, ref, vpsListState.error!)
              : vpsListState.items.isEmpty
                  ? const EmptyState(
                      message: AppStrings.noVps,
                      icon: Icons.cloud_off,
                    )
                  : _buildVpsList(context, ref, vpsListState.items),
      floatingActionButton: Padding(
        padding: const EdgeInsets.only(bottom: _paginationFabOffset, right: 8),
        child: FloatingActionButton(
          onPressed: () => context.go('/console/buy'),
          child: const Icon(Icons.add_shopping_cart),
        ),
      ),
      floatingActionButtonLocation: FloatingActionButtonLocation.endFloat,
      floatingActionButtonAnimator: FloatingActionButtonAnimator.scaling,
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
            onPressed: () => ref.read(vpsListProvider.notifier).refresh(),
            child: const Text(AppStrings.retry),
          ),
        ],
      ),
    );
  }

  Widget _buildVpsList(BuildContext context, WidgetRef ref, List<Map<String, dynamic>> vpsList) {
    final total = vpsList.length;
    final totalPages = (total / _pageSize).ceil().clamp(1, 9999);
    if (_page > totalPages) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) {
          setState(() => _page = totalPages);
        }
      });
    }
    final start = (_page - 1) * _pageSize;
    final end = (start + _pageSize).clamp(0, total);
    final pageItems = vpsList.sublist(
      start.clamp(0, total),
      end,
    );

    return Column(
      children: [
        Expanded(
          child: RefreshIndicator(
            onRefresh: () => ref.read(vpsListProvider.notifier).refresh(),
            child: ListView.builder(
              padding: const EdgeInsets.all(24),
              itemCount: pageItems.length,
              itemBuilder: (context, index) {
                final vps = pageItems[index];
          final id = vps['id'] ?? vps['ID'];
          final name = vps['name'] ?? vps['Name'] ?? 'VPS #$id';
          final status = _resolveStatus(vps);
          final ip = _resolveIp(vps);
          final specText = _normalizeSpec(vps);
          final regionLine = _resolveRegionLine(vps);
          final expireAt = vps['expire_at'] ?? vps['ExpireAt'];
          final destroyInDays = vps['destroy_in_days'] ?? vps['DestroyInDays'];
                return Card(
                  margin: const EdgeInsets.only(bottom: 16),
                  child: ListTile(
                    contentPadding: const EdgeInsets.all(16),
                    title: Row(
                      children: [
                        Expanded(
                          child: Text(
                            '$name',
                            style: const TextStyle(
                              fontSize: 16,
                              fontWeight: FontWeight.bold,
                            ),
                          ),
                        ),
                        StatusTag.vps('$status'),
                      ],
                    ),
                    subtitle: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const SizedBox(height: 8),
                        Text('IP: $ip'),
                        if (regionLine.isNotEmpty) Text('地区: $regionLine'),
                        if (specText.isNotEmpty) Text(specText),
                        const SizedBox(height: 4),
                        Text(
                          '到期时间: ${DateFormatter.formatIso(expireAt)}',
                          style: TextStyle(
                            fontSize: 12,
                            color: AppColors.gray500,
                          ),
                        ),
                        if (destroyInDays != null)
                          Padding(
                            padding: const EdgeInsets.only(top: 2),
                            child: Text(
                              '将在 $destroyInDays 天后自动删除',
                              style: TextStyle(
                                fontSize: 12,
                                color: AppColors.warning,
                              ),
                            ),
                          ),
                      ],
                    ),
                    trailing: const Icon(Icons.arrow_forward_ios, size: 16),
                    onTap: () {
                      if (id != null) {
                        context.go('/console/vps/$id');
                      }
                    },
                  ),
                );
              },
            ),
          ),
        ),
        Padding(
          padding: const EdgeInsets.fromLTRB(24, 0, 24, 20),
          child: PaginationBar(
            currentPage: _page,
            pageSize: _pageSize,
            totalItems: total,
            onPageChanged: (page) {
              setState(() => _page = page);
            },
            onPageSizeChanged: (size) {
              setState(() {
                _pageSize = size;
                _page = 1;
              });
            },
          ),
        ),
      ],
    );
  }

  Map<String, dynamic> _parseJson(dynamic input) {
    if (input == null) return {};
    if (input is Map<String, dynamic>) return input;
    if (input is String) {
      try {
        return Map<String, dynamic>.from(jsonDecode(input));
      } catch (_) {
        return {};
      }
    }
    return {};
  }

  String _resolveIp(Map<String, dynamic> vps) {
    final access = _parseJson(
      vps['access_info'] ??
          vps['AccessInfo'] ??
          vps['access_info_json'] ??
          vps['AccessInfoJSON'],
    );
    final ip = access['remote_ip'] ??
        access['ip'] ??
        access['public_ip'] ??
        access['ipv4'] ??
        access['Ip'] ??
        vps['ip'] ??
        vps['Ip'];
    return ip == null || ip.toString().isEmpty ? '-' : ip.toString();
  }

  String _resolveRegionLine(Map<String, dynamic> vps) {
    final access = _parseJson(
      vps['access_info'] ??
          vps['AccessInfo'] ??
          vps['access_info_json'] ??
          vps['AccessInfoJSON'],
    );
    final region = vps['region'] ?? vps['Region'] ?? '-';
    final line = vps['line'] ??
        vps['Line'] ??
        vps['line_name'] ??
        vps['LineName'] ??
        access['line'] ??
        '';
    return line == null || line.toString().isEmpty ? '$region' : '$region/$line';
  }

  String _normalizeSpec(Map<String, dynamic> vps) {
    dynamic spec = vps['spec'] ?? vps['Spec'] ?? vps['spec_json'] ?? vps['SpecJSON'];
    if (spec == null) spec = vps;
    if (spec is String) {
      try {
        spec = jsonDecode(spec);
      } catch (_) {
        return spec;
      }
    }
    if (spec is! Map) return '';
    final cpu = spec['cpu'] ?? spec['cores'] ?? spec['CPU'] ?? spec['Cores'] ?? vps['cpu'] ?? 0;
    final mem = spec['memory_gb'] ??
        spec['mem_gb'] ??
        spec['MemoryGB'] ??
        vps['memory_gb'] ??
        vps['MemoryGB'] ??
        0;
    final disk = spec['disk_gb'] ?? spec['DiskGB'] ?? vps['disk_gb'] ?? vps['DiskGB'] ?? 0;
    final bw = spec['bandwidth_mbps'] ??
        spec['BandwidthMB'] ??
        spec['bandwidth'] ??
        vps['bandwidth_mbps'];
    final parts = [
      'CPU ${cpu}核',
      '内存 ${mem}G',
      '磁盘 ${disk}G',
      if (bw != null) '带宽 ${bw}M',
    ];
    return parts.join(' / ');
  }

  String _resolveStatus(Map<String, dynamic> vps) {
    final rawStatus = vps['status'] ?? vps['Status'];
    final automationState = vps['automation_state'] ?? vps['AutomationState'];
    if (automationState == null) return rawStatus?.toString() ?? '';
    switch (int.tryParse(automationState.toString()) ?? -1) {
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
        return rawStatus?.toString() ?? '';
    }
  }
}


