import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/constants/app_colors.dart';
import '../../../core/constants/app_strings.dart';
import '../../../core/constants/input_limits.dart';
import '../../providers/ticket_provider.dart';

class TicketDetailPage extends ConsumerStatefulWidget {
  final int id;
  const TicketDetailPage({super.key, required this.id});

  @override
  ConsumerState<TicketDetailPage> createState() => _TicketDetailPageState();
}

class _TicketDetailPageState extends ConsumerState<TicketDetailPage> {
  final _messageController = TextEditingController();
  final _messageScrollController = ScrollController();
  int _lastMessageCount = 0;

  @override
  void initState() {
    super.initState();
    Future.microtask(() async {
      await ref.read(ticketDetailProvider.notifier).fetchDetail(widget.id);
      if (mounted) {
        _jumpToBottomInstant();
      }
    });
  }

  @override
  void dispose() {
    _messageController.dispose();
    _messageScrollController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final loading = ref.watch(ticketDetailProvider.select((s) => s.loading));
    final error = ref.watch(ticketDetailProvider.select((s) => s.error));

    return Scaffold(
      body: loading
          ? const Center(child: CircularProgressIndicator())
          : error != null
          ? Center(child: Text(error))
          : _buildContent(context),
    );
  }

  Widget _buildContent(BuildContext context) {
    final ticket =
        ref.watch(ticketDetailProvider.select((s) => s.ticket)) ?? {};
    final subject = ticket['subject'] ?? ticket['Subject'] ?? '';
    final status = ticket['status'] ?? ticket['Status'] ?? '';
    final createdAt = ticket['created_at'] ?? ticket['CreatedAt'] ?? '';
    final isClosed = status.toString().toLowerCase() == 'closed';
    final messages = ref.watch(ticketDetailProvider.select((s) => s.messages));

    if (messages.length != _lastMessageCount) {
      _lastMessageCount = messages.length;
      _jumpToBottomInstant();
    }

    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.all(16),
          child: Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Expanded(
                        child: Text(
                          subject,
                          style: const TextStyle(
                            fontSize: 18,
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                      ),
                      TextButton.icon(
                        onPressed: status == 'closed'
                            ? null
                            : () => _closeTicket(context),
                        icon: const Icon(Icons.lock_outline),
                        label: const Text(AppStrings.closeTicket),
                      ),
                    ],
                  ),
                  const SizedBox(height: 8),
                  Text(
                    '状态: $status',
                    style: TextStyle(color: AppColors.gray500),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    '创建时间: $createdAt',
                    style: TextStyle(color: AppColors.gray500),
                  ),
                ],
              ),
            ),
          ),
        ),
        Expanded(
          child: _MessageList(
            messages: messages,
            controller: _messageScrollController,
          ),
        ),
        _buildInputBar(context, isClosed: isClosed),
      ],
    );
  }

  Widget _buildInputBar(BuildContext context, {required bool isClosed}) {
    final cs = Theme.of(context).colorScheme;
    final isLight = cs.brightness == Brightness.light;
    return SafeArea(
      child: Container(
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: cs.surface.withValues(alpha: isLight ? 0.98 : 0.92),
          border: Border(
            top: BorderSide(
              color: cs.outlineVariant.withValues(alpha: isLight ? 0.45 : 0.28),
            ),
          ),
        ),
        child: Row(
          children: [
            Expanded(
              child: TextField(
                controller: _messageController,
                enabled: !isClosed,
                maxLength: InputLimits.ticketContent,
                decoration: InputDecoration(
                  hintText: isClosed ? '工单已关闭，无法回复' : '输入回复内容',
                  filled: true,
                  fillColor: cs.surfaceContainerHighest.withValues(
                    alpha: isLight ? 0.72 : 0.45,
                  ),
                  hintStyle: TextStyle(color: cs.onSurfaceVariant),
                  border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(12),
                    borderSide: BorderSide(
                      color: cs.outlineVariant.withValues(
                        alpha: isLight ? 0.45 : 0.3,
                      ),
                    ),
                  ),
                  enabledBorder: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(12),
                    borderSide: BorderSide(
                      color: cs.outlineVariant.withValues(
                        alpha: isLight ? 0.45 : 0.3,
                      ),
                    ),
                  ),
                  focusedBorder: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(12),
                    borderSide: BorderSide(
                      color: cs.primary.withValues(alpha: 0.85),
                    ),
                  ),
                  isDense: true,
                ),
              ),
            ),
            const SizedBox(width: 8),
            ElevatedButton.icon(
              onPressed: isClosed ? null : () => _sendMessage(context),
              icon: const Icon(Icons.send, size: 16),
              label: const Text(AppStrings.sendMessage),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _sendMessage(BuildContext context) async {
    final status =
        ref
            .read(ticketDetailProvider)
            .ticket?['status']
            ?.toString()
            .toLowerCase() ??
        '';
    if (status == 'closed') return;
    final text = _messageController.text.trim();
    if (text.isEmpty) return;
    if (runeLength(text) > InputLimits.ticketContent) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(const SnackBar(content: Text('回复内容长度不能超过 10000 个字符')));
      }
      return;
    }
    try {
      await ref.read(ticketDetailProvider.notifier).addMessage(widget.id, text);
      if (mounted) {
        _messageController.clear();
        _jumpToBottomInstant();
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(e.toString())));
      }
    }
  }

  Future<void> _closeTicket(BuildContext context) async {
    try {
      await ref.read(ticketDetailProvider.notifier).closeTicket(widget.id);
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(const SnackBar(content: Text('工单已关闭')));
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(e.toString())));
      }
    }
  }

  void _jumpToBottomInstant() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      if (!_messageScrollController.hasClients) return;
      final target = _messageScrollController.position.maxScrollExtent;
      _messageScrollController.jumpTo(target);

      // Double-pass to handle late layout updates after rebuild.
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) return;
        if (!_messageScrollController.hasClients) return;
        final target2 = _messageScrollController.position.maxScrollExtent;
        if ((_messageScrollController.offset - target2).abs() > 1) {
          _messageScrollController.jumpTo(target2);
        }
      });
    });
  }
}

