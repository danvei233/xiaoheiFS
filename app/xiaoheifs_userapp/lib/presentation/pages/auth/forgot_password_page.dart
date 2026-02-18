import 'package:flutter/cupertino.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/constants/app_colors.dart';
import '../../../core/constants/input_limits.dart';
import '../../providers/auth_provider.dart';
import '../../widgets/common/app_button.dart';
import '../../widgets/common/app_input.dart';

class ForgotPasswordPage extends ConsumerStatefulWidget {
  const ForgotPasswordPage({super.key});

  @override
  ConsumerState<ForgotPasswordPage> createState() => _ForgotPasswordPageState();
}

class _ForgotPasswordPageState extends ConsumerState<ForgotPasswordPage> {
  int _step = 1;
  bool _loading = false;
  bool _sending = false;

  final _accountController = TextEditingController();
  final _codeController = TextEditingController();
  final _phoneFullController = TextEditingController();
  final _newPasswordController = TextEditingController();
  final _confirmPasswordController = TextEditingController();

  List<String> _channels = const [];
  String _channel = 'email';
  String _maskedPhone = '';
  bool _smsRequiresPhoneFull = false;
  String _resetTicket = '';

  @override
  void dispose() {
    _accountController.dispose();
    _codeController.dispose();
    _phoneFullController.dispose();
    _newPasswordController.dispose();
    _confirmPasswordController.dispose();
    super.dispose();
  }

  Future<void> _loadOptions() async {
    final account = _accountController.text.trim();
    if (account.isEmpty) {
      _showError('请输入账户名/邮箱/手机号');
      return;
    }
    if (account.runes.length > InputLimits.email) {
      _showError('输入长度不能超过 ${InputLimits.email} 个字符');
      return;
    }

    setState(() {
      _loading = true;
    });

    try {
      final data = await ref
          .read(authRepositoryProvider)
          .getPasswordResetOptions(account);
      final channelsRaw = data['channels'];
      final list = channelsRaw is List
          ? channelsRaw.map((e) => e.toString().toLowerCase()).toList()
          : <String>[];
      if (list.isEmpty) {
        _showError('当前账号未绑定可用的找回方式');
        return;
      }

      setState(() {
        _channels = list;
        _channel = list.first;
        _maskedPhone = (data['masked_phone'] ?? '').toString();
        _smsRequiresPhoneFull = data['sms_requires_phone_full'] == true;
        _codeController.clear();
        _phoneFullController.clear();
        _step = 2;
      });
    } catch (e) {
      _showError(e.toString().replaceAll('Exception: ', '').trim());
    } finally {
      if (mounted) {
        setState(() {
          _loading = false;
        });
      }
    }
  }

  Future<void> _sendCode() async {
    if (_sending) return;

    if (_channel == 'sms' && _smsRequiresPhoneFull) {
      final phoneFull = _phoneFullController.text.trim();
      if (phoneFull.isEmpty) {
        _showError('请输入完整手机号');
        return;
      }
      if (!RegExp(r'^[0-9+\-\s]{6,20}$').hasMatch(phoneFull)) {
        _showError('请输入有效手机号');
        return;
      }
    }

    setState(() {
      _sending = true;
    });

    try {
      await ref.read(authRepositoryProvider).sendPasswordResetCode({
        'account': _accountController.text.trim(),
        'channel': _channel,
        if (_channel == 'sms' && _smsRequiresPhoneFull)
          'phone_full': _phoneFullController.text.trim(),
      });
      _showSuccess('验证码已发送');
    } catch (e) {
      _showError(e.toString().replaceAll('Exception: ', '').trim());
    } finally {
      if (mounted) {
        setState(() {
          _sending = false;
        });
      }
    }
  }

