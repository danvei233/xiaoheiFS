import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/constants/app_colors.dart';
import '../../../core/constants/app_strings.dart';
import '../../../core/utils/date_formatter.dart';
import '../../providers/notification_provider.dart';
import '../../providers/refresh_provider.dart';
import '../../widgets/common/empty_state.dart';
import '../../widgets/common/pagination_bar.dart';

class NotificationsPage extends ConsumerStatefulWidget {
  const NotificationsPage({super.key});

  @override
  ConsumerState<NotificationsPage> createState() => _NotificationsPageState();
}

class _NotificationsPageState extends ConsumerState<NotificationsPage> {
  String _status = 'unread';
  int _page = 1;
  int _pageSize = 20;
  ProviderSubscription<RefreshEvent?>? _refreshSub;

  @override
  void initState() {
    super.initState();
    Future.microtask(() {
      _fetch(force: true);
      ref.read(notificationProvider.notifier).fetchUnreadCount();
    });
    _refreshSub = ref.listenManual<RefreshEvent?>(pageRefreshProvider, (_, next) {
      if (next?.route == '/console/notifications') {
        _fetch(force: true);
        ref.read(notificationProvider.notifier).fetchUnreadCount();
      }
    });
  }

  @override
  void dispose() {
    _refreshSub?.close();
    super.dispose();
  }

  Future<void> _fetch({bool force = false}) async {
    await ref.read(notificationProvider.notifier).fetchNotifications(
          status: _status,
          limit: _pageSize,
          offset: (_page - 1) * _pageSize,
          force: force,
        );
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(notificationProvider);

    return Scaffold(
      body: Column(
        children: [
          _buildHeader(context, state.unreadCount),
          Expanded(
            child: RefreshIndicator(
              onRefresh: () => _fetch(force: true),
              child: state.loading
                  ? const Center(child: CircularProgressIndicator())
                  : state.items.isEmpty
                      ? const EmptyState(
                          message: AppStrings.noNotifications,
                          icon: Icons.notifications_off_outlined,
                        )
                      : ListView.separated(
                          padding: const EdgeInsets.all(16),
                          itemCount: state.items.length,
                          separatorBuilder: (_, __) => const SizedBox(height: 12),
                          itemBuilder: (context, index) {
                            final item = state.items[index];
                            final id = item['id'] ?? item['ID'];
                            final title = item['title'] ?? item['type'] ?? AppStrings.notification;
                            final content = item['content'] ?? item['message'] ?? '';
                            final createdAt = item['created_at'] ?? item['CreatedAt'];
                            final readAt = item['read_at'] ?? item['ReadAt'];
                            final isUnread = readAt == null || readAt.toString().isEmpty;
                            return InkWell(
                              borderRadius: BorderRadius.circular(12),
                              onTap: () async {
                                if (id != null) {
                                  await ref.read(notificationProvider.notifier).markRead(id);
                                  ref.read(notificationProvider.notifier).fetchUnreadCount();
                                  await _fetch(force: true);
                                }
                              },
                              child: Container(
                                padding: const EdgeInsets.all(16),
                                decoration: BoxDecoration(
                                  color: Theme.of(context).colorScheme.surface,
                                  borderRadius: BorderRadius.circular(12),
                                  border: Border.all(
                                    color: isUnread
                                        ? AppColors.primary.withOpacity(0.3)
                                        : Theme.of(context).colorScheme.outlineVariant,
                                  ),
                                ),
                                child: Row(
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: [
                                    Container(
                                      width: 36,
                                      height: 36,
                                      decoration: BoxDecoration(
                                        color: AppColors.primary.withOpacity(0.12),
                                        borderRadius: BorderRadius.circular(10),
                                      ),
                                      child: Icon(
                                        Icons.notifications,
                                        color: AppColors.primary,
                                        size: 20,
                                      ),
                                    ),
                                    const SizedBox(width: 12),
                                    Expanded(
                                      child: Column(
                                        crossAxisAlignment: CrossAxisAlignment.start,
                                        children: [
                                          Row(
                                            children: [
                                              Expanded(
                                                child: Text(
                                                  '$title',
                                                  style: TextStyle(
                                                    fontSize: 14,
                                                    fontWeight: isUnread ? FontWeight.bold : FontWeight.w600,
                                                  ),
                                                ),
                                              ),
                                              Text(
                                                DateFormatter.formatIso(createdAt),
                                                style: TextStyle(
                                                  fontSize: 11,
                                                  color: AppColors.gray500,
                                                ),
                                              ),
                                            ],
                                          ),
                                          const SizedBox(height: 6),
                                          Text(
                                            '$content',
                                            maxLines: 3,
                                            overflow: TextOverflow.ellipsis,
                                            style: TextStyle(
                                              fontSize: 12,
                                              color: AppColors.gray400,
                                            ),
                                          ),
                                        ],
                                      ),
                                    ),
                                  ],
                                ),
                              ),
                            );
                          },
                        ),
            ),
          ),
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
            child: PaginationBar(
              currentPage: _page,
              pageSize: _pageSize,
              totalItems: state.total,
              onPageChanged: (page) async {
                setState(() => _page = page);
                await _fetch();
              },
              onPageSizeChanged: (size) async {
                setState(() {
                  _pageSize = size;
                  _page = 1;
                });
                await _fetch(force: true);
              },
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildHeader(BuildContext context, int unreadCount) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
      child: Row(
        children: [
          const Expanded(
            child: Text(
              AppStrings.notifications,
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
            ),
          ),
          TextButton(
            onPressed: () async {
              await ref.read(notificationProvider.notifier).markAllRead();
              ref.read(notificationProvider.notifier).fetchUnreadCount();
            },
            child: const Text(AppStrings.markAllRead),
          ),
          const SizedBox(width: 8),
          _buildSegmented(unreadCount),
        ],
      ),
    );
  }

  Widget _buildSegmented(int unreadCount) {
    return SegmentedButton<String>(
      segments: [
        ButtonSegment<String>(
          value: 'unread',
          label: Text('未读${unreadCount > 0 ? '($unreadCount)' : ''}'),
        ),
        const ButtonSegment<String>(
          value: 'read',
          label: Text('已读'),
        ),
        const ButtonSegment<String>(
          value: 'all',
          label: Text('全部'),
        ),
      ],
      selected: {_status},
      onSelectionChanged: (value) {
        setState(() {
          _status = value.first;
          _page = 1;
        });
        _fetch(force: true);
      },
    );
  }
}
