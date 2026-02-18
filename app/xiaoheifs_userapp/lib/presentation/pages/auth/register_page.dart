import 'dart:async';
import 'dart:convert';

import 'package:flutter/cupertino.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/constants/app_colors.dart';
import '../../../core/constants/input_limits.dart';
import '../../providers/auth_provider.dart';
import '../../widgets/common/app_button.dart';
import '../../widgets/common/app_input.dart';
import 'gt4_helper.dart';

class RegisterPage extends ConsumerStatefulWidget {
  const RegisterPage({super.key});

  @override
  ConsumerState<RegisterPage> createState() => _RegisterPageState();
}

class _RegisterPageState extends ConsumerState<RegisterPage> {
  int _step = 1;

  final _usernameController = TextEditingController();
  final _emailController = TextEditingController();
  final _qqController = TextEditingController();
  final _phoneController = TextEditingController();
  final _passwordController = TextEditingController();
  final _captchaController = TextEditingController();
  final _verifyCodeController = TextEditingController();

  bool _loading = false;
  bool _sendCooling = false;
  int _sendCount = 60;
  Timer? _sendTimer;

  bool _registerEnabled = true;
  List<String> _requiredFields = const ['username', 'password'];
  bool _registerEmailRequired = true;
  List<String> _verifyChannels = const [];
  String _verifyChannel = 'email';

  bool _registerCaptchaEnabled = true;
  String _captchaProvider = 'image';
  String _captchaId = '';
  String _captchaImageBase64 = '';
  bool _geetestLoading = false;
  GeeTestResult _geetestResult = GeeTestResult.empty;

  @override
  void initState() {
    super.initState();
    _loadSettings();
  }

  @override
  void dispose() {
    _sendTimer?.cancel();
    _usernameController.dispose();
    _emailController.dispose();
    _qqController.dispose();
    _phoneController.dispose();
    _passwordController.dispose();
    _captchaController.dispose();
    _verifyCodeController.dispose();
    super.dispose();
  }

  bool _isRequired(String field) {
    final set = _requiredFields.map((e) => e.toLowerCase().trim()).toSet();
    return set.contains(field.toLowerCase());
  }

  bool get _showChannelTabs =>
      _verifyChannels.contains('email') && _verifyChannels.contains('sms');

  bool get _showEmailField =>
      _verifyChannel == 'email' ||
      _registerEmailRequired ||
      _isRequired('email');

  bool get _showPhoneField => _verifyChannel == 'sms' || _isRequired('phone');

  bool get _canSendCode {
    if (_verifyChannel == 'email') {
      return _emailController.text.trim().isNotEmpty;
    }
    if (_verifyChannel == 'sms') {
      return _phoneController.text.trim().isNotEmpty;
    }
    return false;
  }

  Future<void> _loadSettings() async {
    try {
      final settings = await ref.read(authRepositoryProvider).getAuthSettings();
      _registerEnabled = settings['register_enabled'] != false;
      _registerEmailRequired = settings['register_email_required'] != false;

      final fields = settings['register_required_fields'];
      if (fields is List) {
        _requiredFields = fields.map((e) => e.toString()).toList();
      }

      final channels = settings['register_verify_channels'];
      if (channels is List && channels.isNotEmpty) {
        _verifyChannels = channels
            .map((e) => e.toString().toLowerCase())
            .toList();
      } else {
        final type = (settings['register_verify_type'] ?? 'none')
            .toString()
            .toLowerCase();
        if (type == 'email' || type == 'sms') {
          _verifyChannels = [type];
        }
      }

      if (_verifyChannels.isNotEmpty &&
          !_verifyChannels.contains(_verifyChannel)) {
        _verifyChannel = _verifyChannels.first;
      }

      _registerCaptchaEnabled = settings['register_captcha_enabled'] != false;
      final provider = (settings['captcha_provider'] ?? 'image')
          .toString()
          .toLowerCase();
      _captchaProvider = provider == 'geetest' ? 'geetest' : 'image';
    } catch (_) {
      _registerEnabled = true;
      _registerCaptchaEnabled = true;
      _captchaProvider = 'image';
    } finally {
      if (_registerCaptchaEnabled) {
        await _refreshCaptcha();
      }
      if (mounted) {
        setState(() {});
      }
    }
  }

