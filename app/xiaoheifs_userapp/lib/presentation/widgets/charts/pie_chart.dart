import 'dart:math';

import 'package:flutter/material.dart';
import '../../../core/constants/app_colors.dart';
import '../../../core/constants/app_strings.dart';

class PieChart extends StatelessWidget {
  final List<Map<String, dynamic>> data;
  final double height;

  const PieChart({
    super.key,
    required this.data,
    this.height = 180,
  });

  @override
  Widget build(BuildContext context) {
    final items = _normalize(data);
    if (items.isEmpty) {
      return SizedBox(
        height: height,
        child: const Center(child: Text(AppStrings.noData)),
      );
    }

    return Column(
      children: [
        SizedBox(
          height: height,
          width: double.infinity,
          child: CustomPaint(
            painter: _PieChartPainter(items: items),
          ),
        ),
        const SizedBox(height: 12),
        Wrap(
          spacing: 12,
          runSpacing: 8,
          children: items.map((item) {
            return Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Container(
                  width: 8,
                  height: 8,
                  decoration: BoxDecoration(
                    color: item.color,
                    shape: BoxShape.circle,
                  ),
                ),
                const SizedBox(width: 6),
                Text('${item.name} (${item.value})'),
              ],
            );
          }).toList(),
        ),
      ],
    );
  }

  List<_PieItem> _normalize(List<Map<String, dynamic>> raw) {
    final colors = [
      AppColors.primary,
      AppColors.success,
      AppColors.warning,
      AppColors.info,
      const Color(0xFF8B5CF6),
      const Color(0xFFEF4444),
    ];
    final items = <_PieItem>[];
    var index = 0;
    for (final item in raw) {
      final rawName = item['name']?.toString() ?? '';
      final name = _mapStatusLabel(rawName);
      final value = double.tryParse('${item['value'] ?? 0}') ?? 0;
      if (value <= 0) continue;
      items.add(_PieItem(
        name: name.isEmpty ? 'Unknown' : name,
        value: value,
        color: colors[index % colors.length],
      ));
      index++;
    }
    return items;
  }

  String _mapStatusLabel(String status) {
    switch (status) {
      case 'active':
        return '已完成';
      case 'pending_review':
        return '待审核';
      case 'failed':
        return '失败';
      case 'canceled':
      case 'cancelled':
        return '已取消';
      case 'pending':
        return '待支付';
      case 'paid':
        return '已支付';
      case 'refunded':
        return '已退款';
      default:
        return status;
    }
  }
}

class _PieItem {
  final String name;
  final double value;
  final Color color;

  const _PieItem({
    required this.name,
    required this.value,
    required this.color,
  });
}

class _PieChartPainter extends CustomPainter {
  final List<_PieItem> items;

  _PieChartPainter({required this.items});

  @override
  void paint(Canvas canvas, Size size) {
    if (items.isEmpty) return;
    final total = items.fold<double>(0, (sum, item) => sum + item.value);
    if (total <= 0) return;

    final radius = min(size.width, size.height) / 2 - 8;
    final center = Offset(size.width / 2, size.height / 2);
    var start = -pi / 2;

    for (final item in items) {
      final sweep = (item.value / total) * 2 * pi;
      final paint = Paint()
        ..color = item.color
        ..style = PaintingStyle.fill;
      canvas.drawArc(
        Rect.fromCircle(center: center, radius: radius),
        start,
        sweep,
        true,
        paint,
      );
      start += sweep;
    }
  }

  @override
  bool shouldRepaint(covariant _PieChartPainter oldDelegate) {
    return oldDelegate.items != items;
  }
}