  Future<void> _verifyCode() async {
    final code = _codeController.text.trim();
    if (code.isEmpty) {
      _showError('请输入验证码');
      return;
    }
    if (code.length < 4 || code.length > 12) {
      _showError('验证码长度应为 4-12 位');
      return;
    }

    setState(() {
      _loading = true;
    });

    try {
      final data = await ref
          .read(authRepositoryProvider)
          .verifyPasswordResetCode({
            'account': _accountController.text.trim(),
            'channel': _channel,
            'code': code,
          });
      final ticket = (data['reset_ticket'] ?? '').toString();
      if (ticket.isEmpty) {
        _showError('获取重置票据失败');
        return;
      }
      setState(() {
        _resetTicket = ticket;
        _step = 3;
      });
    } catch (e) {
      _showError(e.toString().replaceAll('Exception: ', '').trim());
    } finally {
      if (mounted) {
        setState(() {
          _loading = false;
        });
      }
    }
  }

  Future<void> _submitReset() async {
    final newPassword = _newPasswordController.text;
    final confirmPassword = _confirmPasswordController.text;

    if (newPassword.isEmpty) {
      _showError('请输入新密码');
      return;
    }
    if (newPassword.length < 6 || newPassword.length > InputLimits.password) {
      _showError('密码长度应为 6-${InputLimits.password} 位');
      return;
    }
    if (confirmPassword != newPassword) {
      _showError('两次输入密码不一致');
      return;
    }

    setState(() {
      _loading = true;
    });

    try {
      await ref.read(authRepositoryProvider).confirmPasswordReset({
        'reset_ticket': _resetTicket,
        'new_password': newPassword,
      });
      if (!mounted) return;
      _showSuccess('密码重置成功，请登录');
      context.go('/login');
    } catch (e) {
      _showError(e.toString().replaceAll('Exception: ', '').trim());
    } finally {
      if (mounted) {
        setState(() {
          _loading = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final width = MediaQuery.of(context).size.width;
    final compact = width < 430;

    return Scaffold(
      backgroundColor: AppColors.darkBackground,
      appBar: AppBar(
        title: const Text('找回密码'),
        backgroundColor: AppColors.darkBackground,
        foregroundColor: AppColors.gray100,
        elevation: 0,
        scrolledUnderElevation: 0,
      ),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: EdgeInsets.fromLTRB(
            compact ? 16 : 24,
            22,
            compact ? 16 : 24,
            20,
          ),
          child: Align(
            alignment: Alignment.topCenter,
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 420),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  const Text(
                    '重置账户密码',
                    style: TextStyle(
                      color: Colors.white,
                      fontSize: 22,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  const SizedBox(height: 6),
                  Text(
                    _step == 1
                        ? '输入账号后查询可用找回方式'
                        : _step == 2
                        ? '完成验证码校验后继续'
                        : '设置新密码并完成重置',
                    style: const TextStyle(
                      color: AppColors.gray400,
                      fontSize: 13,
                    ),
                  ),
                  const SizedBox(height: 16),
                  _buildStepHeader(compact),
                  const SizedBox(height: 16),
                  if (_step == 1) _buildStepOne(),
                  if (_step == 2) _buildStepTwo(compact),
                  if (_step == 3) _buildStepThree(),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildStepHeader(bool compact) {
    return Row(
      children: [
        _stepDot(1, '账号', compact),
        Expanded(child: Divider(color: AppColors.gray600.withOpacity(0.5))),
        _stepDot(2, '验证', compact),
        Expanded(child: Divider(color: AppColors.gray600.withOpacity(0.5))),
        _stepDot(3, '新密码', compact),
      ],
    );
  }

  Widget _stepDot(int index, String title, bool compact) {
    final active = _step >= index;
    return Column(
      children: [
        AnimatedContainer(
          duration: const Duration(milliseconds: 200),
          width: compact ? 26 : 28,
          height: compact ? 26 : 28,
          decoration: BoxDecoration(
            color: active ? AppColors.primary : AppColors.gray700,
            borderRadius: BorderRadius.circular(999),
          ),
          child: Center(
            child: Text(
              '$index',
              style: TextStyle(
                fontSize: compact ? 11 : 12,
                color: Colors.white,
                fontWeight: FontWeight.w700,
              ),
            ),
          ),
        ),
        const SizedBox(height: 5),
        Text(
          title,
          style: TextStyle(
            fontSize: compact ? 11 : 12,
            color: active ? AppColors.gray100 : AppColors.gray400,
          ),
        ),
      ],
    );
  }

  Widget _buildStepOne() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        AppInput(
          label: '账户名/邮箱/手机号',
          controller: _accountController,
          maxLength: InputLimits.email,
        ),
        const SizedBox(height: 12),
        AppButton(text: '下一步', onPressed: _loadOptions, isLoading: _loading),
      ],
    );
  }

  Widget _buildStepTwo(bool compact) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        const Text(
          '重置方式',
          style: TextStyle(
            color: AppColors.gray300,
            fontWeight: FontWeight.w600,
          ),
        ),
        const SizedBox(height: 8),
        _buildResetChannelTabs(),
        const SizedBox(height: 20),
        if (_channel == 'sms' && _smsRequiresPhoneFull) ...[
          Container(
            padding: const EdgeInsets.all(10),
            decoration: BoxDecoration(
              color: AppColors.gray800,
              borderRadius: BorderRadius.circular(10),
              border: Border.all(color: AppColors.gray700),
            ),
            child: Text(
              '您的手机号是${_maskedPhone.isEmpty ? '已绑定号码' : _maskedPhone}，请补全后发送验证码',
              style: const TextStyle(color: AppColors.gray200, height: 1.35),
            ),
          ),
          const SizedBox(height: 10),
          AppInput(
            label: '完整手机号（用于校验）',
            controller: _phoneFullController,
            maxLength: InputLimits.phone,
            keyboardType: TextInputType.phone,
          ),
        ],
        const SizedBox(height: 10),
        AppInput(label: '验证码', controller: _codeController, maxLength: 12),
        const SizedBox(height: 12),
        if (compact) ...[
          AppButton(
            text: _channel == 'email' ? '发送邮箱验证码' : '发送短信验证码',
            onPressed: _sendCode,
            isOutlined: true,
            isLoading: _sending,
          ),
          const SizedBox(height: 10),
          AppButton(text: '验证并继续', onPressed: _verifyCode, isLoading: _loading),
        ] else
          Row(
            children: [
              Expanded(
                child: AppButton(
                  text: _channel == 'email' ? '发送邮箱验证码' : '发送短信验证码',
                  onPressed: _sendCode,
                  isOutlined: true,
                  isLoading: _sending,
                ),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: AppButton(
                  text: '验证并继续',
                  onPressed: _verifyCode,
                  isLoading: _loading,
                ),
              ),
            ],
          ),
      ],
    );
  }

  Widget _buildResetChannelTabs() {
    final hasEmail = _channels.contains('email');
    final hasSms = _channels.contains('sms');

    final children = <String, Widget>{};
    if (hasEmail) {
      children['email'] = const Padding(
        padding: EdgeInsets.symmetric(vertical: 6, horizontal: 14),
        child: Text(
          '邮箱',
          style: TextStyle(fontSize: 13, fontWeight: FontWeight.w600),
        ),
      );
    }
    if (hasSms) {
      children['sms'] = const Padding(
        padding: EdgeInsets.symmetric(vertical: 6, horizontal: 14),
        child: Text(
          '手机号',
          style: TextStyle(fontSize: 13, fontWeight: FontWeight.w600),
        ),
      );
    }

    return Theme(
      data: Theme.of(context).copyWith(
        cupertinoOverrideTheme: const CupertinoThemeData(
          primaryColor: AppColors.primary,
          barBackgroundColor: AppColors.gray800,
        ),
      ),
      child: CupertinoSlidingSegmentedControl<String>(
        groupValue: _channel,
        children: children,
        thumbColor: AppColors.primary,
        backgroundColor: AppColors.gray800,
        onValueChanged: (value) {
          if (value == null || value == _channel) return;
          setState(() {
            _channel = value;
          });
        },
      ),
    );
  }

  Widget _buildStepThree() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        AppInput(
          label: '新密码',
          controller: _newPasswordController,
          obscureText: true,
          maxLength: InputLimits.password,
        ),
        const SizedBox(height: 10),
        AppInput(
          label: '确认密码',
          controller: _confirmPasswordController,
          obscureText: true,
          maxLength: InputLimits.password,
        ),
        const SizedBox(height: 12),
        AppButton(text: '重置密码', onPressed: _submitReset, isLoading: _loading),
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
