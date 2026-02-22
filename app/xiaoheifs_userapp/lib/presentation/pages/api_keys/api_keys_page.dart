import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/constants/app_colors.dart';
import '../../../core/constants/app_strings.dart';
import '../../../core/utils/date_formatter.dart';
import '../../providers/api_key_provider.dart';
import '../../providers/refresh_provider.dart';
import '../../widgets/common/empty_state.dart';

class ApiKeysPage extends ConsumerStatefulWidget {
  const ApiKeysPage({super.key});

  @override
  ConsumerState<ApiKeysPage> createState() => _ApiKeysPageState();
}

class _ApiKeysPageState extends ConsumerState<ApiKeysPage> {
  ProviderSubscription<RefreshEvent?>? _refreshSub;
  bool _codeExpanded = false;

  static const String _signSnippet = '''import crypto from "crypto";

const method = "POST";
const path = "/api/v1/open/orders/instant/create";
const query = "";
const ts = new Date().toISOString(); // RFC3339
const nonce = crypto.randomUUID().replace(/-/g, "");
const body = JSON.stringify({
  items: [{ package_id: 1, system_id: 1, qty: 1 }]
});

const bodyHash = crypto.createHash("sha256").update(body).digest("hex");
const canonical = [method.toUpperCase(), path, query, ts, nonce, bodyHash].join("\\n");
const sig = crypto.createHmac("sha256", process.env.OPEN_KEY)
  .update(canonical)
  .digest("hex");''';

  @override
  void initState() {
    super.initState();
    Future.microtask(() => _fetch(force: true));
    _refreshSub = ref.listenManual<RefreshEvent?>(pageRefreshProvider, (
      _,
      next,
    ) {
      if (next?.route == '/console/api-keys') {
        _fetch(force: true);
      }
    });
  }

  @override
  void dispose() {
    _refreshSub?.close();
    super.dispose();
  }

