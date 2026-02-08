import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/config/api_config.dart';
import '../../../core/constants/app_colors.dart';
import '../../../core/constants/app_strings.dart';
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
  final _formKey = GlobalKey<FormState>();
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
    if (!_formKey.currentState!.validate()) {
      return;
    }

    if (mounted) {
      setState(() {
        _errorMessage = null;
      });
    }

    try {
      await ref.read(authProvider.notifier).login(
            username: _usernameController.text.trim(),
            password: _passwordController.text,
            apiUrl: _showAdvanced ? _apiUrlController.text.trim() : null,
          );
    } catch (e) {
      final message = _normalizeError(e.toString());
      _setErrorMessage(message);
      _showErrorSnackBar(message);
    }
  }

  @override
  Widget build(BuildContext context) {
    final authState = ref.watch(authProvider);
    final isLoading = authState.isLoading;
    final screenWidth = MediaQuery.of(context).size.width;
    final isNarrow = screenWidth < 600;
    final formMaxWidth = isNarrow ? double.infinity : 400.0;

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
              padding: const EdgeInsets.all(24),
              child: ConstrainedBox(
                constraints: BoxConstraints(maxWidth: formMaxWidth),
                child: Form(
                  key: _formKey,
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      _buildHeader(),
                    const SizedBox(height: 48),
                    AppInput(
                      label: AppStrings.username,
                      hint: AppStrings.inputUsername,
                      controller: _usernameController,
                      prefixIcon: const Icon(Icons.person_outline),
                      keyboardType: TextInputType.text,
                      textInputAction: TextInputAction.next,
                      validator: Validators.validateUsername,
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
                      validator: Validators.validatePassword,
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
                        validator: Validators.validateUrl,
                      ),
                    ],
                    const SizedBox(height: 8),
                    GestureDetector(
                      onTap: () {
                        setState(() {
                          _showAdvanced = !_showAdvanced;
                        });
                      },
                      child: Row(
                        children: [
                          Icon(
                            _showAdvanced
                                ? Icons.expand_less
                                : Icons.expand_more,
                            size: 20,
                            color: AppColors.gray400,
                          ),
                          const SizedBox(width: 4),
                          Text(
                            '高级设置',
                            style: TextStyle(
                              fontSize: 14,
                              color: AppColors.gray400,
                            ),
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(height: 24),
                    if (_errorMessage != null)
                      Container(
                        padding: const EdgeInsets.all(12),
                        decoration: BoxDecoration(
                          color: AppColors.danger.withOpacity(0.1),
                          borderRadius: BorderRadius.circular(8),
                          border: Border.all(color: AppColors.danger.withOpacity(0.3)),
                        ),
                        child: Row(
                          children: [
                            Icon(Icons.error_outline, color: AppColors.danger, size: 20),
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
                    if (_errorMessage != null) const SizedBox(height: 16),
                    AppButton(
                      text: AppStrings.login,
                      onPressed: _login,
                      isLoading: isLoading,
                    ),
                    const SizedBox(height: 24),
                      Center(
                        child: Text(
                          '${AppStrings.appName} v1.0.0',
                          style: TextStyle(
                            fontSize: 12,
                            color: AppColors.gray400,
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
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
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(message),
        backgroundColor: AppColors.danger,
      ),
    );
  }

  String _normalizeError(String message) {
    return message.replaceAll('Exception: ', '');
  }

  Widget _buildHeader() {
    return Column(
      children: [
        Container(
          width: 80,
          height: 80,
          decoration: BoxDecoration(
            color: AppColors.primary,
            borderRadius: BorderRadius.circular(16),
          ),
          child: const Icon(
            Icons.cloud_outlined,
            size: 48,
            color: Colors.white,
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



