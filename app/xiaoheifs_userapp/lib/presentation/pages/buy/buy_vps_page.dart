
import 'dart:math' as math;
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../core/constants/app_colors.dart';
import '../../../core/utils/money_formatter.dart';
import '../../providers/catalog_provider.dart';
import '../../providers/cart_provider.dart';
import '../../providers/order_provider.dart';
import '../../widgets/common/empty_state.dart';

class BuyVpsPage extends ConsumerStatefulWidget {
  const BuyVpsPage({super.key});

  @override
  ConsumerState<BuyVpsPage> createState() => _BuyVpsPageState();
}

class _BuyVpsPageState extends ConsumerState<BuyVpsPage> {
  int? _goodsTypeId;
  int? _regionId;
  int? _planGroupId;
  int? _packageId;
  int? _systemId;
  int? _billingCycleId;
  int _cycleQty = 1;
  int _qty = 1;
  int _addCores = 0;
  int _addMem = 0;
  int _addDisk = 0;
  int _addBw = 0;

  List<Map<String, dynamic>> _systemImages = [];
  bool _loadingImages = false;
  int? _loadedPlanGroupId;
  ProviderSubscription<CatalogState>? _catalogSub;
  DateTime? _catalogLoadingSince;
  DateTime? _lastCatalogForceFetchAt;

