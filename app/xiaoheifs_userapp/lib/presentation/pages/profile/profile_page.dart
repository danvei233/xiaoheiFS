import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/constants/app_colors.dart';
import '../../../core/constants/app_strings.dart';
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
            // 用户信息卡片
            _buildUserInfoCard(user),
            const SizedBox(height: 24),

            // 编辑表单
            _buildEditForm(context, ref, user),
          ],
        ),
      ),
    );
  }

  Widget _buildUserInfoCard(dynamic user) {
    final avatar = user.avatarUrl ?? user.avatar_url;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Row(
          children: [
            CircleAvatar(
              radius: 40,
              backgroundColor: AppColors.primaryLight,
              backgroundImage: avatar != null && avatar.toString().isNotEmpty
                  ? NetworkImage(avatar.toString())
                  : null,
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
                    style: TextStyle(
                      fontSize: 14,
                      color: AppColors.gray500,
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildEditForm(BuildContext context, WidgetRef ref, dynamic user) {
    final emailController = TextEditingController(text: user.email ?? '');
    final phoneController = TextEditingController(text: user.phone ?? '');
    final qqController = TextEditingController(text: user.qq ?? '');
    final bioController = TextEditingController(text: user.bio ?? '');

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
                    'email': emailController.text,
                    'phone': phoneController.text,
                    'qq': qqController.text,
                    'bio': bioController.text,
                  });
                  if (context.mounted) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(content: Text('保存成功')),
                    );
                  }
                } catch (e) {
                  if (context.mounted) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      SnackBar(content: Text('保存失败: $e')),
                    );
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
