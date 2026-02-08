
import 'dart:async';
import 'dart:convert';
import 'dart:math' as math;

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:url_launcher/url_launcher.dart';
import '../../../core/constants/app_colors.dart';
import '../../../core/constants/app_strings.dart';
import '../../../core/network/api_client.dart';
import '../../../core/storage/storage_service.dart';
import '../../../core/utils/date_formatter.dart';
import '../../../core/utils/desktop_launcher.dart';
import '../../../core/utils/money_formatter.dart';
import '../../../core/utils/platform_utils.dart';
import '../../providers/catalog_provider.dart';
import '../../providers/refresh_provider.dart';
import '../../providers/site_provider.dart';
import '../../providers/vps_provider.dart';
import '../../widgets/charts/line_chart.dart';
import '../../widgets/common/empty_state.dart';
import '../../widgets/common/status_tag.dart';

class VpsDetailPage extends ConsumerStatefulWidget {
  final int id;
  const VpsDetailPage({super.key, required this.id});

  @override
  ConsumerState<VpsDetailPage> createState() => _VpsDetailPageState();
}

class _VpsDetailPageState extends ConsumerState<VpsDetailPage>
    with SingleTickerProviderStateMixin {
  late final TabController _tabController;
  ProviderSubscription<RefreshEvent?>? _refreshSub;
  Timer? _portCandidateTimer;
  List<int> _portCandidates = [];
  bool _portCandidatesLoading = false;
  bool _showOsPassword = false;
  bool _showPanelPassword = false;
  String _currentRoute = '';
  int _tabIndex = 0;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 6, vsync: this);
    _tabController.addListener(_handleTabChanged);

    Future.microtask(() {
      ref.read(vpsDetailProvider.notifier).fetch(widget.id);
      ref.read(catalogProvider.notifier).fetchCatalog();
      ref.read(siteProvider.notifier).fetchSettings();
      ref.read(vpsMonitorStateProvider(widget.id).notifier);
    });

    _refreshSub = ref.listenManual<RefreshEvent?>(pageRefreshProvider, (_, next) {
      if (next == null) return;
      if (next.route == _currentRoute) {
        _refreshCurrentPage();
      }
    });
  }

  @override
  void dispose() {
    _refreshSub?.close();
    _tabController.removeListener(_handleTabChanged);
    _tabController.dispose();
    _portCandidateTimer?.cancel();
    ref.read(vpsMonitorStateProvider(widget.id).notifier).stopPolling();
    super.dispose();
  }

  void _handleTabChanged() {
    if (_tabController.indexIsChanging) return;
    setState(() => _tabIndex = _tabController.index);
    _refreshSecurityTabIfNeeded(_tabController.index);
  }

  void _refreshSecurityTabIfNeeded(int index) {
    if (index == 2) {
      ref.invalidate(vpsFirewallProvider(widget.id));
    } else if (index == 3) {
      ref.invalidate(vpsPortsProvider(widget.id));
    } else if (index == 4) {
      ref.invalidate(vpsSnapshotsProvider(widget.id));
    } else if (index == 5) {
      ref.invalidate(vpsBackupsProvider(widget.id));
    }
  }

  Future<void> _refreshCurrentPage() async {
    await ref.read(vpsDetailProvider.notifier).fetch(widget.id);
    await ref.read(vpsMonitorStateProvider(widget.id).notifier).fetchOnce();
    _refreshSecurityTabIfNeeded(_tabController.index);
  }

  @override
  Widget build(BuildContext context) {
    _currentRoute = GoRouterState.of(context).matchedLocation;
    final detail = ref.watch(vpsDetailProvider.select((s) => s.detail));
    final loading = ref.watch(vpsDetailProvider.select((s) => s.loading));
    final error = ref.watch(vpsDetailProvider.select((s) => s.error));

    return Scaffold(
      floatingActionButton: AnimatedSwitcher(
        duration: const Duration(milliseconds: 120),
        switchInCurve: Curves.easeOut,
        switchOutCurve: Curves.easeIn,
        transitionBuilder: (child, animation) {
          return ScaleTransition(
            scale: animation,
            child: FadeTransition(opacity: animation, child: child),
          );
        },
        child: _buildFab() ?? const SizedBox.shrink(key: ValueKey('fab-none')),
      ),
      body: detail == null && loading
          ? const Center(child: CircularProgressIndicator())
          : detail == null && error != null
              ? Center(child: Text(error))
              : _buildContent(context, detail ?? {}, loading),
    );
  }

  Widget? _buildFab() {
    switch (_tabIndex) {
      case 2:
        return FloatingActionButton(
          key: const ValueKey('fab-firewall'),
          onPressed: () => _openFirewallDialog(context),
          child: const Icon(Icons.add),
        );
      case 3:
        return FloatingActionButton(
          key: const ValueKey('fab-port'),
          onPressed: () => _openPortDialog(context),
          child: const Icon(Icons.add),
        );
      case 4:
        return FloatingActionButton(
          key: const ValueKey('fab-snapshot'),
          onPressed: () async {
            await _operate(
              context,
              () => ref.read(vpsRepositoryProvider).createSnapshot(widget.id),
              '快照已创建',
            );
            ref.invalidate(vpsSnapshotsProvider(widget.id));
          },
          child: const Icon(Icons.add),
        );
      case 5:
        return FloatingActionButton(
          key: const ValueKey('fab-backup'),
          onPressed: () async {
            await _operate(
              context,
              () => ref.read(vpsRepositoryProvider).createBackup(widget.id),
              '备份已创建',
            );
            ref.invalidate(vpsBackupsProvider(widget.id));
          },
          child: const Icon(Icons.add),
        );
      default:
        return null;
    }
  }

  Widget _buildContent(BuildContext context, Map<String, dynamic> detail, bool loading) {
    final spec = _resolveSpec(detail);
    final access = _parseAccessInfo(detail);
    final resolvedStatus = _resolveStatus(detail);
    final siteSettings = ref.watch(siteProvider.select((s) => s.settings));
    final emergencyEligible = _isEmergencyRenewEligible(detail, siteSettings);
    final isExpiringSoon = _isExpiringSoon(detail['expire_at']?.toString());

    return Column(
      children: [
        _buildHeader(context, detail, spec, resolvedStatus, emergencyEligible, loading),
        TabBar(
          controller: _tabController,
          isScrollable: true,
          tabs: const [
            Tab(text: '总览'),
            Tab(text: '实时监控'),
            Tab(text: '防火墙'),
            Tab(text: '端口映射'),
            Tab(text: '快照'),
            Tab(text: '备份'),
          ],
        ),
        Expanded(
          child: TabBarView(
            controller: _tabController,
            children: [
              _buildOverviewTab(context, detail, spec, access, resolvedStatus, emergencyEligible, isExpiringSoon),
              _buildMonitorTab(context, detail, resolvedStatus, isExpiringSoon),
              _buildFirewall(context),
              _buildPorts(context),
              _buildSnapshots(context),
              _buildBackups(context),
            ],
          ),
        ),
      ],
    );
  }

  Widget _buildHeader(
    BuildContext context,
    Map<String, dynamic> detail,
    _SpecInfo spec,
    String resolvedStatus,
    bool emergencyRenewEligible,
    bool loading,
  ) {
    final name = detail['name'] ?? detail['Name'] ?? '加载中...';
    final onSurface = Theme.of(context).colorScheme.onSurface;

    final surface = Theme.of(context).colorScheme.surface;
    final borderColor = Theme.of(context).colorScheme.outlineVariant.withOpacity(0.5);

    final primary = AppColors.primary;
    final actionTextStyle = const TextStyle(fontWeight: FontWeight.w600);
    final pillPadding = const EdgeInsets.symmetric(horizontal: 14, vertical: 10);
    final pillShape = const StadiumBorder();

    final primaryButtonStyle = ElevatedButton.styleFrom(
      backgroundColor: primary,
      foregroundColor: Colors.white,
      elevation: 0,
      shape: pillShape,
      padding: pillPadding,
      textStyle: actionTextStyle,
    );

    final dangerButtonStyle = ElevatedButton.styleFrom(
      backgroundColor: AppColors.danger,
      foregroundColor: Colors.white,
      elevation: 0,
      shape: pillShape,
      padding: pillPadding,
      textStyle: actionTextStyle,
    );

    final outlineButtonStyle = OutlinedButton.styleFrom(
      foregroundColor: primary,
      side: BorderSide(color: primary.withOpacity(0.6)),
      shape: pillShape,
      padding: pillPadding,
      textStyle: actionTextStyle,
    );

    return Container(
      margin: const EdgeInsets.fromLTRB(16, 8, 16, 4),
      padding: const EdgeInsets.fromLTRB(14, 8, 14, 8),
      decoration: BoxDecoration(
        color: surface.withOpacity(0.92),
        borderRadius: BorderRadius.circular(16),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withOpacity(0.18),
            blurRadius: 18,
            offset: const Offset(0, 8),
          ),
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Icon(Icons.dns, size: 20, color: AppColors.primary),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  '$name',
                  style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold, color: onSurface),
                ),
              ),
              StatusTag.vps(resolvedStatus),
            ],
          ),
          const SizedBox(height: 12),
          Wrap(
            spacing: 8,
            runSpacing: 6,
            children: [
              _buildMetaChip('ID', '${detail['id'] ?? ''}'),
              _buildMetaChip('${spec.cpu}核', ''),
              _buildMetaChip('${spec.memoryGb}GB', ''),
              _buildMetaChip('${spec.diskGb}GB', ''),
              _buildMetaChip(spec.bandwidthMbps > 0 ? '${spec.bandwidthMbps}Mbps' : '-', ''),
            ],
          ),
          const SizedBox(height: 14),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              ElevatedButton.icon(
                style: primaryButtonStyle,
                onPressed: () => _openPanel(),
                icon: const Icon(Icons.api_outlined),
                label: const Text('控制面板'),
              ),
              if (emergencyRenewEligible)
                ElevatedButton.icon(
                  style: dangerButtonStyle,
                  onPressed: () => _submitEmergencyRenew(),
                  icon: const Icon(Icons.sync),
                  label: const Text('紧急续费'),
                )
              else
                OutlinedButton.icon(
                  style: outlineButtonStyle,
                  onPressed: () => _openRenewDialog(context),
                  icon: const Icon(Icons.sync),
                  label: const Text('续费'),
                ),
              OutlinedButton.icon(
                style: outlineButtonStyle,
                onPressed: () => _openRemote(),
                icon: const Icon(Icons.computer),
                label: const Text('远程'),
              ),
              OutlinedButton.icon(
                style: outlineButtonStyle,
                onPressed: loading ? null : _refreshCurrentPage,
                icon: const Icon(Icons.refresh),
                label: const Text('刷新'),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildMetaChip(String label, String value) {
    final text = value.isEmpty ? label : '$label $value';
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      decoration: BoxDecoration(
        color: AppColors.darkSurface.withOpacity(0.35),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: AppColors.gray600.withOpacity(0.4)),
      ),
      child: Text(
        text,
        style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w600),
      ),
    );
  }
  Widget _buildOverviewTab(
    BuildContext context,
    Map<String, dynamic> detail,
    _SpecInfo spec,
    _AccessInfo access,
    String resolvedStatus,
    bool emergencyEligible,
    bool isExpiringSoon,
  ) {
    final systemLabel = _resolveSystemLabel(detail);
    final remainingText = _formatRemaining(detail['expire_at']?.toString());
    final createdAt = DateFormatter.formatIso(detail['created_at']?.toString());
    final expireAt = DateFormatter.formatIso(detail['expire_at']?.toString());
    final monthlyPrice = MoneyFormatter.format(_toDouble(detail['monthly_price']));

    return LayoutBuilder(
      builder: (context, constraints) {
        final isWide = constraints.maxWidth >= 1100;
        final cardGap = 16.0;
        final content = isWide
            ? Column(
                children: [
                  Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Expanded(
                        child: Column(
                          children: [
                            _buildCard(
                              title: '实例信息',
                              icon: Icons.dns,
                              child: _buildInstanceInfo(
                                detail,
                                access,
                                resolvedStatus,
                                remainingText,
                                systemLabel,
                              ),
                            ),
                            const SizedBox(height: 16),
                            _buildCard(
                              title: '电源操作',
                              icon: Icons.power,
                              child: _buildPowerActions(),
                            ),
                          ],
                        ),
                      ),
                      SizedBox(width: cardGap),
                      Expanded(
                        child: Column(
                          children: [
                            _buildCard(
                              title: '实例监控',
                              icon: Icons.show_chart,
                              child: _buildMonitorSummary(spec),
                            ),
                            const SizedBox(height: 16),
                            _buildCard(
                              title: '时间与价格',
                              icon: Icons.calendar_today,
                              child: _buildTimePrice(
                                createdAt: createdAt,
                                expireAt: expireAt,
                                remainingText: remainingText,
                                isExpiringSoon: isExpiringSoon,
                                monthlyPrice: monthlyPrice,
                                emergencyEligible: emergencyEligible,
                              ),
                            ),
                          ],
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 16),
                  _buildCard(
                    title: '连接信息',
                    icon: Icons.security,
                    child: _buildConnectionInfo(detail, access, systemLabel),
                  ),
                ],
              )
            : Column(
                children: [
                  _buildCard(
                    title: '实例信息',
                    icon: Icons.dns,
                    child:
                        _buildInstanceInfo(detail, access, resolvedStatus, remainingText, systemLabel),
                  ),
                  const SizedBox(height: 16),
                  _buildCard(
                    title: '实例监控',
                    icon: Icons.show_chart,
                    child: _buildMonitorSummary(spec),
                  ),
                  const SizedBox(height: 16),
                  _buildCard(
                    title: '电源操作',
                    icon: Icons.power,
                    child: _buildPowerActions(),
                  ),
                  const SizedBox(height: 16),
                  _buildCard(
                    title: '时间与价格',
                    icon: Icons.calendar_today,
                    child: _buildTimePrice(
                      createdAt: createdAt,
                      expireAt: expireAt,
                      remainingText: remainingText,
                      isExpiringSoon: isExpiringSoon,
                      monthlyPrice: monthlyPrice,
                      emergencyEligible: emergencyEligible,
                    ),
                  ),
                  const SizedBox(height: 16),
                  _buildCard(
                    title: '连接信息',
                    icon: Icons.security,
                    child: _buildConnectionInfo(detail, access, systemLabel),
                  ),
                ],
              );

        return SingleChildScrollView(
          padding: const EdgeInsets.all(16),
          child: isWide
              ? Align(
                  alignment: Alignment.topCenter,
                  child: ConstrainedBox(
                    constraints: const BoxConstraints(maxWidth: 1200),
                    child: content,
                  ),
                )
              : content,
        );
      },
    );
  }

  Widget _buildCard({required String title, required IconData icon, required Widget child}) {
    return Card(
      elevation: 1,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(icon, color: AppColors.primary, size: 18),
                const SizedBox(width: 8),
                Text(title, style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
              ],
            ),
            const SizedBox(height: 16),
            child,
          ],
        ),
      ),
    );
  }

  Widget _buildInstanceInfo(
    Map<String, dynamic> detail,
    _AccessInfo access,
    String resolvedStatus,
    String remainingText,
    String systemLabel,
  ) {
    final region = detail['region'] ?? detail['Region'] ?? '-';
    return Column(
      children: [
        _infoRow('实例状态', StatusTag.vps(resolvedStatus)),
        _infoRow(
          '远程IP',
          Row(
            children: [
              Expanded(child: Text(access.remoteIp.isEmpty ? '-' : access.remoteIp)),
              IconButton(
                onPressed: access.remoteIp.isEmpty ? null : () => _copyText(access.remoteIp, '远程IP'),
                icon: const Icon(Icons.copy, size: 16),
              ),
            ],
          ),
        ),
        _infoRow(
          '剩余天数',
          Text(
            remainingText,
            style: TextStyle(color: remainingText.contains('天') ? null : AppColors.warning),
          ),
        ),
        _infoRow(
          '系统密码',
          Row(
            children: [
              Expanded(
                child: Text(_showOsPassword ? (access.osPassword.isEmpty ? '-' : access.osPassword) : '••••••••'),
              ),
              IconButton(
                onPressed:
                    access.osPassword.isEmpty ? null : () => _copyText(access.osPassword, '密码'),
                icon: const Icon(Icons.copy, size: 16),
              ),
              IconButton(
                onPressed: () => setState(() => _showOsPassword = !_showOsPassword),
                icon: Icon(_showOsPassword ? Icons.visibility_off : Icons.visibility, size: 16),
              ),
              TextButton(onPressed: () => _openResetPasswordDialog(context), child: const Text('修改')),
            ],
          ),
        ),
        _infoRow('区域', Text('$region')),
        _infoRow(
          '操作系统',
          Row(
            children: [
              Expanded(child: Text(systemLabel)),
              TextButton(onPressed: () => _openReinstallDialog(context), child: const Text('重装')),
            ],
          ),
        ),
      ],
    );
  }

  Widget _buildPowerActions() {
    final baseStyle = OutlinedButton.styleFrom(
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
      textStyle: const TextStyle(fontWeight: FontWeight.w600),
    );

    return Row(
      children: [
        Expanded(
          child: OutlinedButton.icon(
            style: baseStyle.copyWith(
              foregroundColor: MaterialStateProperty.all(AppColors.success),
              side: MaterialStateProperty.all(BorderSide(color: AppColors.success.withOpacity(0.6))),
            ),
            onPressed: () => _operate(
              context,
              () => ref.read(vpsRepositoryProvider).start(widget.id),
              '已触发开机',
            ),
            icon: const Icon(Icons.play_arrow),
            label: const Text('启动'),
          ),
        ),
        const SizedBox(width: 10),
        Expanded(
          child: OutlinedButton.icon(
            style: baseStyle.copyWith(
              foregroundColor: MaterialStateProperty.all(AppColors.warning),
              side: MaterialStateProperty.all(BorderSide(color: AppColors.warning.withOpacity(0.6))),
            ),
            onPressed: () => _operate(
              context,
              () => ref.read(vpsRepositoryProvider).shutdown(widget.id),
              '已触发关机',
            ),
            icon: const Icon(Icons.power_settings_new),
            label: const Text('关机'),
          ),
        ),
        const SizedBox(width: 10),
        Expanded(
          child: OutlinedButton.icon(
            style: baseStyle.copyWith(
              foregroundColor: MaterialStateProperty.all(AppColors.info),
              side: MaterialStateProperty.all(BorderSide(color: AppColors.info.withOpacity(0.6))),
            ),
            onPressed: () => _operate(
              context,
              () => ref.read(vpsRepositoryProvider).reboot(widget.id),
              '已触发重启',
            ),
            icon: const Icon(Icons.restart_alt),
            label: const Text('重启'),
          ),
        ),
      ],
    );
  }

  Widget _buildMonitorSummary(_SpecInfo spec) {
    return Consumer(
      builder: (context, ref, _) {
        final monitor = ref.watch(vpsMonitorStateProvider(widget.id));
        final cpu = monitor.cpu.values.isNotEmpty ? monitor.cpu.values.last : 0.0;
        final memory = monitor.memory.values.isNotEmpty ? monitor.memory.values.last : 0.0;
        final trafficIn = monitor.trafficIn.values.isNotEmpty ? monitor.trafficIn.values.last : 0.0;
        final trafficOut = monitor.trafficOut.values.isNotEmpty ? monitor.trafficOut.values.last : 0.0;

        return Column(
          children: [
            _monitorItem('CPU', cpu, '${spec.cpu}核', _cpuColor(cpu)),
            const SizedBox(height: 12),
            _monitorItem('内存', memory, '${spec.memoryGb}GB', _memoryColor(memory)),
            const SizedBox(height: 12),
            _networkItem(trafficIn, trafficOut),
          ],
        );
      },
    );
  }

  Widget _monitorItem(String label, double value, String spec, Color color) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Text(label, style: const TextStyle(fontSize: 13, color: AppColors.gray600)),
            const Spacer(),
            Text('${value.toStringAsFixed(0)}%',
                style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold, color: color)),
            const SizedBox(width: 8),
            Text(spec, style: const TextStyle(fontSize: 12, color: AppColors.gray500)),
          ],
        ),
        const SizedBox(height: 6),
        ClipRRect(
          borderRadius: BorderRadius.circular(4),
          child: LinearProgressIndicator(
            value: value.clamp(0, 100) / 100,
            minHeight: 6,
            backgroundColor: AppColors.gray200,
            color: color,
          ),
        ),
      ],
    );
  }

  Widget _networkItem(double trafficIn, double trafficOut) {
    return Row(
      children: [
        const Icon(Icons.cloud_download, size: 16, color: AppColors.gray500),
        const SizedBox(width: 6),
        Text('${trafficIn.toStringAsFixed(0)} Mbps', style: const TextStyle(fontWeight: FontWeight.w600)),
        const SizedBox(width: 16),
        const Icon(Icons.cloud_upload, size: 16, color: AppColors.gray500),
        const SizedBox(width: 6),
        Text('${trafficOut.toStringAsFixed(0)} Mbps', style: const TextStyle(fontWeight: FontWeight.w600)),
      ],
    );
  }

  Widget _buildTimePrice({
    required String createdAt,
    required String expireAt,
    required String remainingText,
    required bool isExpiringSoon,
    required String monthlyPrice,
    required bool emergencyEligible,
  }) {
    final resizeEnabled = _resizeEnabled();
    final pillPadding = const EdgeInsets.symmetric(horizontal: 14, vertical: 10);
    final pillShape = const StadiumBorder();
    final actionTextStyle = const TextStyle(fontWeight: FontWeight.w600);

    final primaryButtonStyle = ElevatedButton.styleFrom(
      backgroundColor: AppColors.primary,
      foregroundColor: Colors.white,
      elevation: 0,
      shape: pillShape,
      padding: pillPadding,
      textStyle: actionTextStyle,
    );

    final dangerButtonStyle = ElevatedButton.styleFrom(
      backgroundColor: AppColors.danger,
      foregroundColor: Colors.white,
      elevation: 0,
      shape: pillShape,
      padding: pillPadding,
      textStyle: actionTextStyle,
    );

    final outlineButtonStyle = OutlinedButton.styleFrom(
      foregroundColor: AppColors.primary,
      side: BorderSide(color: AppColors.primary.withOpacity(0.6)),
      shape: pillShape,
      padding: pillPadding,
      textStyle: actionTextStyle,
    );
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _simpleInfo('创建时间', createdAt),
        _simpleInfo('到期时间', expireAt, highlight: isExpiringSoon),
        _simpleInfo('剩余天数', remainingText, highlight: remainingText.contains('天') && isExpiringSoon),
        _simpleInfo('当前价格', '$monthlyPrice /月', highlight: true),
        const Divider(height: 24),
        Wrap(
          spacing: 12,
          runSpacing: 8,
          children: [
            if (emergencyEligible)
              ElevatedButton.icon(
                style: dangerButtonStyle,
                onPressed: () => _submitEmergencyRenew(),
                icon: const Icon(Icons.sync),
                label: const Text('紧急续费'),
              )
            else
              ElevatedButton.icon(
                style: primaryButtonStyle,
                onPressed: () => _openRenewDialog(context),
                icon: const Icon(Icons.sync),
                label: const Text('续费'),
              ),
            if (resizeEnabled)
              OutlinedButton.icon(
                style: outlineButtonStyle,
                onPressed: () => _openResizeDialog(context),
                icon: const Icon(Icons.vertical_align_top),
                label: const Text('升降配'),
              ),
            OutlinedButton.icon(
              style: outlineButtonStyle.copyWith(
                foregroundColor: MaterialStateProperty.all(AppColors.danger),
                side: MaterialStateProperty.all(BorderSide(color: AppColors.danger.withOpacity(0.6))),
              ),
              onPressed: () => _openRefundDialog(context),
              icon: const Icon(Icons.delete_outline),
              label: const Text('退款'),
            ),
          ],
        ),
      ],
    );
  }

  Widget _buildConnectionInfo(Map<String, dynamic> detail, _AccessInfo access, String systemLabel) {
    final isWindows = _isWindowsOS(systemLabel);
    final remote = access.remoteIp.isEmpty ? '-' : access.remoteIp;
    return Column(
      children: [
        _infoRow('操作系统', Text(systemLabel)),
        _infoRow(
          '远程地址',
          Row(
            children: [
              Expanded(child: Text(remote)),
              IconButton(
                onPressed: remote == '-' ? null : () => _copyText(remote, '远程地址'),
                icon: const Icon(Icons.copy, size: 16),
              ),
            ],
          ),
        ),
        _infoRow('系统用户', Text(isWindows ? 'Administrator' : 'root')),
        _infoRow(
          '系统密码',
          Row(
            children: [
              Expanded(
                child: Text(_showOsPassword ? (access.osPassword.isEmpty ? '-' : access.osPassword) : '••••••••'),
              ),
              IconButton(
                onPressed: () => setState(() => _showOsPassword = !_showOsPassword),
                icon: Icon(_showOsPassword ? Icons.visibility_off : Icons.visibility, size: 16),
              ),
              TextButton(onPressed: () => _openResetPasswordDialog(context), child: const Text('修改')),
            ],
          ),
        ),
        _infoRow('面板用户', Text(detail['name']?.toString() ?? '-')),
        _infoRow(
          '面板密码',
          Row(
            children: [
              Expanded(
                child: Text(_showPanelPassword
                    ? (access.panelPassword.isEmpty ? '-' : access.panelPassword)
                    : '••••••••'),
              ),
              IconButton(
                onPressed: () => setState(() => _showPanelPassword = !_showPanelPassword),
                icon: Icon(_showPanelPassword ? Icons.visibility_off : Icons.visibility, size: 16),
              ),
            ],
          ),
        ),
      ],
    );
  }

  Widget _buildMonitorTab(
    BuildContext context,
    Map<String, dynamic> detail,
    String resolvedStatus,
    bool isExpiringSoon,
  ) {
    return Consumer(
      builder: (context, ref, _) {
        final monitor = ref.watch(vpsMonitorStateProvider(widget.id));
        final cpuValues = monitor.cpu.values;
        final memoryValues = monitor.memory.values;
        final currentCpu = cpuValues.isNotEmpty ? cpuValues.last : 0.0;
        final currentMemory = memoryValues.isNotEmpty ? memoryValues.last : 0.0;
        final perfScore = 100 - (currentCpu + currentMemory) / 2;
        final gaugeValue = perfScore.clamp(0, 100).toDouble();

        return LayoutBuilder(
          builder: (context, constraints) {
            final isWide = constraints.maxWidth >= 900;
            final cardWidth = isWide ? (constraints.maxWidth - 32) / 2 : constraints.maxWidth;
            return SingleChildScrollView(
              padding: const EdgeInsets.all(16),
              child: Wrap(
                spacing: 16,
                runSpacing: 16,
                children: [
                  SizedBox(
                    width: constraints.maxWidth,
                    child: _buildCard(
                      title: '系统表现',
                      icon: Icons.security,
                      child: _buildPerfPanel(
                        resolvedStatus,
                        detail,
                        isExpiringSoon,
                        gaugeValue,
                      ),
                    ),
                  ),
                  SizedBox(
                    width: cardWidth,
                    child: _buildCard(
                      title: 'CPU',
                      icon: Icons.show_chart,
                      child: LineChart(
                        values: monitor.cpu.values,
                        labels: monitor.cpu.labels,
                        lineColor: AppColors.primary,
                        height: 160,
                      ),
                    ),
                  ),
                  SizedBox(
                    width: cardWidth,
                    child: _buildCard(
                      title: 'IO',
                      icon: Icons.cloud_upload,
                      child: LineChart(
                        values: monitor.trafficOut.values,
                        labels: monitor.trafficOut.labels,
                        lineColor: AppColors.warning,
                        height: 160,
                      ),
                    ),
                  ),
                  SizedBox(
                    width: cardWidth,
                    child: _buildCard(
                      title: '网络',
                      icon: Icons.cloud_download,
                      child: LineChart(
                        values: monitor.trafficIn.values,
                        labels: monitor.trafficIn.labels,
                        lineColor: AppColors.success,
                        height: 160,
                      ),
                    ),
                  ),
                  SizedBox(
                    width: cardWidth,
                    child: _buildCard(
                      title: '内存',
                      icon: Icons.memory,
                      child: LineChart(
                        values: monitor.memory.values,
                        labels: monitor.memory.labels,
                        lineColor: AppColors.info,
                        height: 160,
                      ),
                    ),
                  ),
                ],
              ),
            );
          },
        );
      },
    );
  }
  Widget _buildPerfPanel(
    String resolvedStatus,
    Map<String, dynamic> detail,
    bool isExpiringSoon,
    double gaugeValue,
  ) {
    final systemLabel = _resolveSystemLabel(detail);
    final expireAt = DateFormatter.formatIso(detail['expire_at']?.toString());

    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              _simpleInfo('实例状态', resolvedStatus.isEmpty ? '-' : resolvedStatus),
              _simpleInfo('操作系统', systemLabel),
              _simpleInfo('到期时间', expireAt, highlight: isExpiringSoon),
            ],
          ),
        ),
        _PerfGauge(value: gaugeValue),
      ],
    );
  }

  Widget _buildFirewall(BuildContext context) {
    final firewallAsync = ref.watch(vpsFirewallProvider(widget.id));
    return firewallAsync.when(
      data: (items) {
        final rules = items.map(_normalizeFirewallRule).toList();
        return ListView(
          padding: const EdgeInsets.all(16),
          children: [
            if (rules.isEmpty)
              const EmptyState(message: AppStrings.noData, icon: Icons.security),
            ...rules.map((rule) {
              return Card(
                child: ListTile(
                  title: Text(
                    '${rule['direction'] == '' ? '-' : rule['direction']} '
                    '${rule['protocol'] == '' ? '-' : rule['protocol']} '
                    '${rule['method'] == '' ? '-' : rule['method']} 端口: '
                    '${rule['port'] == '' ? '-' : rule['port']}',
                  ),
                  subtitle: Text('IP: ${rule['ip'] == '' ? '-' : rule['ip']}'),
                  trailing: IconButton(
                    icon: const Icon(Icons.delete, color: Colors.red),
                    onPressed: rule['id'] == null
                        ? null
                        : () async {
                            await _operate(
                              context,
                              () => ref
                                  .read(vpsRepositoryProvider)
                                  .deleteFirewallRule(widget.id, int.parse('${rule['id']}')),
                              '已删除',
                            );
                            ref.invalidate(vpsFirewallProvider(widget.id));
                          },
                  ),
                ),
              );
            }),
          ],
        );
      },
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => Center(child: Text(e.toString())),
    );
  }

  Widget _buildPorts(BuildContext context) {
    final portsAsync = ref.watch(vpsPortsProvider(widget.id));
    return portsAsync.when(
      data: (items) {
        final ports = items.map(_normalizePortMapping).toList();
        return ListView(
          padding: const EdgeInsets.all(16),
          children: [
            if (ports.isEmpty)
              const EmptyState(message: AppStrings.noData, icon: Icons.swap_horiz),
            ...ports.map((item) {
              final external = _formatPortExternal(item);
              final rawName = (item['name'] ?? '').toString().trim();
              final nameLower = rawName.toLowerCase();
              final protectedNames = {'ssh', '远程桌面', 'rdp', 'remote desktop'};
              final isProtected = protectedNames.contains(nameLower) || protectedNames.contains(rawName);
              return Card(
                child: ListTile(
                  title: Text('${item['name'] == '' ? '-' : item['name']}'),
                  subtitle: Text(
                    '外部地址: $external -> 目标端口: ${item['dport'] == '' ? '-' : item['dport']}',
                  ),
                  trailing: IconButton(
                    icon: Icon(Icons.delete, color: isProtected ? AppColors.gray500 : Colors.red),
                    onPressed: item['id'] == null || isProtected
                        ? null
                        : () async {
                            await _operate(
                              context,
                              () => ref
                                  .read(vpsRepositoryProvider)
                                  .deletePortMapping(widget.id, int.parse('${item['id']}')),
                              '已删除',
                            );
                            ref.invalidate(vpsPortsProvider(widget.id));
                          },
                  ),
                ),
              );
            }),
          ],
        );
      },
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => Center(child: Text(e.toString())),
    );
  }

  Widget _buildSnapshots(BuildContext context) {
    final snapshotAsync = ref.watch(vpsSnapshotsProvider(widget.id));
    return snapshotAsync.when(
      data: (items) {
        final snapshots = items.map(_normalizeSnapshotItem).toList();
        return ListView(
          padding: const EdgeInsets.all(16),
          children: [
            if (snapshots.isEmpty)
              const EmptyState(message: AppStrings.noData, icon: Icons.camera_alt),
            ...snapshots.map((item) {
              return Card(
                child: ListTile(
                  title: Text('${item['name'] == '' ? '-' : item['name']}'),
                  subtitle: Text('创建时间: ${DateFormatter.formatIso(item['created_at'])}'),
                  trailing: Wrap(
                    spacing: 8,
                    children: [
                      TextButton(
                        onPressed: item['id'] == null
                            ? null
                            : () async {
                                await _operate(
                                  context,
                                  () => ref
                                      .read(vpsRepositoryProvider)
                                      .restoreSnapshot(widget.id, int.parse('${item['id']}')),
                                  '已提交恢复',
                                );
                              },
                        child: const Text('恢复'),
                      ),
                      TextButton(
                        onPressed: item['id'] == null
                            ? null
                            : () async {
                                await _operate(
                                  context,
                                  () => ref
                                      .read(vpsRepositoryProvider)
                                      .deleteSnapshot(widget.id, int.parse('${item['id']}')),
                                  '已删除',
                                );
                                ref.invalidate(vpsSnapshotsProvider(widget.id));
                              },
                        child: const Text('删除', style: TextStyle(color: Colors.red)),
                      ),
                    ],
                  ),
                ),
              );
            }),
          ],
        );
      },
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => Center(child: Text(e.toString())),
    );
  }

  Widget _buildBackups(BuildContext context) {
    final backupAsync = ref.watch(vpsBackupsProvider(widget.id));
    return backupAsync.when(
      data: (items) {
        final backups = items.map(_normalizeBackupItem).toList();
        return ListView(
          padding: const EdgeInsets.all(16),
          children: [
            if (backups.isEmpty)
              const EmptyState(message: AppStrings.noData, icon: Icons.cloud),
            ...backups.map((item) {
              return Card(
                child: ListTile(
                  title: Text('${item['name'] == '' ? '-' : item['name']}'),
                  subtitle: Text('创建时间: ${DateFormatter.formatIso(item['created_at'])}'),
                  trailing: Wrap(
                    spacing: 8,
                    children: [
                      TextButton(
                        onPressed: item['id'] == null
                            ? null
                            : () async {
                                await _operate(
                                  context,
                                  () => ref
                                      .read(vpsRepositoryProvider)
                                      .restoreBackup(widget.id, int.parse('${item['id']}')),
                                  '已提交恢复',
                                );
                              },
                        child: const Text('恢复'),
                      ),
                      TextButton(
                        onPressed: item['id'] == null
                            ? null
                            : () async {
                                await _operate(
                                  context,
                                  () => ref
                                      .read(vpsRepositoryProvider)
                                      .deleteBackup(widget.id, int.parse('${item['id']}')),
                                  '已删除',
                                );
                                ref.invalidate(vpsBackupsProvider(widget.id));
                              },
                        child: const Text('删除', style: TextStyle(color: Colors.red)),
                      ),
                    ],
                  ),
                ),
              );
            }),
          ],
        );
      },
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => Center(child: Text(e.toString())),
    );
  }

  Widget _infoRow(String label, Widget value) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.center,
        children: [
          SizedBox(
            width: 80,
            child: Text(
              label,
              style: const TextStyle(color: AppColors.gray600),
              textAlign: TextAlign.left,
            ),
          ),
          Expanded(
            child: Align(
              alignment: Alignment.centerLeft,
              child: value,
            ),
          ),
        ],
      ),
    );
  }

  Widget _simpleInfo(String label, String value, {bool highlight = false}) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Row(
        children: [
          SizedBox(width: 80, child: Text(label, style: const TextStyle(color: AppColors.gray600))),
          Expanded(
            child: Text(
              value,
              style: TextStyle(color: highlight ? AppColors.primary : null, fontWeight: FontWeight.w500),
            ),
          ),
        ],
      ),
    );
  }

  String _resolveStatus(Map<String, dynamic> detail) {
    final rawStatus = detail['status']?.toString().toLowerCase() ?? '';
    final autoState = detail['automation_state'];
    if (autoState == null) return rawStatus;
    switch (int.tryParse(autoState.toString()) ?? -1) {
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
        return rawStatus;
    }
  }

  _SpecInfo _resolveSpec(Map<String, dynamic> detail) {
    dynamic spec = detail['spec'] ?? detail['Spec'] ?? detail['spec_json'] ?? detail['SpecJSON'];
    if (spec is String) {
      try {
        spec = jsonDecode(spec);
      } catch (_) {
        spec = {};
      }
    }
    if (spec is! Map) spec = {};
    return _SpecInfo(
      cpu: _toInt(spec['cpu'] ?? spec['cores'] ?? detail['cpu'] ?? detail['cores'] ?? 0),
      memoryGb: _toInt(spec['memory_gb'] ?? spec['mem_gb'] ?? detail['memory_gb'] ?? detail['mem_gb'] ?? 0),
      diskGb: _toInt(spec['disk_gb'] ?? detail['disk_gb'] ?? 0),
      bandwidthMbps: _toInt(spec['bandwidth_mbps'] ?? spec['bandwidth'] ?? detail['bandwidth_mbps'] ?? 0),
    );
  }

  _AccessInfo _parseAccessInfo(Map<String, dynamic> detail) {
    final info = _parseJson(
      detail['access_info'] ??
          detail['AccessInfo'] ??
          detail['access_info_json'] ??
          detail['AccessInfoJSON'],
    );
    return _AccessInfo(
      remoteIp: _toString(
        info['remote_ip'] ?? info['ip'] ?? info['public_ip'] ?? info['ipv4'] ?? info['Ip'],
      ),
      remotePort: _toString(info['remote_port'] ?? info['port'] ?? info['ssh_port'] ?? info['Port']),
      osPassword: _toString(info['os_password'] ?? info['password'] ?? info['pass'] ?? info['Password']),
      panelPassword: _toString(info['panel_password'] ?? info['panelPassword']),
      vncPassword: _toString(info['vnc_password'] ?? info['vnc']),
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
    if (input is Map) {
      return input.map((key, value) => MapEntry(key.toString(), value));
    }
    return {};
  }

  int _toInt(dynamic value) {
    if (value == null) return 0;
    if (value is num) return value.toInt();
    return int.tryParse(value.toString()) ?? 0;
  }

  double _toDouble(dynamic value) {
    if (value == null) return 0;
    if (value is num) return value.toDouble();
    return double.tryParse(value.toString()) ?? 0;
  }

  String _toString(dynamic value) {
    if (value == null) return '';
    return value.toString();
  }

  String _formatRemaining(String? expireAt) {
    if (expireAt == null || expireAt.isEmpty) return '-';
    final date = DateFormatter.parse(expireAt);
    if (date == null) return expireAt;
    final now = DateTime.now();
    final diff = date.difference(now);
    if (diff.isNegative) return '已过期';
    if (diff.inDays == 0) return '今天到期';
    return '${diff.inDays} 天';
  }

  bool _isExpiringSoon(String? expireAt) {
    final date = DateFormatter.parse(expireAt);
    return DateFormatter.isExpiringSoon(date, days: 7);
  }

  String _resolveSystemLabel(Map<String, dynamic> detail) {
    final systemId = detail['system_id'] ?? detail['systemId'] ?? detail['SystemID'];
    final catalog = ref.read(catalogProvider);
    final image = catalog.systemImages.firstWhere(
      (item) => item['id'] == systemId || item['image_id'] == systemId,
      orElse: () => {},
    );
    final label = image['name'] ?? detail['system_name'] ?? detail['system'] ?? '-';
    return label.toString().isEmpty ? '-' : label.toString();
  }

  bool _isWindowsOS(String label) {
    return label.toLowerCase().contains('windows');
  }

  bool _resizeEnabled() {
    final settings = ref.read(siteProvider).settings;
    final value = settings['resize_enabled'];
    if (value == null) return true;
    if (value is bool) return value;
    final str = value.toString().toLowerCase();
    return str != 'false' && str != '0';
  }

  bool _isEmergencyRenewEligible(Map<String, dynamic> detail, Map<String, dynamic> settings) {
    final enabledRaw = settings['emergency_renew_enabled'];
    final enabled = enabledRaw == null
        ? true
        : (enabledRaw is bool ? enabledRaw : enabledRaw.toString().toLowerCase() != 'false');
    if (!enabled) return false;

    final expireAt = DateFormatter.parse(detail['expire_at']);
    if (expireAt == null) return false;
    final now = DateTime.now();
    if (expireAt.isBefore(now)) return false;

    var windowDays = int.tryParse(settings['emergency_renew_window_days']?.toString() ?? '7') ?? 7;
    var intervalHours = int.tryParse(settings['emergency_renew_interval_hours']?.toString() ?? '720') ?? 720;
    if (windowDays < 0) windowDays = 0;
    if (intervalHours <= 0) intervalHours = 24;

    if (windowDays > 0) {
      final windowStart = expireAt.subtract(Duration(days: windowDays));
      if (now.isBefore(windowStart)) return false;
    }

    final lastAtRaw = detail['last_emergency_renew_at'];
    final lastAt = DateFormatter.parse(lastAtRaw);
    if (lastAt != null) {
      final diffHours = now.difference(lastAt).inHours;
      if (diffHours < intervalHours) return false;
    }

    return true;
  }

  Color _cpuColor(double value) {
    if (value >= 90) return AppColors.danger;
    if (value >= 70) return AppColors.warning;
    if (value >= 50) return AppColors.primary;
    return AppColors.success;
  }

  Color _memoryColor(double value) {
    if (value >= 90) return AppColors.danger;
    if (value >= 75) return AppColors.warning;
    if (value >= 50) return AppColors.primary;
    return AppColors.success;
  }

  Map<String, dynamic> _normalizeFirewallRule(Map<String, dynamic> item) {
    return {
      'id': item['id'] ?? item['ID'] ?? item['rule_id'] ?? item['RuleID'] ?? item['firewall_id'] ?? item['FirewallID'],
      'direction': _sanitizeValue(item['direction'] ?? item['Direction']),
      'protocol': _sanitizeValue(item['protocol'] ?? item['Protocol']),
      'port': _sanitizeValue(item['port'] ?? item['Port'] ?? item['start_port'] ?? item['StartPort']),
      'ip': _sanitizeValue(item['ip'] ?? item['IP'] ?? item['start_ip'] ?? item['StartIP']),
      'method': _sanitizeValue(item['method'] ?? item['Method']),
    };
  }

  Map<String, dynamic> _normalizePortMapping(Map<String, dynamic> item) {
    return {
      'id': item['id'] ?? item['ID'] ?? item['port_id'] ?? item['PortID'],
      'name': _sanitizeValue(item['name'] ?? item['Name'] ?? item['remark']),
      'sport': _sanitizeValue(item['sport'] ?? item['Sport'] ?? item['source_port'] ?? item['SourcePort']),
      'dport': _sanitizeValue(item['dport'] ?? item['Dport'] ?? item['target_port'] ?? item['TargetPort']),
      'api_url': _sanitizeValue(item['api_url'] ?? item['apiUrl'] ?? item['ApiUrl']),
    };
  }

  String _formatPortExternal(Map<String, dynamic> record) {
    final host = _sanitizeValue(record['api_url']);
    final port = _sanitizeValue(record['sport']);
    if (host.isEmpty && port.isEmpty) return '-';
    if (host.isEmpty) return port;
    if (port.isEmpty) return host;
    return '$host:$port';
  }

  Map<String, dynamic> _normalizeSnapshotItem(Map<String, dynamic> item) {
    final id = item['id'] ??
        item['ID'] ??
        item['snapshot_id'] ??
        item['snapshotId'] ??
        item['sid'] ??
        item['SID'] ??
        item['virtuals_id'] ??
        item['virtualsId'];
    return {
      'id': id,
      'name': _sanitizeValue(item['name'] ?? item['Name']) != ''
          ? _sanitizeValue(item['name'] ?? item['Name'])
          : (id != null ? 'snapshot-$id' : 'snapshot'),
      'created_at': _sanitizeValue(item['created_at'] ?? item['create_time'] ?? item['createdAt'] ?? item['createTime']),
    };
  }

  Map<String, dynamic> _normalizeBackupItem(Map<String, dynamic> item) {
    final id = item['id'] ?? item['ID'] ?? item['backup_id'] ?? item['backupId'] ?? item['bid'] ?? item['BID'];
    return {
      'id': id,
      'name': _sanitizeValue(item['name'] ?? item['Name']) != ''
          ? _sanitizeValue(item['name'] ?? item['Name'])
          : (id != null ? 'backup-$id' : 'backup'),
      'created_at': _sanitizeValue(item['created_at'] ?? item['create_time'] ?? item['createdAt'] ?? item['createTime']),
    };
  }

  String _sanitizeValue(dynamic value) {
    if (value == null) return '';
    final text = value.toString();
    if (text == '<nil>') return '';
    return text;
  }

  List<int> _normalizeCandidateList(List<dynamic> raw) {
    final result = <int>[];
    for (final entry in raw) {
      if (entry is num) {
        result.add(entry.toInt());
      } else if (entry is String) {
        final parsed = int.tryParse(entry);
        if (parsed != null) result.add(parsed);
      } else if (entry is Map) {
        final value = entry['port'] ?? entry['value'] ?? entry['Port'] ?? entry['Value'];
        final parsed = int.tryParse(value?.toString() ?? '');
        if (parsed != null) result.add(parsed);
      }
    }
    return result;
  }
  Future<void> _openFirewallDialog(BuildContext context) async {
    String direction = 'In';
    String protocol = 'tcp';
    String method = 'allowed';
    final portController = TextEditingController();
    final ipController = TextEditingController(text: '0.0.0.0');

    await showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('添加防火墙规则'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            DropdownButtonFormField<String>(
              value: direction,
              decoration: const InputDecoration(labelText: '方向'),
              items: const [
                DropdownMenuItem(value: 'In', child: Text('入站')),
                DropdownMenuItem(value: 'Out', child: Text('出站')),
              ],
              onChanged: (v) => direction = v ?? 'In',
            ),
            DropdownButtonFormField<String>(
              value: protocol,
              decoration: const InputDecoration(labelText: '协议'),
              items: const [
                DropdownMenuItem(value: 'tcp', child: Text('TCP')),
                DropdownMenuItem(value: 'udp', child: Text('UDP')),
              ],
              onChanged: (v) => protocol = v ?? 'tcp',
            ),
            DropdownButtonFormField<String>(
              value: method,
              decoration: const InputDecoration(labelText: '策略'),
              items: const [
                DropdownMenuItem(value: 'allowed', child: Text('允许')),
                DropdownMenuItem(value: 'denied', child: Text('拒绝')),
              ],
              onChanged: (v) => method = v ?? 'allowed',
            ),
            TextField(
              controller: portController,
              decoration: const InputDecoration(labelText: '端口'),
            ),
            TextField(
              controller: ipController,
              decoration: const InputDecoration(labelText: 'IP'),
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context), child: const Text(AppStrings.cancel)),
          TextButton(
            onPressed: () async {
              if (portController.text.trim().isEmpty) {
                ScaffoldMessenger.of(context)
                    .showSnackBar(const SnackBar(content: Text('请输入端口')));
                return;
              }
              if (ipController.text.trim().isEmpty) {
                ScaffoldMessenger.of(context)
                    .showSnackBar(const SnackBar(content: Text('请输入IP')));
                return;
              }
              await _operate(
                context,
                () => ref.read(vpsRepositoryProvider).addFirewallRule(widget.id, {
                  'direction': direction,
                  'protocol': protocol,
                  'method': method,
                  'port': portController.text.trim(),
                  'ip': ipController.text.trim(),
                }),
                '已添加',
              );
              if (context.mounted) Navigator.pop(context);
              ref.invalidate(vpsFirewallProvider(widget.id));
            },
            child: const Text(AppStrings.save),
          ),
        ],
      ),
    );
  }

  Future<void> _openPortDialog(BuildContext context) async {
    final nameController = TextEditingController();
    final sportController = TextEditingController();
    final dportController = TextEditingController();

    void scheduleCandidates(String value, StateSetter setModalState) {
      _portCandidateTimer?.cancel();
      _portCandidateTimer = Timer(const Duration(milliseconds: 300), () async {
        final keyword = value.trim();
        if (keyword.isEmpty) {
          if (!mounted) return;
          setModalState(() {
            _portCandidates = [];
            _portCandidatesLoading = false;
          });
          return;
        }
        if (!mounted) return;
        setModalState(() {
          _portCandidatesLoading = true;
        });
        try {
          final rawItems = await ref
              .read(vpsRepositoryProvider)
              .listPortCandidates(widget.id, keywords: keyword);
          final items = _normalizeCandidateList(rawItems);
          if (!mounted) return;
          setModalState(() {
            _portCandidates = items;
          });
        } finally {
          if (!mounted) return;
          setModalState(() {
            _portCandidatesLoading = false;
          });
        }
      });
    }

    await showDialog(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setModalState) => AlertDialog(
          title: const Text('添加端口映射'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: nameController,
                decoration: const InputDecoration(labelText: '名称'),
              ),
              TextField(
                controller: sportController,
                keyboardType: TextInputType.number,
                inputFormatters: [FilteringTextInputFormatter.digitsOnly],
                decoration: const InputDecoration(
                  labelText: '外部端口',
                  hintText: '输入端口后自动匹配',
                ),
                onChanged: (value) => scheduleCandidates(value, setModalState),
              ),
              if (_portCandidatesLoading)
                const Padding(padding: EdgeInsets.all(8), child: CircularProgressIndicator()),
              if (_portCandidates.isNotEmpty)
                Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const SizedBox(height: 6),
                    const Text('可用端口', style: TextStyle(fontSize: 12, color: AppColors.gray500)),
                    const SizedBox(height: 6),
                    Wrap(
                      spacing: 8,
                      runSpacing: 6,
                      children: _portCandidates.map((value) {
                        return ActionChip(
                          label: Text('$value'),
                          onPressed: () {
                            sportController.text = '$value';
                          },
                        );
                      }).toList(),
                    ),
                  ],
                ),
              TextField(
                controller: dportController,
                decoration: const InputDecoration(labelText: '目标端口'),
              ),
            ],
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(context), child: const Text(AppStrings.cancel)),
            TextButton(
              onPressed: () async {
                if (dportController.text.trim().isEmpty) {
                  ScaffoldMessenger.of(context)
                      .showSnackBar(const SnackBar(content: Text('请输入目标端口')));
                  return;
                }
                await _operate(
                  context,
                  () => ref.read(vpsRepositoryProvider).addPortMapping(widget.id, {
                    'name': nameController.text.trim(),
                    'sport': sportController.text.trim(),
                    'dport': int.tryParse(dportController.text.trim()) ?? dportController.text.trim(),
                  }),
                  '已添加',
                );
                if (context.mounted) Navigator.pop(context);
                ref.invalidate(vpsPortsProvider(widget.id));
              },
              child: const Text(AppStrings.save),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _openRenewDialog(BuildContext context) async {
    final catalog = ref.read(catalogProvider);
    final cycles = catalog.billingCycles.where((c) => c['active'] != false).toList();
    int? cycleId = cycles.isNotEmpty ? cycles.first['id'] as int? : null;
    final qtyController = TextEditingController(text: '1');

    await showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('续费'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            DropdownButtonFormField<int>(
              value: cycleId,
              decoration: const InputDecoration(labelText: '周期'),
              items: cycles
                  .map((e) => DropdownMenuItem<int>(
                        value: e['id'] as int?,
                        child: Text(e['name']?.toString() ?? '周期'),
                      ))
                  .toList(),
              onChanged: (value) => cycleId = value,
            ),
            TextField(
              controller: qtyController,
              keyboardType: TextInputType.number,
              decoration: const InputDecoration(labelText: '数量'),
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context), child: const Text(AppStrings.cancel)),
          TextButton(
            onPressed: () async {
              final qty = int.tryParse(qtyController.text.trim()) ?? 1;
              final cycle = cycles.firstWhere((e) => e['id'] == cycleId, orElse: () => {});
              final months = (cycle['months'] ?? 1) * qty;
              try {
                final res = await ref.read(vpsRepositoryProvider).createRenewOrder(widget.id, {
                  'duration_months': months,
                });
                if (context.mounted) Navigator.pop(context);
                final orderId = res['order']?['id'] ?? res['order_id'] ?? res['orderId'] ?? res['id'];
                if (orderId != null) {
                  context.go('/console/orders/$orderId');
                } else if (context.mounted) {
                  ScaffoldMessenger.of(context)
                      .showSnackBar(const SnackBar(content: Text('已生成续费订单')));
                }
              } on DioException catch (e) {
                if (e.response?.statusCode == 409) {
                  final data = e.response?.data ?? {};
                  final orderId = data['order']?['id'] ?? data['order_id'] ?? data['orderId'];
                  if (!context.mounted) return;
                  await _showConflictDialog(
                    context,
                    title: '已有进行中的续费订单',
                    message: data['message']?.toString() ?? '已有进行中的续费订单',
                    orderId: orderId,
                  );
                  return;
                }
                rethrow;
              }
            },
            child: const Text(AppStrings.confirm),
          ),
        ],
      ),
    );
  }

  Future<void> _openResizeDialog(BuildContext context) async {
    final catalog = ref.read(catalogProvider);
    final packages = catalog.packages;
    final detail = ref.read(vpsDetailProvider).detail ?? {};
    final currentPackageId = detail['package_id'];
    final currentPackage = packages.firstWhere(
      (p) => p['id'] == currentPackageId,
      orElse: () => {},
    );
    final planGroupId = currentPackage['plan_group_id'] ?? currentPackage['PlanGroupID'];
    final packageOptions = packages
        .where((p) => (p['plan_group_id'] ?? p['PlanGroupID']) == planGroupId)
        .where((p) => p['active'] != false && p['visible'] != false)
        .toList();

    int? targetPackageId;
    bool resetAddons = false;
    final addCoresController = TextEditingController(text: '0');
    final addMemController = TextEditingController(text: '0');
    final addDiskController = TextEditingController(text: '0');
    final addBwController = TextEditingController(text: '0');
    String scheduleMode = 'now';
    final scheduledAtController = TextEditingController();
    Map<String, dynamic>? quote;

    Future<void> fetchQuote(StateSetter setModalState) async {
      if (targetPackageId == null) return;
      final payload = {
        'target_package_id': targetPackageId,
        'reset_addons': resetAddons,
        'spec': {
          'add_cores': resetAddons ? 0 : int.tryParse(addCoresController.text.trim()) ?? 0,
          'add_mem_gb': resetAddons ? 0 : int.tryParse(addMemController.text.trim()) ?? 0,
          'add_disk_gb': resetAddons ? 0 : int.tryParse(addDiskController.text.trim()) ?? 0,
          'add_bw_mbps': resetAddons ? 0 : int.tryParse(addBwController.text.trim()) ?? 0,
        },
      };
      final res = await ref.read(vpsRepositoryProvider).quoteResize(widget.id, payload);
      setModalState(() {
        quote = res['quote'] is Map<String, dynamic> ? res['quote'] : res;
      });
    }

    await showDialog(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setModalState) => AlertDialog(
          title: const Text('升降配'),
          content: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                DropdownButtonFormField<int>(
                  value: targetPackageId,
                  decoration: const InputDecoration(labelText: '目标套餐'),
                  items: packageOptions
                      .map((e) => DropdownMenuItem<int>(
                            value: e['id'] as int?,
                            child: Text(e['name']?.toString() ?? '套餐'),
                          ))
                      .toList(),
                  onChanged: (value) => setModalState(() => targetPackageId = value),
                ),
                SwitchListTile(
                  title: const Text('重置附加项'),
                  value: resetAddons,
                  onChanged: (v) => setModalState(() => resetAddons = v),
                ),
                TextField(
                  controller: addCoresController,
                  keyboardType: TextInputType.number,
                  decoration: const InputDecoration(labelText: '追加 CPU 核心'),
                  enabled: !resetAddons,
                ),
                TextField(
                  controller: addMemController,
                  keyboardType: TextInputType.number,
                  decoration: const InputDecoration(labelText: '追加内存(GB)'),
                  enabled: !resetAddons,
                ),
                TextField(
                  controller: addDiskController,
                  keyboardType: TextInputType.number,
                  decoration: const InputDecoration(labelText: '追加磁盘(GB)'),
                  enabled: !resetAddons,
                ),
                TextField(
                  controller: addBwController,
                  keyboardType: TextInputType.number,
                  decoration: const InputDecoration(labelText: '追加带宽(Mbps)'),
                  enabled: !resetAddons,
                ),
                DropdownButtonFormField<String>(
                  value: scheduleMode,
                  decoration: const InputDecoration(labelText: '执行方式'),
                  items: const [
                    DropdownMenuItem(value: 'now', child: Text('立即执行')),
                    DropdownMenuItem(value: 'scheduled', child: Text('定时执行')),
                  ],
                  onChanged: (value) => setModalState(() => scheduleMode = value ?? 'now'),
                ),
                if (scheduleMode == 'scheduled')
                  TextField(
                    controller: scheduledAtController,
                    decoration: const InputDecoration(labelText: '执行时间 (YYYY-MM-DD HH:mm:ss)'),
                  ),
                const SizedBox(height: 8),
                if (quote != null)
                  Text('本周期需支付: ${MoneyFormatter.format(_toDouble(quote?['charge_amount'] ?? quote?['chargeAmount']))}'),
              ],
            ),
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(context), child: const Text(AppStrings.cancel)),
            TextButton(onPressed: () => fetchQuote(setModalState), child: const Text('计算报价')),
            TextButton(
              onPressed: () async {
                if (targetPackageId == null) {
                  ScaffoldMessenger.of(context)
                      .showSnackBar(const SnackBar(content: Text('请选择目标套餐')));
                  return;
                }
                final payload = {
                  'target_package_id': targetPackageId,
                  'reset_addons': resetAddons,
                  'spec': {
                    'add_cores': resetAddons ? 0 : int.tryParse(addCoresController.text.trim()) ?? 0,
                    'add_mem_gb': resetAddons ? 0 : int.tryParse(addMemController.text.trim()) ?? 0,
                    'add_disk_gb': resetAddons ? 0 : int.tryParse(addDiskController.text.trim()) ?? 0,
                    'add_bw_mbps': resetAddons ? 0 : int.tryParse(addBwController.text.trim()) ?? 0,
                  },
                };
                if (scheduleMode == 'scheduled' && scheduledAtController.text.trim().isNotEmpty) {
                  payload['scheduled_at'] = scheduledAtController.text.trim();
                }
                try {
                  final res = await ref.read(vpsRepositoryProvider).createResizeOrder(widget.id, payload);
                  if (context.mounted) Navigator.pop(context);
                  final orderId = res['order']?['id'] ?? res['order_id'] ?? res['orderId'] ?? res['id'];
                  if (orderId != null) {
                    context.go('/console/orders/$orderId');
                  } else if (context.mounted) {
                    ScaffoldMessenger.of(context)
                        .showSnackBar(const SnackBar(content: Text('已生成升降配订单')));
                  }
                } on DioException catch (e) {
                  if (e.response?.statusCode == 409) {
                    final data = e.response?.data ?? {};
                    final orderId = data['order']?['id'] ?? data['order_id'] ?? data['orderId'];
                    if (!context.mounted) return;
                    await _showConflictDialog(
                      context,
                      title: '已有进行中的升降配任务/订单',
                      message: data['message']?.toString() ?? '已有进行中的升降配任务/订单',
                      orderId: orderId,
                    );
                    return;
                  }
                  rethrow;
                }
              },
              child: const Text(AppStrings.confirm),
            ),
          ],
        ),
      ),
    );
  }
  Future<void> _openReinstallDialog(BuildContext context) async {
    final catalog = ref.read(catalogProvider);
    final images = catalog.systemImages;
    int? templateId = images.isNotEmpty ? images.first['id'] as int? : null;
    final passwordController = TextEditingController();

    await showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('重装系统'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            DropdownButtonFormField<int>(
              value: templateId,
              decoration: const InputDecoration(labelText: '系统镜像'),
              items: images
                  .map((e) => DropdownMenuItem<int>(
                        value: e['id'] as int?,
                        child: Text(e['name']?.toString() ?? '镜像'),
                      ))
                  .toList(),
              onChanged: (value) => templateId = value,
            ),
            TextField(
              controller: passwordController,
              decoration: const InputDecoration(labelText: '重装密码'),
              obscureText: true,
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context), child: const Text(AppStrings.cancel)),
          TextButton(
            onPressed: () async {
              if (templateId == null) {
                ScaffoldMessenger.of(context)
                    .showSnackBar(const SnackBar(content: Text('请选择镜像')));
                return;
              }
              await _operate(
                context,
                () => ref.read(vpsRepositoryProvider).resetOs(widget.id, {
                  'host_id': widget.id,
                  'template_id': templateId,
                  'password': passwordController.text.trim(),
                }),
                '已放入重装队列',
              );
              if (context.mounted) Navigator.pop(context);
            },
            child: const Text(AppStrings.confirm),
          ),
        ],
      ),
    );
  }

  Future<void> _openResetPasswordDialog(BuildContext context) async {
    final detail = ref.read(vpsDetailProvider).detail ?? {};
    final access = _parseAccessInfo(detail);
    final passwordController = TextEditingController(text: access.osPassword);
    await showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('重置密码'),
        content: TextField(
          controller: passwordController,
          decoration: const InputDecoration(labelText: '新密码'),
          obscureText: true,
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context), child: const Text(AppStrings.cancel)),
          TextButton(
            onPressed: () async {
              if (passwordController.text.trim().isEmpty) {
                ScaffoldMessenger.of(context)
                    .showSnackBar(const SnackBar(content: Text('请输入密码')));
                return;
              }
              final validation = _validateOsPassword(passwordController.text.trim());
              if (validation != null) {
                ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(validation)));
                return;
              }
              await _operate(
                context,
                () => ref.read(vpsRepositoryProvider).resetOsPassword(widget.id, {
                  'password': passwordController.text.trim(),
                }),
                '密码已更新',
              );
              if (context.mounted) Navigator.pop(context);
            },
            child: const Text(AppStrings.confirm),
          ),
        ],
      ),
    );
  }

  Future<void> _openRefundDialog(BuildContext context) async {
    final reasonController = TextEditingController();
    await showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('退款申请'),
        content: TextField(
          controller: reasonController,
          decoration: const InputDecoration(labelText: '退款原因'),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context), child: const Text(AppStrings.cancel)),
          TextButton(
            onPressed: () async {
              if (reasonController.text.trim().isEmpty) {
                ScaffoldMessenger.of(context)
                    .showSnackBar(const SnackBar(content: Text('请填写退款原因')));
                return;
              }
              final res = await ref.read(vpsRepositoryProvider).requestRefund(widget.id, {
                'reason': reasonController.text.trim(),
              });
              final orderId = res['order']?['id'] ?? res['order_id'] ?? res['orderId'];
              if (context.mounted) {
                Navigator.pop(context);
                final message = orderId != null
                    ? '已提交退款申请，订单ID: $orderId'
                    : '已提交退款申请';
                ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(message)));
              }
            },
            child: const Text(AppStrings.confirm),
          ),
        ],
      ),
    );
  }

  Future<void> _submitEmergencyRenew() async {
    await _operate(
      context,
      () => ref.read(vpsRepositoryProvider).emergencyRenew(widget.id),
      '已提交紧急续费',
    );
  }

  Future<void> _operate(
    BuildContext context,
    Future<void> Function() action,
    String successMessage,
  ) async {
    try {
      await action();
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(successMessage)));
        ref.read(vpsDetailProvider.notifier).fetch(widget.id);
      }
    } catch (e) {
      if (context.mounted) {
        final message = _extractErrorMessage(e);
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(message)));
      }
    }
  }

  String _extractErrorMessage(dynamic error) {
    if (error is DioException) {
      final data = error.response?.data;
      if (data is Map<String, dynamic>) {
        final msg = data['error']?.toString() ?? data['message']?.toString();
        if (msg != null && msg.isNotEmpty) return msg;
      }
      if (data is String && data.isNotEmpty) {
        final msg = _extractMsgFromString(data);
        if (msg != null) return msg;
      }
      final fallback = error.message;
      if (fallback != null && fallback.isNotEmpty) return fallback;
    }
    final text = error.toString();
    final msg = _extractMsgFromString(text);
    return msg ?? text;
  }

  String? _extractMsgFromString(String text) {
    if (text.contains('Parameter verification failed')) {
      return '密码不符合要求，请使用更复杂的密码';
    }
    final match = RegExp(r'"msg"\\s*:\\s*"([^"]+)"').firstMatch(text);
    if (match != null) return match.group(1);
    final errIndex = text.indexOf('automation error:');
    if (errIndex >= 0) {
      final end = text.indexOf('|', errIndex);
      if (end > errIndex) {
        return text.substring(errIndex, end).trim();
      }
      return text.substring(errIndex).trim();
    }
    return null;
  }

  String? _validateOsPassword(String value) {
    if (value.length < 8 || value.length > 20) {
      return '系统密码长度需为 8-20 位';
    }
    final hasLower = RegExp(r'[a-z]').hasMatch(value);
    final hasUpper = RegExp(r'[A-Z]').hasMatch(value);
    final hasDigit = RegExp(r'\d').hasMatch(value);
    final hasSpecial =
        RegExp(r'[!@#\$%\^&*()_+\-=\[\]{}|\\:;"<>,.?/`~]').hasMatch(value);
    final categories =
        <bool>[hasLower, hasUpper, hasDigit, hasSpecial].where((e) => e).length;
    if (categories < 3) {
      return '系统密码需包含大小写字母、数字、特殊符号中的至少三类';
    }
    return null;
  }

  Future<void> _copyText(String text, String name) async {
    await Clipboard.setData(ClipboardData(text: text));
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('已复制$name')));
  }

  Future<void> _openPanel() async {
    final token = StorageService.instance.getAccessToken();
    final url = _buildVpsUrl('panel', token: token);
    await _launchUrl(url);
  }

  Future<void> _openVnc() async {
    final token = StorageService.instance.getAccessToken();
    final url = _buildVpsUrl('vnc', token: token);
    await _launchUrl(url);
  }

  Future<void> _openRemote() async {
    final detail = ref.read(vpsDetailProvider).detail ?? {};
    final access = _parseAccessInfo(detail);
    final systemLabel = _resolveSystemLabel(detail);
    final isWindows = _isWindowsOS(systemLabel);
    final remote = _splitRemote(access);
    if (remote.host.isEmpty || remote.port.isEmpty) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('远程地址为空')));
      }
      return;
    }

    final username = isWindows ? 'administrator' : 'root';
    final password = access.osPassword;
    final platform = getPlatformUtils();

    if (platform.isMobile) {
      final url = isWindows
          ? _buildRdpLink(remote.host, remote.port, username, password)
          : _buildSshLink(remote.host, remote.port, username);
      await _launchUrl(url);
      return;
    }

    if (platform.isWeb) {
      if (isWindows) {
        _downloadRdpFile(remote.host, remote.port, username);
      } else {
        _downloadSshFile(remote.host, remote.port, username, password);
      }
      return;
    }

    if (platform.isWindows) {
      final launcher = getDesktopLauncher();
      if (isWindows) {
        await launcher.launchWindowsRdp(
          host: remote.host,
          port: remote.port,
          username: username,
          password: password,
        );
      } else {
        await launcher.launchWindowsSsh(
          host: remote.host,
          port: remote.port,
          username: username,
        );
      }
      return;
    }

    if (isWindows) {
      _downloadRdpFile(remote.host, remote.port, username);
    } else {
      _downloadSshFile(remote.host, remote.port, username, password);
    }
  }

  String _buildVpsUrl(String action, {String? token}) {
    var base = ApiClient.instance.dio.options.baseUrl;
    base = base.trim();
    if (base.endsWith('/')) {
      base = base.substring(0, base.length - 1);
    }
    if (base.endsWith('/api')) {
      base = base.substring(0, base.length - 4);
    }
    final query = token != null ? '?token=${Uri.encodeComponent(token)}' : '';
    return '$base/api/v1/vps/${widget.id}/$action$query';
  }

  _RemoteHost _splitRemote(_AccessInfo access) {
    final raw = access.remoteIp.trim();
    if (raw.isEmpty) {
      return _RemoteHost('', access.remotePort);
    }
    final lastColon = raw.lastIndexOf(':');
    if (lastColon > 0 && lastColon < raw.length - 1) {
      return _RemoteHost(raw.substring(0, lastColon), raw.substring(lastColon + 1));
    }
    return _RemoteHost(raw, access.remotePort);
  }

  String _buildRdpLink(String host, String port, String username, String password) {
    final remote = '$host:$port';
    final encodedRemote = Uri.encodeComponent('full address=s:$remote');
    final encodedUser = Uri.encodeComponent('username=s:$username');
    final encodedPass = Uri.encodeComponent('password=s:$password');
    return 'rdp:$encodedRemote&$encodedUser&$encodedPass';
  }

  String _buildSshLink(String host, String port, String username) {
    return 'ssh://$username@$host:$port';
  }

  void _downloadRdpFile(String host, String port, String username) {
    final content = [
      'full address:s:$host:$port',
      'username:s:$username',
      'prompt for credentials:i:1',
    ].join('\r\n');
    _downloadTextFile('connection.rdp', content, 'application/rdp');
  }

  void _downloadSshFile(String host, String port, String username, String password) {
    final content = [
      '@echo off',
      'ssh $username@$host -p $port',
      'plink -ssh $username@$host -P $port -pw $password',
      'pause',
    ].join('\r\n');
    _downloadTextFile('connection.bat', content, 'application/bat');
  }

  void _downloadTextFile(String filename, String content, String mimeType) {
    downloadTextFile(filename, content, mimeType);
  }

  Future<void> _launchUrl(String url) async {
    final uri = Uri.parse(url);
    if (await canLaunchUrl(uri)) {
      await launchUrl(uri, mode: LaunchMode.platformDefault);
    } else if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('无法打开链接')));
    }
  }

  Future<void> _showConflictDialog(
    BuildContext context, {
    required String title,
    required String message,
    int? orderId,
  }) async {
    await showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(title),
        content: Text(message),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context), child: const Text('我知道了')),
          TextButton(
            onPressed: () {
              Navigator.pop(context);
              if (orderId != null) {
                context.go('/console/orders/$orderId');
              } else {
                context.go('/console/orders');
              }
            },
            child: Text(orderId != null ? '去订单详情' : '去订单列表'),
          ),
        ],
      ),
    );
  }
}