  Future<void> _refreshCaptcha() async {
    if (!_registerCaptchaEnabled) return;
    try {
      final data = await ref.read(authRepositoryProvider).getCaptcha();
      final provider = (data['captcha_provider'] ?? _captchaProvider)
          .toString()
          .toLowerCase();
      _captchaProvider = provider == 'geetest' ? 'geetest' : 'image';
      _captchaId = (data['captcha_id'] ?? '').toString();
      _geetestResult = GeeTestResult.empty;
      _captchaController.clear();

      if (_captchaProvider == 'image') {
        _captchaImageBase64 = (data['image_base64'] ?? '').toString();
      } else {
        _captchaImageBase64 = '';
      }
      if (mounted) setState(() {});
    } catch (_) {
      _captchaId = '';
      _captchaImageBase64 = '';
      _geetestResult = GeeTestResult.empty;
      if (mounted) setState(() {});
    }
  }

  Future<void> _verifyGeeTest() async {
    if (_captchaId.isEmpty) {
      _showError('验证码尚未就绪，请刷新后重试');
      return;
    }
    setState(() {
      _geetestLoading = true;
      _geetestResult = GeeTestResult.empty;
    });
    try {
      final result = await runGeeTestChallenge(_captchaId);
      setState(() {
        _geetestResult = result;
      });
      if (!result.passed) {
        if (result.canceled) {
          _showError('已取消验证');
        } else if (result.message.isNotEmpty) {
          _showError('极验失败: ${result.message}');
        } else if (result.errorCode.isNotEmpty) {
          _showError('极验失败: ${result.errorCode}');
        } else {
          _showError('极验未通过，请重试');
        }
      }
    } catch (e) {
      final msg = e.toString().contains('插件未注册')
          ? '极验插件未加载，需完整重启 App'
          : '极验验证失败，请重试';
      _showError(msg);
    } finally {
      if (mounted) {
        setState(() {
          _geetestLoading = false;
        });
      }
    }
  }

  Future<void> _sendCode() async {
    if (_sendCooling || !_canSendCode) return;

    if (_registerCaptchaEnabled &&
        _captchaProvider == 'geetest' &&
        !_geetestResult.passed) {
      _showError('请先完成极验验证');
      return;
    }

    if (_registerCaptchaEnabled &&
        _captchaProvider == 'image' &&
        _captchaController.text.trim().isEmpty) {
      _showError('请输入验证码');
      return;
    }

    setState(() {
      _sendCooling = true;
      _sendCount = 60;
    });

    try {
      await ref.read(authRepositoryProvider).requestRegisterCode({
        'channel': _verifyChannel,
        'email': _verifyChannel == 'email' ? _emailController.text.trim() : '',
        'phone': _verifyChannel == 'sms' ? _phoneController.text.trim() : '',
        'captcha_id': _captchaId,
        'captcha_code': _captchaController.text.trim(),
        'lot_number': _geetestResult.lotNumber,
        'captcha_output': _geetestResult.captchaOutput,
        'pass_token': _geetestResult.passToken,
        'gen_time': _geetestResult.genTime,
      });

      _showSuccess('验证码已发送');
      _captchaController.clear();
      await _refreshCaptcha();

      _sendTimer?.cancel();
      _sendTimer = Timer.periodic(const Duration(seconds: 1), (timer) {
        if (!mounted) {
          timer.cancel();
          return;
        }
        if (_sendCount <= 1) {
          timer.cancel();
          setState(() {
            _sendCount = 60;
            _sendCooling = false;
          });
          return;
        }
        setState(() {
          _sendCount -= 1;
        });
      });
    } catch (e) {
      setState(() {
        _sendCooling = false;
        _sendCount = 60;
      });
      _showError(e.toString().replaceAll('Exception: ', '').trim());
      await _refreshCaptcha();
    }
  }

