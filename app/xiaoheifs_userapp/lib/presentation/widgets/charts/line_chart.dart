import 'dart:math';

import 'package:flutter/material.dart';
import '../../../core/constants/app_colors.dart';
import '../../../core/constants/app_strings.dart';

class LineChart extends StatelessWidget {
  final List<double> values;
  final List<String> labels;
  final Color lineColor;
  final double height;

  const LineChart({
    super.key,
    required this.values,
    this.labels = const [],
    this.lineColor = AppColors.primary,
    this.height = 180,
  });

  @override
  Widget build(BuildContext context) {
    if (values.isEmpty) {
      return SizedBox(
        height: height,
        child: const Center(child: Text(AppStrings.noData)),
      );
    }

    return SizedBox(
      height: height,
      width: double.infinity,
      child: CustomPaint(
        painter: _LineChartPainter(
          values: values,
          labels: labels,
          lineColor: lineColor,
          textColor: Theme.of(context).colorScheme.onSurface.withOpacity(0.7),
        ),
      ),
    );
  }
}

class _LineChartPainter extends CustomPainter {
  final List<double> values;
  final List<String> labels;
  final Color lineColor;
  final Color textColor;

  _LineChartPainter({
    required this.values,
    required this.labels,
    required this.lineColor,
    required this.textColor,
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
      ..color = lineColor.withOpacity(0.15)
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
    return oldDelegate.values != values || oldDelegate.labels != labels || oldDelegate.lineColor != lineColor;
  }
}