  Future<void> _fetch({bool force = false}) async {
    await ref.read(apiKeyProvider.notifier).fetchApiKeys(force: force);
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(apiKeyProvider);
    final cs = Theme.of(context).colorScheme;

    return Scaffold(
      floatingActionButton: FloatingActionButton.extended(
        onPressed: _openCreateDialog,
        icon: const Icon(Icons.add),
        label: const Text('创建密钥'),
      ),
      body: RefreshIndicator(
        onRefresh: () => _fetch(force: true),
        child: ListView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.all(16),
          children: [
            _buildHeader(),
            const SizedBox(height: 16),
            _buildInfoBanner(cs),
            const SizedBox(height: 16),
            _buildListCard(state),
            const SizedBox(height: 16),
            _buildCodeCard(cs),
            const SizedBox(height: 96),
          ],
        ),
      ),
    );
  }

  Widget _buildHeader() {
    final cs = Theme.of(context).colorScheme;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text(
          'API 密钥管理',
          style: TextStyle(fontSize: 22, fontWeight: FontWeight.w700),
        ),
        const SizedBox(height: 6),
        Text(
          '管理用于开放接口鉴权的 AKID/Key 凭证',
          style: TextStyle(fontSize: 13, color: cs.onSurfaceVariant),
        ),
      ],
    );
  }

  Widget _buildInfoBanner(ColorScheme cs) {
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: AppColors.primary.withValues(alpha: 0.08),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: AppColors.primary.withValues(alpha: 0.2)),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 36,
            height: 36,
            decoration: BoxDecoration(
              color: AppColors.primary,
              borderRadius: BorderRadius.circular(10),
            ),
            child: const Icon(
              Icons.info_outline,
              color: Colors.white,
              size: 20,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  '签名鉴权规范',
                  style: TextStyle(
                    color: cs.onSurface,
                    fontWeight: FontWeight.w600,
                    fontSize: 14,
                  ),
                ),
                const SizedBox(height: 8),
                Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: const [
                    _HeaderChip(label: 'X-AKID'),
                    _HeaderChip(label: 'X-Timestamp'),
                    _HeaderChip(label: 'X-Nonce'),
                    _HeaderChip(label: 'X-Signature'),
                    _HeaderChip(label: '时间窗: ±300 秒', highlight: true),
                  ],
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildListCard(ApiKeyState state) {
    final cs = Theme.of(context).colorScheme;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const Icon(Icons.key_outlined, size: 20),
                const SizedBox(width: 8),
                const Text(
                  '密钥列表',
                  style: TextStyle(fontSize: 16, fontWeight: FontWeight.w700),
                ),
                const Spacer(),
                if (state.items.isNotEmpty)
                  Text(
                    '${state.items.length} 个密钥',
                    style: TextStyle(fontSize: 12, color: cs.onSurfaceVariant),
                  ),
              ],
            ),
            const SizedBox(height: 12),
            if (state.loading && state.items.isEmpty)
              const Padding(
                padding: EdgeInsets.symmetric(vertical: 24),
                child: Center(child: CircularProgressIndicator()),
              )
            else if (state.items.isEmpty)
              EmptyState(
                message: '暂无 API 密钥',
                icon: Icons.vpn_key_outlined,
                actionLabel: '创建第一个密钥',
                onAction: _openCreateDialog,
              )
            else
              ...state.items.map((item) => _buildApiKeyItem(item)),
          ],
        ),
      ),
    );
  }

  Widget _buildApiKeyItem(Map<String, dynamic> item) {
    final cs = Theme.of(context).colorScheme;
    final isLight = cs.brightness == Brightness.light;
    final id = _readInt(item, ['id', 'ID']);
    final name = _readString(item, ['name', 'Name']);
    final akid = _readString(item, ['akid', 'AKID']);
    final status = _readString(item, ['status', 'Status']).toLowerCase();
    final lastUsedAt = _readString(item, ['last_used_at', 'LastUsedAt']);
    final isActive = status == 'active';

    return Container(
      margin: const EdgeInsets.only(bottom: 10),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(12),
        border: Border.all(
          color: cs.outlineVariant.withValues(alpha: isLight ? 0.5 : 0.35),
        ),
        color: cs.surfaceContainerHighest.withValues(
          alpha: isLight ? 0.45 : 0.2,
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(
                  name.isEmpty ? '未命名密钥' : name,
                  style: const TextStyle(
                    fontWeight: FontWeight.w600,
                    fontSize: 15,
                  ),
                ),
              ),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                decoration: BoxDecoration(
                  color: isActive
                      ? AppColors.success.withValues(
                          alpha: isLight ? 0.14 : 0.18,
                        )
                      : cs.surfaceContainerHighest.withValues(
                          alpha: isLight ? 0.85 : 0.6,
                        ),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Text(
                  isActive ? '已启用' : '已停用',
                  style: TextStyle(
                    fontSize: 12,
                    color: isActive ? AppColors.success : cs.onSurfaceVariant,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 10),
          Row(
            children: [
              Expanded(
                child: Text(
                  akid.isEmpty ? '-' : akid,
                  style: TextStyle(
                    fontSize: 12,
                    color: cs.primary,
                    fontFamily: 'monospace',
                  ),
                ),
              ),
              IconButton(
                onPressed: akid.isEmpty
                    ? null
                    : () => _copyText(akid, 'AKID 已复制到剪贴板'),
                icon: const Icon(Icons.copy_outlined, size: 18),
                tooltip: '复制 AKID',
              ),
            ],
          ),
          const SizedBox(height: 4),
          Text(
            '最近使用: ${lastUsedAt.isEmpty ? '—' : DateFormatter.formatIso(lastUsedAt, DateFormatter.formatCompact)}',
            style: TextStyle(fontSize: 12, color: cs.onSurfaceVariant),
          ),
          const SizedBox(height: 8),
          Row(
            children: [
              OutlinedButton.icon(
                onPressed: id == null ? null : () => _toggleStatus(id, status),
                icon: Icon(
                  isActive
                      ? Icons.pause_circle_outline
                      : Icons.play_circle_outline,
                ),
                label: Text(isActive ? '停用' : '启用'),
              ),
              const SizedBox(width: 8),
              OutlinedButton.icon(
                onPressed: id == null ? null : () => _confirmDelete(id, name),
                icon: const Icon(Icons.delete_outline, color: AppColors.danger),
                label: const Text(
                  '删除',
                  style: TextStyle(color: AppColors.danger),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildCodeCard(ColorScheme cs) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          children: [
            InkWell(
              borderRadius: BorderRadius.circular(8),
              onTap: () => setState(() => _codeExpanded = !_codeExpanded),
              child: Padding(
                padding: const EdgeInsets.symmetric(vertical: 4),
                child: Row(
                  children: [
                    const Icon(Icons.code_outlined),
                    const SizedBox(width: 8),
                    const Text(
                      '签名算法示例',
                      style: TextStyle(
                        fontSize: 16,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                    const Spacer(),
                    Container(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 8,
                        vertical: 4,
                      ),
                      decoration: BoxDecoration(
                        color: AppColors.primary.withValues(alpha: 0.12),
                        borderRadius: BorderRadius.circular(8),
                      ),
                      child: Text(
                        'Node.js',
                        style: TextStyle(fontSize: 12, color: cs.primary),
                      ),
                    ),
                    IconButton(
                      onPressed: () => _copyText(_signSnippet, '示例代码已复制'),
                      icon: const Icon(Icons.copy_outlined, size: 18),
                    ),
                    Icon(
                      _codeExpanded ? Icons.expand_less : Icons.expand_more,
                      color: cs.onSurfaceVariant,
                    ),
                  ],
                ),
              ),
            ),
            AnimatedCrossFade(
              duration: const Duration(milliseconds: 180),
              crossFadeState: _codeExpanded
                  ? CrossFadeState.showSecond
                  : CrossFadeState.showFirst,
              firstChild: const SizedBox.shrink(),
              secondChild: Container(
                width: double.infinity,
                margin: const EdgeInsets.only(top: 10),
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: cs.surfaceContainerHighest.withValues(alpha: 0.4),
                  borderRadius: BorderRadius.circular(10),
                ),
                child: SelectableText.rich(
                  _buildHighlightedSnippetText(cs),
                  style: const TextStyle(
                    fontSize: 12,
                    height: 1.55,
                    fontFamily: 'monospace',
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  TextSpan _buildHighlightedSnippetText(ColorScheme cs) {
    final isLight = cs.brightness == Brightness.light;
    final keywordColor = cs.primary;
    final stringColor = isLight
        ? const Color(0xFF15803D)
        : const Color(0xFF22C55E);
    final commentColor = cs.onSurfaceVariant;
    final normalColor = cs.onSurface;
    final pattern = RegExp(
      r'//.*|"(?:\\.|[^"\\])*"|\b(import|const|new|from)\b',
      multiLine: true,
    );

    final spans = <TextSpan>[];
    var last = 0;
    for (final match in pattern.allMatches(_signSnippet)) {
      if (match.start > last) {
        spans.add(
          TextSpan(
            text: _signSnippet.substring(last, match.start),
            style: TextStyle(color: normalColor),
          ),
        );
      }
      final token = _signSnippet.substring(match.start, match.end);
      final isComment = token.startsWith('//');
      final isString = token.startsWith('"');
      spans.add(
        TextSpan(
          text: token,
          style: TextStyle(
            color: isComment
                ? commentColor
                : isString
                ? stringColor
                : keywordColor,
            fontWeight: isComment ? FontWeight.w400 : FontWeight.w600,
          ),
        ),
      );
      last = match.end;
    }

    if (last < _signSnippet.length) {
      spans.add(
        TextSpan(
          text: _signSnippet.substring(last),
          style: TextStyle(color: normalColor),
        ),
      );
    }
    return TextSpan(children: spans);
  }

  Future<void> _openCreateDialog() async {
    final controller = TextEditingController();
    final created = await showDialog<Map<String, String>>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('创建新 API 密钥'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              '生成新的 AKID/Key 凭证对，Key 仅在创建时展示一次。',
              style: TextStyle(
                color: Theme.of(context).colorScheme.onSurfaceVariant,
                fontSize: 13,
              ),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: controller,
              maxLength: 64,
              decoration: const InputDecoration(
                labelText: '密钥名称',
                hintText: '例如：生产环境-订单服务',
              ),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text(AppStrings.cancel),
          ),
          FilledButton(
            onPressed: () async {
              final name = controller.text.trim();
              if (name.isEmpty) {
                _showMessage('请输入密钥名称', isError: true);
                return;
              }
              try {
                final res = await ref
                    .read(apiKeyProvider.notifier)
                    .createApiKey(name);
                final item = _readMap(res, ['item']);
                final akid = _readString(item, ['akid', 'AKID']);
                final key = _readString(res, [
                  'key',
                  'secret',
                  'Key',
                  'Secret',
                ]);
                if (!context.mounted) return;
                Navigator.pop(context, {'akid': akid, 'key': key});
              } catch (e) {
                _showMessage(
                  e.toString().replaceAll('Exception: ', ''),
                  isError: true,
                );
              }
            },
            child: const Text('创建密钥'),
          ),
        ],
      ),
    );

    if (created == null) {
      return;
    }

    await _fetch(force: true);
    if (!mounted) {
      return;
    }
    await _openSecretDialog(
      akid: created['akid'] ?? '',
      key: created['key'] ?? '',
    );
  }

  Future<void> _openSecretDialog({
    required String akid,
    required String key,
  }) async {
    await showDialog<void>(
      context: context,
      barrierDismissible: false,
      builder: (context) => AlertDialog(
        title: const Text('已创建密钥'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Container(
              padding: const EdgeInsets.all(10),
              decoration: BoxDecoration(
                color: AppColors.warning.withValues(alpha: 0.12),
                borderRadius: BorderRadius.circular(10),
                border: Border.all(
                  color: AppColors.warning.withValues(alpha: 0.25),
                ),
              ),
              child: const Text(
                '窗口关闭后将无法再次查看 Key，请务必立即复制保存。',
                style: TextStyle(fontSize: 13, color: AppColors.warning),
              ),
            ),
            const SizedBox(height: 12),
            const Text('AKID', style: TextStyle(fontWeight: FontWeight.w600)),
            const SizedBox(height: 6),
            _SecretField(
              value: akid,
              onCopy: () => _copyText(akid, 'AKID 已复制到剪贴板'),
            ),
            const SizedBox(height: 10),
            const Text('Key', style: TextStyle(fontWeight: FontWeight.w600)),
            const SizedBox(height: 6),
            _SecretField(
              value: key,
              emphasize: true,
              onCopy: () => _copyText(key, 'Key 已复制到剪贴板'),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text(AppStrings.close),
          ),
          FilledButton.icon(
            onPressed: () => _copyText(key, 'Key 已复制到剪贴板'),
            icon: const Icon(Icons.copy_outlined),
            label: const Text('复制 Key'),
          ),
        ],
      ),
    );
  }

  Future<void> _toggleStatus(int id, String status) async {
    try {
      await ref.read(apiKeyProvider.notifier).toggleStatus(id, status);
      await _fetch(force: true);
      _showMessage(status.toLowerCase() == 'active' ? '密钥已停用' : '密钥已启用');
    } catch (e) {
      _showMessage(e.toString().replaceAll('Exception: ', ''), isError: true);
    }
  }

  Future<void> _confirmDelete(int id, String name) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('确认删除'),
        content: Text('确认删除密钥 ${name.isEmpty ? '#$id' : '"$name"'} 吗？删除后无法恢复。'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text(AppStrings.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            style: FilledButton.styleFrom(backgroundColor: AppColors.danger),
            child: const Text('确认删除'),
          ),
        ],
      ),
    );

    if (confirmed != true) {
      return;
    }

    try {
      await ref.read(apiKeyProvider.notifier).deleteApiKey(id);
      await _fetch(force: true);
      _showMessage('密钥已删除');
    } catch (e) {
      _showMessage(e.toString().replaceAll('Exception: ', ''), isError: true);
    }
  }

  Future<void> _copyText(String value, String successText) async {
    await Clipboard.setData(ClipboardData(text: value));
    _showMessage(successText);
  }

  void _showMessage(String message, {bool isError = false}) {
    if (!mounted) {
      return;
    }
    ScaffoldMessenger.of(context)
      ..hideCurrentSnackBar()
      ..showSnackBar(
        SnackBar(
          content: Text(message),
          backgroundColor: isError ? AppColors.danger : AppColors.success,
        ),
      );
  }

  String _readString(Map<String, dynamic> map, List<String> keys) {
    for (final key in keys) {
      final value = map[key];
      if (value != null && value.toString().trim().isNotEmpty) {
        return value.toString();
      }
    }
    return '';
  }

  int? _readInt(Map<String, dynamic> map, List<String> keys) {
    for (final key in keys) {
      final value = map[key];
      if (value is int) {
        return value;
      }
      if (value != null) {
        final parsed = int.tryParse(value.toString());
        if (parsed != null) {
          return parsed;
        }
      }
    }
    return null;
  }

  Map<String, dynamic> _readMap(Map<String, dynamic> map, List<String> keys) {
    for (final key in keys) {
      final value = map[key];
      if (value is Map<String, dynamic>) {
        return value;
      }
      if (value is Map) {
        return value.map((k, v) => MapEntry(k.toString(), v));
      }
    }
    return {};
  }
}

class _HeaderChip extends StatelessWidget {
  const _HeaderChip({required this.label, this.highlight = false});

  final String label;
  final bool highlight;

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;
    final isLight = cs.brightness == Brightness.light;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: highlight
            ? AppColors.success.withValues(alpha: isLight ? 0.14 : 0.18)
            : cs.surfaceContainerHighest.withValues(
                alpha: isLight ? 0.85 : 0.5,
              ),
        borderRadius: BorderRadius.circular(6),
        border: Border.all(
          color: highlight
              ? AppColors.success.withValues(alpha: isLight ? 0.24 : 0.28)
              : cs.outlineVariant.withValues(alpha: isLight ? 0.45 : 0.3),
        ),
      ),
      child: Text(
        label,
        style: TextStyle(
          fontSize: 11,
          color: highlight ? AppColors.success : cs.primary,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

class _SecretField extends StatelessWidget {
  const _SecretField({
    required this.value,
    required this.onCopy,
    this.emphasize = false,
  });

  final String value;
  final VoidCallback onCopy;
  final bool emphasize;

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;
    final isLight = cs.brightness == Brightness.light;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(8),
        border: Border.all(
          color: emphasize
              ? AppColors.warning.withValues(alpha: 0.4)
              : cs.outlineVariant.withValues(alpha: isLight ? 0.5 : 0.35),
        ),
        color: emphasize
            ? AppColors.warning.withValues(alpha: 0.08)
            : cs.surfaceContainerHighest.withValues(
                alpha: isLight ? 0.75 : 0.45,
              ),
      ),
      child: Row(
        children: [
          Expanded(
            child: SelectableText(
              value,
              style: const TextStyle(fontSize: 12, fontFamily: 'monospace'),
            ),
          ),
          IconButton(
            onPressed: onCopy,
            icon: const Icon(Icons.copy_outlined, size: 18),
            tooltip: '复制',
          ),
        ],
      ),
    );
  }
}