class _MessageList extends StatelessWidget {
  final List<Map<String, dynamic>> messages;
  final ScrollController controller;
  const _MessageList({required this.messages, required this.controller});

  @override
  Widget build(BuildContext context) {
    return ListView.builder(
      controller: controller,
      padding: const EdgeInsets.symmetric(horizontal: 16),
      itemCount: messages.length,
      itemBuilder: (context, index) {
        final msg = messages[index];
        final content = msg['content'] ?? msg['Content'] ?? '';
        final role = msg['sender_role'] ?? msg['role'] ?? msg['Role'] ?? '';
        final roleText = role.toString().toLowerCase();
        final user = roleText == 'admin'
            ? '管理员'
            : (msg['sender_name'] ??
                  msg['user_name'] ??
                  msg['UserName'] ??
                  msg['user'] ??
                  '用户');
        final time = msg['created_at'] ?? msg['CreatedAt'] ?? '';
        final isAdmin = roleText == 'admin' || roleText == 'support';
        final bubbleColor = isAdmin
            ? AppColors.darkSurface.withValues(alpha: 0.75)
            : AppColors.primary.withValues(alpha: 0.22);
        final align = isAdmin
            ? CrossAxisAlignment.start
            : CrossAxisAlignment.end;
        final radius = BorderRadius.only(
          topLeft: const Radius.circular(14),
          topRight: const Radius.circular(14),
          bottomLeft: Radius.circular(isAdmin ? 4 : 14),
          bottomRight: Radius.circular(isAdmin ? 14 : 4),
        );
        return Padding(
          padding: const EdgeInsets.only(bottom: 12),
          child: Column(
            crossAxisAlignment: align,
            children: [
              Text(
                user.toString(),
                style: TextStyle(fontSize: 12, color: AppColors.gray500),
              ),
              const SizedBox(height: 4),
              Container(
                constraints: const BoxConstraints(maxWidth: 520),
                padding: const EdgeInsets.symmetric(
                  horizontal: 12,
                  vertical: 10,
                ),
                decoration: BoxDecoration(
                  color: bubbleColor,
                  borderRadius: radius,
                ),
                child: Text(content.toString()),
              ),
              const SizedBox(height: 4),
              Text(
                time.toString(),
                style: TextStyle(color: AppColors.gray500, fontSize: 11),
              ),
            ],
          ),
        );
      },
    );
  }
}
