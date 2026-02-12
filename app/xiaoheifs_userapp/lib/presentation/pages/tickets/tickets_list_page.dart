import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../core/constants/app_strings.dart';
import '../../providers/ticket_provider.dart';
import '../../providers/vps_provider.dart';
import '../../providers/refresh_provider.dart';
import '../../widgets/common/empty_state.dart';
import '../../widgets/common/pagination_bar.dart';

/// 工单列表页面
class TicketsListPage extends ConsumerStatefulWidget {
  const TicketsListPage({super.key});

  @override
  ConsumerState<TicketsListPage> createState() => _TicketsListPageState();
}

class _TicketsListPageState extends ConsumerState<TicketsListPage> {
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
    Future.microtask(() => _fetch());
    _refreshSub = ref.listenManual<RefreshEvent?>(pageRefreshProvider, (
      _,
      next,
    ) {
      if (next?.route == '/console/tickets') {
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
    await ref
        .read(ticketListProvider.notifier)
        .fetchTickets(
          limit: _pageSize,
          offset: (_page - 1) * _pageSize,
          force: force,
        );
  }

  @override
  Widget build(BuildContext context) {
    final ticketListState = ref.watch(ticketListProvider);
    return Scaffold(
      body: ticketListState.loading
          ? const Center(child: CircularProgressIndicator())
          : ticketListState.items.isEmpty
          ? const EmptyState(
              message: AppStrings.noTickets,
              icon: Icons.support_agent_outlined,
            )
          : _buildTicketList(
              context,
              ref,
              ticketListState.items,
              ticketListState.total,
            ),
      floatingActionButton: Padding(
        padding: const EdgeInsets.only(bottom: _paginationFabOffset, right: 8),
        child: FloatingActionButton(
          onPressed: () => _showCreateTicketDialog(context, ref),
          child: const Icon(Icons.add),
        ),
      ),
      floatingActionButtonLocation: FloatingActionButtonLocation.endFloat,
      floatingActionButtonAnimator: FloatingActionButtonAnimator.scaling,
    );
  }

  Widget _buildTicketList(
    BuildContext context,
    WidgetRef ref,
    List<Map<String, dynamic>> tickets,
    int total,
  ) {
    return Column(
      children: [
        Expanded(
          child: RefreshIndicator(
            onRefresh: () => _fetch(force: true),
            child: ListView.builder(
              padding: const EdgeInsets.all(24),
              itemCount: tickets.length,
              itemBuilder: (context, index) {
                final ticket = tickets[index];
                final id = ticket['id'] ?? ticket['ID'];
                final subject = ticket['subject'] ?? ticket['Subject'] ?? '';
                final createdAt =
                    ticket['created_at'] ?? ticket['CreatedAt'] ?? '';
                return Card(
                  margin: const EdgeInsets.only(bottom: 16),
                  child: ListTile(
                    title: Text('$subject'),
                    subtitle: Text('$createdAt'),
                    trailing: const Icon(Icons.arrow_forward_ios, size: 16),
                    onTap: () {
                      if (id != null) {
                        context.go('/console/tickets/$id');
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
    );
  }

  void _showCreateTicketDialog(BuildContext context, WidgetRef ref) {
    final subjectController = TextEditingController();
    final contentController = TextEditingController();
    final vpsList = ref.read(vpsListProvider).items;
    final selectedIds = <int>{};

    showDialog(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setState) => AlertDialog(
          title: const Text(AppStrings.createTicket),
          content: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                TextField(
                  controller: subjectController,
                  decoration: const InputDecoration(
                    labelText: AppStrings.ticketTitle,
                  ),
                ),
                const SizedBox(height: 16),
                TextField(
                  controller: contentController,
                  decoration: const InputDecoration(
                    labelText: AppStrings.ticketContent,
                  ),
                  maxLines: 4,
                ),
                const SizedBox(height: 16),
                if (vpsList.isNotEmpty) ...[
                  Align(
                    alignment: Alignment.centerLeft,
                    child: Text(
                      '关联实例',
                      style: Theme.of(context).textTheme.bodyMedium,
                    ),
                  ),
                  const SizedBox(height: 8),
                  _MultiSelectChips(
                    items: vpsList,
                    selectedIds: selectedIds,
                    onChanged: (ids) {
                      setState(() {
                        selectedIds
                          ..clear()
                          ..addAll(ids);
                      });
                    },
                  ),
                ],
              ],
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text(AppStrings.cancel),
            ),
            TextButton(
              onPressed: () async {
                final subject = subjectController.text.trim();
                final content = contentController.text.trim();
                if (subject.isEmpty || content.isEmpty) {
                  ScaffoldMessenger.of(
                    context,
                  ).showSnackBar(const SnackBar(content: Text('请填写完整内容')));
                  return;
                }
                try {
                  final resources = selectedIds
                      .map(
                        (id) => {
                          'resource_type': 'vps',
                          'resource_id': id,
                          'resource_name':
                              vpsList
                                  .firstWhere(
                                    (item) => (item['id'] ?? item['ID']) == id,
                                    orElse: () => {},
                                  )['name']
                                  ?.toString() ??
                              'VPS-$id',
                        },
                      )
                      .toList();
                  await ref
                      .read(ticketListProvider.notifier)
                      .createTicket(
                        subject: subject,
                        content: content,
                        resources: resources,
                      );
                  if (context.mounted) {
                    Navigator.pop(context);
                    ScaffoldMessenger.of(
                      context,
                    ).showSnackBar(const SnackBar(content: Text('工单创建成功')));
                  }
                } catch (e) {
                  if (context.mounted) {
                    ScaffoldMessenger.of(
                      context,
                    ).showSnackBar(SnackBar(content: Text(e.toString())));
                  }
                }
              },
              child: const Text(AppStrings.submit),
            ),
          ],
        ),
      ),
    );
  }
}

class _MultiSelectChips extends StatefulWidget {
  final List<Map<String, dynamic>> items;
  final Set<int> selectedIds;
  final ValueChanged<Set<int>> onChanged;

  const _MultiSelectChips({
    required this.items,
    required this.selectedIds,
    required this.onChanged,
  });

  @override
  State<_MultiSelectChips> createState() => _MultiSelectChipsState();
}

class _MultiSelectChipsState extends State<_MultiSelectChips> {
  String _keyword = '';

  @override
  Widget build(BuildContext context) {
    final items = widget.items.where((item) {
      if (_keyword.isEmpty) return true;
      final name = item['name'] ?? item['Name'] ?? '';
      final region = item['region'] ?? item['Region'] ?? '';
      final text =
          '${name.toString().toLowerCase()} ${region.toString().toLowerCase()}';
      return text.contains(_keyword.toLowerCase());
    }).toList();

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        TextField(
          decoration: const InputDecoration(
            labelText: '搜索实例',
            prefixIcon: Icon(Icons.search),
          ),
          onChanged: (value) => setState(() => _keyword = value.trim()),
        ),
        const SizedBox(height: 8),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: items.map((vps) {
            final id = vps['id'] ?? vps['ID'];
            final name = vps['name'] ?? vps['Name'] ?? 'VPS';
            final region = vps['region'] ?? vps['Region'];
            final label = region == null || region.toString().isEmpty
                ? name.toString()
                : '${name.toString()} · ${region.toString()}';
            final selected = id != null && widget.selectedIds.contains(id);
            return FilterChip(
              label: Text(label),
              selected: selected,
              onSelected: id == null
                  ? null
                  : (val) {
                      final next = Set<int>.from(widget.selectedIds);
                      if (val) {
                        next.add(id as int);
                      } else {
                        next.remove(id as int);
                      }
                      widget.onChanged(next);
                    },
            );
          }).toList(),
        ),
      ],
    );
  }
}
