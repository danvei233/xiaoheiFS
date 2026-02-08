import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/constants/app_colors.dart';
import '../../../core/constants/app_strings.dart';
import '../../providers/realname_provider.dart';
import '../../widgets/common/app_button.dart';
import '../../widgets/common/app_input.dart';

/// 实名认证页面
class RealnamePage extends ConsumerWidget {
  const RealnamePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final realnameState = ref.watch(realnameProvider);

    return Scaffold(
      body: realnameState.loading
          ? const Center(child: CircularProgressIndicator())
          : realnameState.error != null
              ? Center(child: Text('错误: ${realnameState.error}'))
              : _buildContent(context, ref, realnameState.data ?? {}),
    );
  }

  Widget _buildContent(BuildContext context, WidgetRef ref, Map<String, dynamic> status) {
    final isVerified = status['verified'] == true;
    final verification = status['verification'] is Map<String, dynamic>
        ? status['verification'] as Map<String, dynamic>
        : null;

    return SingleChildScrollView(
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(24),
              child: Row(
                children: [
                  Icon(
                    isVerified ? Icons.verified : Icons.pending_outlined,
                    size: 48,
                    color: isVerified ? AppColors.success : AppColors.warning,
                  ),
                  const SizedBox(width: 16),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const Text(
                          '认证状态',
                          style: TextStyle(
                            fontSize: 14,
                            color: AppColors.gray500,
                          ),
                        ),
                        const SizedBox(height: 4),
                        Text(
                          isVerified
                              ? AppStrings.verificationApproved
                              : AppStrings.verificationNotSubmit,
                          style: const TextStyle(
                            fontSize: 20,
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ),
          if (!isVerified) ...[
            const SizedBox(height: 24),
            const Text(
              '提交实名认证',
              style: TextStyle(
                fontSize: 18,
                fontWeight: FontWeight.bold,
              ),
            ),
            const SizedBox(height: 16),
            _buildVerificationForm(context, ref, verification),
          ] else if (verification != null) ...[
            const SizedBox(height: 24),
            _buildVerificationInfo(verification),
          ],
        ],
      ),
    );
  }

  Widget _buildVerificationForm(BuildContext context, WidgetRef ref, Map<String, dynamic>? verification) {
    final realNameController = TextEditingController();
    final idNumberController = TextEditingController();

    if (verification != null) {
      realNameController.text = verification['real_name'] ?? verification['realName'] ?? '';
      idNumberController.text = verification['id_number'] ?? verification['idNumber'] ?? '';
    }

    final canEdit = verification == null || verification['status'] == 'rejected';

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          children: [
            AppInput(
              label: AppStrings.realname,
              hint: '请输入真实姓名',
              controller: realNameController,
              enabled: canEdit,
            ),
            const SizedBox(height: 16),
            AppInput(
              label: AppStrings.idNumber,
              hint: '请输入身份证号',
              controller: idNumberController,
              enabled: canEdit,
              maxLength: 18,
            ),
            if (verification != null && verification['status'] == 'rejected') ...[
              const SizedBox(height: 16),
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: AppColors.danger.withOpacity(0.1),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Row(
                  children: [
                    Icon(Icons.info_outline, color: AppColors.danger, size: 20),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        '审核未通过: ${verification['remark'] ?? ''}',
                        style: TextStyle(color: AppColors.danger),
                      ),
                    ),
                  ],
                ),
              ),
            ],
            if (canEdit) ...[
              const SizedBox(height: 24),
              AppButton(
                text: AppStrings.submitVerification,
                onPressed: () async {
                  try {
                    await ref.read(realnameProvider.notifier).submit({
                          'real_name': realNameController.text,
                          'id_number': idNumberController.text,
                        });
                    if (context.mounted) {
                      ScaffoldMessenger.of(context).showSnackBar(
                        const SnackBar(content: Text('提交成功')),
                      );
                    }
                  } catch (e) {
                    if (context.mounted) {
                      ScaffoldMessenger.of(context).showSnackBar(
                        SnackBar(content: Text(e.toString())),
                      );
                    }
                  }
                },
              ),
            ],
          ],
        ),
      ),
    );
  }

  Widget _buildVerificationInfo(Map<String, dynamic> verification) {
    final realName = verification['real_name'] ?? verification['realName'];
    final idNumber = verification['id_number'] ?? verification['idNumber'];
    final submittedAt = verification['submitted_at'] ?? verification['submittedAt'];
    final reviewedAt = verification['reviewed_at'] ?? verification['reviewedAt'];
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              '认证信息',
              style: TextStyle(
                fontSize: 18,
                fontWeight: FontWeight.bold,
              ),
            ),
            const SizedBox(height: 16),
            _buildInfoRow('真实姓名', realName?.toString()),
            _buildInfoRow(
              '身份证号',
              _maskIdNumber(idNumber?.toString()),
            ),
            _buildInfoRow('提交时间', submittedAt?.toString()),
            if (reviewedAt != null)
              _buildInfoRow('审核时间', reviewedAt?.toString()),
          ],
        ),
      ),
    );
  }

  Widget _buildInfoRow(String label, String? value) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: Row(
        children: [
          SizedBox(
            width: 80,
            child: Text(
              label,
              style: TextStyle(
                color: AppColors.gray500,
              ),
            ),
          ),
          Expanded(
            child: Text(
              value ?? '-',
              style: const TextStyle(
                fontWeight: FontWeight.w500,
              ),
            ),
          ),
        ],
      ),
    );
  }

  String _maskIdNumber(String? idNumber) {
    final value = (idNumber ?? '').trim();
    if (value.isEmpty) return '-';
    if (value.length <= 8) return value;
    final start = 6;
    final end = value.length - 4;
    if (end <= start) return value;
    return value.replaceRange(start, end, '*' * (end - start));
  }
}
