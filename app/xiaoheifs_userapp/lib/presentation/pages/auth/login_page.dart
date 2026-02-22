import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:go_router/go_router.dart';
import 'package:package_info_plus/package_info_plus.dart';

import '../../../core/config/api_config.dart';
import '../../../core/constants/app_colors.dart';
import '../../../core/constants/app_strings.dart';
import '../../../core/constants/input_limits.dart';
import '../../../core/network/api_client.dart';
import '../../../core/storage/storage_service.dart';
import '../../../core/update/update_service.dart';
import '../../../core/utils/platform_utils.dart';
import '../../../core/utils/validators.dart';
import '../../providers/auth_provider.dart';
import '../../widgets/common/app_button.dart';
import '../../widgets/common/app_input.dart';
import 'gt4_helper.dart';

/// 登录页面
class LoginPage extends ConsumerStatefulWidget {
  const LoginPage({super.key});

  @override
  ConsumerState<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends ConsumerState<LoginPage> {
  final _accountController = TextEditingController();
  final _phoneController = TextEditingController();
  final _passwordController = TextEditingController();
  final _apiUrlController = TextEditingController();
  final _captchaController = TextEditingController();

  bool _obscurePassword = true;
  String? _errorMessage;
  String _appVersionLabel = 'V1.2.0';
  bool _loadingSettings = false;

  String _loginMode = 'account';
  bool _loginCaptchaEnabled = false;
  String _captchaProvider = 'image';
  String _captchaId = '';
  String _captchaImageBase64 = '';

  bool _geetestLoading = false;
  GeeTestResult _geetestResult = GeeTestResult.empty;

  late final ProviderSubscription<AuthState> _authSubscription;
  static bool _hasCheckedUpdateInThisLaunch = false;
  final UpdateService _updateService = UpdateService();

  @override
  void initState() {
    super.initState();
    _authSubscription = ref.listenManual<AuthState>(authProvider, (
      previous,
      next,
    ) {
      if (next.error != null && next.error != previous?.error) {
        final message = _normalizeError(next.error!);
        _setErrorMessage(message);
        _showErrorSnackBar(message);
      }
    });

    WidgetsBinding.instance.addPostFrameCallback((_) {
      final apiUrl = StorageService.instance.getApiBaseUrl();
      if (apiUrl != null && apiUrl.isNotEmpty) {
        _apiUrlController.text = apiUrl;
      }
      _loadAuthSettings();
    });
    _loadAppVersion();
    _checkAppUpdateIfNeeded();
  }

  @override
  void dispose() {
    _authSubscription.close();
    _accountController.dispose();
    _phoneController.dispose();
    _passwordController.dispose();
    _apiUrlController.dispose();
    _captchaController.dispose();
    super.dispose();
  }

  Future<void> _loadAppVersion() async {
    try {
      final info = await PackageInfo.fromPlatform();
      if (!mounted) return;
      setState(() {
        _appVersionLabel = 'V${info.version}';
      });
    } catch (_) {
      // Keep fallback version label.
    }
  }

  Future<void> _checkAppUpdateIfNeeded() async {
    if (_hasCheckedUpdateInThisLaunch) return;
    _hasCheckedUpdateInThisLaunch = true;

    final platform = getPlatformUtils();
    if (!platform.isAndroid) return;

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

  Future<void> _loadAuthSettings() async {
    setState(() {
      _loadingSettings = true;
    });
    try {
      final settings = await ref.read(authRepositoryProvider).getAuthSettings();
      _loginCaptchaEnabled = settings['login_captcha_enabled'] == true;
      final provider = (settings['captcha_provider'] ?? 'image')
          .toString()
          .toLowerCase();
      _captchaProvider = provider == 'geetest' ? 'geetest' : 'image';
    } catch (_) {
      _loginCaptchaEnabled = false;
      _captchaProvider = 'image';
    } finally {
      if (_loginCaptchaEnabled) {
        await _refreshCaptcha();
      }
      if (mounted) {
        setState(() {
          _loadingSettings = false;
        });
      }
    }
  }

  Future<void> _refreshCaptcha() async {
    if (!_loginCaptchaEnabled) return;
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
      _showErrorSnackBar('验证码尚未就绪，请刷新后重试');
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
          _showErrorSnackBar('已取消验证');
        } else if (result.message.isNotEmpty) {
          _showErrorSnackBar('极验失败: ${result.message}');
        } else if (result.errorCode.isNotEmpty) {
          _showErrorSnackBar('极验失败: ${result.errorCode}');
        } else {
          _showErrorSnackBar('极验未通过，请重试');
        }
      }
    } catch (e) {
      final msg = e.toString().contains('插件未注册')
          ? '极验插件未加载，需完整重启 App'
          : '极验验证失败，请重试';
      _showErrorSnackBar(msg);
    } finally {
      if (mounted) {
        setState(() {
          _geetestLoading = false;
        });
      }
    }
  }

  Future<void> _login() async {
    final username = _loginMode == 'phone'
        ? _phoneController.text.trim()
        : _accountController.text.trim();
    final password = _passwordController.text;

    if (username.isEmpty) {
      _showErrorSnackBar(_loginMode == 'phone' ? '请输入手机号' : '请输入账号');
      return;
    }
    if (_loginMode == 'phone' &&
        !RegExp(r'^[0-9+\-\s]{6,20}$').hasMatch(username)) {
      _showErrorSnackBar('请输入有效手机号');
      return;
    }
    if (username.runes.length > InputLimits.email) {
      _showErrorSnackBar('账号长度不能超过 ${InputLimits.email} 个字符');
      return;
    }
    if (password.isEmpty) {
      _showErrorSnackBar('请输入密码');
      return;
    }
    if (password.runes.length > InputLimits.password) {
      _showErrorSnackBar('密码长度不能超过 ${InputLimits.password} 个字符');
      return;
    }

    if (_loginCaptchaEnabled) {
      if (_captchaProvider == 'geetest') {
        if (!_geetestResult.passed) {
          _showErrorSnackBar('请先完成极验验证');
          return;
        }
      } else {
        if (_captchaController.text.trim().isEmpty) {
          _showErrorSnackBar('请输入验证码');
          return;
        }
      }
    }

    if (mounted) {
      setState(() {
        _errorMessage = null;
      });
    }

    try {
      await ref
          .read(authProvider.notifier)
          .login(
            username: username,
            password: password,
            captchaId: _captchaId,
            captchaCode: _captchaController.text.trim(),
            lotNumber: _geetestResult.lotNumber,
            captchaOutput: _geetestResult.captchaOutput,
            passToken: _geetestResult.passToken,
            genTime: _geetestResult.genTime,
          );

      if (!mounted) return;
      final redirect = GoRouterState.of(
        context,
      ).uri.queryParameters['redirect'];
      if (redirect != null && redirect.isNotEmpty) {
        context.go(Uri.decodeComponent(redirect));
      } else {
        context.go('/console');
      }
    } catch (_) {
      if (_loginCaptchaEnabled) {
        await _refreshCaptcha();
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final authState = ref.watch(authProvider);
    final cs = Theme.of(context).colorScheme;
    final isLight = cs.brightness == Brightness.light;

    return Scaffold(
      backgroundColor: isLight ? cs.surface : AppColors.darkBackground,
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.fromLTRB(24, 30, 24, 20),
          child: Align(
            alignment: Alignment.topCenter,
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 420),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Align(
                    alignment: Alignment.centerRight,
                    child: IconButton(
                      tooltip: 'API 设置',
                      onPressed: _openApiSettingsDialog,
                      icon: const Icon(Icons.settings_outlined),
                    ),
                  ),
                  _buildHeader(),
                  const SizedBox(height: 28),
                  _buildModeTabs(),
                  const SizedBox(height: 20),
                  if (_loginMode == 'account')
                    AppInput(
                      label: '账号',
                      hint: '请输入用户名/邮箱',
                      controller: _accountController,
                      prefixIcon: const Icon(Icons.person_outline),
                      maxLength: InputLimits.email,
                      textInputAction: TextInputAction.next,
                    )
                  else
                    AppInput(
                      label: '手机号',
                      hint: '请输入手机号',
                      controller: _phoneController,
                      prefixIcon: const Icon(Icons.phone_outlined),
                      keyboardType: TextInputType.phone,
                      maxLength: InputLimits.phone,
                      textInputAction: TextInputAction.next,
                    ),
                  const SizedBox(height: 16),
                  AppInput(
                    label: AppStrings.password,
                    hint: AppStrings.inputPassword,
                    controller: _passwordController,
                    obscureText: _obscurePassword,
                    prefixIcon: const Icon(Icons.lock_outline),
                    suffixIcon: Icon(
                      _obscurePassword
                          ? Icons.visibility_outlined
                          : Icons.visibility_off_outlined,
                    ),
                    onSuffixIconPressed: () {
                      setState(() {
                        _obscurePassword = !_obscurePassword;
                      });
                    },
                    maxLength: InputLimits.password,
                    textInputAction: TextInputAction.done,
                    onFieldSubmitted: (_) => _login(),
                  ),
                  if (_loadingSettings)
                    const Padding(
                      padding: EdgeInsets.only(top: 8),
                      child: LinearProgressIndicator(minHeight: 2),
                    ),
                  if (_loginCaptchaEnabled) ...[
                    const SizedBox(height: 12),
                    _buildCaptchaWidget(),
                  ],
                  const SizedBox(height: 8),
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      TextButton(
                        onPressed: () => context.push('/forgot-password'),
                        child: const Text('找回密码'),
                      ),
                      TextButton(
                        onPressed: () => context.push('/register'),
                        child: const Text('注册'),
                      ),
                    ],
                  ),
                  const SizedBox(height: 8),
                  if (_errorMessage != null)
                    Container(
                      padding: const EdgeInsets.all(12),
                      decoration: BoxDecoration(
                        color: AppColors.danger.withValues(alpha: 0.12),
                        borderRadius: BorderRadius.circular(8),
                        border: Border.all(
                          color: AppColors.danger.withValues(alpha: 0.5),
                        ),
                      ),
                      child: Row(
                        children: [
                          Icon(
                            Icons.error_outline,
                            color: AppColors.danger,
                            size: 20,
                          ),
                          const SizedBox(width: 8),
                          Expanded(
                            child: Text(
                              _errorMessage!,
                              style: TextStyle(
                                fontSize: 14,
                                color: AppColors.danger,
                              ),
                            ),
                          ),
                        ],
                      ),
                    ),
                  if (_errorMessage != null) const SizedBox(height: 14),
                  AppButton(
                    text: AppStrings.login,
                    onPressed: _login,
                    isLoading: authState.isLoading,
                  ),
                  const SizedBox(height: 24),
                  Center(
                    child: Text(
                      '${AppStrings.appName} $_appVersionLabel',
                      style: TextStyle(
                        fontSize: 12,
                        color: isLight
                            ? cs.onSurfaceVariant
                            : AppColors.gray400,
                      ),
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

  Widget _buildModeTabs() {
    final cs = Theme.of(context).colorScheme;
    final isLight = cs.brightness == Brightness.light;
    final bg = isLight
        ? const Color(0xFFF2F4F7)
        : AppColors.gray800;

    Widget segment({required String keyValue, required String label}) {
      final selected = _loginMode == keyValue;
      return Expanded(
        child: GestureDetector(
          onTap: selected
              ? null
              : () {
                  setState(() {
                    _loginMode = keyValue;
                    _errorMessage = null;
                  });
                },
          child: AnimatedContainer(
            duration: const Duration(milliseconds: 160),
            curve: Curves.easeOutCubic,
            padding: const EdgeInsets.symmetric(vertical: 10),
            decoration: BoxDecoration(
              color: selected
                  ? (isLight
                        ? const Color(0xFFDCEBFF)
                        : cs.primary.withValues(alpha: 0.95))
                  : Colors.transparent,
              borderRadius: BorderRadius.circular(10),
              border: selected
                  ? Border.all(
                      color: isLight
                          ? const Color(0xFFB9D5FF)
                          : cs.primary.withValues(alpha: 0.9),
                    )
                  : null,
            ),
            child: Text(
              label,
              textAlign: TextAlign.center,
              style: TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w700,
                color: selected
                    ? (isLight ? const Color(0xFF1F4E8C) : cs.onPrimary)
                    : (isLight ? const Color(0xFF3F4752) : AppColors.gray300),
              ),
            ),
          ),
        ),
      );
    }

    return Container(
      padding: const EdgeInsets.all(4),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(
          color: isLight
              ? const Color(0xFFD4DAE3)
              : cs.outlineVariant.withValues(alpha: 0.3),
        ),
      ),
      child: Row(
        children: [
          segment(keyValue: 'account', label: '账号登录'),
          const SizedBox(width: 4),
          segment(keyValue: 'phone', label: '手机号登录'),
        ],
      ),
    );
  }

  Future<void> _openApiSettingsDialog() async {
    final cs = Theme.of(context).colorScheme;
    final isLight = cs.brightness == Brightness.light;
    final current =
        StorageService.instance.getApiBaseUrl() ??
        ApiClient.instance.dio.options.baseUrl;
    final controller = TextEditingController(text: current);

    Future<bool> saveApiUrl(String value) async {
      final raw = value.trim();
      if (raw.isEmpty) {
        await StorageService.instance.setApiBaseUrl(ApiConfig.defaultUrl);
        ApiClient.instance.updateBaseUrl(ApiConfig.defaultUrl);
        return true;
      }

      final urlError = Validators.validateUrl(raw);
      if (urlError != null) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(
              content: Text(urlError),
              backgroundColor: AppColors.danger,
            ),
          );
        }
        return false;
      }

      final normalized = _normalizeApiUrl(raw);
      await StorageService.instance.setApiBaseUrl(normalized);
      ApiClient.instance.updateBaseUrl(normalized);
      return true;
    }

    final saved = await showDialog<bool>(
      context: context,
      builder: (context) => Dialog(
        backgroundColor: Colors.transparent,
        insetPadding: const EdgeInsets.symmetric(horizontal: 20, vertical: 24),
        child: Container(
          padding: const EdgeInsets.fromLTRB(18, 16, 18, 14),
          decoration: BoxDecoration(
            color: isLight ? cs.surface : AppColors.gray900,
            borderRadius: BorderRadius.circular(4),
            border: Border.all(
              color: isLight ? cs.outlineVariant : AppColors.gray700,
            ),
            boxShadow: [
              BoxShadow(
                color: Colors.black.withValues(alpha: isLight ? 0.12 : 0.35),
                blurRadius: 24,
                offset: const Offset(0, 12),
              ),
            ],
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Container(
                    width: 34,
                    height: 34,
                    decoration: BoxDecoration(
                      color: AppColors.primary.withValues(alpha: 0.18),
                      borderRadius: BorderRadius.circular(4),
                    ),
                    child: const Icon(
                      Icons.settings_ethernet_rounded,
                      color: AppColors.primaryLight,
                      size: 20,
                    ),
                  ),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Text(
                      'API 地址设置',
                      style: TextStyle(
                        color: isLight ? cs.onSurface : Colors.white,
                        fontWeight: FontWeight.w700,
                        fontSize: 17,
                      ),
                    ),
                  ),
                  IconButton(
                    tooltip: '关闭',
                    onPressed: () => Navigator.of(context).pop(false),
                    icon: Icon(
                      Icons.close_rounded,
                      color: isLight ? cs.onSurfaceVariant : AppColors.gray300,
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 4),
              Text(
                '修改后会立即刷新认证配置与验证码。',
                style: TextStyle(
                  color: isLight ? cs.onSurfaceVariant : AppColors.gray400,
                  fontSize: 12.5,
                ),
              ),
              const SizedBox(height: 14),
              TextField(
                controller: controller,
                keyboardType: TextInputType.url,
                style: TextStyle(color: isLight ? cs.onSurface : Colors.white),
                decoration: InputDecoration(
                  hintText: ApiConfig.defaultUrl,
                  hintStyle: TextStyle(
                    color: isLight ? cs.onSurfaceVariant : AppColors.gray500,
                  ),
                  prefixIcon: const Icon(Icons.link_rounded),
                  filled: true,
                  fillColor: isLight
                      ? cs.surfaceContainerHighest
                      : AppColors.gray800,
                  helperText: '示例: http://192.168.1.10:8080/api',
                  helperStyle: TextStyle(
                    color: isLight ? cs.onSurfaceVariant : AppColors.gray500,
                  ),
                  border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(4),
                    borderSide: BorderSide.none,
                  ),
                ),
              ),
              const SizedBox(height: 12),
              Row(
                children: [
                  Expanded(
                    child: OutlinedButton(
                      onPressed: () async {
                        final ok = await saveApiUrl('');
                        if (ok && context.mounted) {
                          Navigator.of(context).pop(true);
                        }
                      },
                      style: OutlinedButton.styleFrom(
                        foregroundColor: isLight
                            ? cs.onSurfaceVariant
                            : AppColors.gray300,
                        side: BorderSide(
                          color: isLight
                              ? cs.outlineVariant
                              : AppColors.gray600,
                        ),
                        shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(4),
                        ),
                      ),
                      child: const Text('恢复默认'),
                    ),
                  ),
                  const SizedBox(width: 10),
                  Expanded(
                    child: FilledButton(
                      onPressed: () async {
                        final ok = await saveApiUrl(controller.text);
                        if (ok && context.mounted) {
                          Navigator.of(context).pop(true);
                        }
                      },
                      style: FilledButton.styleFrom(
                        backgroundColor: AppColors.primary,
                        shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(4),
                        ),
                      ),
                      child: const Text('保存'),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );

    if (saved == true) {
      _apiUrlController.text =
          StorageService.instance.getApiBaseUrl() ??
          ApiClient.instance.dio.options.baseUrl;
      await _loadAuthSettings();
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(const SnackBar(content: Text('API 地址已保存')));
      }
    }
  }

  String _normalizeApiUrl(String value) {
    var url = value.trim();
    if (url.endsWith('/')) {
      url = url.substring(0, url.length - 1);
    }
    if (!url.endsWith('/api')) {
      url = '$url/api';
    }
    return url;
  }

  Widget _buildCaptchaWidget() {
    final cs = Theme.of(context).colorScheme;
    final isLight = cs.brightness == Brightness.light;
    if (_captchaProvider == 'geetest') {
      return Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            '行为验证码',
            style: TextStyle(
              color: isLight ? cs.onSurfaceVariant : AppColors.gray300,
            ),
          ),
          const SizedBox(height: 8),
          _buildGeeTestActionButton(),
        ],
      );
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          '图形验证码',
          style: TextStyle(
            color: isLight ? cs.onSurfaceVariant : AppColors.gray300,
          ),
        ),
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
                  border: Border.all(
                    color: isLight ? cs.outlineVariant : AppColors.gray700,
                  ),
                  color: isLight
                      ? cs.surfaceContainerHighest
                      : AppColors.gray900,
                ),
                clipBehavior: Clip.hardEdge,
                child: _captchaImageBase64.isEmpty
                    ? Center(
                        child: Text(
                          '点击刷新',
                          style: TextStyle(
                            color: isLight
                                ? cs.onSurfaceVariant
                                : AppColors.gray400,
                            fontSize: 12,
                          ),
                        ),
                      )
                    : Image.memory(
                        base64Decode(_captchaImageBase64),
                        fit: BoxFit.cover,
                        errorBuilder: (_, __, ___) => Center(
                          child: Text(
                            '点击刷新',
                            style: TextStyle(
                              color: isLight
                                  ? cs.onSurfaceVariant
                                  : AppColors.gray400,
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
    final isLight =
        Theme.of(context).colorScheme.brightness == Brightness.light;
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
            color: (passed ? AppColors.success : AppColors.primary).withValues(
              alpha: isLight ? 0.18 : 0.28,
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
                ? SizedBox(
                    width: 22,
                    height: 22,
                    child: CircularProgressIndicator(
                      strokeWidth: 2.2,
                      backgroundColor: Colors.white.withValues(alpha: 0.28),
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

  void _setErrorMessage(String message) {
    if (!mounted) return;
    setState(() {
      _errorMessage = message;
    });
  }

  void _showErrorSnackBar(String message) {
    if (!mounted) return;
    ScaffoldMessenger.of(context)
      ..hideCurrentSnackBar()
      ..showSnackBar(
        SnackBar(content: Text(message), backgroundColor: AppColors.danger),
      );
  }

  String _normalizeError(String message) {
    final text = message.replaceAll('Exception: ', '').trim();
    final lower = text.toLowerCase();

    if (lower.contains('connection refused')) {
      return '连接被拒绝，请检查 API 地址是否可访问。';
    }
    if (lower.contains('connection error') ||
        lower.contains('failed host lookup')) {
      return '网络连接失败，请检查手机与服务端网络。';
    }
    if (lower.contains('invalid credentials')) {
      return '账号或密码错误';
    }
    return text;
  }

  Widget _buildHeader() {
    final cs = Theme.of(context).colorScheme;
    final isLight = cs.brightness == Brightness.light;
    return Column(
      children: [
        SizedBox(
          width: 100,
          height: 100,
          child: ClipRRect(
            borderRadius: BorderRadius.circular(18),
            child: SvgPicture.asset('assets/app_icon.svg', fit: BoxFit.contain),
          ),
        ),
        const SizedBox(height: 24),
        Text(
          AppStrings.appTitle,
          style: TextStyle(
            fontSize: 28,
            fontWeight: FontWeight.bold,
            color: isLight ? cs.onSurface : AppColors.gray100,
          ),
        ),
        const SizedBox(height: 8),
        Text(
          '请登录您的账户',
          style: TextStyle(
            fontSize: 14,
            color: isLight ? cs.onSurfaceVariant : AppColors.gray400,
          ),
        ),
      ],
    );
  }
}