class _SpecInfo {
  final int cpu;
  final int memoryGb;
  final int diskGb;
  final int bandwidthMbps;

  const _SpecInfo({
    required this.cpu,
    required this.memoryGb,
    required this.diskGb,
    required this.bandwidthMbps,
  });
}

class _AccessInfo {
  final String remoteIp;
  final String remotePort;
  final String osPassword;
  final String panelPassword;
  final String vncPassword;

  const _AccessInfo({
    required this.remoteIp,
    required this.remotePort,
    required this.osPassword,
    required this.panelPassword,
    required this.vncPassword,
  });
}

class _RemoteHost {
  final String host;
  final String port;
  const _RemoteHost(this.host, this.port);
}

class _PerfGauge extends StatelessWidget {
  final double value;

  const _PerfGauge({required this.value});

  @override
  Widget build(BuildContext context) {
    final label = value >= 80
        ? '优'
        : value >= 60
            ? '良'
            : value >= 40
                ? '中'
                : '差';
    final color = value >= 80
        ? AppColors.success
        : value >= 60
            ? AppColors.primary
            : value >= 40
                ? AppColors.warning
                : AppColors.danger;

    return Column(
      children: [
        CustomPaint(
          size: const Size(120, 60),
          painter: _GaugePainter(value: value, color: color),
        ),
        const SizedBox(height: 6),
        Text(label, style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold, color: color)),
        const Text('系统表现', style: TextStyle(fontSize: 12, color: AppColors.gray500)),
      ],
    );
  }
}

