import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:qr_flutter/qr_flutter.dart';

import '../../../core/constants/app_colors.dart';
import '../../../core/constants/input_limits.dart';
import '../../../core/network/api_client.dart';
import '../../../core/utils/avatar_url.dart';
import '../../providers/auth_provider.dart';
import '../../widgets/common/app_button.dart';
import '../../widgets/common/app_input.dart';

class SecurityCenterPage extends ConsumerStatefulWidget {
  const SecurityCenterPage({super.key});

  @override
  ConsumerState<SecurityCenterPage> createState() => _SecurityCenterPageState();
}

class _SecurityCenterPageState extends ConsumerState<SecurityCenterPage> {
  final _username = TextEditingController();
  final _usernameTotp = TextEditingController();
  final _pwdCurrent = TextEditingController();
  final _pwdNew = TextEditingController();
  final _pwdConfirm = TextEditingController();
  final _pwdTotp = TextEditingController();
  final _twofaPassword = TextEditingController();
  final _twofaCurrentCode = TextEditingController();
  final _twofaConfirmCode = TextEditingController();
  final _emailValue = TextEditingController();
  final _emailCode = TextEditingController();
  final _emailPassword = TextEditingController();
  final _emailTotp = TextEditingController();
  final _phoneValue = TextEditingController();
  final _phoneCode = TextEditingController();
  final _phonePassword = TextEditingController();
  final _phoneTotp = TextEditingController();

  Timer? _emailTimer;
  Timer? _phoneTimer;
  int _emailCd = 0;
  int _phoneCd = 0;

  bool _loading = false;
  bool _twofaEnabled = false;
  bool _emailBound = false;
  bool _phoneBound = false;
  String _emailMasked = '';
  String _phoneMasked = '';
  String _emailTicket = '';
  String _phoneTicket = '';
  String _otpAuthUrl = '';
  String _otpSecret = '';

