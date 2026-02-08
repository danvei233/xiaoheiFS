import 'dart:math';

import 'package:flutter/material.dart';

class PaginationBar extends StatelessWidget {
  final int currentPage;
  final int totalItems;
  final int pageSize;
  final List<int> pageSizeOptions;
  final ValueChanged<int> onPageChanged;
  final ValueChanged<int> onPageSizeChanged;
  final bool dense;

  const PaginationBar({
    super.key,
    required this.currentPage,
    required this.totalItems,
    required this.pageSize,
    required this.onPageChanged,
    required this.onPageSizeChanged,
    this.pageSizeOptions = const [10, 20, 50],
    this.dense = false,
  });

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final totalPages = max(1, (totalItems / pageSize).ceil());
    final safePage = currentPage.clamp(1, totalPages);
    final isFirst = safePage <= 1;
    final isLast = safePage >= totalPages;

    return Container(
      padding: EdgeInsets.symmetric(horizontal: 12, vertical: dense ? 8 : 12),
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: colorScheme.outlineVariant),
      ),
      child: Row(
        children: [
          Text(
            '共 $totalItems 条',
            style: TextStyle(fontSize: 12, color: colorScheme.onSurface.withOpacity(0.6)),
          ),
          const Spacer(),
          IconButton(
            onPressed: isFirst ? null : () => onPageChanged(safePage - 1),
            icon: const Icon(Icons.chevron_left),
            tooltip: '上一页',
          ),
          Text(
            '$safePage / $totalPages',
            style: TextStyle(
              fontSize: 13,
              fontWeight: FontWeight.w600,
              color: colorScheme.onSurface,
            ),
          ),
          IconButton(
            onPressed: isLast ? null : () => onPageChanged(safePage + 1),
            icon: const Icon(Icons.chevron_right),
            tooltip: '下一页',
          ),
          const SizedBox(width: 8),
          DropdownButtonHideUnderline(
            child: DropdownButton<int>(
              value: pageSize,
              dropdownColor: colorScheme.surface,
              items: pageSizeOptions
                  .map((size) => DropdownMenuItem<int>(
                        value: size,
                        child: Text(
                          '$size / 页',
                          style: TextStyle(color: colorScheme.onSurface),
                        ),
                      ))
                  .toList(),
              onChanged: (value) {
                if (value != null && value != pageSize) {
                  onPageSizeChanged(value);
                }
              },
            ),
          ),
        ],
      ),
    );
  }
}