  bool _isCatalogEmpty(CatalogState s) {
    return s.goodsTypes.isEmpty &&
        s.regions.isEmpty &&
        s.planGroups.isEmpty &&
        s.packages.isEmpty &&
        s.billingCycles.isEmpty;
  }

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      ref.read(catalogProvider.notifier).fetchCatalog(force: true);
    });
    _catalogSub = ref.listenManual<CatalogState>(catalogProvider, (prev, next) {
      if (!mounted) return;
      if (next.loading && !(prev?.loading ?? false)) {
        _catalogLoadingSince = DateTime.now();
      } else if (!next.loading) {
        _catalogLoadingSince = null;
      }
      _syncDefaults(next);
    });
  }

  @override
  void dispose() {
    _catalogSub?.close();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final catalog = ref.watch(catalogProvider);
    _ensureCatalogFetch(catalog);

    // Align with frontend BuyVps.vue:
    // no full-page loading gate; render page as soon as any catalog data exists.
    if (catalog.error != null && _isCatalogEmpty(catalog)) {
      return EmptyState(
        message: '加载购买配置失败',
        actionLabel: '重试',
        onAction: () => ref.read(catalogProvider.notifier).fetchCatalog(force: true),
      );
    }

    final goodsTypes = catalog.goodsTypes.where((g) => g['active'] != false).toList();
    final regions = catalog.regions.where((r) {
      if (r['active'] == false) return false;
      if (_goodsTypeId == null) return false;
      return '${r['goods_type_id']}' == '$_goodsTypeId';
    }).toList();
    final planGroups = catalog.planGroups.where((g) {
      if (g['active'] == false || g['visible'] == false) return false;
      if (_regionId == null) return false;
      if (_goodsTypeId != null && '${g['goods_type_id']}' != '$_goodsTypeId') return false;
      return g['region_id'] == _regionId;
    }).toList();
    final packages = catalog.packages.where((p) {
      if (p['active'] == false || p['visible'] == false) return false;
      if (_planGroupId == null) return false;
      final groupId = p['plan_group_id'] ?? p['PlanGroupID'] ?? p['planGroupId'];
      return '$groupId' == '$_planGroupId';
    }).toList();
    final billingCycles = catalog.billingCycles.where((c) => c['active'] != false).toList();

    _scheduleAutoSelect(
      regions: regions,
      planGroups: planGroups,
      packages: packages,
      billingCycles: billingCycles,
    );

    final selectedRegion = _safeFirstWhere(regions, (r) => r['id'] == _regionId);
    final selectedPlanGroup = _safeFirstWhere(planGroups, (g) => g['id'] == _planGroupId);
    final selectedPackage = _safeFirstWhere(packages, (p) => p['id'] == _packageId);
    final selectedCycle = _safeFirstWhere(billingCycles, (c) => c['id'] == _billingCycleId);
    final selectedSystem = _safeFirstWhere(_systemImages, (s) => s['id'] == _systemId);

    final addonRule = _buildAddonRule(selectedPlanGroup);
    _scheduleAddonNormalize(addonRule);
    final basePrice = _asDouble(selectedPackage['monthly_price']);
    final addonPrice = _computeAddonPrice(selectedPlanGroup);
    final cycleMultiplier = _cycleMultiplier(selectedCycle, _cycleQty);
    final total = (basePrice + addonPrice) * cycleMultiplier * _qty;

    return Scaffold(
      body: ListView(
        padding: const EdgeInsets.fromLTRB(16, 16, 16, 24),
        children: [
          _buildHeader(isLoading: catalog.loading),
          const SizedBox(height: 16),
          _buildSteps(_currentStep()),
          const SizedBox(height: 16),
          _buildSectionCard(
            title: '基础配置',
            child: Column(
              children: [
                _buildDropdown('商品类型', goodsTypes, _goodsTypeId, (v) {
                  setState(() {
                    _goodsTypeId = v;
                    _regionId = null;
                    _planGroupId = null;
                    _packageId = null;
                    _systemId = null;
                    _systemImages = [];
                  });
                }),
                _buildDropdown('地区', regions, _regionId, (v) {
                  setState(() {
                    _regionId = v;
                    _planGroupId = null;
                    _packageId = null;
                    _systemId = null;
                    _systemImages = [];
                  });
                }),
                _buildPlanGroupSelector(planGroups),
                const SizedBox(height: 12),
                _buildPackageSelector(packages),
                const SizedBox(height: 12),
                _buildSystemImageSelector(),
              ],
            ),
          ),
          const SizedBox(height: 16),
          _buildSectionCard(
            title: '附加配置',
            child: Column(
              children: [
                _buildAddonSlider(
                  title: 'CPU 核心',
                  value: _addCores,
                  rule: addonRule,
                  unitKey: 'unit_core',
                  minKey: 'add_core_min',
                  maxKey: 'add_core_max',
                  stepKey: 'add_core_step',
                  onChanged: (v) => setState(() => _addCores = v),
                  suffix: '核',
                ),
                const SizedBox(height: 12),
                _buildAddonSlider(
                  title: '内存',
                  value: _addMem,
                  rule: addonRule,
                  unitKey: 'unit_mem',
                  minKey: 'add_mem_min',
                  maxKey: 'add_mem_max',
                  stepKey: 'add_mem_step',
                  onChanged: (v) => setState(() => _addMem = v),
                  suffix: 'GB',
                ),
                const SizedBox(height: 12),
                _buildAddonSlider(
                  title: '磁盘空间',
                  value: _addDisk,
                  rule: addonRule,
                  unitKey: 'unit_disk',
                  minKey: 'add_disk_min',
                  maxKey: 'add_disk_max',
                  stepKey: 'add_disk_step',
                  onChanged: (v) => setState(() => _addDisk = v),
                  suffix: 'GB',
                ),
                const SizedBox(height: 12),
                _buildAddonSlider(
                  title: '带宽',
                  value: _addBw,
                  rule: addonRule,
                  unitKey: 'unit_bw',
                  minKey: 'add_bw_min',
                  maxKey: 'add_bw_max',
                  stepKey: 'add_bw_step',
                  onChanged: (v) => setState(() => _addBw = v),
                  suffix: 'Mbps',
                ),
              ],
            ),
          ),
          const SizedBox(height: 16),
          _buildSectionCard(
            title: '购买周期',
            child: Column(
              children: [
                _buildDropdown('计费周期', billingCycles, _billingCycleId, (v) {
                  setState(() => _billingCycleId = v);
                },
                    display: (item) =>
                        '${item['name'] ?? ''} (${item['months'] ?? 1}个月)'),
                Row(
                  children: [
                    Expanded(
                      child: _buildNumberStepper(
                        label: '周期数量',
                        value: _cycleQty,
                        min: _cycleQtyMin(selectedCycle),
                        max: _cycleQtyMax(selectedCycle),
                        onChanged: (v) => setState(() => _cycleQty = v),
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: _buildNumberStepper(
                        label: '购买数量',
                        value: _qty,
                        min: 1,
                        max: 10,
                        onChanged: (v) => setState(() => _qty = v),
                      ),
                    ),
                  ],
                ),
              ],
            ),
          ),
          const SizedBox(height: 16),
          _buildSummaryCard(
            region: selectedRegion,
            planGroup: selectedPlanGroup,
            package: selectedPackage,
            system: selectedSystem,
            cycle: selectedCycle,
            cycleQty: _cycleQty,
          ),
          const SizedBox(height: 16),
          _buildPriceCard(
            basePrice: basePrice,
            addonPrice: addonPrice,
            cycleMultiplier: cycleMultiplier,
            qty: _qty,
            total: total,
          ),
          const SizedBox(height: 16),
          Row(
            children: [
              Expanded(
                child: OutlinedButton(
                  onPressed: _canCheckout ? () => _addToCart(context) : null,
                  style: OutlinedButton.styleFrom(
                    padding: const EdgeInsets.symmetric(vertical: 14),
                    shape: const StadiumBorder(),
                    side: const BorderSide(color: AppColors.primary),
                  ),
                  child: const Text('加入购物车'),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: ElevatedButton(
                  onPressed: _canCheckout ? () => _createOrderNow(context) : null,
                  style: ElevatedButton.styleFrom(
                    padding: const EdgeInsets.symmetric(vertical: 14),
                    shape: const StadiumBorder(),
                    backgroundColor: AppColors.primary,
                  ),
                  child: const Text('立即下单'),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  void _ensureCatalogFetch(CatalogState catalog) {
    final now = DateTime.now();
    final empty = _isCatalogEmpty(catalog);
    final lastKick = _lastCatalogForceFetchAt;
    final canKick = lastKick == null || now.difference(lastKick).inSeconds >= 3;

    if (empty && !catalog.loading && canKick) {
      _lastCatalogForceFetchAt = now;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) return;
        ref.read(catalogProvider.notifier).fetchCatalog(force: true);
      });
      return;
    }

    if (empty && catalog.loading && canKick && _catalogLoadingSince != null) {
      final loadingSeconds = now.difference(_catalogLoadingSince!).inSeconds;
      if (loadingSeconds >= 12) {
        _lastCatalogForceFetchAt = now;
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (!mounted) return;
          ref.read(catalogProvider.notifier).fetchCatalog(force: true);
        });
      }
    }
  }

  Widget _buildHeader({bool isLoading = false}) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                '购买 VPS',
                style: TextStyle(fontSize: 20, fontWeight: FontWeight.w600),
              ),
              SizedBox(height: 4),
              Text(
                '按需选择资源配置并自动计算价格',
                style: TextStyle(fontSize: 13, color: AppColors.gray500),
              ),
            ],
          ),
        ),
        if (isLoading)
          const Padding(
            padding: EdgeInsets.only(top: 2),
            child: SizedBox(
              width: 18,
              height: 18,
              child: CircularProgressIndicator(strokeWidth: 2),
            ),
          ),
      ],
    );
  }

  Widget _buildSteps(int step) {
    final steps = const ['商品类型', '地区', '线路', '套餐', '系统镜像', '确认'];
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: AppColors.darkSurface.withOpacity(0.35),
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: AppColors.gray700.withOpacity(0.4)),
      ),
      child: Row(
        children: steps.asMap().entries.map((entry) {
          final index = entry.key;
          final title = entry.value;
          final active = index <= step;
          return Expanded(
            child: Row(
              children: [
                Container(
                  width: 18,
                  height: 18,
                  decoration: BoxDecoration(
                    color: active ? AppColors.primary : AppColors.gray700,
                    shape: BoxShape.circle,
                  ),
                  alignment: Alignment.center,
                  child: Text(
                    '${index + 1}',
                    style: const TextStyle(fontSize: 11, color: Colors.white),
                  ),
                ),
                const SizedBox(width: 6),
                Expanded(
                  child: Text(
                    title,
                    style: TextStyle(
                      fontSize: 12,
                      color: active ? Colors.white : AppColors.gray500,
                    ),
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                if (index != steps.length - 1)
                  Container(
                    width: 12,
                    height: 1,
                    margin: const EdgeInsets.symmetric(horizontal: 4),
                    color: AppColors.gray600,
                  ),
              ],
            ),
          );
        }).toList(),
      ),
    );
  }

  Widget _buildSectionCard({required String title, required Widget child}) {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppColors.darkSurface.withOpacity(0.6),
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: AppColors.gray700.withOpacity(0.4)),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withOpacity(0.12),
            blurRadius: 20,
            offset: const Offset(0, 10),
          ),
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(title, style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w600)),
          const SizedBox(height: 12),
          child,
        ],
      ),
    );
  }

  Widget _buildDropdown(
    String label,
    List<Map<String, dynamic>> items,
    int? value,
    void Function(int?) onChanged, {
    String Function(Map<String, dynamic>)? display,
  }) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: DropdownButtonFormField<int>(
        value: value,
        decoration: InputDecoration(labelText: label),
        items: items
            .map((e) => DropdownMenuItem<int>(
                  value: e['id'] as int?,
                  child: Text(display?.call(e) ?? (e['name']?.toString() ?? '选项')),
                ))
            .toList(),
        onChanged: items.isEmpty ? null : onChanged,
      ),
    );
  }

  Widget _buildPlanGroupSelector(List<Map<String, dynamic>> groups) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text('线路选择', style: TextStyle(fontSize: 13, color: AppColors.gray400)),
        const SizedBox(height: 8),
        Wrap(
          spacing: 10,
          runSpacing: 10,
          children: groups.map((group) {
            final disabled = _isPlanGroupDisabled(group);
            final selected = group['id'] == _planGroupId;
            return ChoiceChip(
              label: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(group['name']?.toString() ?? '线路'),
                  if (_capacityLabel(group) != null) ...[
                    const SizedBox(width: 6),
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                      decoration: BoxDecoration(
                        color: _capacityColor(group).withOpacity(0.2),
                        borderRadius: BorderRadius.circular(10),
                      ),
                      child: Text(
                        _capacityLabel(group) ?? '',
                        style: TextStyle(fontSize: 11, color: _capacityColor(group)),
                      ),
                    ),
                  ]
                ],
              ),
              selected: selected,
              onSelected: disabled
                  ? null
                  : (_) {
                      setState(() {
                        _planGroupId = group['id'] as int?;
                        _packageId = null;
                        _systemId = null;
                        _systemImages = [];
                      });
                      _fetchSystemImages(_planGroupId);
                    },
              selectedColor: AppColors.primary.withOpacity(0.2),
              backgroundColor: AppColors.darkSurface.withOpacity(0.6),
              labelStyle: TextStyle(color: selected ? Colors.white : AppColors.gray300),
              side: BorderSide(color: selected ? AppColors.primary : AppColors.gray700),
            );
          }).toList(),
        ),
      ],
    );
  }
  Widget _buildPackageSelector(List<Map<String, dynamic>> packages) {
    if (_planGroupId == null) {
      return const SizedBox.shrink();
    }
    if (packages.isEmpty) {
      return const EmptyState(message: '暂无可用套餐', icon: Icons.inventory_2_outlined);
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text('套餐选择', style: TextStyle(fontSize: 13, color: AppColors.gray400)),
        const SizedBox(height: 8),
        SizedBox(
          height: 190,
          child: ListView.separated(
            scrollDirection: Axis.horizontal,
            itemCount: packages.length,
            separatorBuilder: (_, __) => const SizedBox(width: 12),
            itemBuilder: (context, index) {
              final pkg = packages[index];
              final disabled = _isPackageDisabled(pkg);
              final selected = pkg['id'] == _packageId;
              return GestureDetector(
                onTap: disabled
                    ? null
                    : () {
                        setState(() => _packageId = pkg['id'] as int?);
                      },
                child: Container(
                  width: 220,
                  padding: const EdgeInsets.all(14),
                  decoration: BoxDecoration(
                    color: selected
                        ? AppColors.primary.withOpacity(0.12)
                        : AppColors.darkSurface.withOpacity(0.7),
                    borderRadius: BorderRadius.circular(16),
                    border: Border.all(
                      color: selected ? AppColors.primary : AppColors.gray700,
                      width: selected ? 2 : 1,
                    ),
                    boxShadow: [
                      BoxShadow(
                        color: Colors.black.withOpacity(0.15),
                        blurRadius: 16,
                        offset: const Offset(0, 8),
                      ),
                    ],
                  ),
                  child: Opacity(
                    opacity: disabled ? 0.5 : 1,
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Row(
                          children: [
                            Expanded(
                              child: Text(
                                pkg['name']?.toString() ?? '套餐',
                                style: const TextStyle(
                                  fontSize: 14,
                                  fontWeight: FontWeight.w600,
                                ),
                                overflow: TextOverflow.ellipsis,
                              ),
                            ),
                            if (selected)
                              const Icon(Icons.check_circle, color: AppColors.success, size: 18),
                          ],
                        ),
                        const SizedBox(height: 12),
                        _buildSpecRow('CPU', '${pkg['cores'] ?? '-'} 核'),
                        _buildSpecRow('内存', '${pkg['memory_gb'] ?? '-'} GB'),
                        _buildSpecRow('磁盘', '${pkg['disk_gb'] ?? '-'} GB'),
                        _buildSpecRow('带宽', '${pkg['bandwidth_mbps'] ?? '-'} Mbps'),
                        const Spacer(),
                        Row(
                          children: [
                            const Text('¥', style: TextStyle(color: AppColors.danger)),
                            Text(
                              _asDouble(pkg['monthly_price']).toStringAsFixed(2),
                              style: const TextStyle(
                                fontSize: 18,
                                fontWeight: FontWeight.w700,
                                color: AppColors.danger,
                              ),
                            ),
                            const Text('/月', style: TextStyle(color: AppColors.gray400)),
                          ],
                        ),
                      ],
                    ),
                  ),
                ),
              );
            },
          ),
        ),
      ],
    );
  }

  Widget _buildSystemImageSelector() {
    if (_packageId == null) {
      return const SizedBox.shrink();
    }
    if (_loadingImages) {
      return const Padding(
        padding: EdgeInsets.symmetric(vertical: 8),
        child: LinearProgressIndicator(),
      );
    }
    if (_systemImages.isEmpty) {
      return const EmptyState(message: '暂无可用系统镜像', icon: Icons.desktop_windows_outlined);
    }
    return _buildDropdown(
      '系统镜像',
      _systemImages,
      _systemId,
      (v) => setState(() => _systemId = v),
      display: (item) => '${item['name'] ?? ''} (${item['type'] ?? ''})',
    );
  }

  Widget _buildAddonSlider({
    required String title,
    required int value,
    required Map<String, dynamic> rule,
    required String unitKey,
    required String minKey,
    required String maxKey,
    required String stepKey,
    required ValueChanged<int> onChanged,
    required String suffix,
  }) {
    final addon = _resolveAddonRule(rule[minKey], rule[maxKey], rule[stepKey], _getDefaultMax(maxKey));
    final disabled = addon['disabled'] == true;
    final min = _asDouble(addon['min']);
    final max = _asDouble(addon['max']);
    final step = _asDouble(addon['step']);
    final unitPrice = _asDouble(rule[unitKey]);
    final divisions = (!disabled && max > min && step > 0) ? ((max - min) / step).round() : null;
    final normalizedValue = disabled
        ? 0
        : _clampAddonValue(
            value,
            addon,
          );

    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: AppColors.darkSurface.withOpacity(0.6),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: AppColors.gray700.withOpacity(0.4)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Text(title, style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w500)),
              const Spacer(),
              if (disabled)
                const Text('已禁用', style: TextStyle(color: AppColors.gray500, fontSize: 12))
              else if (normalizedValue > 0)
                Text(
                  '+$normalizedValue$suffix · +¥${(normalizedValue * unitPrice).toStringAsFixed(2)}/月',
                  style: const TextStyle(color: AppColors.success, fontSize: 12),
                )
              else
                const Text('不添加', style: TextStyle(color: AppColors.gray500, fontSize: 12)),
            ],
          ),
          Slider(
            value: normalizedValue.toDouble().clamp(min, max),
            min: min,
            max: max,
            divisions: divisions,
            label: '$normalizedValue$suffix',
            onChanged: disabled
                ? null
                : (v) => onChanged(
                      _clampAddonValue(v.round(), addon),
                    ),
          ),
        ],
      ),
    );
  }

  Widget _buildSummaryCard({
    required Map<String, dynamic> region,
    required Map<String, dynamic> planGroup,
    required Map<String, dynamic> package,
    required Map<String, dynamic> system,
    required Map<String, dynamic> cycle,
    required int cycleQty,
  }) {
    return _buildSectionCard(
      title: '配置摘要',
      child: Column(
        children: [
          _summaryRow('地区', region['name']),
          _summaryRow('线路', planGroup['name']),
          _summaryRow('套餐', package['name']),
          if (package['port_num'] != null) _summaryRow('端口', '${package['port_num']}'),
          _summaryRow('系统', system['name']),
          _summaryRow('周期', '${cycle['name'] ?? '-'} × $cycleQty'),
          if (_hasAddons)
            _summaryRow(
              '附加',
              [
                if (_addCores > 0) '+${_addCores}核',
                if (_addMem > 0) '+${_addMem}G',
                if (_addDisk > 0) '+${_addDisk}G',
                if (_addBw > 0) '+${_addBw}M',
              ].join(' '),
            ),
        ],
      ),
    );
  }

  Widget _summaryRow(String label, dynamic value) {
    final textValue = value?.toString().isNotEmpty == true ? value.toString() : '-';
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Row(
        children: [
          Expanded(
            child: Text(
              label,
              style: const TextStyle(color: AppColors.gray400),
            ),
          ),
          SizedBox(
            width: 132,
            child: Text(
              textValue,
              style: const TextStyle(fontWeight: FontWeight.w600),
              textAlign: TextAlign.right,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildPriceCard({
    required double basePrice,
    required double addonPrice,
    required double cycleMultiplier,
    required int qty,
    required double total,
  }) {
    return _buildSectionCard(
      title: '价格预估',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _summaryRow('基础月费', MoneyFormatter.format(basePrice)),
          _summaryRow('附加配置', MoneyFormatter.format(addonPrice)),
          _summaryRow('计费倍数', cycleMultiplier.toStringAsFixed(2)),
          _summaryRow('数量', qty),
          const Divider(height: 20),
          Text(
            '总计 ${MoneyFormatter.format(total)}',
            style: const TextStyle(fontSize: 18, fontWeight: FontWeight.bold, color: AppColors.primary),
          ),
        ],
      ),
    );
  }

  Widget _buildSpecRow(String label, String value) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 6),
      child: Row(
        children: [
          Text(label, style: const TextStyle(color: AppColors.gray400, fontSize: 12)),
          const Spacer(),
          Text(value, style: const TextStyle(fontSize: 12)),
        ],
      ),
    );
  }

  Widget _buildNumberStepper({
    required String label,
    required int value,
    required int min,
    required int max,
    required ValueChanged<int> onChanged,
  }) {
    final canDec = value > min;
    final canInc = value < max;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: AppColors.darkSurface.withOpacity(0.6),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: AppColors.gray700.withOpacity(0.4)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(label, style: const TextStyle(fontSize: 12, color: AppColors.gray400)),
          const SizedBox(height: 6),
          Row(
            children: [
              IconButton(
                icon: const Icon(Icons.remove_circle_outline),
                onPressed: canDec ? () => onChanged(value - 1) : null,
              ),
              Text('$value', style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
              IconButton(
                icon: const Icon(Icons.add_circle_outline),
                onPressed: canInc ? () => onChanged(value + 1) : null,
              ),
            ],
          ),
        ],
      ),
    );
  }
  void _scheduleAutoSelect({
    required List<Map<String, dynamic>> regions,
    required List<Map<String, dynamic>> planGroups,
    required List<Map<String, dynamic>> packages,
    required List<Map<String, dynamic>> billingCycles,
  }) {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      bool changed = false;

      final regionExists = _regionId != null && regions.any((r) => r['id'] == _regionId);
      if (!regionExists && regions.isNotEmpty) {
        _regionId = regions.first['id'] as int?;
        _planGroupId = null;
        _packageId = null;
        _systemId = null;
        _systemImages = [];
        changed = true;
      }

      final planGroupExists =
          _planGroupId != null && planGroups.any((g) => g['id'] == _planGroupId);
      if (!planGroupExists && planGroups.isNotEmpty) {
        final next = planGroups.firstWhere(
          (g) => !_isPlanGroupDisabled(g),
          orElse: () => planGroups.first,
        );
        _planGroupId = next['id'] as int?;
        _packageId = null;
        _systemId = null;
        _systemImages = [];
        changed = true;
        _fetchSystemImages(_planGroupId);
      }

      final packageExists = _packageId != null && packages.any((p) => p['id'] == _packageId);
      if (!packageExists && packages.isNotEmpty) {
        final next = packages.firstWhere(
          (p) => !_isPackageDisabled(p),
          orElse: () => packages.first,
        );
        _packageId = next['id'] as int?;
        changed = true;
      }

      final cycleExists =
          _billingCycleId != null && billingCycles.any((c) => c['id'] == _billingCycleId);
      if (!cycleExists && billingCycles.isNotEmpty) {
        _billingCycleId = billingCycles.first['id'] as int?;
        changed = true;
      }

      if (changed) {
        setState(() {});
      }
    });
  }

  bool get _hasAddons => _addCores > 0 || _addMem > 0 || _addDisk > 0 || _addBw > 0;

  bool get _canCheckout => _packageId != null && _systemId != null;

  Map<String, dynamic> _safeFirstWhere(
    List<Map<String, dynamic>>? list,
    bool Function(Map<String, dynamic>) test,
  ) {
    if (list == null || list.isEmpty) return const {};
    return list.firstWhere(test, orElse: () => const {});
  }

  int _currentStep() {
    if (_goodsTypeId == null) return 0;
    if (_regionId == null) return 1;
    if (_planGroupId == null) return 2;
    if (_packageId == null) return 3;
    if (_systemId == null) return 4;
    return 5;
  }

  double _asDouble(dynamic value) {
    if (value is num) return value.toDouble();
    return double.tryParse(value?.toString() ?? '') ?? 0;
  }

  double _cycleMultiplier(Map<String, dynamic> cycle, int qty) {
    final base = _asDouble(cycle['multiplier'] ?? cycle['months'] ?? 1);
    return base * qty;
  }

  double _computeAddonPrice(Map<String, dynamic> planGroup) {
    return _addCores * _asDouble(planGroup['unit_core']) +
        _addMem * _asDouble(planGroup['unit_mem']) +
        _addDisk * _asDouble(planGroup['unit_disk']) +
        _addBw * _asDouble(planGroup['unit_bw']);
  }

  int _cycleQtyMin(Map<String, dynamic> cycle) {
    final min = cycle['min_qty'] ?? cycle['minQty'];
    if (min is int && min > 0) return min;
    return 1;
  }

  int _cycleQtyMax(Map<String, dynamic> cycle) {
    final max = cycle['max_qty'] ?? cycle['maxQty'];
    if (max is int && max > 0) return max;
    return 12;
  }

  Map<String, dynamic> _buildAddonRule(Map<String, dynamic> group) {
    return {
      'unit_core': group['unit_core'] ?? 0,
      'unit_mem': group['unit_mem'] ?? 0,
      'unit_disk': group['unit_disk'] ?? 0,
      'unit_bw': group['unit_bw'] ?? 0,
      'add_core_min': group['add_core_min'] ?? 0,
      'add_core_max': group['add_core_max'],
      'add_core_step': group['add_core_step'] ?? 1,
      'add_mem_min': group['add_mem_min'] ?? 0,
      'add_mem_max': group['add_mem_max'],
      'add_mem_step': group['add_mem_step'] ?? 1,
      'add_disk_min': group['add_disk_min'] ?? 0,
      'add_disk_max': group['add_disk_max'],
      'add_disk_step': group['add_disk_step'] ?? 10,
      'add_bw_min': group['add_bw_min'] ?? 0,
      'add_bw_max': group['add_bw_max'],
      'add_bw_step': group['add_bw_step'] ?? 10,
    };
  }

  Map<String, dynamic> _resolveAddonRule(
    dynamic minRaw,
    dynamic maxRaw,
    dynamic stepRaw,
    double fallbackMax,
  ) {
    final min = _asDouble(minRaw);
    final max = _asDouble(maxRaw);
    final step = math.max(1, _asDouble(stepRaw == null ? 1 : stepRaw)).toDouble();
    if (min == -1 || max == -1) {
      return {'disabled': true, 'min': 0.0, 'max': 0.0, 'step': 1.0};
    }
    final effectiveMin = min > 0 ? min : 0.0;
    final effectiveMax = max > 0 ? max : fallbackMax;
    return {
      'disabled': false,
      'min': effectiveMin,
      'max': math.max(effectiveMin, effectiveMax).toDouble(),
      'step': step,
    };
  }

  int _clampAddonValue(int value, Map<String, dynamic> rule) {
    if (rule['disabled'] == true) return 0;
    final step = math.max(1, _asDouble(rule['step'] == null ? 1 : rule['step'])).toDouble();
    final min = _asDouble(rule['min']);
    final max = math.max(min, _asDouble(rule['max'] ?? min));
    var next = value.toDouble();
    next = math.max(min, math.min(max, next));
    next = min + ((next - min) / step).round() * step;
    if (next > max) next = max;
    if (next < min) next = min;
    return next.round();
  }

  void _scheduleAddonNormalize(Map<String, dynamic> rule) {
    final coreRule =
        _resolveAddonRule(rule['add_core_min'], rule['add_core_max'], rule['add_core_step'], _getDefaultMax('add_core_max'));
    final memRule =
        _resolveAddonRule(rule['add_mem_min'], rule['add_mem_max'], rule['add_mem_step'], _getDefaultMax('add_mem_max'));
    final diskRule =
        _resolveAddonRule(rule['add_disk_min'], rule['add_disk_max'], rule['add_disk_step'], _getDefaultMax('add_disk_max'));
    final bwRule =
        _resolveAddonRule(rule['add_bw_min'], rule['add_bw_max'], rule['add_bw_step'], _getDefaultMax('add_bw_max'));

    final nextCores = _clampAddonValue(_addCores, coreRule);
    final nextMem = _clampAddonValue(_addMem, memRule);
    final nextDisk = _clampAddonValue(_addDisk, diskRule);
    final nextBw = _clampAddonValue(_addBw, bwRule);
    if (nextCores == _addCores && nextMem == _addMem && nextDisk == _addDisk && nextBw == _addBw) {
      return;
    }
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      setState(() {
        _addCores = nextCores;
        _addMem = nextMem;
        _addDisk = nextDisk;
        _addBw = nextBw;
      });
    });
  }

  double _getDefaultMax(String maxKey) {
    switch (maxKey) {
      case 'add_core_max':
        return 64;
      case 'add_mem_max':
        return 256;
      case 'add_disk_max':
        return 2000;
      case 'add_bw_max':
        return 1000;
      default:
        return 256;
    }
  }

  String? _capacityLabel(Map<String, dynamic> item) {
    final remaining = _asDouble(item['capacity_remaining'] ?? item['capacityRemaining']);
    if (remaining.isNaN) return null;
    if (remaining < 0) return '不限';
    if (remaining == 0) return '售罄';
    return '余量 ${remaining.toInt()}';
  }

  Color _capacityColor(Map<String, dynamic> item) {
    final label = _capacityLabel(item);
    if (label == '售罄') return AppColors.danger;
    if (label == '不限') return AppColors.success;
    return AppColors.info;
  }

  bool _isPlanGroupDisabled(Map<String, dynamic> group) {
    if (group['active'] == false || group['visible'] == false) return true;
    final remaining = _asDouble(group['capacity_remaining'] ?? group['capacityRemaining']);
    return remaining == 0;
  }

  bool _isPackageDisabled(Map<String, dynamic> pkg) {
    if (pkg['active'] == false || pkg['visible'] == false) return true;
    final remaining = _asDouble(pkg['capacity_remaining'] ?? pkg['capacityRemaining']);
    return remaining == 0;
  }

  Future<void> _fetchSystemImages(int? planGroupId) async {
    if (planGroupId == null) {
      setState(() {
        _systemImages = [];
        _systemId = null;
      });
      return;
    }
    if (_loadedPlanGroupId == planGroupId && _systemImages.isNotEmpty) return;
    setState(() {
      _loadingImages = true;
      _systemImages = [];
      _systemId = null;
    });
    try {
      final repo = ref.read(catalogRepositoryProvider);
      final images =
          await repo.listSystemImages(planGroupId: planGroupId).timeout(const Duration(seconds: 20));
      final enabled = images.where((img) => img['enabled'] != false).toList();
      if (!mounted) return;
      setState(() {
        _systemImages = enabled;
        _loadingImages = false;
        _loadedPlanGroupId = planGroupId;
        if (enabled.isNotEmpty) {
          _systemId = enabled.first['id'] as int?;
        }
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _loadingImages = false;
      });
    }
  }

  void _syncDefaults(CatalogState catalog) {
    bool changed = false;
    final goodsTypes = catalog.goodsTypes.where((g) => g['active'] != false).toList();
    if (_goodsTypeId == null && goodsTypes.isNotEmpty) {
      _goodsTypeId = goodsTypes.first['id'] as int?;
      changed = true;
    }

    final regions = catalog.regions.where((r) {
      if (r['active'] == false) return false;
      if (_goodsTypeId == null) return false;
      return '${r['goods_type_id']}' == '$_goodsTypeId';
    }).toList();
    if (_regionId == null && regions.isNotEmpty) {
      _regionId = regions.first['id'] as int?;
      changed = true;
    }

    final planGroups = catalog.planGroups.where((g) {
      if (g['active'] == false || g['visible'] == false) return false;
      if (_regionId == null) return false;
      if (_goodsTypeId != null && '${g['goods_type_id']}' != '$_goodsTypeId') return false;
      return g['region_id'] == _regionId;
    }).toList();
    if (_planGroupId == null && planGroups.isNotEmpty) {
      final next = planGroups.firstWhere((g) => !_isPlanGroupDisabled(g), orElse: () => planGroups.first);
      _planGroupId = next['id'] as int?;
      changed = true;
      _fetchSystemImages(_planGroupId);
    }

    final packages = catalog.packages.where((p) {
      if (p['active'] == false || p['visible'] == false) return false;
      if (_planGroupId == null) return false;
      final groupId = p['plan_group_id'] ?? p['PlanGroupID'] ?? p['planGroupId'];
      return '$groupId' == '$_planGroupId';
    }).toList();
    if (_packageId == null && packages.isNotEmpty) {
      final next = packages.firstWhere((p) => !_isPackageDisabled(p), orElse: () => packages.first);
      _packageId = next['id'] as int?;
      changed = true;
    }

    final cycles = catalog.billingCycles.where((c) => c['active'] != false).toList();
    if (_billingCycleId == null && cycles.isNotEmpty) {
      _billingCycleId = cycles.first['id'] as int?;
      changed = true;
    }

    if (changed && mounted) {
      setState(() {});
    }
  }

  Future<void> _addToCart(BuildContext context) async {
    if (!_canCheckout) {
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('请先选择套餐、系统镜像和计费周期')));
      return;
    }
    final catalog = ref.read(catalogProvider);
    final cycle = catalog.billingCycles
        .firstWhere((c) => c['id'] == _billingCycleId, orElse: () => {});
    final months = int.tryParse('${cycle['months'] ?? 1}') ?? 1;

    await ref.read(cartProvider.notifier).addItem({
      'package_id': _packageId,
      'system_id': _systemId,
      'spec': {
        'add_cores': _addCores,
        'add_mem_gb': _addMem,
        'add_disk_gb': _addDisk,
        'add_bw_mbps': _addBw,
        'billing_cycle_id': _billingCycleId,
        'cycle_qty': _cycleQty,
        'duration_months': months * _cycleQty,
      },
      'qty': _qty,
    });

    if (context.mounted) {
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('已加入购物车')));
    }
  }

  Future<void> _createOrderNow(BuildContext context) async {
    if (!_canCheckout) {
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('请先选择套餐、系统镜像和计费周期')));
      return;
    }
    final catalog = ref.read(catalogProvider);
    final cycle = catalog.billingCycles
        .firstWhere((c) => c['id'] == _billingCycleId, orElse: () => {});
    final months = int.tryParse('${cycle['months'] ?? 1}') ?? 1;

    final res = await ref.read(orderRepositoryProvider).createOrderItems({
      'items': [
        {
          'package_id': _packageId,
          'system_id': _systemId,
          'spec': {
            'add_cores': _addCores,
            'add_mem_gb': _addMem,
            'add_disk_gb': _addDisk,
            'add_bw_mbps': _addBw,
            'billing_cycle_id': _billingCycleId,
            'cycle_qty': _cycleQty,
            'duration_months': months * _cycleQty,
          },
          'qty': _qty,
        }
      ]
    }, idempotencyKey: 'order-${DateTime.now().millisecondsSinceEpoch}');

    final orderId = res['order']?['id'] ?? res['id'] ?? res['order_id'];
    if (context.mounted) {
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('订单创建成功')));
      if (orderId != null) {
        context.go('/console/orders/$orderId');
      }
    }
  }
}

