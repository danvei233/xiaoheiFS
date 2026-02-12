import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_svg/flutter_svg.dart';
import '../../../core/config/api_config.dart';
import '../../../core/constants/app_colors.dart';
import '../../../core/constants/app_strings.dart';
import '../../../core/network/api_client.dart';
import '../../../core/storage/storage_service.dart';
import '../../../core/utils/validators.dart';
import '../../providers/auth_provider.dart';
import '../../widgets/common/app_button.dart';
import '../../widgets/common/app_input.dart';

/// 登录页面
class LoginPage extends ConsumerStatefulWidget {
  const LoginPage({super.key});

  @override
  ConsumerState<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends ConsumerState<LoginPage> {
  final _usernameController = TextEditingController();
  final _passwordController = TextEditingController();
  final _apiUrlController = TextEditingController();

  bool _obscurePassword = true;
  bool _showAdvanced = false;
  String? _errorMessage;
  late final ProviderSubscription<AuthState> _authSubscription;

  @override
  void initState() {
    super.initState();
    _authSubscription = ref.listenManual<AuthState>(authProvider, (previous, next) {
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
    });
  }

  @override
  void dispose() {
    _authSubscription.close();
    _usernameController.dispose();
    _passwordController.dispose();
    _apiUrlController.dispose();
    super.dispose();
  }

  Future<void> _login() async {
    final username = _usernameController.text.trim();
    final password = _passwordController.text;

    if (username.isEmpty) {
      _showErrorSnackBar('请输入用户名');
      return;
    }
    if (password.isEmpty) {
      _showErrorSnackBar('请输入密码');
      return;
    }

    String? apiUrl;
    if (_showAdvanced) {
      final raw = _apiUrlController.text.trim();
      if (raw.isNotEmpty) {
        final urlError = Validators.validateUrl(raw);
        if (urlError != null) {
          _showErrorSnackBar(urlError);
          return;
        }
        apiUrl = raw;
      }
    }

    if (mounted) {
      setState(() {
        _errorMessage = null;
      });
    }

    try {
      await ref.read(authProvider.notifier).login(
            username: username,
            password: password,
            apiUrl: apiUrl,
          );
    } catch (_) {
      // Error is handled by authProvider listener to avoid duplicate toasts.
    }
  }

  @override
  Widget build(BuildContext context) {
    ref.watch(authProvider);

    final darkTheme = Theme.of(context).copyWith(
      brightness: Brightness.dark,
      colorScheme: ColorScheme.fromSeed(
        seedColor: AppColors.primary,
        brightness: Brightness.dark,
      ),
    );

    return Theme(
      data: darkTheme,
      child: Scaffold(
        backgroundColor: AppColors.darkBackground,
        body: SafeArea(
          child: Center(
            child: SingleChildScrollView(
              padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 20),
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  _buildHeader(),
                  const SizedBox(height: 40),
                  AppInput(
                    label: AppStrings.username,
                    hint: AppStrings.inputUsername,
                    controller: _usernameController,
                    prefixIcon: const Icon(Icons.person_outline),
                    keyboardType: TextInputType.text,
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
                    textInputAction: TextInputAction.done,
                    onFieldSubmitted: (_) => _login(),
                  ),
                  if (_showAdvanced) ...[
                    const SizedBox(height: 16),
                    AppInput(
                      label: AppStrings.apiUrl,
                      hint: ApiConfig.defaultUrl,
                      controller: _apiUrlController,
                      prefixIcon: const Icon(Icons.link_outlined),
                      keyboardType: TextInputType.url,
                    ),
                  ],
                  const SizedBox(height: 8),
                  InkWell(
                    onTap: () {
                      setState(() {
                        _showAdvanced = !_showAdvanced;
                      });
                    },
                    borderRadius: BorderRadius.circular(8),
                    child: Padding(
                      padding: const EdgeInsets.symmetric(vertical: 6),
                      child: Row(
                        children: [
                          Icon(
                            _showAdvanced ? Icons.expand_less : Icons.expand_more,
                            size: 20,
                            color: AppColors.gray400,
                          ),
                          const SizedBox(width: 4),
                          Text(
                            '高级设置',
                            style: TextStyle(fontSize: 14, color: AppColors.gray400),
                          ),
                        ],
                      ),
                    ),
                  ),
                  const SizedBox(height: 18),
                  if (_errorMessage != null)
                    Container(
                      padding: const EdgeInsets.all(12),
                      decoration: BoxDecoration(
                        color: AppColors.danger.withOpacity(0.12),
                        borderRadius: BorderRadius.circular(8),
                        border: Border.all(color: AppColors.danger.withOpacity(0.5)),
                      ),
                      child: Row(
                        children: [
                          Icon(Icons.error_outline, color: AppColors.danger, size: 20),
                          const SizedBox(width: 8),
                          Expanded(
                            child: Text(
                              _errorMessage!,
                              style: TextStyle(fontSize: 14, color: AppColors.danger),
                            ),
                          ),
                        ],
                      ),
                    ),
                  if (_errorMessage != null) const SizedBox(height: 14),
                  AppButton(
                    text: AppStrings.login,
                    onPressed: _login,
                  ),
                  const SizedBox(height: 24),
                  Center(
                    child: Text(
                      '${AppStrings.appName} v1.0.0',
                      style: TextStyle(fontSize: 12, color: AppColors.gray400),
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
        SnackBar(
          content: Text(message),
          backgroundColor: AppColors.danger,
        ),
      );
  }

  String _normalizeError(String message) {
    final text = message.replaceAll('Exception: ', '').trim();
    final lower = text.toLowerCase();
    final currentBaseUrl = ApiClient.instance.dio.options.baseUrl;

    if (lower.contains('connection refused')) {
      return '连接被拒绝：$currentBaseUrl 无法连接。请确认后端监听地址不是 localhost，并且手机可访问该 IP。';
    }
    if (lower.contains('connection error') || lower.contains('failed host lookup')) {
      return '网络连接失败：请检查手机与服务端是否同一网络，当前地址：$currentBaseUrl';
    }
    if (lower.contains('localhost') || lower.contains('127.0.0.1')) {
      return 'Android 设备不能访问手机自身 localhost，请改成电脑局域网 IP。';
    }
    if (lower.contains('invalid credentials')) {
      return '无效登录凭据';
    }
    return text;
  }

  Widget _buildHeader() {
    return Column(
      children: [
        SizedBox(
          width: 100,
          height: 100,
          child: SvgPicture.asset(
            'assets/app_icon.svg',
            fit: BoxFit.contain,
          ),
        ),
        const SizedBox(height: 24),
        Text(
          AppStrings.appTitle,
          style: const TextStyle(
            fontSize: 28,
            fontWeight: FontWeight.bold,
            color: AppColors.gray100,
          ),
        ),
        const SizedBox(height: 8),
        Text(
          '请登录您的账户',
          style: TextStyle(
            fontSize: 14,
            color: AppColors.gray400,
          ),
        ),
      ],
    );
  }
}
