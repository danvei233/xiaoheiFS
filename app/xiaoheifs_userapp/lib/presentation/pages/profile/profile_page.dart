import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/constants/app_colors.dart';
import '../../../core/constants/input_limits.dart';
import '../../../core/network/api_client.dart';
import '../../../core/utils/avatar_url.dart';
import '../../providers/auth_provider.dart';
import '../../widgets/common/app_button.dart';
import '../../widgets/common/app_input.dart';

/// 个人设置页面（仅支持修改 QQ 和个人简介）
class ProfilePage extends ConsumerStatefulWidget {
  const ProfilePage({super.key});

  @override
  ConsumerState<ProfilePage> createState() => _ProfilePageState();
}

class _ProfilePageState extends ConsumerState<ProfilePage> {
  final _qqController = TextEditingController();
  final _bioController = TextEditingController();
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final user = ref.read(authProvider).user;
      if (user == null) return;
      _qqController.text = _sanitizeQq(user.qq?.toString());
      _bioController.text = _sanitizeProfileValue(user.bio?.toString());
    });
  }

  @override
  void dispose() {
    _qqController.dispose();
    _bioController.dispose();
    super.dispose();
  }

  static String _sanitizeProfileValue(String? raw) {
    final value = (raw ?? '').trim();
    if (value.isEmpty) return '';
    if (value.contains('�')) return '';
    if (RegExp(r'[\u0080-\u009f]').hasMatch(value)) return '';
    return value;
  }

  static String _sanitizeQq(String? raw) {
    final digits = (raw ?? '').replaceAll(RegExp(r'[^0-9]'), '');
    if (digits.isEmpty) return '';
    if (digits.length >= 5 && digits.length <= 11 && !digits.startsWith('0')) {
      return digits;
    }
    return '';
  }

  int _runeLength(String text) => text.runes.length;

  Future<void> _save() async {
    if (_saving) return;
    final qq = _qqController.text.trim();
    final bio = _bioController.text.trim();

    if (_runeLength(qq) > InputLimits.qq) {
      _showError('QQ 长度不能超过 ${InputLimits.qq} 个字符');
      return;
    }
    if (_runeLength(bio) > InputLimits.bio) {
      _showError('个人简介长度不能超过 ${InputLimits.bio} 个字符');
      return;
    }

    setState(() {
      _saving = true;
    });
    try {
      await ref.read(authProvider.notifier).updateUserInfo({
        'qq': qq,
        'bio': bio,
      });
      _showSuccess('保存成功');
    } catch (e) {
      _showError(e.toString().replaceAll('Exception: ', '').trim());
    } finally {
      if (mounted) {
        setState(() {
          _saving = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final user = ref.watch(authProvider).user;
    if (user == null) {
      return const Scaffold(body: Center(child: Text('请先登录')));
    }
    final avatarUrl = resolveUserAvatarUrl(
      baseUrl: ApiClient.instance.dio.options.baseUrl,
      qq: user.qq?.toString(),
      avatarUrl: user.avatarUrl?.toString(),
      avatar: user.avatar?.toString(),
    );

    return Scaffold(
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              _buildProfileHeader(user, avatarUrl),
              const SizedBox(height: 16),
              _buildEditableSection(),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildProfileHeader(dynamic user, String avatarUrl) {
    return Row(
      children: [
        avatarUrl.isNotEmpty
            ? CircleAvatar(radius: 28, backgroundImage: NetworkImage(avatarUrl))
            : CircleAvatar(
                radius: 28,
                backgroundColor: AppColors.primary,
                child: Text(
                  (user.username ?? 'U')[0].toUpperCase(),
                  style: const TextStyle(
                    fontSize: 20,
                    fontWeight: FontWeight.bold,
                    color: Colors.white,
                  ),
                ),
              ),
        const SizedBox(width: 12),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                user.username ?? '',
                style: const TextStyle(
                  fontSize: 17,
                  fontWeight: FontWeight.w700,
                ),
              ),
              const SizedBox(height: 3),
              Text(
                user.email?.toString().trim().isEmpty == true
                    ? '未绑定邮箱'
                    : user.email.toString(),
                style: const TextStyle(color: AppColors.gray500, fontSize: 12),
              ),
              Text(
                user.phone?.toString().trim().isEmpty == true
                    ? '未绑定手机号'
                    : user.phone.toString(),
                style: const TextStyle(color: AppColors.gray500, fontSize: 12),
              ),
            ],
          ),
        ),
      ],
    );
  }

  Widget _buildEditableSection() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        AppInput(
          label: 'QQ',
          hint: '请输入QQ号',
          controller: _qqController,
          maxLength: InputLimits.qq,
          keyboardType: TextInputType.number,
          inputFormatters: [FilteringTextInputFormatter.digitsOnly],
        ),
        const SizedBox(height: 12),
        AppInput(
          label: '个人简介',
          hint: '请输入个人简介',
          controller: _bioController,
          maxLength: InputLimits.bio,
          maxLines: 4,
        ),
        const SizedBox(height: 14),
        AppButton(text: '保存资料', onPressed: _save, isLoading: _saving),
      ],
    );
  }

  void _showError(String message) {
    if (!mounted) return;
    ScaffoldMessenger.of(context)
      ..hideCurrentSnackBar()
      ..showSnackBar(
        SnackBar(content: Text(message), backgroundColor: AppColors.danger),
      );
  }

  void _showSuccess(String message) {
    if (!mounted) return;
    ScaffoldMessenger.of(context)
      ..hideCurrentSnackBar()
      ..showSnackBar(
        SnackBar(content: Text(message), backgroundColor: AppColors.success),
      );
  }
}
