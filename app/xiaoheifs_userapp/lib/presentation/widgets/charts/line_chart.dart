import 'dart:math';

import 'package:flutter/material.dart';
import '../../../core/constants/app_colors.dart';
import '../../../core/constants/app_strings.dart';

class LineChart extends StatefulWidget {
  final List<double> values;
  final List<String> labels;
  final Color lineColor;
  final double height;
  final bool enablePointSelection;

  const LineChart({
    super.key,
    required this.values,
    this.labels = const [],
    this.lineColor = AppColors.primary,
    this.height = 180,
    this.enablePointSelection = false,
  });

  @override
  State<LineChart> createState() => _LineChartState();
}

class _LineChartState extends State<LineChart> {
  int? _selectedIndex;

  @override
  void didUpdateWidget(covariant LineChart oldWidget) {
    super.didUpdateWidget(oldWidget);
    final dataChanged = oldWidget.values != widget.values || oldWidget.labels != widget.labels;
    if (dataChanged) {
      _selectedIndex = null;
      return;
    }
    if (widget.values.isEmpty || (_selectedIndex != null && _selectedIndex! >= widget.values.length)) {
      _selectedIndex = null;
    }
  }

  List<Offset> _buildPoints(Size size) {
    const padding = EdgeInsets.fromLTRB(8, 12, 8, 20);
    final chartWidth = size.width - padding.left - padding.right;
    final chartHeight = size.height - padding.top - padding.bottom;
    if (widget.values.isEmpty || chartWidth <= 0 || chartHeight <= 0) return const [];

    var minValue = widget.values.reduce(min);
    var maxValue = widget.values.reduce(max);
    if (minValue == maxValue) {
      minValue -= 1;
      maxValue += 1;
    }

    final points = <Offset>[];
    for (var i = 0; i < widget.values.length; i++) {
      final dx = padding.left + (chartWidth * i / max(1, widget.values.length - 1));
      final normalized = (widget.values[i] - minValue) / (maxValue - minValue);
      final dy = padding.top + (1 - normalized) * chartHeight;
      points.add(Offset(dx, dy));
    }
    return points;
  }

  int _pickNearestIndex(Offset localPosition, Size size) {
    final points = _buildPoints(size);
    if (points.isEmpty) return 0;
    var nearestIndex = 0;
    var nearestDistance = double.infinity;
    for (var i = 0; i < points.length; i++) {
      final distance = (points[i] - localPosition).distanceSquared;
      if (distance < nearestDistance) {
        nearestDistance = distance;
        nearestIndex = i;
      }
    }
    return nearestIndex;
  }

  void _handleTapDown(TapDownDetails details, BoxConstraints constraints) {
    if (!widget.enablePointSelection || widget.values.isEmpty) return;
    final size = Size(constraints.maxWidth, widget.height);
    final index = _pickNearestIndex(details.localPosition, size);
    setState(() => _selectedIndex = index);
  }

  @override
  Widget build(BuildContext context) {
    if (widget.values.isEmpty) {
      return SizedBox(
        height: widget.height,
        child: const Center(child: Text(AppStrings.noData)),
      );
    }

    return LayoutBuilder(
      builder: (context, constraints) {
        final chart = SizedBox(
          height: widget.height,
          width: double.infinity,
          child: CustomPaint(
            painter: _LineChartPainter(
              values: widget.values,
              labels: widget.labels,
              lineColor: widget.lineColor,
              textColor: Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.7),
              selectedIndex: widget.enablePointSelection ? _selectedIndex : null,
            ),
          ),
        );

        if (!widget.enablePointSelection) {
          return chart;
        }

        return TapRegion(
          onTapOutside: (_) {
            if (_selectedIndex != null) {
              setState(() => _selectedIndex = null);
            }
          },
          child: GestureDetector(
            behavior: HitTestBehavior.opaque,
            onTapDown: (details) => _handleTapDown(details, constraints),
            child: chart,
          ),
        );
      },
    );
  }
}

class _LineChartPainter extends CustomPainter {
  final List<double> values;
  final List<String> labels;
  final Color lineColor;
  final Color textColor;
  final int? selectedIndex;

  _LineChartPainter({
    required this.values,
    required this.labels,
    required this.lineColor,
    required this.textColor,
    required this.selectedIndex,
  });

