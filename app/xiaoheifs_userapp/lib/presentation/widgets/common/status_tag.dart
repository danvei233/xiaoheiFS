import 'package:flutter/material.dart';
import '../../../core/constants/app_colors.dart';

/// 状态标签组件
class StatusTag extends StatelessWidget {
  final String text;
  final Color? backgroundColor;
  final Color? textColor;

  const StatusTag({
    super.key,
    required this.text,
    this.backgroundColor,
    this.textColor,
  });

  /// VPS 状态标签
  factory StatusTag.vps(String? status) {
    Color bgColor;
    Color textColor;

    switch (status?.toLowerCase()) {
      case 'running':
        bgColor = AppColors.vpsRunning.withOpacity(0.1);
        textColor = AppColors.vpsRunning;
        break;
      case 'stopped':
        bgColor = AppColors.vpsStopped.withOpacity(0.1);
        textColor = AppColors.vpsStopped;
        break;
      case 'pending':
      case 'provisioning':
      case 'reinstalling':
      case 'deleting':
        bgColor = AppColors.vpsPending.withOpacity(0.1);
        textColor = AppColors.vpsPending;
        break;
      case 'reinstall_failed':
      case 'failed':
      case 'error':
        bgColor = AppColors.danger.withOpacity(0.1);
        textColor = AppColors.danger;
        break;
      case 'locked':
        bgColor = AppColors.warning.withOpacity(0.1);
        textColor = AppColors.warning;
        break;
      case 'suspended':
        bgColor = AppColors.vpsSuspended.withOpacity(0.1);
        textColor = AppColors.vpsSuspended;
        break;
      default:
        bgColor = AppColors.gray200;
        textColor = AppColors.gray600;
    }

    return StatusTag(
      text: _getVpsStatusText(status),
      backgroundColor: bgColor,
      textColor: textColor,
    );
  }

  /// 订单状态标签
  factory StatusTag.order(String? status) {
    Color bgColor;
    Color textColor;

    switch (status?.toLowerCase()) {
      case 'pending_payment':
        bgColor = AppColors.orderPending.withOpacity(0.1);
        textColor = AppColors.orderPending;
        break;
      case 'active':
        bgColor = AppColors.orderPaid.withOpacity(0.1);
        textColor = AppColors.orderPaid;
        break;
      case 'pending_review':
        bgColor = AppColors.orderPending.withOpacity(0.1);
        textColor = AppColors.orderPending;
        break;
      case 'pending':
        bgColor = AppColors.orderPending.withOpacity(0.1);
        textColor = AppColors.orderPending;
        break;
      case 'failed':
        bgColor = AppColors.danger.withOpacity(0.1);
        textColor = AppColors.danger;
        break;
      case 'paid':
        bgColor = AppColors.orderPaid.withOpacity(0.1);
        textColor = AppColors.orderPaid;
        break;
      case 'cancelled':
      case 'canceled':
        bgColor = AppColors.orderCancelled.withOpacity(0.1);
        textColor = AppColors.orderCancelled;
        break;
      case 'refunded':
        bgColor = AppColors.orderRefunded.withOpacity(0.1);
        textColor = AppColors.orderRefunded;
        break;
      case 'completed':
        bgColor = AppColors.orderCompleted.withOpacity(0.1);
        textColor = AppColors.orderCompleted;
        break;
      default:
        bgColor = AppColors.gray200;
        textColor = AppColors.gray600;
    }

    return StatusTag(
      text: _getOrderStatusText(status),
      backgroundColor: bgColor,
      textColor: textColor,
    );
  }

  static String _getVpsStatusText(String? status) {
    switch (status?.toLowerCase()) {
      case 'running':
        return '运行中';
      case 'stopped':
        return '关机';
      case 'pending':
      case 'provisioning':
        return '创建中';
      case 'reinstalling':
        return '重装中';
      case 'reinstall_failed':
        return '重装失败';
      case 'locked':
        return '锁定';
      case 'deleting':
        return '删除中';
      case 'failed':
      case 'error':
        return '异常';
      case 'suspended':
        return '暂停';
      default:
        return status ?? '未知';
    }
  }

  static String _getOrderStatusText(String? status) {
    switch (status?.toLowerCase()) {
      case 'pending_payment':
        return '等待支付';
      case 'active':
        return '生效中';
      case 'pending_review':
        return '审核中';
      case 'pending':
        return '待支付';
      case 'failed':
        return '失败';
      case 'paid':
        return '已支付';
      case 'cancelled':
      case 'canceled':
        return '已取消';
      case 'refunded':
        return '已退款';
      case 'completed':
        return '已完成';
      default:
        return status ?? '未知';
    }
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: backgroundColor ?? AppColors.gray200,
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        text,
        style: TextStyle(
          fontSize: 12,
          fontWeight: FontWeight.w500,
          color: textColor ?? AppColors.gray700,
        ),
      ),
    );
  }
}
