import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:provider/provider.dart';
import 'package:qr_flutter/qr_flutter.dart';

import '../app_state.dart';
import '../models/auth_tokens.dart';
import '../services/admin_auth.dart';
import '../services/api_client.dart';
import '../services/app_storage.dart';
import '../services/update_service.dart';
import 'root_scaffold.dart';

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen>
    with SingleTickerProviderStateMixin {
  final _apiKeyFormKey = GlobalKey<FormState>();
  final _passwordFormKey = GlobalKey<FormState>();

  final _apiUrlController = TextEditingController(text: 'https://api.example.com');
  final _apiKeyController = TextEditingController();
  final _usernameController = TextEditingController(text: 'Admin');

  final _loginUserController = TextEditingController(text: 'admin');
  final _loginPasswordController = TextEditingController();

  late final TabController _tabController;
  bool _saving = false;
  String? _error;
  static bool _hasCheckedUpdateInThisLaunch = false;
  final UpdateService _updateService = UpdateService();
  final AppStorage _storage = AppStorage();

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 2, vsync: this);
    _loadLastSession();
    _checkAppUpdateIfNeeded();
  }

  @override
  void dispose() {
    _apiUrlController.dispose();
    _apiKeyController.dispose();
    _usernameController.dispose();
    _loginUserController.dispose();
    _loginPasswordController.dispose();
    _tabController.dispose();
    super.dispose();
  }

  Future<void> _loadLastSession() async {
    final session = await _storage.loadSession();
    final savedApiUrl = await _storage.loadApiUrl();
    if (savedApiUrl != null && savedApiUrl.isNotEmpty) {
      _apiUrlController.text = savedApiUrl;
    }
    if (session == null) return;
    _apiUrlController.text = session.apiUrl;
    _usernameController.text = session.username;
    if (session.apiKey != null && session.apiKey!.isNotEmpty) {
      _apiKeyController.text = session.apiKey!;
    }
  }

  Future<void> _checkAppUpdateIfNeeded() async {
    if (_hasCheckedUpdateInThisLaunch) return;
    _hasCheckedUpdateInThisLaunch = true;
    if (kIsWeb || defaultTargetPlatform != TargetPlatform.android) return;

    try {
      final info = await PackageInfo.fromPlatform();
      final buildNumber = int.tryParse(info.buildNumber) ?? 0;
      final update = await _updateService.checkForUpdate(
        packageName: info.packageName,
        versionCode: buildNumber,
      );
      if (!mounted || update == null || !update.hasUpdate) return;
      await _showUpdateDialog(update);
    } catch (_) {
      // Keep login flow unaffected if update check fails.
    }
  }

  Future<void> _showUpdateDialog(AppUpdateInfo update) async {
    await showDialog<void>(
      context: context,
      barrierDismissible: !update.forceUpdate,
      builder: (context) {
        return PopScope(
          canPop: !update.forceUpdate,
          child: AlertDialog(
            title: Text('发现新版本 V${update.latestVersion}'),
            content: SingleChildScrollView(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text('版本号: ${update.latestVersionCode}'),
                  const SizedBox(height: 10),
                  const Text('更新说明:'),
                  const SizedBox(height: 6),
                  Text(update.changelog.isEmpty ? '暂无更新说明' : update.changelog),
                ],
              ),
            ),
            actions: [
              if (!update.forceUpdate)
                TextButton(
                  onPressed: () => Navigator.of(context).pop(),
                  child: const Text('稍后'),
                ),
              FilledButton(
                onPressed: () async {
                  await _updateService.openUpdateLink(update.apkUrl);
                },
                child: Text(update.forceUpdate ? '立即更新(必需)' : '立即更新'),
              ),
            ],
          ),
        );
      },
    );
  }

  Future<void> _loginWithApiKey() async {
    if (!_apiKeyFormKey.currentState!.validate()) return;
    final apiUrl = _apiUrlController.text.trim();
    if (apiUrl.isEmpty) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('请先在右上角设置 API 地址')),
      );
      return;
    }
    final ok = await _confirmBeforeLogin(mode: 'API Key');
    if (ok != true) return;
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      final state = context.read<AppState>();
      await state.loginWithApiKey(
        apiUrl: apiUrl,
        apiKey: _apiKeyController.text.trim(),
        username: _usernameController.text.trim().isEmpty
            ? 'Admin'
            : _usernameController.text.trim(),
      );
      if (!mounted) return;
      _goHomeAfterLogin();
    } catch (e) {
      _error = _mapLoginError(e.toString());
    } finally {
      if (mounted) {
        setState(() {
          _saving = false;
        });
      }
    }
  }

  Future<void> _loginWithPassword() async {
    if (!_passwordFormKey.currentState!.validate()) return;
    final apiUrl = _apiUrlController.text.trim();
    if (apiUrl.isEmpty) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('请先在右上角设置 API 地址')),
      );
      return;
    }
    final ok = await _confirmBeforeLogin(mode: '密码');
    if (ok != true) return;
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      final auth = AdminAuthService();
      final login = await auth.login(
        apiUrl: apiUrl,
        username: _loginUserController.text.trim(),
        password: _loginPasswordController.text.trim(),
      );
      final tokens = await _resolveMfaIfNeeded(
        auth: auth,
        apiUrl: apiUrl,
        login: login,
        password: _loginPasswordController.text.trim(),
      );
      if (tokens == null) {
        return;
      }
      String username = _loginUserController.text.trim();
      String? email;
      try {
        final client = ApiClient(baseUrl: apiUrl, token: tokens.accessToken);
        final profile = await client.getJson('/admin/api/v1/profile');
        final profileData = (profile['data'] is Map<String, dynamic>)
            ? profile['data'] as Map<String, dynamic>
            : profile;
        final u = (profileData['username'] as String?)?.trim();
        if (u != null && u.isNotEmpty) {
          username = u;
        }
        email = profileData['email'] as String?;
      } catch (_) {
        // Keep login successful even if profile endpoint is temporarily unavailable.
      }
      await context.read<AppState>().loginWithPassword(
        apiUrl: apiUrl,
        tokens: tokens,
        username: username.isNotEmpty ? username : 'Admin',
        email: email,
      );
      if (!mounted) return;
      _goHomeAfterLogin();
    } on AuthException catch (e) {
      _error = _mapLoginError(e.message);
    } catch (e) {
      _error = _mapLoginError(e.toString());
    } finally {
      if (mounted) {
        setState(() {
          _saving = false;
        });
      }
    }
  }

  String _mapLoginError(String raw) {
    final message = raw.toLowerCase();
    if (message.contains('invalid credentials')) {
      return '无效登录凭据';
    }
    return '登录失败: $raw';
  }

  Future<bool?> _confirmBeforeLogin({required String mode}) {
    return showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('确认登录'),
        content: Text('确认使用$mode方式登录当前后台吗？'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(context, true), child: const Text('确认')),
        ],
      ),
    );
  }

  Future<void> _openApiUrlSettings() async {
    final ctl = TextEditingController(text: _apiUrlController.text.trim());
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('API 地址设置'),
        content: TextField(
          controller: ctl,
          decoration: const InputDecoration(
            labelText: 'API 地址',
            hintText: 'https://api.example.com',
          ),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(context, true), child: const Text('保存')),
        ],
      ),
    );
    if (ok != true) return;
    final url = ctl.text.trim();
    if (url.isEmpty) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('API 地址不能为空')));
      return;
    }
    setState(() => _apiUrlController.text = url);
    await _storage.saveApiUrl(url);
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('API 地址已保存')));
  }

  void _goHomeAfterLogin() {
    Navigator.of(context).pushAndRemoveUntil(
      MaterialPageRoute(builder: (_) => const RootScaffold()),
      (_) => false,
    );
  }

  Future<AuthTokens?> _resolveMfaIfNeeded({
    required AdminAuthService auth,
    required String apiUrl,
    required AdminLoginResult login,
    required String password,
  }) async {
    if (login.mfaUnlocked || (!login.mfaRequired && !login.mfaBindRequired)) {
      return login.tokens;
    }

    var accessToken = login.tokens.accessToken;
    if (login.mfaBindRequired) {
      final setup = await auth.setup2FA(
        apiUrl: apiUrl,
        accessToken: accessToken,
        password: password,
      );
      if (!mounted) return null;
      final bindCode = await _showTotpDialog(
        title: '绑定管理员 2FA',
        message: '请在身份验证器中添加账号后输入 6 位验证码',
        secret: setup.secret,
        otpauthUrl: setup.otpauthUrl,
      );
      if (bindCode == null) return null;
      await auth.confirm2FA(
        apiUrl: apiUrl,
        accessToken: accessToken,
        code: bindCode,
      );
      final unlocked = await auth.unlock2FA(
        apiUrl: apiUrl,
        accessToken: accessToken,
        totpCode: bindCode,
      );
      return unlocked;
    }

    final unlockCode = await _showTotpDialog(
      title: '管理员 2FA 验证',
      message: '请输入身份验证器中的 6 位动态验证码',
    );
    if (unlockCode == null) return null;
    final unlocked = await auth.unlock2FA(
      apiUrl: apiUrl,
      accessToken: accessToken,
      totpCode: unlockCode,
    );
    return unlocked;
  }

  Future<String?> _showTotpDialog({
    required String title,
    required String message,
    String? secret,
    String? otpauthUrl,
  }) async {
    if (!mounted) return null;
    return Navigator.of(context).push<String>(
      MaterialPageRoute(
        fullscreenDialog: true,
        builder: (_) => _TotpInputScreen(
          title: title,
          message: message,
          secret: secret,
          otpauthUrl: otpauthUrl,
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      body: SafeArea(
        child: Container(
          decoration: const BoxDecoration(
            gradient: LinearGradient(
              begin: Alignment.topCenter,
              end: Alignment.bottomCenter,
              colors: [Color(0xFFF8FBFF), Color(0xFFF2F7FF)],
            ),
          ),
          child: Stack(
            children: [
              Positioned(
                top: -120,
                right: -60,
                child: Container(
                  width: 280,
                  height: 280,
                  decoration: const BoxDecoration(
                    shape: BoxShape.circle,
                    color: Color(0x221E88E5),
                  ),
                ),
              ),
              Positioned(
                bottom: -100,
                left: -40,
                child: Container(
                  width: 220,
                  height: 220,
                  decoration: const BoxDecoration(
                    shape: BoxShape.circle,
                    color: Color(0x1A42A5F5),
                  ),
                ),
              ),
              Center(
                child: SingleChildScrollView(
                  padding: const EdgeInsets.all(24),
                  child: ConstrainedBox(
                    constraints: const BoxConstraints(maxWidth: 560),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Row(
                          children: [
                            const _AppLogo(),
                            const SizedBox(width: 12),
                            Expanded(
                              child: Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  Text(
                                    '小黑财务后台管理系统',
                                    style: theme.textTheme.headlineSmall?.copyWith(
                                      fontWeight: FontWeight.w800,
                                      letterSpacing: 0.3,
                                    ),
                                  ),
                                  const SizedBox(height: 4),
                                  Text(
                                    'Secure Cloud Console',
                                    style: theme.textTheme.bodyMedium?.copyWith(
                                      color: theme.colorScheme.onSurfaceVariant,
                                    ),
                                  ),
                                ],
                              ),
                            ),
                            IconButton(
                              tooltip: 'API 地址设置',
                              onPressed: _saving ? null : _openApiUrlSettings,
                              icon: const Icon(Icons.settings_outlined),
                            ),
                          ],
                        ),
                        const SizedBox(height: 24),
                        Text(
                          'API: ${_apiUrlController.text.trim().isEmpty ? '未配置（点右上角设置）' : _apiUrlController.text.trim()}',
                          style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.onSurfaceVariant),
                        ),
                        const SizedBox(height: 16),
                        Container(
                          decoration: BoxDecoration(
                            color: Colors.white.withOpacity(0.72),
                            borderRadius: BorderRadius.circular(12),
                            border: Border.all(color: const Color(0xFFDAE5F3)),
                          ),
                          child: TabBar(
                            controller: _tabController,
                            indicatorSize: TabBarIndicatorSize.tab,
                            indicatorPadding: EdgeInsets.zero,
                            labelPadding: EdgeInsets.zero,
                            indicator: BoxDecoration(
                              color: const Color(0xFF1E88E5),
                              borderRadius: BorderRadius.circular(10),
                            ),
                            labelColor: Colors.white,
                            unselectedLabelColor: const Color(0xFF2A3B52),
                            tabs: const [
                              Tab(
                                child: SizedBox(
                                  height: 42,
                                  child: Center(child: Text('密码登录')),
                                ),
                              ),
                              Tab(
                                child: SizedBox(
                                  height: 42,
                                  child: Center(child: Text('API Key')),
                                ),
                              ),
                            ],
                          ),
                        ),
                        const SizedBox(height: 40),
                        SizedBox(
                          height: 260,
                          child: TabBarView(
                            controller: _tabController,
                            children: [
                              Form(
                                key: _passwordFormKey,
                                child: Padding(
                                  padding: const EdgeInsets.only(top: 10),
                                  child: Column(
                                    children: [
                                    TextFormField(
                                      controller: _loginUserController,
                                      decoration: const InputDecoration(labelText: '用户名'),
                                      validator: (value) {
                                        if (value == null || value.trim().isEmpty) {
                                          return '请输入用户名';
                                        }
                                        return null;
                                      },
                                    ),
                                    const SizedBox(height: 12),
                                    TextFormField(
                                      controller: _loginPasswordController,
                                      obscureText: true,
                                      decoration: const InputDecoration(labelText: '密码'),
                                      validator: (value) {
                                        if (value == null || value.trim().isEmpty) {
                                          return '请输入密码';
                                        }
                                        return null;
                                      },
                                    ),
                                    const Spacer(),
                                    SizedBox(
                                      width: double.infinity,
                                      child: FilledButton(
                                        onPressed: _saving ? null : _loginWithPassword,
                                        style: FilledButton.styleFrom(
                                          padding: const EdgeInsets.symmetric(vertical: 13),
                                        ),
                                        child: _saving
                                            ? const SizedBox(
                                                height: 20,
                                                width: 20,
                                                child: CircularProgressIndicator(strokeWidth: 2),
                                              )
                                            : const Text('登录'),
                                      ),
                                    ),
                                    ],
                                  ),
                                ),
                              ),
                              Form(
                                key: _apiKeyFormKey,
                                child: Padding(
                                  padding: const EdgeInsets.only(top: 10),
                                  child: Column(
                                    children: [
                                    TextFormField(
                                      controller: _apiKeyController,
                                      decoration: const InputDecoration(
                                        labelText: 'API Key',
                                        hintText: 'ak_live_xxx',
                                      ),
                                      validator: (value) {
                                        if (value == null || value.trim().isEmpty) {
                                          return '请输入 API Key';
                                        }
                                        return null;
                                      },
                                    ),
                                    const SizedBox(height: 12),
                                    TextFormField(
                                      controller: _usernameController,
                                      decoration: const InputDecoration(
                                        labelText: '展示名称',
                                        hintText: '管理员',
                                      ),
                                    ),
                                    const Spacer(),
                                    SizedBox(
                                      width: double.infinity,
                                      child: FilledButton(
                                        onPressed: _saving ? null : _loginWithApiKey,
                                        style: FilledButton.styleFrom(
                                          padding: const EdgeInsets.symmetric(vertical: 13),
                                        ),
                                        child: _saving
                                            ? const SizedBox(
                                                height: 20,
                                                width: 20,
                                                child: CircularProgressIndicator(strokeWidth: 2),
                                              )
                                            : const Text('登录'),
                                      ),
                                    ),
                                    ],
                                  ),
                                ),
                              ),
                            ],
                          ),
                        ),
                        if (_error != null) ...[
                          const SizedBox(height: 12),
                          Text(
                            _error!,
                            style: theme.textTheme.bodyMedium?.copyWith(
                              color: Colors.redAccent,
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                        ],
                      ],
                    ),
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _AppLogo extends StatelessWidget {
  const _AppLogo();

  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(12),
      child: SvgPicture.asset(
        'assets/app.svg',
        width: 56,
        height: 56,
        fit: BoxFit.contain,
      ),
    );
  }
}

class _TotpInputScreen extends StatefulWidget {
  final String title;
  final String message;
  final String? secret;
  final String? otpauthUrl;

  const _TotpInputScreen({
    required this.title,
    required this.message,
    this.secret,
    this.otpauthUrl,
  });

  @override
  State<_TotpInputScreen> createState() => _TotpInputScreenState();
}

class _TotpInputScreenState extends State<_TotpInputScreen> {
  final TextEditingController _controller = TextEditingController();
  String? _error;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _submit() {
    final code = _controller.text.trim();
    if (!RegExp(r'^\d{6}$').hasMatch(code)) {
      setState(() => _error = '请输入 6 位数字验证码');
      return;
    }
    Navigator.of(context).pop(code);
  }

  @override
  Widget build(BuildContext context) {
    final hasSecret = widget.secret != null && widget.secret!.isNotEmpty;
    final hasOtpUrl =
        widget.otpauthUrl != null && widget.otpauthUrl!.isNotEmpty;
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.title),
        leading: IconButton(
          icon: const Icon(Icons.close),
          onPressed: () => Navigator.of(context).pop(),
        ),
      ),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(widget.message),
              if (hasSecret) ...[
                const SizedBox(height: 16),
                if (hasOtpUrl) ...[
                  const Text('扫码绑定'),
                  const SizedBox(height: 8),
                  Center(
                    child: Container(
                      color: Colors.white,
                      padding: const EdgeInsets.all(8),
                      child: QrImageView(
                        data: widget.otpauthUrl!,
                        version: QrVersions.auto,
                        size: 200,
                        gapless: false,
                      ),
                    ),
                  ),
                  const SizedBox(height: 12),
                ],
                const Text('手动密钥'),
                const SizedBox(height: 6),
                SelectableText(widget.secret!),
                TextButton(
                  onPressed: () async {
                    await Clipboard.setData(
                      ClipboardData(text: widget.secret!),
                    );
                    if (!mounted) return;
                    ScaffoldMessenger.maybeOf(
                      context,
                    )?.showSnackBar(const SnackBar(content: Text('密钥已复制')));
                  },
                  child: const Text('复制密钥'),
                ),
              ],
              const SizedBox(height: 12),
              TextField(
                controller: _controller,
                keyboardType: TextInputType.number,
                maxLength: 6,
                autofocus: true,
                decoration: InputDecoration(
                  labelText: '2FA 验证码',
                  errorText: _error,
                ),
                onSubmitted: (_) => _submit(),
              ),
              const SizedBox(height: 12),
              SizedBox(
                width: double.infinity,
                child: FilledButton(
                  onPressed: _submit,
                  child: const Text('确认'),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