  @override
  void paint(Canvas canvas, Size size) {
    final padding = const EdgeInsets.fromLTRB(8, 12, 8, 20);
    final chartWidth = size.width - padding.left - padding.right;
    final chartHeight = size.height - padding.top - padding.bottom;
    if (chartWidth <= 0 || chartHeight <= 0) return;

    double minValue = values.reduce(min);
    double maxValue = values.reduce(max);
    if (minValue == maxValue) {
      minValue -= 1;
      maxValue += 1;
    }

    final points = <Offset>[];
    for (var i = 0; i < values.length; i++) {
      final dx = padding.left + (chartWidth * i / max(1, values.length - 1));
      final normalized = (values[i] - minValue) / (maxValue - minValue);
      final dy = padding.top + (1 - normalized) * chartHeight;
      points.add(Offset(dx, dy));
    }

    final gridPaint = Paint()
      ..color = AppColors.gray200
      ..strokeWidth = 1;

    canvas.drawLine(
      Offset(padding.left, padding.top + chartHeight),
      Offset(padding.left + chartWidth, padding.top + chartHeight),
      gridPaint,
    );

    canvas.drawLine(
      Offset(padding.left, padding.top + chartHeight * 0.5),
      Offset(padding.left + chartWidth, padding.top + chartHeight * 0.5),
      gridPaint,
    );

    final linePaint = Paint()
      ..color = lineColor
      ..strokeWidth = 2
      ..style = PaintingStyle.stroke;

    final fillPaint = Paint()
      ..color = lineColor.withValues(alpha: 0.15)
      ..style = PaintingStyle.fill;

    final path = Path();
    for (var i = 0; i < points.length; i++) {
      if (i == 0) {
        path.moveTo(points[i].dx, points[i].dy);
      } else {
        path.lineTo(points[i].dx, points[i].dy);
      }
    }

    final fillPath = Path.from(path)
      ..lineTo(points.last.dx, padding.top + chartHeight)
      ..lineTo(points.first.dx, padding.top + chartHeight)
      ..close();

    canvas.drawPath(fillPath, fillPaint);
    canvas.drawPath(path, linePaint);

    final dotPaint = Paint()..color = lineColor;
    for (final point in points) {
      canvas.drawCircle(point, 3, dotPaint);
    }

    if (selectedIndex != null && selectedIndex! >= 0 && selectedIndex! < points.length) {
      final selectedPoint = points[selectedIndex!];
      final selectedValue = values[selectedIndex!];
      final selectedLabel = selectedIndex! < labels.length ? labels[selectedIndex!] : '';
      final valueText = selectedValue.toStringAsFixed(2);
      final tooltipText = selectedLabel.isEmpty ? valueText : '$selectedLabel  $valueText';

      final guidePaint = Paint()
        ..color = lineColor.withValues(alpha: 0.25)
        ..strokeWidth = 1;
      canvas.drawLine(
        Offset(selectedPoint.dx, padding.top),
        Offset(selectedPoint.dx, padding.top + chartHeight),
        guidePaint,
      );

      final selectedDotPaint = Paint()..color = lineColor;
      canvas.drawCircle(selectedPoint, 5, selectedDotPaint);
      canvas.drawCircle(
        selectedPoint,
        2.3,
        Paint()..color = Colors.white,
      );

      final textPainter = TextPainter(
        text: TextSpan(
          text: tooltipText,
          style: const TextStyle(
            fontSize: 10.5,
            color: Colors.white,
            fontWeight: FontWeight.w600,
          ),
        ),
        textDirection: TextDirection.ltr,
      )..layout(maxWidth: max(80, chartWidth - 8));

      const tooltipPadding = EdgeInsets.symmetric(horizontal: 8, vertical: 4);
      final tooltipWidth = textPainter.width + tooltipPadding.horizontal;
      final tooltipHeight = textPainter.height + tooltipPadding.vertical;
      final left = (selectedPoint.dx - tooltipWidth / 2).clamp(
        padding.left,
        padding.left + chartWidth - tooltipWidth,
      );
      final top = (selectedPoint.dy - tooltipHeight - 10).clamp(
        padding.top,
        padding.top + chartHeight - tooltipHeight,
      );
      final rect = RRect.fromRectAndRadius(
        Rect.fromLTWH(left, top, tooltipWidth, tooltipHeight),
        const Radius.circular(6),
      );
      canvas.drawRRect(
        rect,
        Paint()..color = AppColors.gray900.withValues(alpha: 0.9),
      );
      textPainter.paint(canvas, Offset(left + tooltipPadding.left, top + tooltipPadding.top));
    }

    if (labels.isNotEmpty) {
      _drawLabel(canvas, labels.first, Offset(padding.left, padding.top + chartHeight + 4));
      if (labels.length > 1) {
        _drawLabel(
          canvas,
          labels.last,
          Offset(padding.left + chartWidth - 32, padding.top + chartHeight + 4),
        );
      }
    }
  }

  void _drawLabel(Canvas canvas, String text, Offset offset) {
    final painter = TextPainter(
      text: TextSpan(
        text: text,
        style: TextStyle(fontSize: 10, color: textColor),
      ),
      textDirection: TextDirection.ltr,
    )..layout(maxWidth: 64);
    painter.paint(canvas, offset);
  }

  @override
  bool shouldRepaint(covariant _LineChartPainter oldDelegate) {
    return oldDelegate.values != values ||
        oldDelegate.labels != labels ||
        oldDelegate.lineColor != lineColor ||
        oldDelegate.selectedIndex != selectedIndex;
  }
}