  bool _busyUser = false;
  bool _busyPwd = false;
  bool _busyTwofaSetup = false;
  bool _busyTwofaConfirm = false;
  bool _busyEmailTicket = false;
  bool _busyPhoneTicket = false;
  bool _busyEmailSend = false;
  bool _busyEmailConfirm = false;
  bool _busyPhoneSend = false;
  bool _busyPhoneConfirm = false;
  bool _emailPasswordVerified = false;
  bool _phonePasswordVerified = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _username.text = ref.read(authProvider).user?.username ?? '';
      _load();
    });
  }

  @override
  void dispose() {
    _emailTimer?.cancel();
    _phoneTimer?.cancel();
    for (final c in [
      _username,
      _usernameTotp,
      _pwdCurrent,
      _pwdNew,
      _pwdConfirm,
      _pwdTotp,
      _twofaPassword,
      _twofaCurrentCode,
      _twofaConfirmCode,
      _emailValue,
      _emailCode,
      _emailPassword,
      _emailTotp,
      _phoneValue,
      _phoneCode,
      _phonePassword,
      _phoneTotp,
    ]) {
      c.dispose();
    }
    super.dispose();
  }

  Future<void> _load() async {
    setState(() => _loading = true);
    try {
      final repo = ref.read(authRepositoryProvider);
      final data = await Future.wait([
        repo.getTwoFAStatus(),
        repo.getMySecurityContacts(),
      ]);
      final twofa = data[0];
      final contacts = data[1];
      _twofaEnabled =
          twofa['enabled'] == true ||
          twofa['totp_enabled'] == true ||
          contacts['totp_enabled'] == true;
      _emailBound = contacts['email_bound'] == true;
      _phoneBound = contacts['phone_bound'] == true;
      _emailMasked = (contacts['email_masked'] ?? '').toString();
      _phoneMasked = (contacts['phone_masked'] ?? '').toString();
    } catch (_) {
      final user = ref.read(authProvider).user;
      _emailMasked = user?.email ?? '';
      _phoneMasked = user?.phone ?? '';
      _emailBound = _emailMasked.isNotEmpty;
      _phoneBound = _phoneMasked.isNotEmpty;
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  int get _score {
    final user = ref.read(authProvider).user;
    var s = 10;
    if ((user?.username ?? '').isNotEmpty) s += 15;
    if ((user?.qq ?? '').isNotEmpty) s += 5;
    if ((user?.bio ?? '').isNotEmpty) s += 5;
    if (_emailBound) s += 20;
    if (_phoneBound) s += 20;
    if (_twofaEnabled) s += 25;
    return s.clamp(0, 100);
  }

  Future<void> _saveUsername() async {
    final name = _username.text.trim();
    if (name.isEmpty) return _err('请输入用户名');
    if (name.runes.length > InputLimits.username) {
      return _err('用户名长度不能超过 ${InputLimits.username}');
    }
    if (_twofaEnabled &&
        !RegExp(r'^\d{6}$').hasMatch(_usernameTotp.text.trim())) {
      return _err('请输入6位2FA验证码');
    }
    setState(() => _busyUser = true);
    try {
      await ref.read(authProvider.notifier).updateUserInfo({
        'username': name,
        if (_twofaEnabled) 'totp_code': _usernameTotp.text.trim(),
      });
      _ok('用户名已更新');
    } catch (e) {
      _err(e.toString().replaceAll('Exception: ', '').trim());
    } finally {
      if (mounted) {
        setState(() => _busyUser = false);
      }
    }
  }

  Future<void> _changePwd() async {
    if (_pwdCurrent.text.isEmpty ||
        _pwdNew.text.isEmpty ||
        _pwdConfirm.text.isEmpty) {
      return _err('请完整填写密码字段');
    }
    if (_pwdNew.text != _pwdConfirm.text) return _err('两次新密码不一致');
    if (_twofaEnabled && !RegExp(r'^\d{6}$').hasMatch(_pwdTotp.text.trim())) {
      return _err('请输入6位2FA验证码');
    }
    setState(() => _busyPwd = true);
    try {
      await ref.read(authRepositoryProvider).changeMyPassword({
        'current_password': _pwdCurrent.text,
        'new_password': _pwdNew.text,
        if (_twofaEnabled) 'totp_code': _pwdTotp.text.trim(),
      });
      _pwdCurrent.clear();
      _pwdNew.clear();
      _pwdConfirm.clear();
      _pwdTotp.clear();
      _ok('密码已更新');
    } catch (e) {
      _err(e.toString().replaceAll('Exception: ', '').trim());
    } finally {
      if (mounted) {
        setState(() => _busyPwd = false);
      }
    }
  }

  Future<void> _setupTwofa() async {
    if (!_twofaEnabled && _twofaPassword.text.trim().isEmpty) {
      return _err('请输入当前登录密码');
    }
    if (_twofaEnabled &&
        !RegExp(r'^\d{6}$').hasMatch(_twofaCurrentCode.text.trim())) {
      return _err('请输入当前2FA验证码');
    }
    setState(() => _busyTwofaSetup = true);
    try {
      final data = await ref.read(authRepositoryProvider).setupTwoFA({
        if (!_twofaEnabled) 'password': _twofaPassword.text.trim(),
        if (_twofaEnabled) 'current_code': _twofaCurrentCode.text.trim(),
      });
      _otpAuthUrl = (data['otpauth_url'] ?? '').toString();
      _otpSecret = (data['secret'] ?? '').toString().trim().isNotEmpty
          ? (data['secret'] ?? '').toString().trim()
          : _extractSecretFromOtpUrl(_otpAuthUrl);
      if (_otpAuthUrl.isEmpty) return _err('未获取到绑定信息');
      setState(() {});
      _ok('已生成绑定信息，请完成第2步验证');
    } catch (e) {
      _err(e.toString().replaceAll('Exception: ', '').trim());
    } finally {
      if (mounted) {
        setState(() => _busyTwofaSetup = false);
      }
    }
  }

  Future<void> _confirmTwofa() async {
    if (!RegExp(r'^\d{6}$').hasMatch(_twofaConfirmCode.text.trim())) {
      return _err('请输入6位验证码');
    }
    setState(() => _busyTwofaConfirm = true);
    try {
      await ref.read(authRepositoryProvider).confirmTwoFA({
        'code': _twofaConfirmCode.text.trim(),
      });
      _otpAuthUrl = '';
      _otpSecret = '';
      _twofaPassword.clear();
      _twofaCurrentCode.clear();
      _twofaConfirmCode.clear();
      await _load();
      _ok('2FA 已启用');
    } catch (e) {
      _err(e.toString().replaceAll('Exception: ', '').trim());
    } finally {
      if (mounted) {
        setState(() => _busyTwofaConfirm = false);
      }
    }
  }

  Future<void> _verifyEmailTicket() async {
    if (!RegExp(r'^\d{6}$').hasMatch(_emailTotp.text.trim())) {
      return _err('请输入6位2FA验证码');
    }
    setState(() => _busyEmailTicket = true);
    try {
      final data = await ref.read(authRepositoryProvider).verifyMyEmailBind2FA({
        'totp_code': _emailTotp.text.trim(),
      });
      _emailTicket = (data['security_ticket'] ?? '').toString();
      if (_emailTicket.isEmpty) return _err('2FA 校验失败');
      setState(() {});
      _ok('邮箱绑定2FA校验通过');
    } catch (e) {
      _err(e.toString().replaceAll('Exception: ', '').trim());
    } finally {
      if (mounted) {
        setState(() => _busyEmailTicket = false);
      }
    }
  }

  Future<void> _verifyPhoneTicket() async {
    if (!RegExp(r'^\d{6}$').hasMatch(_phoneTotp.text.trim())) {
      return _err('请输入6位2FA验证码');
    }
    setState(() => _busyPhoneTicket = true);
    try {
      final data = await ref.read(authRepositoryProvider).verifyMyPhoneBind2FA({
        'totp_code': _phoneTotp.text.trim(),
      });
      _phoneTicket = (data['security_ticket'] ?? '').toString();
      if (_phoneTicket.isEmpty) return _err('2FA 校验失败');
      setState(() {});
      _ok('手机绑定2FA校验通过');
    } catch (e) {
      _err(e.toString().replaceAll('Exception: ', '').trim());
    } finally {
      if (mounted) {
        setState(() => _busyPhoneTicket = false);
      }
    }
  }

  Future<void> _sendEmailCode() async {
    final v = _emailValue.text.trim();
    if (!RegExp(r'^[^\s@]+@[^\s@]+\.[^\s@]+$').hasMatch(v)) {
      return _err('请输入有效邮箱');
    }
    if (_twofaEnabled && _emailTicket.isEmpty) return _err('请先完成2FA校验');
    if (!_twofaEnabled && _emailPassword.text.trim().isEmpty) {
      return _err('请输入当前登录密码');
    }
    setState(() => _busyEmailSend = true);
    try {
      await ref.read(authRepositoryProvider).sendMyEmailBindCode({
        'value': v,
        if (_twofaEnabled) 'security_ticket': _emailTicket,
        if (!_twofaEnabled) 'current_password': _emailPassword.text.trim(),
      });
      _startCd(email: true);
      _ok('邮箱验证码已发送');
    } catch (e) {
      _err(e.toString().replaceAll('Exception: ', '').trim());
    } finally {
      if (mounted) {
        setState(() => _busyEmailSend = false);
      }
    }
  }

  Future<void> _confirmEmail() async {
    if (_emailValue.text.trim().isEmpty || _emailCode.text.trim().length < 4) {
      return _err('请输入邮箱和有效验证码');
    }
    setState(() => _busyEmailConfirm = true);
    try {
      await ref.read(authRepositoryProvider).confirmMyEmailBind({
        'value': _emailValue.text.trim(),
        'code': _emailCode.text.trim(),
        if (_twofaEnabled) 'security_ticket': _emailTicket,
      });
      _emailValue.clear();
      _emailCode.clear();
      _emailPassword.clear();
      _emailTotp.clear();
      _emailTicket = '';
      await ref.read(authProvider.notifier).refreshUser();
      await _load();
      _ok('邮箱绑定已更新');
    } catch (e) {
      _err(e.toString().replaceAll('Exception: ', '').trim());
    } finally {
      if (mounted) {
        setState(() => _busyEmailConfirm = false);
      }
    }
  }

  Future<void> _sendPhoneCode() async {
    final v = _phoneValue.text.trim();
    if (!RegExp(r'^[0-9+\-\s]{6,20}$').hasMatch(v)) return _err('请输入有效手机号');
    if (_twofaEnabled && _phoneTicket.isEmpty) return _err('请先完成2FA校验');
    if (!_twofaEnabled && _phonePassword.text.trim().isEmpty) {
      return _err('请输入当前登录密码');
    }
    setState(() => _busyPhoneSend = true);
    try {
      await ref.read(authRepositoryProvider).sendMyPhoneBindCode({
        'value': v,
        if (_twofaEnabled) 'security_ticket': _phoneTicket,
        if (!_twofaEnabled) 'current_password': _phonePassword.text.trim(),
      });
      _startCd(email: false);
      _ok('短信验证码已发送');
    } catch (e) {
      _err(e.toString().replaceAll('Exception: ', '').trim());
    } finally {
      if (mounted) {
        setState(() => _busyPhoneSend = false);
      }
    }
  }

  Future<void> _confirmPhone() async {
    if (_phoneValue.text.trim().isEmpty || _phoneCode.text.trim().length < 4) {
      return _err('请输入手机号和有效验证码');
    }
    setState(() => _busyPhoneConfirm = true);
    try {
      await ref.read(authRepositoryProvider).confirmMyPhoneBind({
        'value': _phoneValue.text.trim(),
        'code': _phoneCode.text.trim(),
        if (_twofaEnabled) 'security_ticket': _phoneTicket,
      });
      _phoneValue.clear();
      _phoneCode.clear();
      _phonePassword.clear();
      _phoneTotp.clear();
      _phoneTicket = '';
      await ref.read(authProvider.notifier).refreshUser();
      await _load();
      _ok('手机号绑定已更新');
    } catch (e) {
      _err(e.toString().replaceAll('Exception: ', '').trim());
    } finally {
      if (mounted) setState(() => _busyPhoneConfirm = false);
    }
  }

  void _startCd({required bool email}) {
    if (email) {
      _emailTimer?.cancel();
      _emailCd = 60;
      _emailTimer = Timer.periodic(const Duration(seconds: 1), (t) {
        if (!mounted || _emailCd <= 1) {
          t.cancel();
          if (mounted) setState(() => _emailCd = 0);
          return;
        }
        setState(() => _emailCd -= 1);
      });
    } else {
      _phoneTimer?.cancel();
      _phoneCd = 60;
      _phoneTimer = Timer.periodic(const Duration(seconds: 1), (t) {
        if (!mounted || _phoneCd <= 1) {
          t.cancel();
          if (mounted) setState(() => _phoneCd = 0);
          return;
        }
        setState(() => _phoneCd -= 1);
      });
    }
    setState(() {});
  }

  @override
  Widget build(BuildContext context) {
    final user = ref.watch(authProvider).user;
    if (user == null) {
      return const Scaffold(body: Center(child: Text('请先登录')));
    }
    final avatarUrl = resolveUserAvatarUrl(
      baseUrl: ApiClient.instance.dio.options.baseUrl,
      qq: user.qq,
      avatarUrl: user.avatarUrl,
      avatar: user.avatar,
    );
    return Scaffold(
      body: SafeArea(
        child: RefreshIndicator(
          onRefresh: _load,
          child: _loading
              ? ListView(
                  physics: const AlwaysScrollableScrollPhysics(),
                  children: const [
                    SizedBox(height: 220),
                    Center(child: CircularProgressIndicator()),
                  ],
                )
              : ListView(
                  physics: const AlwaysScrollableScrollPhysics(),
                  padding: const EdgeInsets.all(16),
                  children: [
                    _buildTopProfileRow(user, avatarUrl),
                    const SizedBox(height: 16),
                    _buildSecurityScoreStrip(),
                    const SizedBox(height: 16),
                    _buildActionRow(
                      icon: Icons.person_outline,
                      title: '用户名',
                      subtitle: user.username?.trim().isEmpty == true
                          ? '未设置'
                          : user.username!,
                      onTap: () => _openPanel(
                        '修改用户名',
                        (refresh) => _buildUsernamePanel(refresh),
                      ),
                    ),
                    _buildActionRow(
                      icon: Icons.lock_outline,
                      title: '登录密码',
                      subtitle: _twofaEnabled ? '已启用2FA校验' : '可直接修改',
                      onTap: () => _openPanel(
                        '修改密码',
                        (refresh) => _buildPasswordPanel(refresh),
                      ),
                    ),
                    _buildActionRow(
                      icon: Icons.shield_outlined,
                      title: '双重验证 (2FA)',
                      subtitle: _twofaEnabled ? '已启用' : '未启用',
                      onTap: () {
                        setState(() {
                          _otpAuthUrl = '';
                          _twofaConfirmCode.clear();
                        });
                        _openPanel(
                          '2FA 设置',
                          (refresh) => _buildTwofaPanel(refresh),
                        );
                      },
                    ),
                    _buildActionRow(
                      icon: Icons.mail_outline,
                      title: '邮箱绑定',
                      subtitle: _emailBound
                          ? (_emailMasked.isEmpty ? '已绑定' : _emailMasked)
                          : '未绑定',
                      onTap: () {
                        setState(() {
                          _emailTicket = '';
                          _emailPasswordVerified = false;
                        });
                        _openPanel(
                          '邮箱绑定 / 换绑',
                          (refresh) => _buildEmailPanel(refresh),
                        );
                      },
                    ),
                    _buildActionRow(
                      icon: Icons.phone_outlined,
                      title: '手机绑定',
                      subtitle: _phoneBound
                          ? (_phoneMasked.isEmpty ? '已绑定' : _phoneMasked)
                          : '未绑定',
                      onTap: () {
                        setState(() {
                          _phoneTicket = '';
                          _phonePasswordVerified = false;
                        });
                        _openPanel(
                          '手机号绑定 / 换绑',
                          (refresh) => _buildPhonePanel(refresh),
                        );
                      },
                    ),
                  ],
                ),
        ),
      ),
    );
  }

  Widget _buildTopProfileRow(dynamic user, String avatarUrl) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Row(
          children: [
            avatarUrl.isNotEmpty
                ? CircleAvatar(
                    radius: 28,
                    backgroundImage: NetworkImage(avatarUrl),
                  )
                : CircleAvatar(
                    radius: 28,
                    backgroundColor: AppColors.primaryLight,
                    child: Text(
                      (user.username ?? 'U')[0].toUpperCase(),
                      style: const TextStyle(
                        color: Colors.white,
                        fontWeight: FontWeight.bold,
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
                      fontSize: 16,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    _emailMasked.isEmpty ? '未绑定邮箱' : _emailMasked,
                    style: TextStyle(color: AppColors.gray500, fontSize: 12),
                  ),
                  const SizedBox(height: 2),
                  Text(
                    _phoneMasked.isEmpty ? '未绑定手机号' : _phoneMasked,
                    style: TextStyle(color: AppColors.gray500, fontSize: 12),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildSecurityScoreStrip() {
    final color = _score >= 85
        ? AppColors.success
        : (_score >= 60 ? AppColors.warning : AppColors.danger);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          children: [
            SizedBox(
              width: 88,
              height: 88,
              child: Stack(
                alignment: Alignment.center,
                children: [
                  SizedBox(
                    width: 88,
                    height: 88,
                    child: CircularProgressIndicator(
                      value: _score / 100,
                      strokeWidth: 8,
                      backgroundColor: Theme.of(context)
                          .colorScheme
                          .surfaceContainerHighest
                          .withValues(
                            alpha:
                                Theme.of(context).colorScheme.brightness ==
                                    Brightness.light
                                ? 0.9
                                : 0.62,
                          ),
                      valueColor: AlwaysStoppedAnimation(color),
                    ),
                  ),
                  Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Text(
                        '$_score',
                        style: TextStyle(
                          color: color,
                          fontSize: 20,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                      const Text(
                        '/ 100',
                        style: TextStyle(
                          fontSize: 11,
                          color: AppColors.gray500,
                        ),
                      ),
                    ],
                  ),
                ],
              ),
            ),
            const SizedBox(width: 14),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Text(
                    '安全评分',
                    style: TextStyle(fontWeight: FontWeight.w700, fontSize: 15),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    _score >= 85
                        ? '安全等级高'
                        : _score >= 60
                        ? '安全等级中'
                        : '安全等级低',
                    style: TextStyle(color: color, fontWeight: FontWeight.w600),
                  ),
                  const SizedBox(height: 2),
                  const Text(
                    '建议开启2FA并绑定邮箱/手机号',
                    style: TextStyle(color: AppColors.gray500, fontSize: 12),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildActionRow({
    required IconData icon,
    required String title,
    required String subtitle,
    required VoidCallback onTap,
  }) {
    return Card(
      child: ListTile(
        leading: Icon(icon),
        title: Text(title),
        subtitle: Text(subtitle, maxLines: 1, overflow: TextOverflow.ellipsis),
        trailing: const Icon(Icons.arrow_forward_ios, size: 16),
        onTap: onTap,
      ),
    );
  }

  Future<void> _openPanel(
    String title,
    Widget Function(VoidCallback refreshPanel) childBuilder,
  ) async {
    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      showDragHandle: true,
      builder: (sheetContext) {
        final bottom = MediaQuery.of(sheetContext).viewInsets.bottom;
        return StatefulBuilder(
          builder: (context, setModalState) {
            void refreshPanel() => setModalState(() {});
            return SafeArea(
              child: Padding(
                padding: EdgeInsets.fromLTRB(16, 8, 16, bottom + 16),
                child: SingleChildScrollView(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      Text(
                        title,
                        style: const TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                      const SizedBox(height: 14),
                      childBuilder(refreshPanel),
                    ],
                  ),
                ),
              ),
            );
          },
        );
      },
    );
  }

  Widget _buildStepLine({required int current, required List<String> labels}) {
    final cs = Theme.of(context).colorScheme;
    final isLight = cs.brightness == Brightness.light;
    return Row(
      children: List.generate(labels.length, (index) {
        final step = index + 1;
        final active = current >= step;
        return Expanded(
          child: Row(
            children: [
              Container(
                width: 22,
                height: 22,
                decoration: BoxDecoration(
                  color: active
                      ? cs.primary
                      : cs.surfaceContainerHighest.withValues(
                          alpha: isLight ? 0.95 : 0.62,
                        ),
                  shape: BoxShape.circle,
                  border: Border.all(
                    color: active
                        ? cs.primary.withValues(alpha: 0.75)
                        : cs.outlineVariant.withValues(
                            alpha: isLight ? 0.48 : 0.3,
                          ),
                  ),
                ),
                alignment: Alignment.center,
                child: Text(
                  '$step',
                  style: TextStyle(
                    color: active ? Colors.white : cs.onSurfaceVariant,
                    fontSize: 11,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ),
              const SizedBox(width: 6),
              Expanded(
                child: Text(
                  labels[index],
                  style: TextStyle(
                    fontSize: 12,
                    color: active ? cs.onSurface : cs.onSurfaceVariant,
                  ),
                ),
              ),
            ],
          ),
        );
      }),
    );
  }

  Widget _buildUsernamePanel(VoidCallback refreshPanel) => Column(
    crossAxisAlignment: CrossAxisAlignment.stretch,
    children: [
      AppInput(
        label: '新用户名',
        controller: _username,
        maxLength: InputLimits.username,
      ),
      if (_twofaEnabled) ...[
        const SizedBox(height: 10),
        AppInput(
          label: '2FA 验证码',
          controller: _usernameTotp,
          maxLength: 6,
          keyboardType: TextInputType.number,
        ),
      ],
      const SizedBox(height: 10),
      AppButton(
        text: '保存用户名',
        onPressed: () async {
          await _saveUsername();
          refreshPanel();
        },
        isLoading: _busyUser,
      ),
    ],
  );

  Widget _buildPasswordPanel(VoidCallback refreshPanel) => Column(
    crossAxisAlignment: CrossAxisAlignment.stretch,
    children: [
      AppInput(
        label: '当前密码',
        controller: _pwdCurrent,
        obscureText: true,
        maxLength: InputLimits.password,
      ),
      const SizedBox(height: 10),
      AppInput(
        label: '新密码',
        controller: _pwdNew,
        obscureText: true,
        maxLength: InputLimits.password,
      ),
      const SizedBox(height: 10),
      AppInput(
        label: '确认新密码',
        controller: _pwdConfirm,
        obscureText: true,
        maxLength: InputLimits.password,
      ),
      if (_twofaEnabled) ...[
        const SizedBox(height: 10),
        AppInput(
          label: '2FA 验证码',
          controller: _pwdTotp,
          maxLength: 6,
          keyboardType: TextInputType.number,
        ),
      ],
      const SizedBox(height: 10),
      AppButton(
        text: '更新密码',
        onPressed: () async {
          await _changePwd();
          refreshPanel();
        },
        isLoading: _busyPwd,
      ),
    ],
  );

  Widget _buildTwofaPanel(VoidCallback refreshPanel) {
    final step = _otpAuthUrl.isEmpty ? 1 : 2;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _buildStepLine(current: step, labels: const ['身份验证', '扫码确认']),
        const SizedBox(height: 10),
        if (step == 1) ...[
          !_twofaEnabled
              ? AppInput(
                  label: '当前登录密码',
                  controller: _twofaPassword,
                  obscureText: true,
                  maxLength: InputLimits.password,
                )
              : AppInput(
                  label: '当前2FA验证码',
                  controller: _twofaCurrentCode,
                  maxLength: 6,
                  keyboardType: TextInputType.number,
                ),
          const SizedBox(height: 10),
          AppButton(
            text: _twofaEnabled ? '生成新绑定信息' : '生成绑定信息',
            onPressed: () async {
              await _setupTwofa();
              refreshPanel();
            },
            isLoading: _busyTwofaSetup,
          ),
        ],
        if (step == 2) ...[
          Center(
            child: Container(
              width: 220,
              height: 220,
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: AppColors.gray100.withValues(alpha: 0.08),
                borderRadius: BorderRadius.circular(10),
              ),
              child: Center(
                child: SizedBox.square(
                  dimension: 180,
                  child: QrImageView(
                    data: _otpAuthUrl,
                    size: 180,
                    backgroundColor: Colors.white,
                  ),
                ),
              ),
            ),
          ),
          const SizedBox(height: 10),
          if (_otpSecret.isNotEmpty) ...[
            SelectableText(
              '手动绑定密钥: $_otpSecret',
              style: const TextStyle(fontSize: 12, color: AppColors.gray300),
            ),
            const SizedBox(height: 8),
            Row(
              children: [
                Expanded(
                  child: AppButton(
                    text: '复制密钥',
                    isOutlined: true,
                    onPressed: () async {
                      await Clipboard.setData(ClipboardData(text: _otpSecret));
                      _ok('密钥已复制');
                    },
                  ),
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: AppButton(
                    text: '复制绑定链接',
                    isOutlined: true,
                    onPressed: () async {
                      await Clipboard.setData(ClipboardData(text: _otpAuthUrl));
                      _ok('绑定链接已复制');
                    },
                  ),
                ),
              ],
            ),
            const SizedBox(height: 10),
          ],
          AppInput(
            label: '确认验证码',
            controller: _twofaConfirmCode,
            maxLength: 6,
            keyboardType: TextInputType.number,
          ),
          const SizedBox(height: 10),
          AppButton(
            text: '确认开启2FA',
            onPressed: () async {
              await _confirmTwofa();
              refreshPanel();
            },
            isLoading: _busyTwofaConfirm,
          ),
          const SizedBox(height: 8),
          TextButton(
            onPressed: () {
              _otpAuthUrl = '';
              refreshPanel();
            },
            child: const Text('返回上一步'),
          ),
        ],
      ],
    );
  }

  Widget _buildEmailPanel(VoidCallback refreshPanel) {
    final identityReady = _twofaEnabled
        ? _emailTicket.isNotEmpty
        : _emailPasswordVerified;
    final step = identityReady ? 2 : 1;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _buildStepLine(current: step, labels: const ['身份校验', '验证码绑定']),
        const SizedBox(height: 10),
        AppInput(
          label: '新邮箱',
          controller: _emailValue,
          maxLength: InputLimits.email,
          keyboardType: TextInputType.emailAddress,
        ),
        const SizedBox(height: 10),
        if (step == 1) ...[
          if (_twofaEnabled) ...[
            AppInput(
              label: '2FA验证码（用于身份校验）',
              controller: _emailTotp,
              maxLength: 6,
              keyboardType: TextInputType.number,
            ),
            const SizedBox(height: 8),
            AppButton(
              text: '验证并继续',
              onPressed: () async {
                await _verifyEmailTicket();
                refreshPanel();
              },
              isLoading: _busyEmailTicket,
            ),
          ] else ...[
            AppInput(
              label: '当前登录密码',
              controller: _emailPassword,
              obscureText: true,
              maxLength: InputLimits.password,
            ),
            const SizedBox(height: 8),
            AppButton(
              text: '验证并继续',
              onPressed: () {
                if (_emailPassword.text.trim().isEmpty) {
                  _err('请输入当前登录密码');
                  return;
                }
                _emailPasswordVerified = true;
                refreshPanel();
              },
            ),
          ],
        ],
        if (step == 2) ...[
          AppInput(
            label: '邮箱验证码',
            controller: _emailCode,
            maxLength: 12,
            keyboardType: TextInputType.number,
          ),
          const SizedBox(height: 10),
          Row(
            children: [
              Expanded(
                child: AppButton(
                  text: _emailCd > 0 ? '${_emailCd}s' : '发送验证码',
                  onPressed: (_emailCd > 0 || _busyEmailSend)
                      ? null
                      : () async {
                          await _sendEmailCode();
                          refreshPanel();
                        },
                  isOutlined: true,
                  isLoading: _busyEmailSend,
                ),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: AppButton(
                  text: '确认绑定',
                  onPressed: () async {
                    await _confirmEmail();
                    refreshPanel();
                  },
                  isLoading: _busyEmailConfirm,
                ),
              ),
            ],
          ),
        ],
      ],
    );
  }

  Widget _buildPhonePanel(VoidCallback refreshPanel) {
    final identityReady = _twofaEnabled
        ? _phoneTicket.isNotEmpty
        : _phonePasswordVerified;
    final step = identityReady ? 2 : 1;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _buildStepLine(current: step, labels: const ['身份校验', '验证码绑定']),
        const SizedBox(height: 10),
        AppInput(
          label: '新手机号',
          controller: _phoneValue,
          maxLength: InputLimits.phone,
          keyboardType: TextInputType.phone,
        ),
        const SizedBox(height: 10),
        if (step == 1) ...[
          if (_twofaEnabled) ...[
            AppInput(
              label: '2FA验证码（用于身份校验）',
              controller: _phoneTotp,
              maxLength: 6,
              keyboardType: TextInputType.number,
            ),
            const SizedBox(height: 8),
            AppButton(
              text: '验证并继续',
              onPressed: () async {
                await _verifyPhoneTicket();
                refreshPanel();
              },
              isLoading: _busyPhoneTicket,
            ),
          ] else ...[
            AppInput(
              label: '当前登录密码',
              controller: _phonePassword,
              obscureText: true,
              maxLength: InputLimits.password,
            ),
            const SizedBox(height: 8),
            AppButton(
              text: '验证并继续',
              onPressed: () {
                if (_phonePassword.text.trim().isEmpty) {
                  _err('请输入当前登录密码');
                  return;
                }
                _phonePasswordVerified = true;
                refreshPanel();
              },
            ),
          ],
        ],
        if (step == 2) ...[
          AppInput(
            label: '短信验证码',
            controller: _phoneCode,
            maxLength: 12,
            keyboardType: TextInputType.number,
          ),
          const SizedBox(height: 10),
          Row(
            children: [
              Expanded(
                child: AppButton(
                  text: _phoneCd > 0 ? '${_phoneCd}s' : '发送验证码',
                  onPressed: (_phoneCd > 0 || _busyPhoneSend)
                      ? null
                      : () async {
                          await _sendPhoneCode();
                          refreshPanel();
                        },
                  isOutlined: true,
                  isLoading: _busyPhoneSend,
                ),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: AppButton(
                  text: '确认绑定',
                  onPressed: () async {
                    await _confirmPhone();
                    refreshPanel();
                  },
                  isLoading: _busyPhoneConfirm,
                ),
              ),
            ],
          ),
        ],
      ],
    );
  }

  void _err(String m) {
    if (!mounted) return;
    ScaffoldMessenger.of(context)
      ..hideCurrentSnackBar()
      ..showSnackBar(
        SnackBar(content: Text(m), backgroundColor: AppColors.danger),
      );
  }

  void _ok(String m) {
    if (!mounted) return;
    ScaffoldMessenger.of(context)
      ..hideCurrentSnackBar()
      ..showSnackBar(
        SnackBar(content: Text(m), backgroundColor: AppColors.success),
      );
  }

  String _extractSecretFromOtpUrl(String url) {
    try {
      final uri = Uri.parse(url);
      return (uri.queryParameters['secret'] ?? '').trim();
    } catch (_) {
      return '';
    }
  }
}