  Future<void> _submit() async {
    if (_loading) return;
    if (!_registerEnabled) {
      _showError('当前已关闭注册');
      return;
    }

    final username = _usernameController.text.trim();
    final email = _emailController.text.trim();
    final qq = _qqController.text.trim();
    final phone = _phoneController.text.trim();
    final password = _passwordController.text;
    final verifyCode = _verifyCodeController.text.trim();

    if (username.isEmpty) {
      _showError('请输入用户名');
      return;
    }
    if (username.runes.length > InputLimits.username) {
      _showError('用户名长度不能超过 ${InputLimits.username} 个字符');
      return;
    }

    if ((_showEmailField || _registerEmailRequired) && email.isEmpty) {
      _showError('请输入邮箱');
      return;
    }
    if (email.runes.length > InputLimits.email) {
      _showError('邮箱长度不能超过 ${InputLimits.email} 个字符');
      return;
    }

    if (_isRequired('qq') && qq.isEmpty) {
      _showError('请输入QQ');
      return;
    }
    if (qq.runes.length > InputLimits.qq) {
      _showError('QQ长度不能超过 ${InputLimits.qq} 个字符');
      return;
    }

    if (_showPhoneField && phone.isEmpty) {
      _showError('请输入手机号');
      return;
    }
    if (phone.runes.length > InputLimits.phone) {
      _showError('手机号长度不能超过 ${InputLimits.phone} 个字符');
      return;
    }

    if (password.isEmpty) {
      _showError('请输入密码');
      return;
    }
    if (password.runes.length > InputLimits.password) {
      _showError('密码长度不能超过 ${InputLimits.password} 个字符');
      return;
    }

    if (_verifyChannels.isNotEmpty && verifyCode.isEmpty) {
      _showError('请输入验证码');
      return;
    }

    if (_registerCaptchaEnabled) {
      if (_captchaProvider == 'geetest' && !_geetestResult.passed) {
        _showError('请先完成极验验证');
        return;
      }
      if (_captchaProvider == 'image' &&
          _captchaController.text.trim().isEmpty) {
        _showError('请输入验证码');
        return;
      }
    }

    setState(() {
      _loading = true;
    });

    try {
      await ref.read(authRepositoryProvider).register({
        'username': username,
        'email': _verifyChannel == 'email' ? email : '',
        'qq': qq,
        'phone': _verifyChannel == 'sms' ? phone : '',
        'password': password,
        'verify_channel': _verifyChannel,
        'verify_code': verifyCode,
        'captcha_id': _captchaId,
        'captcha_code': _captchaController.text.trim(),
        'lot_number': _geetestResult.lotNumber,
        'captcha_output': _geetestResult.captchaOutput,
        'pass_token': _geetestResult.passToken,
        'gen_time': _geetestResult.genTime,
      });

      if (!mounted) return;
      _showSuccess('注册成功，请登录');
      context.go('/login');
    } catch (e) {
      _showError(e.toString().replaceAll('Exception: ', '').trim());
      await _refreshCaptcha();
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
        title: const Text('用户注册'),
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
                    '创建新账号',
                    style: TextStyle(
                      color: Colors.white,
                      fontSize: 22,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  const SizedBox(height: 6),
                  Text(
                    _step == 1
                        ? '填写账号基础信息'
                        : _step == 2
                        ? '完成验证码校验'
                        : '设置密码并提交注册',
                    style: const TextStyle(
                      color: AppColors.gray400,
                      fontSize: 13,
                    ),
                  ),
                  const SizedBox(height: 20),
                  _buildStepHeader(compact),
                  const SizedBox(height: 26),
                  if (!_registerEnabled)
                    Container(
                      padding: const EdgeInsets.all(10),
                      margin: const EdgeInsets.only(bottom: 14),
                      decoration: BoxDecoration(
                        color: AppColors.warning.withOpacity(0.1),
                        borderRadius: BorderRadius.circular(8),
                        border: Border.all(
                          color: AppColors.warning.withOpacity(0.45),
                        ),
                      ),
                      child: const Text(
                        '当前已关闭注册',
                        style: TextStyle(color: AppColors.gray200),
                      ),
                    ),
                  if (_step == 1) _buildStepOne(),
                  if (_step == 2) _buildStepTwo(compact),
                  if (_step == 3) _buildStepThree(),
                  const SizedBox(height: 12),
                  Center(
                    child: TextButton(
                      onPressed: () => context.go('/login'),
                      child: const Text('已有账号？去登录'),
                    ),
                  ),
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
        _stepDot(3, '完成', compact),
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
          label: '用户名',
          controller: _usernameController,
          maxLength: InputLimits.username,
          onChanged: (_) => setState(() {}),
        ),
        if (_showChannelTabs) ...[
          const SizedBox(height: 8),
          _buildVerifyChannelTabs(),
        ],
        if (_showEmailField) ...[
          const SizedBox(height: 10),
          AppInput(
            label: (_registerEmailRequired || _verifyChannel == 'email')
                ? '邮箱'
                : '邮箱（选填）',
            controller: _emailController,
            maxLength: InputLimits.email,
            keyboardType: TextInputType.emailAddress,
            onChanged: (_) => setState(() {}),
          ),
        ],
        const SizedBox(height: 10),
        AppInput(
          label: _isRequired('qq') ? 'QQ' : 'QQ（选填）',
          controller: _qqController,
          maxLength: InputLimits.qq,
        ),
        if (_showPhoneField) ...[
          const SizedBox(height: 10),
          AppInput(
            label: _verifyChannel == 'sms' || _isRequired('phone')
                ? '手机号'
                : '手机号（选填）',
            controller: _phoneController,
            maxLength: InputLimits.phone,
            keyboardType: TextInputType.phone,
            onChanged: (_) => setState(() {}),
          ),
        ],
        const SizedBox(height: 12),
        AppButton(text: '下一步', onPressed: _nextFromStepOne),
      ],
    );
  }

  Widget _buildStepTwo(bool compact) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (_registerCaptchaEnabled) ...[
          _buildCaptchaWidget(),
          const SizedBox(height: 12),
        ] else
          const Text(
            '当前无需图形/行为验证码',
            style: TextStyle(color: AppColors.gray400, fontSize: 13),
          ),
        if (_verifyChannels.isNotEmpty) ...[
          AppInput(
            label: _verifyChannel == 'email' ? '邮箱验证码' : '短信验证码',
            controller: _verifyCodeController,
            maxLength: 12,
          ),
          const SizedBox(height: 10),
          AppButton(
            text: _sendCooling ? '${_sendCount}s' : '发送验证码',
            onPressed: (_sendCooling || !_canSendCode) ? null : _sendCode,
            isOutlined: true,
          ),
          const SizedBox(height: 12),
        ] else
          const SizedBox(height: 6),
        if (compact) ...[
          AppButton(text: '下一步', onPressed: _nextFromStepTwo),
          const SizedBox(height: 10),
          AppButton(
            text: '上一步',
            onPressed: () => setState(() => _step = 1),
            isOutlined: true,
          ),
        ] else
          Row(
            children: [
              Expanded(
                child: AppButton(
                  text: '上一步',
                  onPressed: () => setState(() => _step = 1),
                  isOutlined: true,
                ),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: AppButton(text: '下一步', onPressed: _nextFromStepTwo),
              ),
            ],
          ),
      ],
    );
  }

  Widget _buildStepThree() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        AppInput(
          label: '密码',
          controller: _passwordController,
          obscureText: true,
          maxLength: InputLimits.password,
        ),
        const SizedBox(height: 12),
        Row(
          children: [
            Expanded(
              child: AppButton(
                text: '上一步',
                onPressed: () => setState(() => _step = 2),
                isOutlined: true,
              ),
            ),
            const SizedBox(width: 10),
            Expanded(
              child: AppButton(
                text: '提交注册',
                onPressed: _submit,
                isLoading: _loading,
              ),
            ),
          ],
        ),
      ],
    );
  }

  Widget _buildVerifyChannelTabs() {
    final children = <String, Widget>{
      'email': const Padding(
        padding: EdgeInsets.symmetric(vertical: 6, horizontal: 14),
        child: Text(
          '邮箱注册',
          style: TextStyle(fontSize: 13, fontWeight: FontWeight.w600),
        ),
      ),
      'sms': const Padding(
        padding: EdgeInsets.symmetric(vertical: 6, horizontal: 14),
        child: Text(
          '手机号注册',
          style: TextStyle(fontSize: 13, fontWeight: FontWeight.w600),
        ),
      ),
    };

    return Theme(
      data: Theme.of(context).copyWith(
        cupertinoOverrideTheme: const CupertinoThemeData(
          primaryColor: AppColors.primary,
          barBackgroundColor: AppColors.gray800,
        ),
      ),
      child: CupertinoSlidingSegmentedControl<String>(
        groupValue: _verifyChannel,
        children: children,
        thumbColor: AppColors.primary,
        backgroundColor: AppColors.gray800,
        onValueChanged: (value) {
          if (value == null || value == _verifyChannel) return;
          setState(() {
            _verifyChannel = value;
            _verifyCodeController.clear();
          });
        },
      ),
    );
  }

  void _nextFromStepOne() {
    final username = _usernameController.text.trim();
    if (username.isEmpty) {
      _showError('请输入用户名');
      return;
    }
    if (username.runes.length > InputLimits.username) {
      _showError('用户名长度不能超过 ${InputLimits.username} 个字符');
      return;
    }

    final email = _emailController.text.trim();
    if ((_showEmailField || _registerEmailRequired) && email.isEmpty) {
      _showError('请输入邮箱');
      return;
    }
    if (email.runes.length > InputLimits.email) {
      _showError('邮箱长度不能超过 ${InputLimits.email} 个字符');
      return;
    }

    final qq = _qqController.text.trim();
    if (_isRequired('qq') && qq.isEmpty) {
      _showError('请输入QQ');
      return;
    }
    if (qq.runes.length > InputLimits.qq) {
      _showError('QQ长度不能超过 ${InputLimits.qq} 个字符');
      return;
    }

    final phone = _phoneController.text.trim();
    if (_showPhoneField && phone.isEmpty) {
      _showError('请输入手机号');
      return;
    }
    if (phone.runes.length > InputLimits.phone) {
      _showError('手机号长度不能超过 ${InputLimits.phone} 个字符');
      return;
    }

    setState(() {
      _step = 2;
    });
  }

  void _nextFromStepTwo() {
    if (_registerCaptchaEnabled) {
      if (_captchaProvider == 'geetest' && !_geetestResult.passed) {
        _showError('请先完成极验验证');
        return;
      }
      if (_captchaProvider == 'image' &&
          _captchaController.text.trim().isEmpty) {
        _showError('请输入验证码');
        return;
      }
    }

    if (_verifyChannels.isNotEmpty &&
        _verifyCodeController.text.trim().isEmpty) {
      _showError('请输入验证码');
      return;
    }

    setState(() {
      _step = 3;
    });
  }

  Widget _buildCaptchaWidget() {
    if (_captchaProvider == 'geetest') {
      return Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text('行为验证码'),
          const SizedBox(height: 8),
          _buildGeeTestActionButton(),
        ],
      );
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text('图形验证码'),
        const SizedBox(height: 8),
        Row(
          children: [
            Expanded(
              child: AppInput(
                hint: '请输入验证码',
                controller: _captchaController,
                maxLength: 12,
              ),
            ),
            const SizedBox(width: 8),
            InkWell(
              onTap: _refreshCaptcha,
              child: Container(
                width: 120,
                height: 40,
                decoration: BoxDecoration(
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: AppColors.gray700),
                  color: AppColors.gray900,
                ),
                clipBehavior: Clip.hardEdge,
                child: _captchaImageBase64.isEmpty
                    ? const Center(
                        child: Text(
                          '点击刷新',
                          style: TextStyle(
                            color: AppColors.gray400,
                            fontSize: 12,
                          ),
                        ),
                      )
                    : Image.memory(
                        base64Decode(_captchaImageBase64),
                        fit: BoxFit.cover,
                        errorBuilder: (context, error, stackTrace) =>
                            const Center(
                              child: Text(
                                '点击刷新',
                                style: TextStyle(
                                  color: AppColors.gray400,
                                  fontSize: 12,
                                ),
                              ),
                            ),
                      ),
              ),
            ),
          ],
        ),
      ],
    );
  }

  Widget _buildGeeTestActionButton() {
    final passed = _geetestResult.passed;
    final loading = _geetestLoading;
    final enabled = !loading && !passed;

    return AnimatedContainer(
      duration: const Duration(milliseconds: 260),
      curve: Curves.easeOutCubic,
      height: 46,
      decoration: BoxDecoration(
        color: passed ? AppColors.success : AppColors.primary,
        borderRadius: BorderRadius.circular(10),
        boxShadow: [
          BoxShadow(
            color: (passed ? AppColors.success : AppColors.primary).withOpacity(
              0.28,
            ),
            blurRadius: passed ? 16 : 12,
            offset: const Offset(0, 6),
          ),
        ],
      ),
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          borderRadius: BorderRadius.circular(10),
          onTap: enabled ? _verifyGeeTest : null,
          child: Center(
            child: loading
                ? const SizedBox(
                    width: 22,
                    height: 22,
                    child: CircularProgressIndicator(
                      strokeWidth: 2.2,
                      valueColor: AlwaysStoppedAnimation<Color>(Colors.white),
                    ),
                  )
                : AnimatedSwitcher(
                    duration: const Duration(milliseconds: 240),
                    switchInCurve: Curves.easeOutBack,
                    switchOutCurve: Curves.easeIn,
                    transitionBuilder: (child, animation) {
                      return FadeTransition(
                        opacity: animation,
                        child: ScaleTransition(scale: animation, child: child),
                      );
                    },
                    child: passed
                        ? const Row(
                            key: ValueKey('gt_passed'),
                            mainAxisAlignment: MainAxisAlignment.center,
                            children: [
                              Icon(
                                Icons.check_circle_rounded,
                                color: Colors.white,
                                size: 20,
                              ),
                              SizedBox(width: 8),
                              Text(
                                '验证成功',
                                style: TextStyle(
                                  color: Colors.white,
                                  fontSize: 15,
                                  fontWeight: FontWeight.w700,
                                ),
                              ),
                            ],
                          )
                        : const Row(
                            key: ValueKey('gt_idle'),
                            mainAxisAlignment: MainAxisAlignment.center,
                            children: [
                              Icon(
                                Icons.verified_user_outlined,
                                color: Colors.white,
                                size: 20,
                              ),
                              SizedBox(width: 8),
                              Text(
                                '点击完成极验验证',
                                style: TextStyle(
                                  color: Colors.white,
                                  fontSize: 15,
                                  fontWeight: FontWeight.w600,
                                ),
                              ),
                            ],
                          ),
                  ),
          ),
        ),
      ),
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