class _GaugePainter extends CustomPainter {
  final double value;
  final Color color;

  _GaugePainter({required this.value, required this.color});

  @override
  void paint(Canvas canvas, Size size) {
    final center = Offset(size.width / 2, size.height);
    final radius = size.width / 2;

    final basePaint = Paint()
      ..color = AppColors.gray200
      ..style = PaintingStyle.stroke
      ..strokeWidth = 8;

    canvas.drawArc(Rect.fromCircle(center: center, radius: radius), -math.pi, math.pi, false, basePaint);

    final sweep = (value.clamp(0, 100) / 100) * math.pi;
    final valuePaint = Paint()
      ..color = color
      ..style = PaintingStyle.stroke
      ..strokeWidth = 8;

    canvas.drawArc(Rect.fromCircle(center: center, radius: radius), -math.pi, sweep, false, valuePaint);

    final needleAngle = -math.pi + sweep;
    final needlePaint = Paint()
      ..color = AppColors.gray700
      ..strokeWidth = 2;

    final needleEnd = Offset(
      center.dx + (radius - 6) * math.cos(needleAngle),
      center.dy + (radius - 6) * math.sin(needleAngle),
    );
    canvas.drawLine(center, needleEnd, needlePaint);
    canvas.drawCircle(center, 3, needlePaint);
  }

  @override
  bool shouldRepaint(covariant _GaugePainter oldDelegate) {
    return oldDelegate.value != value || oldDelegate.color != color;
  }
}
