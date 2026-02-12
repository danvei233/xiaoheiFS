import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/constants/app_colors.dart';
import '../../../core/constants/app_strings.dart';
import '../../../core/network/api_client.dart';
import '../../../core/utils/avatar_url.dart';
import '../../providers/auth_provider.dart';
import '../../widgets/common/app_button.dart';
import '../../widgets/common/app_input.dart';

/// 个人设置页面
class ProfilePage extends ConsumerWidget {
  const ProfilePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authState = ref.watch(authProvider);
    final user = authState.user;

    if (user == null) {
      return const Center(child: Text('请先登录'));
    }

    final emailController = TextEditingController(
      text: _sanitizeProfileValue(user.email?.toString()),
    );
    final phoneController = TextEditingController(
      text: _sanitizePhone(user.phone?.toString()),
    );
    final qqController = TextEditingController(
      text: _sanitizeQq(user.qq?.toString()),
    );
    final bioController = TextEditingController(
      text: _sanitizeProfileValue(user.bio?.toString()),
    );
    return Scaffold(
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              AppStrings.profileSettings,
              style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w700),
            ),
            const SizedBox(height: 16),
            _buildUserInfoCard(user),
            const SizedBox(height: 24),
            _buildEditForm(
              context,
              ref,
              emailController,
              phoneController,
              qqController,
              bioController,
            ),
          ],
        ),
      ),
    );
  }

  static String _sanitizeProfileValue(String? raw) {
    final value = (raw ?? '').trim();
    if (value.isEmpty) return '';

    if (value.contains('�')) return '';
    if (RegExp(r'[\u0080-\u009f]').hasMatch(value)) return '';

    final lowered = value.toLowerCase();
    if (lowered.contains('鍙') ||
        lowered.contains('閫€') ||
        lowered.contains('鏆')) {
      return '';
    }

    return value;
  }

  static String _sanitizePhone(String? raw) {
    final digits = (raw ?? '').replaceAll(RegExp(r'[^0-9]'), '');
    if (digits.isEmpty) return '';
    if (digits.length == 11 && digits.startsWith('1')) return digits;
    return '';
  }

  static String _sanitizeQq(String? raw) {
    final digits = (raw ?? '').replaceAll(RegExp(r'[^0-9]'), '');
    if (digits.isEmpty) return '';
    if (digits.length >= 5 && digits.length <= 11 && !digits.startsWith('0')) {
      return digits;
    }
    return '';
  }

  Widget _buildUserInfoCard(dynamic user) {
    final avatarUrl = resolveUserAvatarUrl(
      baseUrl: ApiClient.instance.dio.options.baseUrl,
      qq: user?.qq?.toString(),
      avatarUrl: user?.avatarUrl?.toString(),
      avatar: user?.avatar?.toString(),
    );
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Row(
          children: [
            if (avatarUrl.isNotEmpty)
              CircleAvatar(radius: 40, backgroundImage: NetworkImage(avatarUrl))
            else
              CircleAvatar(
                radius: 40,
                backgroundColor: AppColors.primaryLight,
                child: Text(
                  (user.username ?? 'U')[0].toUpperCase(),
                  style: const TextStyle(
                    fontSize: 32,
                    fontWeight: FontWeight.bold,
                    color: Colors.white,
                  ),
                ),
              ),
            const SizedBox(width: 16),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    user.username ?? '',
                    style: const TextStyle(
                      fontSize: 20,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    user.email ?? '',
                    style: TextStyle(fontSize: 14, color: AppColors.gray500),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildEditForm(
    BuildContext context,
    WidgetRef ref,
    TextEditingController emailController,
    TextEditingController phoneController,
    TextEditingController qqController,
    TextEditingController bioController,
  ) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            AppInput(
              label: AppStrings.email,
              hint: '请输入邮箱',
              controller: emailController,
              keyboardType: TextInputType.emailAddress,
              prefixIcon: const Icon(Icons.email_outlined),
            ),
            const SizedBox(height: 16),
            AppInput(
              label: AppStrings.phone,
              hint: '请输入手机号',
              controller: phoneController,
              keyboardType: TextInputType.phone,
              prefixIcon: const Icon(Icons.phone_outlined),
            ),
            const SizedBox(height: 16),
            AppInput(
              label: AppStrings.qq,
              hint: '请输入QQ号',
              controller: qqController,
              keyboardType: TextInputType.number,
              prefixIcon: const Icon(Icons.chat_bubble_outline),
            ),
            const SizedBox(height: 16),
            AppInput(
              label: AppStrings.bio,
              hint: '请输入个人简介',
              controller: bioController,
              maxLines: 3,
              prefixIcon: const Icon(Icons.edit_note),
            ),
            const SizedBox(height: 24),
            AppButton(
              text: AppStrings.save,
              onPressed: () async {
                try {
                  await ref.read(authProvider.notifier).updateUserInfo({
                    'email': emailController.text.trim(),
                    'phone': phoneController.text.trim(),
                    'qq': qqController.text.trim(),
                    'bio': bioController.text.trim(),
                  });
                  if (context.mounted) {
                    ScaffoldMessenger.of(
                      context,
                    ).showSnackBar(const SnackBar(content: Text('保存成功')));
                  }
                } catch (e) {
                  if (context.mounted) {
                    ScaffoldMessenger.of(
                      context,
                    ).showSnackBar(SnackBar(content: Text('保存失败: $e')));
                  }
                }
              },
            ),
          ],
        ),
      ),
    );
  }
}
